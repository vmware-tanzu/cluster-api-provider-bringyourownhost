// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package installer

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

const (
	osNotDetected = "could not detect OS correctly"
)

// oSDetector contains all the logic for detecting the OS version.
type osDetector struct {
	cachedNormalizedOS string
}

// Detect returns the os info in normalized format.
// The format is as follows: <os>_<ver>_<arch>
// Example with Ubuntu 21.04.3 64bit: Ubuntu_20.04.3_x64
func (osd *osDetector) Detect() (string, error) {
	return osd.DetectByHostnamectl(func() (string, error) { return osd.getHostnamectl() })
}

// DetectByHostnamectl is a helper method to enable testing of detect with mock methods.
func (osd *osDetector) DetectByHostnamectl(f func() (string, error)) (string, error) {
	if osd.cachedNormalizedOS != "" {
		return osd.cachedNormalizedOS, nil
	}

	systemInfo, err := f()
	if err != nil {
		return "", err
	}
	osDetails := parseHostnamectl(systemInfo)
	os := osDetails[0]
	ver := osDetails[1]
	arch := osDetails[2]
	if os == "" || ver == "" || arch == "" {
		return "", errors.New(osNotDetected)
	}

	osd.cachedNormalizedOS = normalizeOSName(os, ver, arch)
	return osd.cachedNormalizedOS, nil
}

// normalizeOsName normalizes given os, arch and k8s version to the correct format.
// Takes as arguments os, ver and arch then returns string in the format <os>_<ver>_<arch>
func normalizeOSName(os, ver, arch string) string {
	osName := fmt.Sprintf("%s_%s_%s", os, ver, arch)
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
func (osd *osDetector) getHostnamectl() (string, error) {
	out, err := exec.Command("hostnamectl").Output()

	if err != nil {
		return "", err
	}

	return string(out), nil
}

// Method that extracts the important information from getHostSystemInfo.
func parseHostnamectl(systemInfo string) [3]string {
	const strIndicatingOSline string = "Operating System: "
	const strIndicatingArchline string = "Architecture: "

	var os, ver, arch string

	osRegex := regexp.MustCompile(strIndicatingOSline + `[a-zA-Z]+[ a-zA-Z]*[a-zA-Z]+`)
	locOS := osRegex.FindIndex([]byte(systemInfo))
	if locOS != nil {
		os = systemInfo[locOS[0]+len(strIndicatingOSline) : locOS[1]]
	}

	verRegex := regexp.MustCompile(strIndicatingOSline + `[a-zA-Z]+[ a-zA-Z]* (\d+(\.\d+)*)`)
	locVer := verRegex.FindIndex([]byte(systemInfo))
	if locVer != nil {
		ver = systemInfo[locOS[1]+1 : locVer[1]]
	}

	archRegex := regexp.MustCompile(strIndicatingArchline + `[a-zA-Z]+[ a-zA-Z0-9-]*`)
	locArch := archRegex.FindIndex([]byte(systemInfo))
	if locArch != nil {
		arch = systemInfo[locArch[0]+len(strIndicatingArchline) : locArch[1]]
	}

	return [3]string{os, ver, arch}
}
