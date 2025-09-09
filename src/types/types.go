// dvol
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/06/09 09:42

// Original filename: src/utils/types.go

package types

var DockerHost = "unix:///var/run/docker.sock"
var Image = "alpine:latest"
var APIVersion = ""
var FallbackAPIVersion = "1.50"
var NoCleanup = false
var Quiet = false
var LogLevel = "none"
var FastfailTimeout = 60 // HttpTimeout
var Timeout = 30

type DockerVolume struct {
	Name       string            `json:"Name"`
	Driver     string            `json:"Driver"`
	Mountpoint string            `json:"Mountpoint"`
	Labels     map[string]string `json:"Labels"`
	Scope      string            `json:"Scope"`
	Options    map[string]string `json:"Options"`
	CreatedAt  string            `json:"CreatedAt,omitempty"` // if present
}

// Timestamp layout is used by emit() in state.go
const timeLayout = "2006-01-02 15:04:05"

// LogLevel ordering: None < Error < Info < Debug
type ErrorCodes int
