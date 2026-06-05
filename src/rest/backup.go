// dvol
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Updated: 2025/08/09
// Original filename: src/rest/backup.go

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

	"dvol/types"

	ce "github.com/jeanfrancoisgratton/customError/v3"
	hflog "github.com/jeanfrancoisgratton/helperFunctions/v5/logging"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// BackupVolume streams /containers/{id}/archive (download) to a local file (optionally gzipped).
// We request ?path=/ so the tar stream contains entries rooted at "/" (e.g. "data/pg_data/...").
// This matches the restore side which also PUTs to ?path=/, causing "data/" to land at /data/.
func BackupVolume(client *http.Client, base, version, volumeName, archivePath string) *ce.CustomError {
	image := types.Image

	if !types.Quiet {
		fmt.Println(hftx.InProgressSign(fmt.Sprintf("Backing up %s to %s", volumeName, archivePath)))
		if types.HandshakeTimeout != 60 {
			fmt.Println(hftx.NoteSign(fmt.Sprintf("HTTP handshake (fast-fail) timeout set to %d seconds", types.HandshakeTimeout)))
		}
		if types.SessionTimeout != 60 {
			fmt.Println(hftx.NoteSign(fmt.Sprintf("HTTP session timeout set to %d minutes", types.SessionTimeout)))
		}
	}
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
			fmt.Println(hftx.InProgressSign(fmt.Sprintf("Temporarily stopping the running container(s) using %s", volumeName)))
		}
		if e := stopContainers(client, base, version, attachedContainers); e != nil {
			return e
		}
	}

	// Create a temp container bound to the volume and start it
	if !types.Quiet {
		fmt.Println(hftx.InProgressSign(fmt.Sprintf("Creating and starting a temp Alpine container to temporarily attach the volume %s", volumeName)))
	}
	containerID, cerr := createTempContainer(client, base, version, image, volumeName)
	if cerr != nil {
		return cerr
	}
	if e := startContainer(client, base, version, containerID); e != nil {
		return e
	}

	// Request path=/ so entries in the tar stream are rooted at "/" (i.e. "data/...").
	// On restore we also PUT to path=/ so "data/" extracts cleanly to /data/ inside the volume.
	copyURL := APIPath(base, version, "containers", containerID, "archive") + "?path=/"
	req, rerr := http.NewRequest(http.MethodGet, copyURL, nil)
	if rerr != nil {
		e := ce.CustomError{Title: "Failed to build archive request", Message: rerr.Error(), Code: 401}
		hflog.Errorf(e.ErrorNoColor())
		return &e
	}

	// Apply overall stream timeout for the whole backup transfer
	sctx, cancel := context.WithTimeout(context.Background(), time.Duration(types.SessionTimeout)*time.Minute)
	defer cancel()
	req = req.WithContext(sctx)

	resp, doErr := client.Do(req)
	if doErr != nil {
		e := ce.CustomError{Title: "Failed to retrieve archive path from container", Message: doErr.Error(), Code: 401}
		hflog.Errorf(e.ErrorNoColor())
		return &e
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		e := ce.CustomError{Title: "Error retrieving archive", Message: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)), Code: 402}
		hflog.Errorf(e.ErrorNoColor())
		return &e
	}

	// Open destination file
	outf, ferr := os.Create(archivePath)
	if ferr != nil {
		e := ce.CustomError{Title: "Failed to create archive file", Message: ferr.Error(), Code: 403}
		hflog.Errorf(e.ErrorNoColor())
		return &e
	}
	defer outf.Close()

	// If .tar.gz/.tgz, wrap the destination in gzip; DO NOT re-tar — the daemon already sends a tar stream
	var writer io.Writer = outf
	var gz *gzip.Writer
	if strings.HasSuffix(archivePath, ".tar.gz") || strings.HasSuffix(archivePath, ".tgz") {
		gz = gzip.NewWriter(outf)
		defer gz.Close()
		writer = gz
	}

	// Directly copy the daemon's tar stream to the destination (raw or gz-wrapped)
	if _, werr := io.Copy(writer, resp.Body); werr != nil {
		e := ce.CustomError{Title: "Failed to write data to archive", Message: werr.Error(), Code: 405}
		hflog.Errorf(e.ErrorNoColor())
		return &e
	}

	if !types.NoCleanup {
		if !types.Quiet {
			fmt.Println(hftx.InProgressSign("Cleanup: stoping and removing the temp Alpine container"))
		}
		if e := stopAndRemoveContainer(client, base, version, containerID); e != nil {
			return e
		}
	}

	// restart the containers that were stopped before the backup
	if !types.Quiet {
		fmt.Println(hftx.InProgressSign("Restarting the containers that were stopped before the backup"))
	}
	if scerr := startContainers(client, base, version, attachedContainers); scerr != nil {
		return scerr
	}
	if !types.Quiet {
		fmt.Printf("%s volume %s backed up as %s\n", hftx.EnabledSign(""), hftx.Blue(volumeName), hftx.Blue(archivePath))
	}
	return nil
}
