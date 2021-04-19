package cmd

import (
	"fmt"

	"github.com/jandubois/docker-machine-driver-hyperkit/pkg/hyperkit"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Long: `Prints version and git commit information for this docker-machine
hyperkit driver in the format produced by the legacy version of the driver.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("version:", hyperkit.GetVersion())
		fmt.Println("commit:", hyperkit.GetGitCommitID())
	},
}
