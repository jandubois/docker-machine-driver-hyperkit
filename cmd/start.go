package cmd

import (
	"encoding/json"
	"path"

	"github.com/docker/machine/libmachine"
	"github.com/docker/machine/libmachine/drivers"
	"github.com/rancher-sandbox/docker-machine-driver-hyperkit/pkg/hyperkit"
	"github.com/spf13/cobra"
)

var (
	isoURL       string
	cpuCount     int
	memorySize   int
	diskSize     int
	mountRoot    string
	volumeMounts []string

	startCmd = &cobra.Command{
		Use:   "start",
		Short: "starts a vm",
		Long:  `Create and start a new VM.`,
		RunE:  startCommand,
	}
)

func init() {
	rootCmd.AddCommand(startCmd)

	startCmd.Flags().StringVar(&isoURL, "iso-url", "", "URL of the boot2docker.iso")
	startCmd.Flags().IntVar(&cpuCount, "cpus", 2, "Number of cpus")
	startCmd.Flags().IntVar(&memorySize, "memory", 4096, "Memory size in MB")
	startCmd.Flags().IntVar(&diskSize, "disk-size", 40000, "Disk size in MB")
	startCmd.Flags().StringVar(&mountRoot, "mount-root", "/nfsshares", "NFS mount root")
	startCmd.Flags().StringArrayVar(&volumeMounts, "volume", []string{}, "Paths to mount via NFS")
}

func startCommand(cmd *cobra.Command, args []string) error {
	api := newAPI()
	defer api.Close()

	driver, err := newDriver(machineName, storagePath)
	data, err := json.Marshal(driver)
	if err != nil {
		return err
	}
	host, err := api.NewHost("hyperkit", data)
	if err != nil {
		return err
	}
	return api.Create(host)
}

func newAPI() *libmachine.Client {
	return libmachine.NewClient(storagePath, path.Join(storagePath, "certs"))
}

func newDriver(machineName, storePath string) (interface{}, error) {
	return &hyperkit.Driver{
		BaseDriver: &drivers.BaseDriver{
			MachineName: machineName,
			StorePath:   storePath,
			SSHUser:     "docker",
		},
		Boot2DockerURL: isoURL,
		DiskSize:       diskSize,
		Memory:         memorySize,
		CPU:            cpuCount,
		NFSSharesRoot:  mountRoot,
		NFSShares:      volumeMounts,
	}, nil
}
