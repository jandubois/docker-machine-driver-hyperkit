package cmd

import (
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/docker/machine/libmachine"
	"github.com/docker/machine/libmachine/drivers"
	"github.com/rancher-sandbox/docker-machine-driver-hyperkit/pkg/hyperkit"
	"github.com/spf13/cobra"
)

var (
	cmdline      string
	cpuCount     int
	diskSize     int
	hyperkitPath string
	isoURL       string
	memorySize   int
	mountRoot    string
	volumeMounts []string

	startCmd = &cobra.Command{
		Use:   "start",
		Short: "starts a vm",
		Long:  `Create and start a new VM.`,
		RunE:  startCommand,
	}

	// TODO(jandubois) these boot options are a subset of what minikube uses right now
	// TODO audit the settings and document why each one is being used!
	// The "noembed" option is required on boot2docker.iso (TinyCoreLinux) to make sure
	// the system doesn't run out of a ramdisk; otherwise pivot_root will fail.
	defaultCmdline = "loglevel=3 console=ttyS0 console=tty0 noembed nomodeset norestore random.trust_cpu=on hw_rng_model=virtio base"
)

func init() {
	rootCmd.AddCommand(startCmd)

	startCmd.Flags().StringVar(&cmdline, "boot-options", defaultCmdline, "Boot commandline options")
	startCmd.Flags().IntVar(&cpuCount, "cpus", 2, "Number of cpus")
	startCmd.Flags().IntVar(&diskSize, "disk-size", 40000, "Disk size in MB")
	startCmd.Flags().StringVar(&hyperkitPath, "hyperkit", "", "Path to hyperkit executable")
	startCmd.Flags().StringVar(&isoURL, "iso-url", "", "URL of the boot2docker.iso")
	startCmd.Flags().IntVar(&memorySize, "memory", 4096, "Memory size in MB")
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
	// TODO(jandubois) AuthOptions.StorePath should be unused, but provision.ConfigureAuth
	// TODO copies the client certs into this directory? Not sure if they are used for
	// TODO anything, as it looks like a race condition when multiple machines are provisioned.
	// https://github.com/docker/machine/pull/2730 does not inspire confidence in that code...
	host.HostOptions.AuthOptions.StorePath = storagePath

	return api.Create(host)
}

func newAPI() *libmachine.Client {
	return libmachine.NewClient(storagePath, path.Join(storagePath, "certs"))
}

func newDriver(machineName, storePath string) (interface{}, error) {
	if hyperkitPath != "" {
		realPath, err := filepath.EvalSymlinks(hyperkitPath)
		if err != nil {
			Abort("Cannot evaluate symlinks in %s", hyperkitPath)
		}
		hyperkitPath = realPath
	}
	driver := hyperkit.Driver{
		BaseDriver: &drivers.BaseDriver{
			MachineName: machineName,
			StorePath:   storePath,
			SSHUser:     "docker",
		},
		Boot2DockerURL: isoURL,
		DiskSize:       diskSize,
		Hyperkit:       hyperkitPath,
		Memory:         memorySize,
		CPU:            cpuCount,
		NFSSharesRoot:  mountRoot,
		NFSShares:      volumeMounts,
		Cmdline:        cmdline,
	}

	// If the user-provided cmdline starts with a "+" then it is supposed to be appended to the defaultCmdline.
	if strings.HasPrefix(cmdline, "+") {
		driver.Cmdline = fmt.Sprintf("%s %s", defaultCmdline, cmdline[1:])
	}
	return &driver, nil
}
