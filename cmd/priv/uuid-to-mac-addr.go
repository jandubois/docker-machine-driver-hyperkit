package priv

import (
	"fmt"
	"os"

	"github.com/rancher-sandbox/docker-machine-driver-hyperkit/cmd"
	"github.com/rancher-sandbox/docker-machine-driver-hyperkit/pkg/hyperkit"
)

func UUIDtoMacAddr() {
	if len(os.Args) != 3 {
		cmd.Abort("usage: uuid-to-mac-addr UUID")
	}
	mac, err := hyperkit.GetMACAddressFromUUID(os.Args[2])
	if err != nil {
		cmd.Abort("Getting MAC address from UUID failed: %v", err)
	}
	fmt.Println(mac)
	os.Exit(0)
}
