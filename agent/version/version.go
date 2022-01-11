// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package version

import (
	"fmt"
	"runtime"
	"strings"
)

var (
	Version   string
	BuildDate string
)

// Info exposes information about the version used for the current running code.
type Info struct {
	Major     string `json:"major,omitempty"`
	Minor     string `json:"minor,omitempty"`
	Patch     string `json:"patch,omitempty"`
	BuildDate string `json:"BuildDate,omitempty"`
	GoVersion string `json:"goVersion,omitempty"`
	Platform  string `json:"platform,omitempty"`
	Compiler  string `json:"compiler,omitempty"`
}

// Get returns an Info object with all the information about the current running code.
func Get() Info {
	var major, minor, patch string
	extractVersion(&major, &minor, &patch)
	return Info{
		Major:     major,
		Minor:     minor,
		Patch:     patch,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
		Compiler:  runtime.Compiler,
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

func extractVersion(major, minor, patch *string) {

	if Version == "dev" {
		*major = "dev"
		return
	}

	version := strings.Split(Version, ".")
	if len(version) != 3 {
		return
	}

	*major = version[0]
	*minor = version[1]
	*patch = version[2]

}
