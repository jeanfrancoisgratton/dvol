// dvol
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/07/25 17:17
// Original filename: src/rest/volumes.go

package rest

import (
	"bytes"
	"context"
	"dvol/types"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	ce "github.com/jeanfrancoisgratton/customError/v3"
	hfl "github.com/jeanfrancoisgratton/helperFunctions/v4/logging"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v4/terminalfx"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

func DeleteVolume(client *http.Client, base, apiVersion, volumeName string) *ce.CustomError {
	if !types.Quiet {
		fmt.Printf("Deleting volume %s\n", volumeName)
	}
	hfl.Debugf("Deleting volume %s\n", volumeName)

	url := APIPath(base, apiVersion, "volumes", volumeName)

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		cerr := ce.CustomError{Title: "Error building http request", Message: err.Error(), Code: 100}
		hfl.Errorf(cerr.ErrorNoColor())
		return &cerr
	}
	resp, err := client.Do(req)
	if err != nil {
		cerr := ce.CustomError{Title: "Error executing http request", Message: err.Error(), Code: 101}
		hfl.Errorf(cerr.ErrorNoColor())
		return &cerr
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		cerr := ce.CustomError{Title: "Delete failed", Message: fmt.Sprintf("http error code: %d", resp.StatusCode), Code: 102}
		hfl.Errorf(cerr.ErrorNoColor())
		return &cerr
	}
	if !types.Quiet {
		fmt.Printf("Volume %s %s.\n", volumeName, hftx.Green("deleted"))
	}
	return nil
}

func ListVolumes(client *http.Client, base, apiVersion string) ([]types.DockerVolume, *ce.CustomError) {
	url := APIPath(base, apiVersion, "volumes")

	resp, err := client.Get(url)
	if err != nil {
		cerr := ce.CustomError{Title: "Unable to list volumes", Message: err.Error(), Code: 103}
		hfl.Errorf(cerr.ErrorNoColor())
		return nil, &cerr
	}
	defer resp.Body.Close()

	var payload struct {
		Volumes []types.DockerVolume `json:"Volumes"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		cerr := ce.CustomError{Title: "Unable to the JSON payload", Message: err.Error(), Code: 201}
		hfl.Errorf(cerr.ErrorNoColor())
		return nil, &cerr
	}
	return payload.Volumes, nil
}

func ShowVols(vols []types.DockerVolume) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Name", "Driver", "Creation time"})

	for _, v := range vols {
		t.AppendRow(table.Row{v.Name, v.Driver, parseTimeString(v.CreatedAt).Format("2006.01.02 15:04:05")})
	}
	t.SetStyle(table.StyleBold)
	t.Style().Options.DrawBorder = true
	t.Style().Options.SeparateColumns = true
	t.Style().Format.Header = text.FormatDefault
	t.Render()
}

func parseTimeString(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

func deleteAndRecreateVolume(client *http.Client, base, version, volume string) *ce.CustomError {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(types.HandshakeTimeout)*time.Second)
	defer cancel()

	if !types.Quiet {
		fmt.Println(hftx.InProgressSign(fmt.Sprintf("Deleting and recreating volume %s", hftx.Green(volume))))
	}
	hfl.Infof("Deleting volume %s\n", volume)
	// DELETE /volumes/{name}
	deleteURL := APIPath(base, version, "volumes", volume)
	req, err := http.NewRequest(http.MethodDelete, deleteURL, nil)
	if err != nil {
		title := "Error building http request"
		message := err.Error()
		rerr := ce.CustomError{Title: title, Message: message, Code: 601}
		hfl.Errorf(rerr.ErrorNoColor())
		if !types.Quiet {
			fmt.Println(hftx.ErrorSign(fmt.Sprintf("%s: %s", title, message)))
		}
		return &rerr
	}
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
		title := "Error executing http request"
		message := err.Error()
		derr := ce.CustomError{Title: title, Message: message, Code: 602}
		hfl.Errorf(derr.ErrorNoColor())
		if !types.Quiet {
			fmt.Println(hftx.SkullBonesSign(fmt.Sprintf("%s: %s", title, message)))
		}
		return &derr
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	// POST /volumes/create with {"Name": volume}
	createURL := APIPath(base, version, "volumes", "create")
	body := map[string]string{"Name": volume}
	var buf []byte
	var jerr error
	if buf, jerr = json.Marshal(body); jerr != nil {
		message := "Error marshalling json"
		title := jerr.Error()
		err := ce.CustomError{Title: title, Message: message, Code: 103}
		hfl.Errorf(err.ErrorNoColor())
		if !types.Quiet {
			fmt.Println(hftx.SkullBonesSign(fmt.Sprintf("%s: %s", title, message)))
		}
		return &err
	}

	resp, err = client.Post(createURL, "application/json", bytes.NewReader(buf))
	if err != nil {
		message := fmt.Sprintf("Error re-creating the volume %s", volume)
		title := err.Error()
		cerr := ce.CustomError{Title: title, Message: message, Code: 603}
		hfl.Errorf(cerr.ErrorNoColor())
		if !types.Quiet {
			fmt.Println(hftx.SkullBonesSign(fmt.Sprintf("%s: %s", title, message)))
		}
		return &cerr
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		if rsp, raerr := io.ReadAll(resp.Body); raerr != nil {
			title := "Volume creation error"
			message := fmt.Sprintf("HTTP error code: %d : %s", resp.StatusCode, rsp)
			cerr := ce.CustomError{Message: message, Title: title, Code: 604}
			hfl.Errorf(cerr.ErrorNoColor())
			if !types.Quiet {
				fmt.Println(hftx.SkullBonesSign(fmt.Sprintf("%s: %s", title, message)))
			}
			return &cerr
		}
	}
	return nil
}
