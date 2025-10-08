// dvol
// src/cmd/root.go

package cmd

import (
	"dvol/rest"
	"dvol/types"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	ce "github.com/jeanfrancoisgratton/customError/v3"
	hfl "github.com/jeanfrancoisgratton/helperFunctions/v3/logging"
	hftfx "github.com/jeanfrancoisgratton/helperFunctions/v3/terminalfx"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "dvol",
	Short:   "Volume backup/restore utility for Docker/Podman",
	Long:    "Backup and restore Docker/Podman volumes using the REST API, with optional gzip compression.",
	Version: hftfx.White(fmt.Sprintf("2.10.00-%s (2025.10.08)", runtime.GOARCH)),
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if types.LogLevel != "none" {
			if err := hfl.Init(filepath.Join(os.Getenv("HOME"), ".local", "state", "dvol.log"),
				hfl.ParseLevel(types.LogLevel), "USER", false, true); err != nil {
				cerr := ce.CustomError{Title: "Failed to init logging", Message: err.Error(), Code: 1}
				fmt.Println(cerr.Error())
				os.Exit(1)
			}
		}
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if types.LogLevel != "none" {
			hfl.Close()
		}
	},
}

var backupCmd = &cobra.Command{
	Use:   "backup <volume> <archive.tar[.gz]>",
	Short: "Backup a volume into a tarball",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		client, base, negotiatedAPI, err := rest.NewClient()
		if err != nil {
			hfl.Errorf(err.Error())
			fmt.Println(err.Error())
			os.Exit(err.Code)
		}
		tarball := args[1]
		if !strings.HasSuffix(tarball, "xz") && !strings.HasSuffix(tarball, "gz") && !strings.HasSuffix(tarball, "tar") {
			tarball += ".tar"
		}
		if err := rest.BackupVolume(client, base, negotiatedAPI, args[0], tarball); err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
	},
}

var restoreCmd = &cobra.Command{
	Use:   "restore <volume> <archive.tar[.{.g|x}z]>",
	Short: "Restore a volume from a tarball",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		client, base, negotiatedAPI, err := rest.NewClient()
		if err != nil {
			hfl.Errorf(err.Error())
			fmt.Println(err.Error())
			os.Exit(err.Code)
		}
		tarball := args[1]
		if !strings.HasSuffix(tarball, "xz") && !strings.HasSuffix(tarball, "gz") && !strings.HasSuffix(tarball, "tar") {
			tarball += ".tar"
		}
		if err := rest.RestoreVolume(client, base, negotiatedAPI, args[0], tarball); err != nil {
			fmt.Println(err.Error())
		}
	},
}

var lsVolCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all Docker volumes",
	Run: func(cmd *cobra.Command, args []string) {
		client, base, negotiatedAPI, err := rest.NewClient()
		vols, err := rest.ListVolumes(client, base, negotiatedAPI)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(err.Code)
		}
		rest.ShowVols(vols)
	},
}

var rmVolCmd = &cobra.Command{
	Use:   "delete <volume>",
	Short: "Delete a volume",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client, base, negotiatedAPI, err := rest.NewClient()
		if err != nil {
			//return err
		}
		if err := rest.DeleteVolume(client, base, negotiatedAPI, args[0]); err != nil {
			fmt.Println(err.Error())
			os.Exit(err.Code)
		}
	},
}

var clCmd = &cobra.Command{
	Use:     "changelog",
	Aliases: []string{"cl"},
	Short:   "Shows the Changelog",
	Run: func(cmd *cobra.Command, args []string) {
		changeLog()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		code := 1

		// Prefer your CustomError when present so we can map its internal Code.
		if cee, ok := err.(*ce.CustomError); ok {
			// Print the detailed error (keeps your rich info); map only the shell exit code.
			fmt.Fprintln(os.Stderr, strings.TrimSpace(cee.ErrorNoColor()))
			code = types.ShellExitCode(cee.Code)
		} else {
			// Non-CustomError: print and fall back to 1
			fmt.Fprintln(os.Stderr, strings.TrimSpace(err.Error()))
			code = 1
		}

		// Ensure exit code is within 0..255 (POSIX shells only observe the low 8 bits).
		if code < 0 {
			code = 1
		} else if code > 255 {
			code = 255
		}
		os.Exit(code)
	}
}

func init() {
	rootCmd.DisableAutoGenTag = true
	rootCmd.CompletionOptions.DisableDefaultCmd = false
	rootCmd.AddCommand(completionCmd, clCmd, restoreCmd, backupCmd, lsVolCmd, rmVolCmd)

	rootCmd.PersistentFlags().StringVarP(&types.LogLevel, "loglevel", "l", "none", "Log level: none|debug|info|error")
	rootCmd.PersistentFlags().StringVarP(&types.DockerHost, "host", "H", types.DockerHost, "Docker daemon host")
	rootCmd.PersistentFlags().StringVarP(&types.Image, "image", "i", types.Image, "Docker image to use")
	rootCmd.PersistentFlags().BoolVarP(&types.NoCleanup, "no-cleanup", "n", false, "Do not remove the temp container/image")
	rootCmd.PersistentFlags().BoolVarP(&types.Quiet, "quiet", "q", false, "Quiet output")
	rootCmd.PersistentFlags().StringVarP(&types.APIVersion, "api", "a", "", "Pin Docker API version (e.g., 1.50)")

	rootCmd.PersistentFlags().IntVarP(&types.FastfailTimeout, "fastfail-timeout", "f", 60, "HTTP handshake timeout value in seconds")
	rootCmd.PersistentFlags().IntVarP(&types.Timeout, "timeout", "t", 30, "HTTP streaming timeout value in minutes")

}

func changeLog() {
	fmt.Printf("\x1bc")
	fmt.Println("CHANGELOG")
	fmt.Print(`
VERSION		DATE			COMMENT
-------		----------		-------
2.10.00		2025.10.08		Go version update, builddeps update, verbose output, added {ba,z}sh completion
2.00.00		2025.08.24		Full rewrite
1.11.00		2025.06.19		Fixed unix:// usage, added the -q flag
1.10.00		2025.06.17		Fixed backup, volumes are now destroyed before restore
1.05.00		2025.06.11		Code is now API version-agnostic
1.02.00		2025.06.10		Added volume listing function
1.01.00		2025.06.09		Added cleanup routines, packaging scripts cleanup
1.00.00		2025.06.06		Initial release
`)
}
