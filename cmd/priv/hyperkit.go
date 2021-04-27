package priv

import (
	"encoding/json"
	"fmt"
	"os"

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

	_, err = h.Start(os.Args[4])
	if err != nil {
		cmd.Abort("Failed to start hyperkit: %v", err)
	}
	_, _ = fmt.Fprintln(os.Stderr, "Hyperkit started successfully")
	os.Exit(0)
}
