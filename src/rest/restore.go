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
	hfl "github.com/jeanfrancoisgratton/helperFunctions/v3/logging"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v3/terminalfx"
	"github.com/ulikunitz/xz"

	"dvol/types"
)

// RestoreVolume uploads a tar(.gz|.xz|.txz) archive to /containers/{id}/archive (upload)
// NOTE: requires: import "context" and "time"
func RestoreVolume(client *http.Client, base, version, volumeName, archivePath string) *ce.CustomError {
	image := types.Image
	if !types.Quiet {
		fmt.Println(hftx.InProgressGlyph(fmt.Sprintf("Restoring %s from %s", hftx.Blue(volumeName), hftx.Blue(archivePath))))
		if types.HandshakeTimeout != 60 {
			fmt.Println(hftx.NoteGlyph(fmt.Sprintf("HTTP handshake (fast-fail) timeout set to %d seconds", types.HandshakeTimeout)))
		}
		if types.SessionTimeout != 60 {
			fmt.Println(hftx.NoteGlyph(fmt.Sprintf("HTTP session timeout set to %d minutes", types.SessionTimeout)))
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
			attachedResult = hftx.InfoGlyph("No running containers were attached to the volume to be backed up")
		} else {
			attachedResult = hftx.InfoGlyph(fmt.Sprintf("%d running containers are attached to the volume. They will be restarted after the backup",
				len(attachedContainers)))
		}
		fmt.Println(attachedResult)
	}
	if err != nil {
		return err
	}
	if len(attachedContainers) > 0 {
		if !types.Quiet {
			fmt.Println(hftx.InProgressGlyph(fmt.Sprintf("Stopping the %d container(s) attached to %s", len(attachedContainers), volumeName)))
		}
		if e := stopContainers(client, base, version, attachedContainers); e != nil {
			return e
		}
	}

	// Destroy & recreate the volume before restore
	if !types.Quiet {
		fmt.Println(hftx.InProgressGlyph(fmt.Sprintf("Deleting the volume %s before creating a brand-new restored volume", volumeName)))
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
		e := ce.CustomError{Title: "Unable to open archive", Message: oerr.Error(), Code: 701}
		hfl.Errorf(e.ErrorNoColor())
		return &e
	}
	defer file.Close()

	var body io.Reader = file
	switch {
	case strings.HasSuffix(archivePath, ".tar.gz") || strings.HasSuffix(archivePath, ".tgz"):
		gr, gerr := gzip.NewReader(file)
		if gerr != nil {
			e := ce.CustomError{Title: "Unable to create gzip reader", Message: gerr.Error(), Code: 702}
			hfl.Errorf(e.ErrorNoColor())
			return &e
		}
		defer gr.Close()
		body = gr
	case strings.HasSuffix(archivePath, ".xz") || strings.HasSuffix(archivePath, ".txz"):
		// If your project includes .xz support here, wire it exactly as you had:
		xzr, xerr := xz.NewReader(file)
		if xerr != nil {
			e := ce.CustomError{Title: "Unable to create xz reader", Message: xerr.Error(), Code: 703}
			hfl.Errorf(e.ErrorNoColor())
			return &e
		}
		body = xzr
	}

	// Build streaming PUT request — Content-Type must be tar; apply overall stream timeout
	putURL := APIPath(base, version, "containers", containerID, "archive") + "?path=/data"
	req, rerr := http.NewRequest(http.MethodPut, putURL, body)
	if rerr != nil {
		e := ce.CustomError{Title: "Unable to build restore http request", Message: rerr.Error(), Code: 801}
		hfl.Errorf(e.ErrorNoColor())
		return &e
	}
	req.Header.Set("Content-Type", "application/x-tar")

	sctx, cancel := context.WithTimeout(context.Background(), time.Duration(types.SessionTimeout)*time.Minute)
	defer cancel()
	req = req.WithContext(sctx)

	resp, derr := client.Do(req)
	if derr != nil {
		e := ce.CustomError{Title: "Unable to perform restore http request", Message: derr.Error(), Code: 802}
		hfl.Errorf(e.ErrorNoColor())
		return &e
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, raerr := io.ReadAll(resp.Body)
		if raerr != nil {
			err := ce.CustomError{Title: "Unable to read restore http response",
				Message: fmt.Sprintf("HTTP %d (and read error: %s)", resp.StatusCode, raerr.Error()), Code: 803}
			hfl.Errorf(err.ErrorNoColor())
			return &err
		}
		err := ce.CustomError{Title: "Restore failed",
			Message: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(b)), Code: 803}
		hfl.Errorf(err.ErrorNoColor())
		return &err
	}

	if !types.NoCleanup {
		if e := stopAndRemoveContainer(client, base, version, containerID); e != nil {
			return e
		}
	}

	return startContainers(client, base, version, attachedContainers)
}
