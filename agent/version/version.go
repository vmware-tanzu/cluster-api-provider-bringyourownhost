// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package version

import (
	"fmt"
	"runtime"
)

var (
	GitMajor     string // major version, always numeric
	GitMinor     string // minor version, numeric possibly followed by "+"
	GitVersion   string // semantic version, derived by build scripts
	GitCommit    string // sha1 from git, output of $(git rev-parse HEAD)
	GitTreeState string // state of git tree, either "clean" or "dirty"
	BuildDate    string // build date in ISO8601 format, output of $(date -u +'%Y-%m-%dT%H:%M:%SZ')
)

// Info exposes information about the version used for the current running code.
type Info struct {
	Major        string `json:"major,omitempty"`
	Minor        string `json:"minor,omitempty"`
	GitVersion   string `json:"gitVersion,omitempty"`
	GitCommit    string `json:"gitCommit,omitempty"`
	GitTreeState string `json:"gitTreeState,omitempty"`
	BuildDate    string `json:"buildDate,omitempty"`
	GoVersion    string `json:"goVersion,omitempty"`
	Compiler     string `json:"compiler,omitempty"`
	Platform     string `json:"platform,omitempty"`
}

// Get returns an Info object with all the information about the current running code.
func Get() Info {
	return Info{
		Major:        GitMajor,
		Minor:        GitMinor,
		GitVersion:   GitVersion,
		GitCommit:    GitCommit,
		GitTreeState: GitTreeState,
		BuildDate:    BuildDate,
		GoVersion:    runtime.Version(),
		Compiler:     runtime.Compiler,
		Platform:     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}
