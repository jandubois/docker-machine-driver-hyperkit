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
	"github.com/rancher-sandbox/docker-machine-driver-hyperkit/cmd/priv"
	"github.com/rancher-sandbox/docker-machine-driver-hyperkit/pkg/hyperkit"
)

func main() {
	if syscall.Geteuid() != 0 {
		executable, err := os.Executable()
		if err != nil {
			cmd.Abort("Cannot determine name of executable: %v", err)
		}

		permErr := "%s needs to run with elevated permissions. " +
			"Please run the following command, then try again: " +
			"sudo chown root:wheel %s && sudo chmod u+s %s"

		cmd.Abort(permErr, filepath.Base(executable), executable, executable)
	}

	if len(os.Args) > 1 {
		// All of the privileged commands will call os.Exit() and never return
		switch os.Args[1] {
		case "hyperkit":
			priv.Hyperkit()
		case "nfs-exports":
			priv.NFSExports()
		case "uuid-to-mac-addr":
			priv.UUIDtoMacAddr()
		}
	}

	// Drop root privileges before running driver mode, or commands via cobra
	if err := syscall.Setuid(syscall.Getuid()); err != nil {
		cmd.Abort("Cannot drop privileges: %v", err)
	}

	if os.Getenv(localbinary.PluginEnvKey) == localbinary.PluginEnvVal {
		plugin.RegisterDriver(hyperkit.NewDriver("", ""))
		return
	}

	// Add the directory name of the current executable to the front of the PATH to load
	// the driver from the same directory (the current executable may be a symlink to the
	// driver, living in a different directory).
	executable, err := os.Executable()
	if err != nil {
		cmd.Abort("Cannot determine absolute path to current executable: %v", err)
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		cmd.Abort("Cannot evaluate symlinks in path to current executable: %v", err)
	}
	err = os.Setenv("PATH", os.ExpandEnv(fmt.Sprintf("%s:$PATH", filepath.Dir(executable))))
	if err != nil {
		cmd.Abort("Cannot update PATH: %v", err)
	}

	err = cmd.Execute()
	if err != nil {
		cmd.Abort("Command failed: %v", err)
	}
}
