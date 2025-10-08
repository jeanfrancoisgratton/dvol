// rest/client.go
// dvol
// src/rest/client.go

package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	ce "github.com/jeanfrancoisgratton/customError/v3"
	hfl "github.com/jeanfrancoisgratton/helperFunctions/v3/logging"

	"dvol/types"
)

// NewClient builds an HTTP client and resolves base URL + API version.
func NewClient() (*http.Client, string, string, *ce.CustomError) {
	var transport *http.Transport
	var base string

	// Fast-fail dialer derived from types.FastfailTimeout
	fast := time.Duration(types.FastfailTimeout) * time.Second

	if types.DockerHost == "" || strings.HasPrefix(types.DockerHost, "unix://") {
		socket := "/var/run/docker.sock"
		if types.DockerHost != "" {
			socket = strings.TrimPrefix(types.DockerHost, "unix://")
		}

		transport = &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				d := net.Dialer{Timeout: fast}
				return d.Dial("unix", socket)
			},
			ResponseHeaderTimeout: fast,
			TLSHandshakeTimeout:   fast,
			DisableCompression:    false,
		}
		base = "http://d" // dummy host for unix transport; path builds actual socket calls
	} else if strings.HasPrefix(types.DockerHost, "tcp://") || strings.HasPrefix(types.DockerHost, "http://") || strings.HasPrefix(types.DockerHost, "https://") {
		host := strings.TrimPrefix(strings.TrimPrefix(types.DockerHost, "tcp://"), "http://")
		host = strings.TrimPrefix(host, "https://")

		transport = &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: fast,
			}).DialContext,
			ResponseHeaderTimeout: fast,
			TLSHandshakeTimeout:   fast,
			DisableCompression:    false,
		}
		// Preserve scheme if provided, default to http
		if strings.HasPrefix(types.DockerHost, "https://") {
			base = "https://" + host
		} else {
			base = "http://" + host
		}
	} else {
		return nil, "", "", &ce.CustomError{
			Title:   "Invalid Docker host",
			Message: fmt.Sprintf("Unsupported host format: %s", types.DockerHost),
			Code:    101,
		}
	}

	// Single shared client; Timeout=0 to avoid killing long streams mid-transfer.
	client := &http.Client{
		Transport: transport,
		Timeout:   0,
	}

	version := types.APIVersion
	if version == "" {
		v, err := negotiateAPIVersion(client, base)
		if err != nil {
			if err.Fatality == ce.Continuable {
				hfl.Infof(err.ErrorNoColor())
			}
			hfl.Errorf(err.ErrorNoColor())
			return nil, "", "", err
		}
		version = v
	}

	return client, base, version, nil
}

func negotiateAPIVersion(client *http.Client, base string) (string, *ce.CustomError) {
	req, err := http.NewRequest(http.MethodGet, base+"/version", nil)
	if err != nil {
		return "", &ce.CustomError{Title: "Failed to build /version request", Message: err.Error(), Code: 100}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(types.FastfailTimeout)*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	resp, rerr := client.Do(req)
	if rerr != nil {
		return "", &ce.CustomError{Title: "Failed to query /version", Message: rerr.Error(), Code: 101}
	}
	defer resp.Body.Close()

	var payload struct {
		APIVersion string `json:"ApiVersion"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", &ce.CustomError{Title: "Failed to decode JSON payload", Message: err.Error(), Code: 102}
	}
	if payload.APIVersion == "" {
		return "", &ce.CustomError{Title: "API version negotiation failed",
			Message: "Version field is missing from response ", Code: 503}
	}

	hfl.Infof("API version now set to v%s", payload.APIVersion)
	return "v" + payload.APIVersion, nil
}
