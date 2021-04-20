package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(deleteCmd)
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Remove a machine.",
	Long:  `Remove a machine.`,
	RunE:  deleteCommand,
}

func deleteCommand(cmd *cobra.Command, args []string) error {
	api := newAPI()
	defer api.Close()

	host, err := api.Load(machineName)
	if err != nil {
		return err
	}

	fmt.Println("Powering down machine now...")
	if err = host.Stop(); err != nil {
		return err
	}
	return api.Remove(machineName)
}
