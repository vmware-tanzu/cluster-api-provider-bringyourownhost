// +build darwin

package gateway

import (
	"net"
	"os/exec"
)

func discoverGatewayOSSpecific() (net.IP, error) {
	routeCmd := exec.Command("/sbin/route", "-n", "get", "0.0.0.0")
	output, err := routeCmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	return parseDarwinRouteGet(output)
}

func discoverGatewayInterfaceOSSpecific() (ip net.IP, err error) {
	return nil, errNotImplemented
}
