// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package version

import (
	"fmt"
	"runtime"
	"strings"
)

var (
	// Version is the version of the agent.
	Version string
	// BuildDate is the date the agent was built.
	BuildDate string
)

const (
	// Dev development version string
	Dev          = "dev"
	gitTagLength = 3
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
	if Version == Dev {
		*major = Dev
		return
	}

	version := strings.Split(Version, ".")
	if len(version) != gitTagLength {
		return
	}

	// The git tag is preceded by a 'v', eg. v1.2.3
	if len(version[0]) != 2 || version[0][0:1] != "v" {
		return
	}

	*major = version[0][1:2]
	*minor = version[1]
	*patch = version[2]
}
