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

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/docker/machine/libmachine/drivers/plugin"
	"github.com/docker/machine/libmachine/drivers/plugin/localbinary"
	"github.com/rancher-sandbox/docker-machine-driver-hyperkit/cmd"
	"github.com/rancher-sandbox/docker-machine-driver-hyperkit/pkg/hyperkit"
)

func main() {
	if len(os.Args) == 3 && os.Args[1] == "uuid-to-mac-addr" {
		mac, err := hyperkit.GetMACAddressFromUUID(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "getting MAC address from UUID: %v", err)
			os.Exit(1)
		}
		fmt.Println(mac)
		return
	}

	if len(os.Args) > 3 && os.Args[1] == "nfs-exports" {
		var err error
		if os.Args[2] == "add" {
			err = hyperkit.AddNFSExports(os.Args[3:]...)
		} else if os.Args[2] == "remove" {
			err = hyperkit.RemoveNFSExports(os.Args[3:]...)
		} else {
			err = fmt.Errorf("unknown nfs-export subcommand: %s", os.Args[2])
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "nfs-export %s failed: %v", os.Args[2], err)
			os.Exit(1)
		}
		return
	}

	if os.Getenv(localbinary.PluginEnvKey) == localbinary.PluginEnvVal {
		plugin.RegisterDriver(hyperkit.NewDriver("", ""))
		return
	}

	// Drop root privileges before running commands via cobra
	if err := syscall.Setuid(syscall.Getuid()); err != nil {
		fmt.Fprintf(os.Stderr, "cannot drop privileges: %v", err)
		os.Exit(1)
	}

	// Add the directory name of the current executable to the front of the PATH to load
	// the driver from the same directory (the current executable may be a symlink to the
	// driver, living in a different directory).
	executable, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot determine absolute path to current executable: %v", err)
		os.Exit(1)
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot evaluate symlinks in path to current executable: %v", err)
		os.Exit(1)
	}
	os.Setenv("PATH", os.ExpandEnv(fmt.Sprintf("%s:$PATH", filepath.Dir(executable))))

	cmd.Execute()
}
