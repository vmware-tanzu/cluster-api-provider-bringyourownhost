// +build linux

package gateway

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
)

const (
	// See http://man7.org/linux/man-pages/man8/route.8.html
	file = "/proc/net/route"
)

func discoverGatewayOSSpecific() (ip net.IP, err error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("Can't access %s", file)
	}
	defer f.Close()

	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("Can't read %s", file)
	}
	return parseLinuxGatewayIP(bytes)
}

func discoverGatewayInterfaceOSSpecific() (ip net.IP, err error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("Can't access %s", file)
	}
	defer f.Close()

	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("Can't read %s", file)
	}
	return parseLinuxInterfaceIP(bytes)
}
