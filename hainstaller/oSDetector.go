package hainstaller

import (
	"errors"
	"log"
	"os/exec"
	"strings"
)

// oSDetector contains all the logic for detecting the OS version.
type oSDetector struct {
}

// newOSDetector is a constructor for OSDetector
func newOSDetector() *oSDetector {
	return &oSDetector{}
}

// detect returns the os info in normalized format.
// The format is as follows: <os>_<ver>_<arch>
// Example with Ubuntu 21.04.3 64bit: Ubuntu_20.04.3_x64
func (osd *oSDetector) detect() (string, error) {
	systemInfo, err := osd.getHostSystemInfo()
	if err != nil {
		return "", err
	}
	os, ver, arch := osd.filterSystemInfo(systemInfo)
	if os == "" || ver == "" || arch == "" {
		err := "Could not detect OS correctly."
		log.Print(err)
		return "", errors.New(err)
	}

	normalizedOS := osd.normalizeOsName(os, ver, arch)
	return normalizedOS, nil
}

// normalizeOsName normalizes given os, arch and k8s version to the correct format.
// Takes as arguments os, ver and arch then returns string in the format <os>_<ver>_<arch>
func (osd *oSDetector) normalizeOsName(os, ver, arch string) string {
	osName := os + " " + ver
	if arch == "x86-64" {
		osName += "_x64"
	} else {
		osName += "_x32"
	}

	osName = strings.ReplaceAll(osName, " ", "_")

	return osName
}

// getHostSystemInfo returns the result after executing a preset command.
// Exact output format varies between different distributions but the important
// part is the line starting with the string  "Operating system:"  which  shows
// the exact version of the operating  system.  This  information  is  used  to
// identify the correct installer that needs to be used. Also used is the  line
// starting with "Architecture: " to identify whether we need the 32 or 64  bit
// bundle.
//
// Example output for running the command on Ubuntu:
//
//  Static hostname: ubuntu
//        Icon name: computer-vm
//          Chassis: vm
//       Machine ID: 242642b0e734472abaf8c5337e1174c4
//          Boot ID: 181f08d651b76h39be5b138231427c5c
//   Virtualization: vmware
// Operating System: Ubuntu 20.04.3 LTS
//           Kernel: Linux 5.11.0-27-generic
//     Architecture: x86-64
func (osd *oSDetector) getHostSystemInfo() (string, error) {
	out, err := exec.Command("hostnamectl").Output()

	if err != nil {
		log.Print(err)
		return "", err
	}

	return string(out), nil
}

// Method that extracts the important information from getHostSystemInfo.
func (osd *oSDetector) filterSystemInfo(systemInfo string) (string, string, string) {
	const strIndicatingOSline string = "Operating System: "
	const strIndicatingArchline string = "Architecture: "
	var os, ver, arch string

	i := strings.LastIndex(systemInfo, strIndicatingOSline) + len(strIndicatingOSline)
	for ; !(systemInfo[i] >= '0' && systemInfo[i] <= '9') && systemInfo[i] != '\n'; i++ {
		os += string(systemInfo[i])
	}
	for ; (systemInfo[i] >= '0' && systemInfo[i] <= '9') || systemInfo[i] == '.'; i++ {
		ver += string(systemInfo[i])
	}
	i = strings.LastIndex(systemInfo, strIndicatingArchline) + len(strIndicatingArchline)
	for ; systemInfo[i] != '\n'; i++ {
		arch += string(systemInfo[i])
	}
	return strings.TrimSpace(os), ver, arch
}
