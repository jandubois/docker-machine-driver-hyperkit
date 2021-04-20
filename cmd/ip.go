package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(ipCmd)
}

var ipCmd = &cobra.Command{
	Use:   "ip",
	Short: "Display the IP address of the machine.",
	Long:  `Display the IP address of the machine.`,
	RunE:  ipCommand,
}

func ipCommand(cmd *cobra.Command, args []string) error {
	api := newAPI()
	defer api.Close()

	host, err := api.Load(machineName)
	if err != nil {
		return err
	}

	ip, err := host.Driver.GetIP()
	if err == nil {
		fmt.Println(ip)
	}
	return err
}
