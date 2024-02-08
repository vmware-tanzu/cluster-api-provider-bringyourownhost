package gateway

import (
	"errors"
	"net"
	"runtime"
)

var (
	errNoGateway      = errors.New("no gateway found")
	errCantParse      = errors.New("can't parse string output")
	errNotImplemented = errors.New("not implemented for OS: " + runtime.GOOS)
)

// DiscoverGateway is the OS independent function to get the default gateway
func DiscoverGateway() (ip net.IP, err error) {
	return discoverGatewayOSSpecific()
}

// DiscoverInterface is the OS independent function to call to get the default network interface IP that uses the default gateway
func DiscoverInterface() (ip net.IP, err error) {
	return discoverGatewayInterfaceOSSpecific()
}
