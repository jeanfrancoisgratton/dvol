// dvol
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Updated: 2025/08/02
// Original filename: src/rest/restore.go

package rest

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	ce "github.com/jeanfrancoisgratton/customError/v3"
	hfl "github.com/jeanfrancoisgratton/helperFunctions/v5/logging"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
	"github.com/ulikunitz/xz"

	"dvol/types"
)

// RestoreVolume uploads a tar(.gz|.xz|.txz) archive to /containers/{id}/archive (upload).
// The archive was created by BackupVolume with ?path=/, so entries are rooted at "/"
// (e.g. "data/pg_data/..."). We PUT to ?path=/ so "data/" extracts to /data/ inside the volume,
// matching the original mount point exactly.
func RestoreVolume(client *http.Client, base, version, volumeName, archivePath string) *ce.CustomError {
	image := types.Image
	if !types.Quiet {
		fmt.Println(hftx.InProgressSign(fmt.Sprintf("Restoring %s from %s", hftx.Blue(volumeName), hftx.Blue(archivePath))))
		if types.HandshakeTimeout != 60 {
			fmt.Println(hftx.NoteSign(fmt.Sprintf("HTTP handshake (fast-fail) timeout set to %d seconds", types.HandshakeTimeout)))
		}
		if types.SessionTimeout != 60 {
			fmt.Println(hftx.NoteSign(fmt.Sprintf("HTTP session timeout set to %d minutes", types.SessionTimeout)))
		}
	}
	if err := ensureImageExists(client, base, version, image); err != nil {
		return err
	}

	// Stop containers using the volume
	attachedContainers, err := getContainersUsingVolume(client, base, version, volumeName)
	if !types.Quiet {
		attachedResult := ""
		if len(attachedContainers) == 0 {
			attachedResult = hftx.InfoSign("No running containers were attached to the volume to be backed up")
		} else {
			attachedResult = hftx.InfoSign(fmt.Sprintf("%d running containers are attached to the volume. They will be restarted after the backup",
				len(attachedContainers)))
		}
		fmt.Println(attachedResult)
	}
	if err != nil {
		return err
	}
	if len(attachedContainers) > 0 {
		if !types.Quiet {
			fmt.Println(hftx.InProgressSign(fmt.Sprintf("Stopping the %d container(s) attached to %s", len(attachedContainers), volumeName)))
		}
		if e := stopContainers(client, base, version, attachedContainers); e != nil {
			return e
		}
	}

	// Destroy & recreate the volume before restore
	if !types.Quiet {
		fmt.Println(hftx.InProgressSign(fmt.Sprintf("Deleting the volume %s before creating a brand-new restored volume", volumeName)))
	}
	if e := deleteAndRecreateVolume(client, base, version, volumeName); e != nil {
		return e
	}

	// Create & start temp container
	containerID, cerr := createTempContainer(client, base, version, image, volumeName)
	if cerr != nil {
		return cerr
	}
	if e := startContainer(client, base, version, containerID); e != nil {
		return e
	}

	// Open source archive; if compressed, decompress to expose a raw tar stream to the PUT
	file, oerr := os.Open(archivePath)
	if oerr != nil {
		title := "Unable to open archive"
		message := oerr.Error()
		if !types.Quiet {
			fmt.Println(hftx.SkullBonesSign(fmt.Sprintf("%s %s", title, message)))
		}
		e := ce.CustomError{Title: title, Message: message, Code: 701}
		hfl.Errorf(e.ErrorNoColor())
		return &e
	}
	defer file.Close()

	var body io.Reader = file
	switch {
	case strings.HasSuffix(archivePath, ".tar.gz") || strings.HasSuffix(archivePath, ".tgz"):
		gr, gerr := gzip.NewReader(file)
		if gerr != nil {
			title := "Unable to create the gzip reader"
			message := gerr.Error()
			e := ce.CustomError{Title: title, Message: message, Code: 702}
			hfl.Errorf(e.ErrorNoColor())
			if !types.Quiet {
				fmt.Println(hftx.SkullBonesSign(fmt.Sprintf("%s %s", title, message)))
			}
			return &e
		}
		defer gr.Close()
		body = gr
	case strings.HasSuffix(archivePath, ".xz") || strings.HasSuffix(archivePath, ".txz"):
		xzr, xerr := xz.NewReader(file)
		if xerr != nil {
			title := "Unable to create the xz reader"
			message := xerr.Error()
			e := ce.CustomError{Title: title, Message: message, Code: 703}
			hfl.Errorf(e.ErrorNoColor())
			if !types.Quiet {
				fmt.Println(hftx.SkullBonesSign(fmt.Sprintf("%s %s", title, message)))
			}
			return &e
		}
		body = xzr
	}

	// PUT to path=/ so "data/" entries from the archive land at /data/ inside the volume.
	// The archive was created with path=/ (entries rooted at "/"), so this is a symmetric operation.
	putURL := APIPath(base, version, "containers", containerID, "archive") + "?path=/"
	req, rerr := http.NewRequest(http.MethodPut, putURL, body)
	if rerr != nil {
		title := "Unable to build the restore http request"
		message := rerr.Error()
		e := ce.CustomError{Title: title, Message: message, Code: 801}
		hfl.Errorf(e.ErrorNoColor())
		if !types.Quiet {
			fmt.Println(hftx.SkullBonesSign(fmt.Sprintf("%s %s", title, message)))
		}
		return &e
	}
	req.Header.Set("Content-Type", "application/x-tar")

	sctx, cancel := context.WithTimeout(context.Background(), time.Duration(types.SessionTimeout)*time.Minute)
	defer cancel()
	req = req.WithContext(sctx)

	resp, derr := client.Do(req)
	if derr != nil {
		title := "Unable to perform the restore http request"
		message := derr.Error()
		e := ce.CustomError{Title: title, Message: message, Code: 802}
		hfl.Errorf(e.ErrorNoColor())
		if !types.Quiet {
			fmt.Println(hftx.SkullBonesSign(fmt.Sprintf("%s %s", title, message)))
		}
		return &e
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, raerr := io.ReadAll(resp.Body)
		if raerr != nil {
			title := "Unable to read the restore http response"
			message := fmt.Sprintf("HTTP %d (and read error: %s)", resp.StatusCode, raerr.Error())
			err := ce.CustomError{Title: title, Message: message, Code: 803}
			hfl.Errorf(err.ErrorNoColor())
			if !types.Quiet {
				fmt.Println(hftx.SkullBonesSign(fmt.Sprintf("%s %s", title, message)))
			}
			return &err
		}
		title := "Restore failed"
		message := fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(b))
		err := ce.CustomError{Title: title, Message: message, Code: 803}
		hfl.Errorf(err.ErrorNoColor())
		if !types.Quiet {
			fmt.Println(hftx.SkullBonesSign(fmt.Sprintf("%s %s", title, message)))
		}
		return &err
	}

	if !types.NoCleanup {
		if e := stopAndRemoveContainer(client, base, version, containerID); e != nil {
			return e
		}
	}

	return startContainers(client, base, version, attachedContainers)
}
