// dvol
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/08/14 20:06
// Original filename: src/rest/images.go

package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/docker/docker/pkg/jsonmessage"
	ce "github.com/jeanfrancoisgratton/customError/v2"
	hfl "github.com/jeanfrancoisgratton/helperFunctions/v2/logging"

	"dvol/types"
)

// ensureImageExists pulls image if missing.
// Timeouts policy:
//   - Fast-fail (types.FastfailTimeout) for the quick inspect call
//   - Streaming overall cap (types.Timeout) for the pull request itself
func ensureImageExists(client *http.Client, base, version, image string) *ce.CustomError {
	// Try inspect
	inspectURL := APIPath(base, version, "images", "json") + "?all=true"
	{
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(types.FastfailTimeout)*time.Second)
		defer cancel()
		req, _ := http.NewRequest(http.MethodGet, inspectURL, nil)
		req = req.WithContext(ctx)

		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			var imgs []struct {
				RepoTags []string `json:"RepoTags"`
			}
			if derr := json.NewDecoder(resp.Body).Decode(&imgs); derr == nil {
				for _, im := range imgs {
					for _, t := range im.RepoTags {
						if t == image {
							return nil
						}
					}
				}
			}
		} else if resp != nil {
			resp.Body.Close()
		}
	}

	// Pull (can be long): use overall stream timeout (types.Timeout)
	pullURL := APIPath(base, version, "images", "create") + "?fromImage=" + image
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(types.Timeout)*time.Minute)
	defer cancel()
	req, err := http.NewRequest(http.MethodPost, pullURL, nil)
	if err != nil {
		perr := ce.CustomError{Title: "Error building image pull request", Message: err.Error(), Code: 200}
		hfl.Errorf(perr.ErrorNoColor())
		return &perr
	}
	req = req.WithContext(ctx)

	resp, err := client.Do(req)
	if err != nil {
		perr := ce.CustomError{Title: "Error when pulling image", Message: err.Error(), Code: 201}
		hfl.Errorf(perr.ErrorNoColor())
		return &perr
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		perr := ce.CustomError{Title: "Error when pulling image",
			Message: fmt.Sprintf("HTTP code: %d : %s", resp.StatusCode, string(body)), Code: 201}
		hfl.Errorf(perr.ErrorNoColor())
		return &perr
	}

	// Render JSON progress (TTY-aware optional)
	termFd := os.Stdout.Fd()
	_, _ = termFd, jsonmessage.DisplayJSONMessagesStream // avoid unused if quiet
	dec := json.NewDecoder(resp.Body)
	for {
		var jm jsonmessage.JSONMessage
		if err := dec.Decode(&jm); err != nil {
			if err == io.EOF {
				break
			}
			perr := ce.CustomError{Title: "Error decoding image pull stream", Message: err.Error(), Code: 202}
			hfl.Errorf(perr.ErrorNoColor())
			return &perr
		}
		// Keep it simple; respect quiet mode externally if desired.
		if jm.Status != "" {
			hfl.Infof("pull: %s", jm.Status)
		}
	}
	return nil
}
