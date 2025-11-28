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
	ce "github.com/jeanfrancoisgratton/customError/v3"
	hfl "github.com/jeanfrancoisgratton/helperFunctions/v4/logging"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v4/terminalfx"

	"dvol/types"
)

// ensureImageExists pulls image if missing.
// Timeouts policy:
//   - Fast-fail (types.HandshakeTimeout) for the quick inspect call
//   - Streaming overall cap (types.SessionTimeout) for the pull request itself
func ensureImageExists(client *http.Client, base, version, image string) *ce.CustomError {
	// Try inspect
	inspectURL := APIPath(base, version, "images", "json") + "?all=true"
	{
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(types.HandshakeTimeout)*time.Second)
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
							if !types.Quiet {
								fmt.Println(hftx.InfoSign(fmt.Sprintf("Image %s is already present, no need to pull it", image)))
							}
							return nil
						}
					}
				}
			}
		} else if resp != nil {
			resp.Body.Close()
		}
	}

	// The image is not on the daemon, we need to pull it
	// Pull (can be long): use overall stream timeout (types.SessionTimeout)
	pullURL := APIPath(base, version, "images", "create") + "?fromImage=" + image
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(types.SessionTimeout)*time.Minute)
	defer cancel()
	req, err := http.NewRequest(http.MethodPost, pullURL, nil)
	if err != nil {
		title := "Error building the image pull request"
		message := err.Error()
		if !types.Quiet {
			fmt.Println(hftx.SkullBonesSign(fmt.Sprintf("%s: %s", title, message)))
		}
		perr := ce.CustomError{Title: title, Message: message, Code: 200}
		hfl.Errorf(perr.ErrorNoColor())
		return &perr
	}
	req = req.WithContext(ctx)

	resp, err := client.Do(req)
	if err != nil {
		title := "Error when pulling the image"
		message := err.Error()
		if !types.Quiet {
			fmt.Println(hftx.SkullBonesSign(fmt.Sprintf("%s: %s", title, message)))
		}
		perr := ce.CustomError{Title: title, Message: message, Code: 201}
		hfl.Errorf(perr.ErrorNoColor())
		return &perr
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		title := "Error pulling the image"
		message := fmt.Sprintf("HTTP code: %d : %s", resp.StatusCode, string(body))
		perr := ce.CustomError{Title: title, Message: message, Code: 201}
		hfl.Errorf(perr.ErrorNoColor())
		if !types.Quiet {
			fmt.Println(hftx.SkullBonesSign(fmt.Sprintf("%s: %s", title, message)))
		}

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
			title := "Error decoding the image pull stream"
			message := err.Error()
			perr := ce.CustomError{Title: title, Message: message, Code: 202}
			hfl.Errorf(perr.ErrorNoColor())
			if !types.Quiet {
				fmt.Println(hftx.SkullBonesSign(fmt.Sprintf("%s: %s", title, message)))
			}
			return &perr
		}
		// Keep it simple; respect quiet mode externally if desired.
		if jm.Status != "" {
			hfl.Infof("pull: %s", jm.Status)
		}
	}
	return nil
}
