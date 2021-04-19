package cmd

import (
	"github.com/jandubois/docker-machine-driver-hyperkit/pkg/hyperkit"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "docker-machine-driver-hyperkit",
		Short: "A driver and control program for hyperkit",
		Long: `This program is both a docker-machine compatible driver for hyperkit,
as well as a lightweight control program to create/access/delete virtual machines
running via hyperkit.`,
		Version: hyperkit.GetVersion(),
	}
)

func Execute() error {
	return rootCmd.Execute()
}
