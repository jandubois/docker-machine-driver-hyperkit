// +build darwin

/*
Copyright 2016 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package hyperkit

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/docker/machine/libmachine/drivers"
	"github.com/docker/machine/libmachine/log"
	"github.com/docker/machine/libmachine/mcnflag"
	"github.com/docker/machine/libmachine/state"
	"github.com/google/uuid"
	"github.com/johanneswuerbach/nfsexports"
	ps "github.com/mitchellh/go-ps"
	hyperkit "github.com/moby/hyperkit/go"
	"github.com/pkg/errors"
	pkgdrivers "github.com/rancher-sandbox/docker-machine-driver-hyperkit/pkg/drivers"
)

const (
	isoFilename     = "boot2docker.iso"
	pidFileName     = "hyperkit.pid"
	machineFileName = "hyperkit.json"

	defaultCPUs     = 1
	defaultDiskSize = 20000
	defaultMemory   = 1024
	defaultSSHUser  = "docker"
)

// Driver is the machine driver for Hyperkit
type Driver struct {
	*drivers.BaseDriver
	*pkgdrivers.CommonDriver
	Boot2DockerURL string
	BootInitrd     string
	BootKernel     string
	CPU            int
	Cmdline        string
	DiskSize       int
	Hyperkit       string
	Memory         int
	NFSShares      []string
	NFSSharesRoot  string
	UUID           string
	VSockPorts     []string
	VpnKitSock     string
}

// NewDriver creates a new driver for a host
func NewDriver(machineName, storePath string) *Driver {
	return &Driver{
		// Don't init BaseDriver values here. They are overwritten by API .SetConfigRaw() call.
		CommonDriver: &pkgdrivers.CommonDriver{},
		DiskSize:     defaultDiskSize,
	}
}

// GetCreateFlags registers the flags this driver adds to
// "docker hosts create"
func (d *Driver) GetCreateFlags() []mcnflag.Flag {
	return []mcnflag.Flag{
		mcnflag.StringFlag{
			EnvVar: "HYPERKIT_BOOT2DOCKER_URL",
			Name:   "hyperkit-boot2docker-url",
			Usage:  "The URL of the boot2docker image. Defaults to the latest available version",
			Value:  "",
		},
		mcnflag.IntFlag{
			EnvVar: "HYPERKIT_CPU_COUNT",
			Name:   "hyperkit-cpu-count",
			Usage:  "Number of CPUs for the host.",
			Value:  defaultCPUs,
		},
		mcnflag.IntFlag{
			EnvVar: "HYPERKIT_DISK_SIZE",
			Name:   "hyperkit-disk-size",
			Usage:  "Size of disk for host in MB.",
			Value:  defaultDiskSize,
		},
		mcnflag.IntFlag{
			EnvVar: "HYPERKIT_MEMORY_SIZE",
			Name:   "hyperkit-memory-size",
			Usage:  "Memory size for host in MB.",
			Value:  defaultMemory,
		},
	}
}

// SetConfigFromFlags sets the machine config
func (d *Driver) SetConfigFromFlags(flags drivers.DriverOptions) error {
	d.Boot2DockerURL = flags.String("hyperkit-boot2docker-url")
	d.CPU = flags.Int("hyperkit-cpu-count")
	d.DiskSize = int(flags.Int("hyperkit-disk-size"))
	d.Memory = flags.Int("hyperkit-memory-size")

	return nil
}

// PreCreateCheck is called to enforce pre-creation steps
func (d *Driver) PreCreateCheck() error {
	return nil
}

func self(args ...string) (string, error) {
	self, err := os.Executable()
	if err != nil {
		return "", err
	}
	cmd := exec.Command(self, args...)
	log.Debugf("Running command: %s", cmd)
	out, err := cmd.CombinedOutput()
	output := strings.TrimSuffix(string(out), "\n")
	log.Debugf("Output:\n%s\nError: %v", output, err)
	return output, err
}

// Create a host using the driver's config
func (d *Driver) Create() error {
	d.SSHUser = defaultSSHUser

	// TODO: handle different disk types.
	if err := pkgdrivers.MakeDiskImage(d.BaseDriver, d.Boot2DockerURL, d.DiskSize); err != nil {
		return errors.Wrap(err, "making disk image")
	}

	isoPath := d.ResolveStorePath(isoFilename)
	if err := d.extractKernel(isoPath); err != nil {
		return errors.Wrap(err, "extracting kernel")
	}

	return d.Start()
}

// DriverName returns the name of the driver
func (d *Driver) DriverName() string {
	return "hyperkit"
}

// GetSSHHostname returns hostname for use with ssh
func (d *Driver) GetSSHHostname() (string, error) {
	return d.IPAddress, nil
}

// GetURL returns a Docker URL inside this host
// e.g. tcp://1.2.3.4:2376
// more info https://github.com/docker/machine/blob/b170508bf44c3405e079e26d5fdffe35a64c6972/libmachine/provision/utils.go#L159_L175
func (d *Driver) GetURL() (string, error) {
	ip, err := d.GetIP()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("tcp://%s:2376", ip), nil
}

// Return the state of the hyperkit pid
func pidState(pid int) (state.State, error) {
	if pid == 0 {
		return state.Stopped, nil
	}
	p, err := ps.FindProcess(pid)
	if err != nil {
		return state.Error, err
	}
	if p == nil {
		log.Debugf("hyperkit pid %d missing from process table", pid)
		return state.Stopped, nil
	}
	// hyperkit or com.docker.hyper
	if !strings.Contains(p.Executable(), "hyper") {
		log.Debugf("pid %d is stale, and is being used by %s", pid, p.Executable())
		return state.Stopped, nil
	}
	return state.Running, nil
}

// GetState returns the state that the host is in (running, stopped, etc)
func (d *Driver) GetState() (state.State, error) {
	pid := d.getPid()
	log.Debugf("hyperkit pid from json: %d", pid)
	return pidState(pid)
}

// Kill stops a host forcefully
func (d *Driver) Kill() error {
	return d.sendSignal(syscall.SIGKILL)
}

// Remove a host
func (d *Driver) Remove() error {
	s, err := d.GetState()
	if err != nil || s == state.Error {
		log.Debugf("Error checking machine status: %v, assuming it has been removed already", err)
	}
	if s == state.Running {
		if err := d.Stop(); err != nil {
			return err
		}
	}
	return nil
}

// Restart a host
func (d *Driver) Restart() error {
	return pkgdrivers.Restart(d)
}

func (d *Driver) createHost() (*hyperkit.HyperKit, error) {
	stateDir := d.ResolveStorePath("")
	h, err := hyperkit.New(d.Hyperkit, d.VpnKitSock, stateDir)
	if err != nil {
		return nil, errors.Wrap(err, "new-ing Hyperkit")
	}

	// TODO: handle the rest of our settings.
	h.Kernel = d.BootKernel
	h.Initrd = d.BootInitrd
	h.VMNet = true
	h.ISOImages = []string{d.ResolveStorePath(isoFilename)}
	h.Console = hyperkit.ConsoleFile
	if d.CPU > defaultCPUs {
		h.CPUs = d.CPU
	}
	if d.Memory > defaultMemory {
		h.Memory = d.Memory
	}
	h.UUID = d.UUID
	if h.UUID == "" {
		h.UUID = uuid.NewSHA1(uuid.Nil, []byte(stateDir)).String()
	}

	if vsockPorts, err := d.extractVSockPorts(); err != nil {
		return nil, err
	} else if len(vsockPorts) >= 1 {
		h.VSock = true
		h.VSockPorts = vsockPorts
	}

	h.Disks = []hyperkit.Disk{
		&hyperkit.RawDisk{
			Path: pkgdrivers.GetDiskPath(d.BaseDriver),
			Size: d.DiskSize,
			Trim: true,
		},
	}

	return h, nil
}

// Start a host
func (d *Driver) Start() error {
	if err := d.recoverFromUncleanShutdown(); err != nil {
		return err
	}

	h, err := d.createHost()
	if err != nil {
		return err
	}

	log.Debugf("Using UUID %s", h.UUID)
	mac, err := self("uuid-to-mac-addr", h.UUID)
	if err != nil {
		return errors.Wrap(err, "getting MAC address from UUID")
	}

	// Need to strip 0's
	mac = trimMacAddress(mac)
	log.Debugf("Generated MAC %s", mac)

	// Marshal h.Disks separately because they will need to be unmarshaled as hyperkit.RawDisk types
	// because hyperkit.Disk is just an interface.
	disks, err := json.Marshal(h.Disks)
	if err != nil {
		return errors.Wrap(err, "exporting hyperkit disks struct to JSON")
	}
	h.Disks = []hyperkit.Disk{}
	hyperkit, err := json.Marshal(h)
	if err != nil {
		return errors.Wrap(err, "exporting hyperkit struct to JSON")
	}

	log.Debugf("Starting with cmdline: %s\nhyperkit is %s\ndisks is %s", d.Cmdline, string(hyperkit), string(disks))
	out, err := self("hyperkit", string(hyperkit), string(disks), d.Cmdline)
	if err != nil {
		return errors.Wrapf(err, "failed to start hyperkit with cmd line: %s\nError: %v\n%s", d.Cmdline, err, out)
	}

	if err := d.setupIP(mac); err != nil {
		return err
	}

	if err := d.setupNFSMounts(); err != nil {
		return err
	}
	return nil
}

func (d *Driver) setupIP(mac string) error {
	getIP := func() error {
		st, err := d.GetState()
		if err != nil {
			return errors.Wrap(err, "get state")
		}
		if st == state.Error || st == state.Stopped {
			return fmt.Errorf("hyperkit crashed! command line:\n  hyperkit %s", d.Cmdline)
		}

		d.IPAddress, err = GetIPAddressByMACAddress(mac)
		if err != nil {
			return &tempError{err}
		}
		return nil
	}

	var err error

	// Implement a retry loop without calling any minikube code
	for i := 0; i < 30; i++ {
		log.Debugf("Attempt %d", i)
		err = getIP()
		if err == nil {
			break
		}
		if _, ok := err.(*tempError); !ok {
			return err
		}
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		return fmt.Errorf("IP address never found in dhcp leases file %v", err)
	}
	log.Debugf("IP: %s", d.IPAddress)

	return nil
}

func (d *Driver) setupNFSMounts() error {
	var err error

	if len(d.NFSShares) > 0 {
		log.Info("Setting up NFS mounts")
		if err := drivers.WaitForSSH(d); err != nil {
			return err
		}
		err = d.setupNFSShare()
		if err != nil {
			// TODO(tstromberg): Check that logging an and error and return it is appropriate. Seems weird.
			log.Errorf("NFS setup failed: %v", err)
			return err
		}
	}

	return nil
}

type tempError struct {
	Err error
}

func (t tempError) Error() string {
	return "Temporary error: " + t.Err.Error()
}

//recoverFromUncleanShutdown searches for an existing hyperkit.pid file in
//the machine directory. If it can't find it, a clean shutdown is assumed.
//If it finds the pid file, it checks for a running hyperkit process with that pid
//as the existence of a file might not indicate an unclean shutdown but an actual running
//hyperkit server. This is an error situation - we shouldn't start minikube as there is likely
//an instance running already. If the PID in the pidfile does not belong to a running hyperkit
//process, we can safely delete it, and there is a good chance the machine will recover when restarted.
func (d *Driver) recoverFromUncleanShutdown() error {
	pidFile := d.ResolveStorePath(pidFileName)

	if _, err := os.Stat(pidFile); err != nil {
		if os.IsNotExist(err) {
			log.Debugf("clean start, hyperkit pid file doesn't exist: %s", pidFile)
			return nil
		}
		return errors.Wrap(err, "stat")
	}

	log.Warnf("minikube might have been shutdown in an unclean way, the hyperkit pid file still exists: %s", pidFile)
	bs, err := ioutil.ReadFile(pidFile)
	if err != nil {
		return errors.Wrapf(err, "reading pidfile %s", pidFile)
	}
	content := strings.TrimSpace(string(bs))
	pid, err := strconv.Atoi(content)
	if err != nil {
		return errors.Wrapf(err, "parsing pidfile %s", pidFile)
	}

	st, err := pidState(pid)
	if err != nil {
		return errors.Wrap(err, "pidState")
	}

	log.Debugf("pid %d is in state %q", pid, st)
	if st == state.Running {
		return nil
	}
	log.Debugf("Removing stale pid file %s...", pidFile)
	if err := os.Remove(pidFile); err != nil {
		return errors.Wrap(err, fmt.Sprintf("removing pidFile %s", pidFile))
	}
	return nil
}

// Stop a host gracefully
func (d *Driver) Stop() error {
	d.cleanupNfsExports()
	err := d.sendSignal(syscall.SIGTERM)
	if err != nil {
		return errors.Wrap(err, "hyperkit sigterm failed")
	}

	// wait 5s for graceful shutdown
	for i := 0; i < 5; i++ {
		log.Debug("waiting for graceful shutdown")
		time.Sleep(time.Second * 1)
		s, err := d.GetState()
		if err != nil {
			return errors.Wrap(err, "hyperkit waiting graceful shutdown failed")
		}
		if s == state.Stopped {
			return nil
		}
	}

	log.Debug("sending sigkill")
	return d.Kill()
}

func (d *Driver) extractKernel(isoPath string) error {
	files, err := ISOExtractBootFiles(isoPath, d.ResolveStorePath(""))
	if err != nil {
		return err
	}

	if files.KernelPath == "" {
		return errors.Wrapf(err, "failed to extract kernel boot image from iso")
	}
	d.BootKernel = files.KernelPath

	if files.InitrdPath == "" {
		return errors.Wrapf(err, "failed to extract initial ram disk from iso")
	}
	d.BootInitrd = files.InitrdPath

	if files.IsoLinuxCfgPath == "" {
		return errors.Wrapf(err, "failed to extract isolinux config")
	}

	return nil
}

// InvalidPortNumberError implements the Error interface.
// It is used when a VSockPorts port number cannot be recognised as an integer.
type InvalidPortNumberError string

// Error returns an Error for InvalidPortNumberError
func (port InvalidPortNumberError) Error() string {
	return fmt.Sprintf("vsock port '%s' is not an integer", string(port))
}

func (d *Driver) extractVSockPorts() ([]int, error) {
	vsockPorts := make([]int, 0, len(d.VSockPorts))

	for _, port := range d.VSockPorts {
		p, err := strconv.Atoi(port)
		if err != nil {
			return nil, InvalidPortNumberError(port)
		}
		vsockPorts = append(vsockPorts, p)
	}

	return vsockPorts, nil
}

func (d *Driver) setupNFSShare() error {
	user, err := user.Current()
	if err != nil {
		return err
	}

	hostIP, err := GetNetAddr()
	if err != nil {
		return err
	}

	mountCommands := "#/bin/bash\\n"
	// TODO(jandubois) nfs-client utils are not running by default on TinyCoreLinux (boot2docker)
	mountCommands += "[ -f /usr/local/etc/init.d/nfs-client ] && sudo /usr/local/etc/init.d/nfs-client start\\n"
	log.Info(d.IPAddress)

	exportsAddCmd := []string{"nfs-exports", "add", user.Username}

	for _, share := range d.NFSShares {
		sharePaths := strings.Split(share, ":")
		localPath := sharePaths[0]
		if !path.IsAbs(localPath) {
			localPath = d.ResolveStorePath(localPath)
		}
		localPath, err := filepath.EvalSymlinks(localPath)
		if err != nil {
			log.Errorf("cannot evaluate symlinks in share path '%s': %v", sharePaths[0], err)
			return err
		}
		// nfsExportIdentifier() is called with `share` and not `localPath` to keep the exports cleanup code simple
		exportsAddCmd = append(exportsAddCmd, d.nfsExportIdentifier(share), localPath, d.IPAddress)

		var mountPoint string
		if len(sharePaths) < 2 {
			mountPoint = filepath.Join(d.NFSSharesRoot, localPath)
		} else {
			// TODO(jandubois) Should we validate that the mountpoint is an absolute path?
			mountPoint = sharePaths[1]
		}
		mountCommands += fmt.Sprintf("sudo mkdir -p %s\\n", mountPoint)
		mountCommands += fmt.Sprintf("sudo mount -t nfs -o vers=3,noacl,async '%s:%s' %s\\n", hostIP, localPath, mountPoint)
	}

	if _, err := self(exportsAddCmd...); err != nil {
		return err
	}

	writeScriptCmd := fmt.Sprintf("echo -e \"%s\" | sh", mountCommands)
	if _, err := drivers.RunSSHCommandFromDriver(d, writeScriptCmd); err != nil {
		return err
	}

	return nil
}

func (d *Driver) nfsExportIdentifier(path string) string {
	return fmt.Sprintf("docker-machine-driver-hyperkit %s-%s", d.MachineName, path)
}

func (d *Driver) sendSignal(s os.Signal) error {
	pid := d.getPid()
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	return proc.Signal(s)
}

func (d *Driver) getPid() int {
	pidPath := d.ResolveStorePath(machineFileName)

	f, err := os.Open(pidPath)
	if err != nil {
		log.Warnf("Error reading pid file: %v", err)
		return 0
	}
	dec := json.NewDecoder(f)

	var config struct {
		Pid int `json:"pid"`
	}

	if err := dec.Decode(&config); err != nil {
		log.Warnf("Error decoding pid file: %v", err)
		return 0
	}

	return config.Pid
}

func (d *Driver) cleanupNfsExports() {
	if len(d.NFSShares) > 0 {
		exportsRemoveCmd := []string{"nfs-exports", "remove"}
		for _, share := range d.NFSShares {
			exportsRemoveCmd = append(exportsRemoveCmd, d.nfsExportIdentifier(share))
		}
		self(exportsRemoveCmd...)
	}
}

// nfsexports.ReloadDaemon uses `sudo` which will prompt for a password; we are already running as root
func reloadNFSDaemon() error {
	uid := syscall.Getuid()
	syscall.Setuid(0)
	out, err := exec.Command("/sbin/nfsd", "restart").Output()
	syscall.Setreuid(uid, 0)
	if err != nil {
		return fmt.Errorf("Reloading nfsd failed: %s\n%s", err.Error(), out)
	}
	return nil
}

func AddNFSExports(args ...string) error {
	user := args[0]
	args = args[1:]

	if len(args)%3 != 0 {
		return fmt.Errorf("there should be 3 arguments for each export")
	}

	for len(args) > 0 {
		ident := args[0]
		path := args[1]
		ip := args[2]
		args = args[3:]

		export := fmt.Sprintf("%q %s -alldirs -mapall=%s", path, ip, user)
		if _, err := nfsexports.Add("", ident, export); err != nil {
			if strings.Contains(err.Error(), "conflicts with existing export") {
				fmt.Fprintf(os.Stderr, "Conflicting NFS Share not setup and ignored: %v", err)
				continue
			}
			return err
		}
	}
	return reloadNFSDaemon()
}

func RemoveNFSExports(args ...string) error {
	for _, ident := range args {
		if _, err := nfsexports.Remove("", ident); err != nil {
			fmt.Fprintf(os.Stderr, "failed removing nfs share (%s): %v", ident, err)
		}
	}
	if err := reloadNFSDaemon(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to reload the nfs daemon: %v", err)
	}
	return nil
}
