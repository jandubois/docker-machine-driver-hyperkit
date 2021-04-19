package cmd

import (
	"fmt"

	"github.com/docker/machine/libmachine/ssh"
	"github.com/docker/machine/libmachine/state"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(sshCmd)
}

var sshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "Log into or run a command on a machine with SSH.",
	Long:  `Log into or run a command on a machine with SSH.`,
	RunE:  sshCommand,
}

func sshCommand(cmd *cobra.Command, args []string) error {
	api := newAPI()
	defer api.Close()

	host, err := api.Load(machineName)
	if err != nil {
		return err
	}

	currentState, err := host.Driver.GetState()
	if err != nil {
		return err
	}

	if currentState != state.Running {
		return fmt.Errorf("cannot run SSH command: Host %q is not running", host.Name)
	}

	ssh.SetDefaultClient(ssh.Native)
	client, err := host.CreateSSHClient()
	if err != nil {
		return err
	}

	if len(args) == 0 {
		args = []string{"bash", "-i"}
	}
	return client.Shell(args...)
}
