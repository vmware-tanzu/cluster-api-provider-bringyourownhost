// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package algo

import (
	"fmt"
	"path/filepath"
	"strings"
)

// NewAptStep returns a new step to install apt package
func NewAptStep(k *BaseK8sInstaller, aptPkg string) Step {
	return NewAptStepEx(k, aptPkg, false)
}

// NewAptStepOptional optional step to install apt package
func NewAptStepOptional(k *BaseK8sInstaller, aptPkg string) Step {
	return NewAptStepEx(k, aptPkg, true)
}

// NewAptStepEx step to install apt packages
func NewAptStepEx(k *BaseK8sInstaller, aptPkg string, optional bool) Step {
	pkgName := strings.Split(aptPkg, ".")[0] // leave only pkg name, strip .deb
	pkgAbsolutePath := filepath.Join(k.BundlePath, aptPkg)

	condCmd := "%s"
	if optional {
		condCmd = fmt.Sprintf("if [ -f %s ]; then %%s; fi", pkgAbsolutePath)
	}
	// apt-mark hold will prevent the package from being automatically installed, upgraded or removed.
	// This is done to prevent unexpected upgrades of the external package in order to ensure that
	// the working environment is stable. If needed the package can be manually upgraded.
	doCmd := fmt.Sprintf("dpkg --install '%s' && apt-mark hold %s", pkgAbsolutePath, pkgName)
	// When uninstalling the package the hold is removed automatically.
	undoCmd := fmt.Sprintf("dpkg --purge %s", pkgName)

	return &ShellStep{
		BaseK8sInstaller: k,
		Desc:             pkgName,
		DoCmd:            fmt.Sprintf(condCmd, doCmd),
		UndoCmd:          fmt.Sprintf(condCmd, undoCmd)}
}
