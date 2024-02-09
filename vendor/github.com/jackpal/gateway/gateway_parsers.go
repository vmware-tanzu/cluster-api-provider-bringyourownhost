package gateway

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

type windowsRouteStruct struct {
	Destination string
	Netmask     string
	Gateway     string
	Interface   string
	Metric      string
}

type linuxRouteStruct struct {
	Iface       string
	Destination string
	Gateway     string
	Flags       string
	RefCnt      string
	Use         string
	Metric      string
	Mask        string
	MTU         string
	Window      string
	IRTT        string
}

func parseToWindowsRouteStruct(output []byte) (windowsRouteStruct, error) {
	// Windows route output format is always like this:
	// ===========================================================================
	// Interface List
	// 8 ...00 12 3f a7 17 ba ...... Intel(R) PRO/100 VE Network Connection
	// 1 ........................... Software Loopback Interface 1
	// ===========================================================================
	// IPv4 Route Table
	// ===========================================================================
	// Active Routes:
	// Network Destination        Netmask          Gateway       Interface  Metric
	//           0.0.0.0          0.0.0.0      192.168.1.1    192.168.1.100     20
	// ===========================================================================
	//
	// Windows commands are localized, so we can't just look for "Active Routes:" string
	// I'm trying to pick the active route,
	// then jump 2 lines and get the row
	// Not using regex because output is quite standard from Windows XP to 8 (NEEDS TESTING)
	lines := strings.Split(string(output), "\n")
	sep := 0
	for idx, line := range lines {
		if sep == 3 {
			// We just entered the 2nd section containing "Active Routes:"
			if len(lines) <= idx+2 {
				return windowsRouteStruct{}, errNoGateway
			}

			fields := strings.Fields(lines[idx+2])
			if len(fields) < 5 {
				return windowsRouteStruct{}, errCantParse
			}

			return windowsRouteStruct{
				Destination: fields[0],
				Netmask:     fields[1],
				Gateway:     fields[2],
				Interface:   fields[3],
				Metric:      fields[4],
			}, nil
		}
		if strings.HasPrefix(line, "=======") {
			sep++
			continue
		}
	}
	return windowsRouteStruct{}, errNoGateway
}

func parseToLinuxRouteStruct(output []byte) (linuxRouteStruct, error) {
	// parseLinuxProcNetRoute parses the route file located at /proc/net/route
	// and returns the IP address of the default gateway. The default gateway
	// is the one with Destination value of 0.0.0.0.
	//
	// The Linux route file has the following format:
	//
	// $ cat /proc/net/route
	//
	// Iface   Destination Gateway     Flags   RefCnt  Use Metric  Mask
	// eno1    00000000    C900A8C0    0003    0   0   100 00000000    0   00
	// eno1    0000A8C0    00000000    0001    0   0   100 00FFFFFF    0   00
	const (
		sep              = "\t" // field separator
		destinationField = 1    // field containing hex destination address
		gatewayField     = 2    // field containing hex gateway address
	)
	scanner := bufio.NewScanner(bytes.NewReader(output))

	// Skip header line
	if !scanner.Scan() {
		return linuxRouteStruct{}, errors.New("Invalid linux route file")
	}

	for scanner.Scan() {
		row := scanner.Text()
		tokens := strings.Split(row, sep)
		if len(tokens) < 11 {
			return linuxRouteStruct{}, fmt.Errorf("invalid row '%s' in route file: doesn't have 11 fields", row)
		}

		// Cast hex destination address to int
		destinationHex := "0x" + tokens[destinationField]
		destination, err := strconv.ParseInt(destinationHex, 0, 64)
		if err != nil {
			return linuxRouteStruct{}, fmt.Errorf(
				"parsing destination field hex '%s' in row '%s': %w",
				destinationHex,
				row,
				err,
			)
		}

		// The default interface is the one that's 0
		if destination != 0 {
			continue
		}

		return linuxRouteStruct{
			Iface:       tokens[0],
			Destination: tokens[1],
			Gateway:     tokens[2],
			Flags:       tokens[3],
			RefCnt:      tokens[4],
			Use:         tokens[5],
			Metric:      tokens[6],
			Mask:        tokens[7],
			MTU:         tokens[8],
			Window:      tokens[9],
			IRTT:        tokens[10],
		}, nil
	}
	return linuxRouteStruct{}, errors.New("interface with default destination not found")
}

func parseWindowsGatewayIP(output []byte) (net.IP, error) {
	parsedOutput, err := parseToWindowsRouteStruct(output)
	if err != nil {
		return nil, err
	}

	ip := net.ParseIP(parsedOutput.Gateway)
	if ip == nil {
		return nil, errCantParse
	}
	return ip, nil
}

func parseWindowsInterfaceIP(output []byte) (net.IP, error) {
	parsedOutput, err := parseToWindowsRouteStruct(output)
	if err != nil {
		return nil, err
	}

	ip := net.ParseIP(parsedOutput.Interface)
	if ip == nil {
		return nil, errCantParse
	}
	return ip, nil
}

func parseLinuxGatewayIP(output []byte) (net.IP, error) {

	parsedStruct, err := parseToLinuxRouteStruct(output)
	if err != nil {
		return nil, err
	}

	destinationHex := "0x" + parsedStruct.Destination
	gatewayHex := "0x" + parsedStruct.Gateway

	// cast hex address to uint32
	d, err := strconv.ParseInt(gatewayHex, 0, 64)
	if err != nil {
		return nil, fmt.Errorf(
			"parsing default interface address field hex '%s': %w",
			destinationHex,
			err,
		)
	}
	// make net.IP address from uint32
	ipd32 := make(net.IP, 4)
	binary.LittleEndian.PutUint32(ipd32, uint32(d))

	// format net.IP to dotted ipV4 string
	return net.IP(ipd32), nil
}

func parseLinuxInterfaceIP(output []byte) (net.IP, error) {
	parsedStruct, err := parseToLinuxRouteStruct(output)
	if err != nil {
		return nil, err
	}

	iface, err := net.InterfaceByName(parsedStruct.Iface)
	if err != nil {
		return nil, err
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return nil, err
	}

	// split when its 192.168.8.8/24
	ipString := strings.Split(addrs[0].String(), "/")[0]
	ip := net.ParseIP(ipString)
	if ip == nil {
		return nil, fmt.Errorf("invalid addr %s", ipString)
	}
	return ip, nil
}

func parseDarwinRouteGet(output []byte) (net.IP, error) {
	// Darwin route out format is always like this:
	//    route to: default
	// destination: default
	//        mask: default
	//     gateway: 192.168.1.1
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "gateway:" {
			ip := net.ParseIP(fields[1])
			if ip != nil {
				return ip, nil
			}
		}
	}

	return nil, errNoGateway
}

func parseBSDSolarisNetstat(output []byte) (net.IP, error) {
	// netstat -rn produces the following on FreeBSD:
	// Routing tables
	//
	// Internet:
	// Destination        Gateway            Flags      Netif Expire
	// default            10.88.88.2         UGS         em0
	// 10.88.88.0/24      link#1             U           em0
	// 10.88.88.148       link#1             UHS         lo0
	// 127.0.0.1          link#2             UH          lo0
	//
	// Internet6:
	// Destination                       Gateway                       Flags      Netif Expire
	// ::/96                             ::1                           UGRS        lo0
	// ::1                               link#2                        UH          lo0
	// ::ffff:0.0.0.0/96                 ::1                           UGRS        lo0
	// fe80::/10                         ::1                           UGRS        lo0
	// ...
	outputLines := strings.Split(string(output), "\n")
	for _, line := range outputLines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "default" {
			ip := net.ParseIP(fields[1])
			if ip != nil {
				return ip, nil
			}
		}
	}

	return nil, errNoGateway
}
