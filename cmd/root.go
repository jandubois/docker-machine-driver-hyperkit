package cmd

import (
	"os"

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
