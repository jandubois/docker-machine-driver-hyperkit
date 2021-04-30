package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/docker/machine/libmachine/state"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(cpCmd)
}

var cpCmd = &cobra.Command{
	Use:   "cp [SOURCE] [DEST]",
	Short: "Copy files into or out of a machine.",
	Long:  `Copy files into or out of a machine.`,
	Args:  cobra.ExactArgs(2),
	RunE:  cpCommand,
}

func cpCommand(cmd *cobra.Command, args []string) error {
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
		return fmt.Errorf("cannot run cp command: Host %q is not running", host.Name)
	}

	port, err := host.Driver.GetSSHPort()
	if err != nil {
		return err
	}

	scpArgs := []string{
		"-P", strconv.Itoa(port),
		"-i", host.Driver.GetSSHKeyPath(),
		"-o", "IdentitiesOnly=yes",
		"-o", "LogLevel=quiet",
		"-o", "PasswordAuthentication=no",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
	}

	sshUserame := host.Driver.GetSSHUsername()
	sshHostname, err := host.Driver.GetSSHHostname()
	if err != nil {
		return err
	}

	for _, arg := range args {
		if strings.HasPrefix(arg, ":") {
			scpArgs = append(scpArgs, fmt.Sprintf("%s@%s%s", sshUserame, sshHostname, arg))
		} else {
			scpArgs = append(scpArgs, arg)
		}
	}

	command := exec.Command("scp", scpArgs...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}
