package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Print status of a machine.",
	Long:  `Print status of a machine.`,
	RunE:  statusCommand,
}

func statusCommand(cmd *cobra.Command, args []string) error {
	api := newAPI()
	defer api.Close()

	host, err := api.Load(machineName)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "does not exist") {
			fmt.Println("Does not exist")
			return nil
		}
		return fmt.Errorf("error loading config for host %s: %v", machineName, err)
	}

	currentState, err := host.Driver.GetState()
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			fmt.Println("Not found")
			return nil
		}
		return fmt.Errorf("error getting state for host %s: %s", host.Name, err)
	}

	fmt.Println(currentState)
	return nil
}
