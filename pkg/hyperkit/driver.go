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
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/docker/machine/libmachine/drivers"
	"github.com/docker/machine/libmachine/log"
	"github.com/docker/machine/libmachine/state"
	hyperkit "github.com/moby/hyperkit/go"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"
	vmnet "github.com/zchee/go-vmnet"
	pkgdrivers "k8s.io/minikube/pkg/drivers"
	commonutil "k8s.io/minikube/pkg/util"
)

const (
	isoFilename     = "boot2docker.iso"
	pidFileName     = "hyperkit.pid"
	machineFileName = "hyperkit.json"
)

type Driver struct {
	*drivers.BaseDriver
	*pkgdrivers.CommonDriver
	Boot2DockerURL string
	DiskSize       int
	CPU            int
	Memory         int
	Cmdline        string
}

func NewDriver(hostName, storePath string) *Driver {
	return &Driver{
		BaseDriver: &drivers.BaseDriver{
			SSHUser: "docker",
		},
		CommonDriver: &pkgdrivers.CommonDriver{},
	}
}

func (d *Driver) Create() error {
	// TODO: handle different disk types.
	if err := pkgdrivers.MakeDiskImage(d.BaseDriver, d.Boot2DockerURL, d.DiskSize); err != nil {
		return errors.Wrap(err, "making disk image")
	}

	isoPath := d.ResolveStorePath(isoFilename)
	if err := d.extractKernel(isoPath); err != nil {
		return err
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

// GetURL returns a Docker compatible host URL for connecting to this host
// e.g. tcp://1.2.3.4:2376
func (d *Driver) GetURL() (string, error) {
	ip, err := d.GetIP()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("tcp://%s:2376", ip), nil
}

// GetState returns the state that the host is in (running, stopped, etc)
func (d *Driver) GetState() (state.State, error) {
	pid := d.getPid()
	if pid == 0 {
		return state.Stopped, nil
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return state.Error, err
	}

	// Sending a signal of 0 can be used to check the existence of a process.
	if err := p.Signal(syscall.Signal(0)); err != nil {
		return state.Stopped, nil
	}
	if p == nil {
		return state.Stopped, nil
	}
	return state.Running, nil
}

// Kill stops a host forcefully
func (d *Driver) Kill() error {
	return d.sendSignal(syscall.SIGKILL)
}

// Remove a host
func (d *Driver) Remove() error {
	s, err := d.GetState()
	if err != nil || s == state.Error {
		log.Infof("Error checking machine status: %s, assuming it has been removed already", err)
	}
	if s == state.Running {
		if err := d.Stop(); err != nil {
			return err
		}
	}
	return nil
}

func (d *Driver) Restart() error {
	return pkgdrivers.Restart(d)
}

// Start a host
func (d *Driver) Start() error {
	h, err := hyperkit.New("", "", filepath.Join(d.StorePath, "machines", d.MachineName))
	if err != nil {
		return err
	}

	// TODO: handle the rest of our settings.
	h.Kernel = d.ResolveStorePath("bzimage")
	h.Initrd = d.ResolveStorePath("initrd")
	h.VMNet = true
	h.ISOImage = d.ResolveStorePath(isoFilename)
	h.Console = hyperkit.ConsoleFile
	h.CPUs = d.CPU
	h.Memory = d.Memory

	// Set UUID
	h.UUID = uuid.NewUUID().String()
	log.Infof("Generated UUID %s", h.UUID)
	mac, err := vmnet.GetMACAddressFromUUID(h.UUID)
	if err != nil {
		return err
	}

	// Need to strip 0's
	mac = trimMacAddress(mac)
	log.Infof("Generated MAC %s", mac)
	h.Disks = []hyperkit.DiskConfig{
		{
			Path:   pkgdrivers.GetDiskPath(d.BaseDriver),
			Size:   d.DiskSize,
			Driver: "virtio-blk",
		},
	}
	log.Infof("Starting with cmdline: %s", d.Cmdline)
	if err := h.Start(d.Cmdline); err != nil {
		return err
	}

	getIP := func() error {
		var err error
		d.IPAddress, err = GetIPAddressByMACAddress(mac)
		if err != nil {
			return &commonutil.RetriableError{Err: err}
		}
		return nil
	}

	if err := commonutil.RetryAfter(30, getIP, 2*time.Second); err != nil {
		return fmt.Errorf("IP address never found in dhcp leases file %v", err)
	}
	return nil
}

// Stop a host gracefully
func (d *Driver) Stop() error {
	return d.sendSignal(syscall.SIGTERM)
}

func (d *Driver) extractKernel(isoPath string) error {
	for _, f := range []struct {
		pathInIso string
		destPath  string
	}{
		{"/boot/bzimage", "bzimage"},
		{"/boot/initrd", "initrd"},
		{"/isolinux/isolinux.cfg", "isolinux.cfg"},
	} {
		fullDestPath := d.ResolveStorePath(f.destPath)
		if err := ExtractFile(isoPath, f.pathInIso, fullDestPath); err != nil {
			return err
		}
	}
	return nil
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
		log.Warnf("Error reading pid file: %s", err)
		return 0
	}
	dec := json.NewDecoder(f)
	config := hyperkit.HyperKit{}
	if err := dec.Decode(&config); err != nil {
		log.Warnf("Error decoding pid file: %s", err)
		return 0
	}

	return config.Pid
}
