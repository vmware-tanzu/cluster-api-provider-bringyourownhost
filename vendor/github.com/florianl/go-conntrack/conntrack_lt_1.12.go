//+build !go1.12

package conntrack

import (
	"log"

	"github.com/florianl/go-conntrack/internal/unix"

	"github.com/mdlayher/netlink"
)

// Open a connection to the conntrack subsystem
func Open(config *Config) (*Nfct, error) {
	var nfct Nfct

	con, err := netlink.Dial(unix.NETLINK_NETFILTER, &netlink.Config{NetNS: config.NetNS, DisableNSLockThread: config.DisableNSLockThread})
	if err != nil {
		return nil, err
	}
	nfct.Con = con

	if config.Logger == nil {
		nfct.logger = log.New(new(devNull), "", 0)
	} else {
		nfct.logger = config.Logger
	}
	nfct.setWriteTimeout = func() error { return nil }

	return &nfct, nil
}
