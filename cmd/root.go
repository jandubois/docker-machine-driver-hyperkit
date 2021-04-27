package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/machine/libmachine/log"
	"github.com/rancher-sandbox/docker-machine-driver-hyperkit/pkg/hyperkit"
	"github.com/spf13/cobra"
)

var (
	storagePath string
	debugMode   bool
	machineName string

	rootCmd = &cobra.Command{
		Use:   "docker-machine-driver-hyperkit",
		Short: "A driver and control program for hyperkit",
		Long: `This program is both a docker-machine compatible driver for hyperkit,
as well as a lightweight control program to create/access/delete virtual machines
running via hyperkit.`,
		Version: hyperkit.GetVersion(),
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Don't display usage information when a RunE command returns an error.
			// Just print the error and exit.
			cmd.SilenceUsage = true
		},
	}
)

func init() {
	cobra.OnInitialize(onInit)
	rootCmd.PersistentFlags().StringVarP(&storagePath, "storage-path", "s", os.ExpandEnv("$HOME/.hyperkit"), "Config directory")
	rootCmd.PersistentFlags().StringVarP(&machineName, "machine-name", "n", "default", "Machine name")
	rootCmd.PersistentFlags().BoolVarP(&debugMode, "debug", "D", false, "Enable debug mode")
}

func onInit() {
	if debugMode {
		log.SetDebug(true)
	}
}

func Execute() error {
	return rootCmd.Execute()
}

// Abort prints the formatted error message to stdout and exits with a non-zero status.
func Abort(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	_, _ = os.Stderr.WriteString(msg)
	if !strings.HasSuffix(msg, "\n") {
		_, _ = os.Stderr.WriteString("\n")
	}
	os.Exit(1)
}

// DriverDir returns the absolute path to the directory containing the running executable
// after resolving symbolic links. Calls cmd.Abort() on failure; doesn't return an error.
func DriverDir() string {
	executable, err := os.Executable()
	if err != nil {
		Abort("Cannot determine absolute path to current executable: %v", err)
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		Abort("Cannot evaluate symlinks in path to current executable: %v", err)
	}
	return filepath.Dir(executable)
}
