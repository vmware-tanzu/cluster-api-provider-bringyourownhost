// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package algo

import (
	"fmt"
	"path/filepath"
	"strings"
)

func NewAptStep(k *BaseK8sInstaller, aptPkg string) Step {
	return NewAptStepEx(k, aptPkg, false)
}

// The step will run only if aptPkg is available
func NewAptStepOptional(k *BaseK8sInstaller, aptPkg string) Step {
	return NewAptStepEx(k, aptPkg, true)
}

func NewAptStepEx(k *BaseK8sInstaller, aptPkg string, optional bool) Step {
	pkgName := strings.Split(aptPkg, ".")[0] // strip deb
	pkgAbsolutePath := filepath.Join(k.BundlePath, aptPkg)

	condCmd := "%s"
	if optional {
		condCmd = fmt.Sprintf("if [ -f %s ]; then %%s; fi", pkgAbsolutePath)
	}

	doCmd := fmt.Sprintf("dpkg --install '%s'", pkgAbsolutePath)
	undoCmd :=fmt.Sprintf("dpkg --purge %s", pkgName)

	return &ShellStep{
		BaseK8sInstaller: k,
		Desc:             pkgName,
		DoCmd:            fmt.Sprintf(condCmd, doCmd),
		UndoCmd:          fmt.Sprintf(condCmd, undoCmd)}
}
