package priv

import (
	"fmt"
	"os"

	"github.com/rancher-sandbox/docker-machine-driver-hyperkit/cmd"
	"github.com/rancher-sandbox/docker-machine-driver-hyperkit/pkg/hyperkit"
)

func NFSExports() {
	if len(os.Args) < 4 {
		cmd.Abort("usage: nfs-exports [add|remove] arguments")
	}

	var err error
	switch os.Args[2] {
	case "add":
		err = hyperkit.AddNFSExports(os.Args[3:]...)
	case "remove":
		err = hyperkit.RemoveNFSExports(os.Args[3:]...)
	default:
		err = fmt.Errorf("Unknown nfs-export subcommand: %s", os.Args[2])
	}
	if err != nil {
		cmd.Abort("nfs-export %s failed: %v", os.Args[2], err)
	}
	os.Exit(0)
}
