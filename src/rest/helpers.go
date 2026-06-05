// dvol
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/08/17 16:03
// Original filename: src/rest/helpers.go

package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	ce "github.com/jeanfrancoisgratton/customError/v3"
	hfl "github.com/jeanfrancoisgratton/helperFunctions/v5/logging"
)

func APIPath(base, version string, parts ...string) string {
	p := path.Join(append([]string{"/" + version}, parts...)...)
	return base + p
}

func ID2Name(client *http.Client, base, version, id string) (string, *ce.CustomError) {
	url := APIPath(base, version, "containers", id, "json")

	resp, err := client.Get(url)
	if err != nil {
		cerr := ce.CustomError{
			Fatality: ce.Continuable,
			Title:    fmt.Sprintf("GET %s failed", url),
			Message:  err.Error(),
			Code:     351,
		}
		hfl.Errorf(cerr.ErrorNoColor())
		return "", &cerr
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		cerr := ce.CustomError{
			Title:   fmt.Sprintf("Inspect container %s failed", id),
			Message: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body))),
			Code:    452,
		}
		return "", &cerr
	}

	var info struct {
		Name string `json:"Name"` // e.g. "/my-container"
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		cerr := ce.CustomError{
			Title:   "Decode container inspect JSON failed",
			Message: err.Error(),
			Code:    453,
		}
		return "", &cerr
	}

	name := strings.TrimPrefix(info.Name, "/")
	if name == "" {
		if len(id) > 12 {
			return id[:12], nil
		}
		return id, nil
	}
	return name, nil
}
