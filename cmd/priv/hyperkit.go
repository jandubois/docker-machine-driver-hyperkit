package priv

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	hyperkit "github.com/moby/hyperkit/go"
	"github.com/rancher-sandbox/docker-machine-driver-hyperkit/cmd"
)

func Hyperkit() {
	if len(os.Args) != 5 {
		cmd.Abort("usage: hyperkit CONFIG DISKS CMDLINE")
	}

	var h hyperkit.HyperKit
	err := json.Unmarshal([]byte(os.Args[2]), &h)
	if err != nil {
		cmd.Abort("Unmarshalling hyperkit structure failed: %v", err)
	}

	var disks []hyperkit.RawDisk
	err = json.Unmarshal([]byte(os.Args[3]), &disks)
	if err != nil {
		cmd.Abort("Unmarshalling hyperkit disks structure failed: %v", err)
	}

	// Type conversion from hyperkit.RawDisk to hyperkit.Disk
	for _, disk := range disks {
		h.Disks = append(h.Disks, &disk)
	}

	// If a file called "hyperkit" exists in the same directory as the driver,
	// then don't allow invoking any other binary.
	executable := filepath.Join(cmd.DriverDir(), "hyperkit")
	if _, err := os.Stat(executable); err == nil {
		if h.HyperKit != executable {
			cmd.Abort("Cannot invoke any other hyperkit executable than %s", executable)
		}
	}

	// hyperkit executable must be owned by root (or group owned by either wheel or admin)
	var stat syscall.Stat_t
	if err := syscall.Stat(h.HyperKit, &stat); err != nil {
		cmd.Abort("Cannot stat %s", h.HyperKit)
	}
	if stat.Uid != 0 && stat.Gid != 0 && stat.Gid != 80 {
		cmd.Abort("Executable %s must be owned by root, or have group ownership by wheel(0) or admin(80)", h.HyperKit)
	}

	_, err = h.Start(os.Args[4])
	if err != nil {
		cmd.Abort("Failed to start hyperkit: %v", err)
	}
	_, _ = fmt.Fprintln(os.Stderr, "Hyperkit started successfully")
	os.Exit(0)
}
