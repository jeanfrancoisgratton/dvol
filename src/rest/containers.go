// dvol
// rest/containers.go
// Uses Docker CLI's progress renderer for image pulls (with AUX digest), plus container/volume helpers.
// rest/containers.go
// dvol
// src/rest/containers.go

package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	ce "github.com/jeanfrancoisgratton/customError/v2"
	hf "github.com/jeanfrancoisgratton/helperFunctions/v2"
	hfl "github.com/jeanfrancoisgratton/helperFunctions/v2/logging"

	"dvol/types"
)

// Returns IDs of RUNNING containers that have a volume mount matching volumeName.
func getContainersUsingVolume(client *http.Client, base, version, volumeName string) ([]string, *ce.CustomError) {
	url := APIPath(base, version, "containers", "json") + "?all=true"
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(types.FastfailTimeout)*time.Second)
	defer cancel()
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req = req.WithContext(ctx)

	resp, err := client.Do(req)
	if err != nil {
		gerr := ce.CustomError{Title: "Error listing the containers using the volume", Message: err.Error(), Code: 301}
		hfl.Errorf(gerr.ErrorNoColor())
		return nil, &gerr
	}
	defer resp.Body.Close()

	var containers []struct {
		ID     string `json:"Id"`
		Mounts []struct {
			Type   string `json:"Type"`
			Source string `json:"Source"`
			Name   string `json:"Name"`
		} `json:"Mounts"`
		State string `json:"State"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&containers); err != nil {
		perr := ce.CustomError{Title: "Error decoding containers list", Message: err.Error(), Code: 302}
		hfl.Errorf(perr.ErrorNoColor())
		return nil, &perr
	}

	result := []string{}
	for _, c := range containers {
		if c.State != "running" {
			continue
		}
		for _, m := range c.Mounts {
			if m.Type == "volume" && (m.Name == volumeName || m.Source == volumeName) {
				result = append(result, c.ID)
				break
			}
		}
	}
	return result, nil
}

func stopContainers(client *http.Client, base, version string, ids []string) *ce.CustomError {
	for _, id := range ids {
		url := APIPath(base, version, "containers", id, "stop")
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(types.FastfailTimeout)*time.Second)
		defer cancel()
		req, _ := http.NewRequest(http.MethodPost, url, nil)
		req = req.WithContext(ctx)

		resp, err := client.Do(req)
		if err != nil {
			gerr := ce.CustomError{Title: "Error stopping container", Message: err.Error(), Code: 303}
			hfl.Errorf(gerr.ErrorNoColor())
			return &gerr
		}
		resp.Body.Close()
	}
	return nil
}

func startContainers(client *http.Client, base, version string, ids []string) *ce.CustomError {
	for _, id := range ids {
		url := APIPath(base, version, "containers", id, "start")
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(types.FastfailTimeout)*time.Second)
		defer cancel()
		req, _ := http.NewRequest(http.MethodPost, url, nil)
		req = req.WithContext(ctx)

		resp, err := client.Do(req)
		if err != nil {
			gerr := ce.CustomError{Title: "Error starting container", Message: err.Error(), Code: 304}
			hfl.Errorf(gerr.ErrorNoColor())
			return &gerr
		}
		resp.Body.Close()
	}
	return nil
}

func createTempContainer(client *http.Client, base, version, image, volumeName string) (string, *ce.CustomError) {
	// ensure image exists (may stream JSON; use stream-timeout because pulls can be long)
	if err := ensureImageExists(client, base, version, image); err != nil {
		return "", err
	}

	payload := map[string]any{
		"Image": image,
		"Cmd":   []string{"sh", "-lc", "sleep infinity"},
		"HostConfig": map[string]any{
			"Binds": []string{fmt.Sprintf("%s:/data", volumeName)},
		},
		"Tty": true,
	}
	data, merr := json.Marshal(payload)
	if merr != nil {
		return "", &ce.CustomError{Title: "Error building container create payload", Message: merr.Error(), Code: 351}
	}

	url := APIPath(base, version, "containers", "create")
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(types.FastfailTimeout)*time.Second)
	defer cancel()
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctx)

	resp, err := client.Do(req)
	if err != nil {
		return "", &ce.CustomError{Title: "Error creating temp container", Message: err.Error(), Code: 353}
	}
	defer resp.Body.Close()

	var res struct {
		ID string `json:"Id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil || res.ID == "" {
		return "", &ce.CustomError{Title: "Error reading container create reply", Message: "empty ID or decode failure", Code: 354}
	}

	hfl.Infof("Temp container created: %s", hf.Blue(res.ID[:12]))
	return res.ID, nil
}

func startContainer(client *http.Client, base, version, id string) *ce.CustomError {
	url := APIPath(base, version, "containers", id, "start")
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(types.FastfailTimeout)*time.Second)
	defer cancel()
	req, _ := http.NewRequest(http.MethodPost, url, nil)
	req = req.WithContext(ctx)

	resp, err := client.Do(req)
	if err != nil {
		return &ce.CustomError{Title: "Error starting temp container", Message: err.Error(), Code: 355}
	}
	resp.Body.Close()
	return nil
}

func stopAndRemoveContainer(client *http.Client, base, version, id string) *ce.CustomError {
	if !types.Quiet {
		fmt.Println("Cleaning up")
	}
	hfl.Debugf("Removing temporary container %s", id)
	stop := APIPath(base, version, "containers", id, "stop")
	_, err := client.Post(stop, "", nil)
	if err != nil {
		perr := ce.CustomError{Title: fmt.Sprintf("Error stopping %s", id), Message: err.Error(), Code: 303}
		hfl.Errorf(perr.ErrorNoColor())
		return &perr
	}

	rm := APIPath(base, version, "containers", id)
	req, _ := http.NewRequest(http.MethodDelete, rm, nil)
	_, _ = client.Do(req)
	return nil
}
