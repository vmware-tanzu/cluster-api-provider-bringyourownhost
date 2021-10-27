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
	pkgName := strings.Split(aptPkg, ".")[0]
	pkgAbsolutePath := filepath.Join(k.BundlePath, aptPkg)
	pkgCheck := ""
	if optional {
		pkgCheck = fmt.Sprintf("test -e %s && ", pkgAbsolutePath)
	}

	return &ShellStep{
		BaseK8sInstaller: k,
		Desc:             pkgName,
		DoCmd:            fmt.Sprintf("%sapt install -y '%s'", pkgCheck, pkgAbsolutePath),
		UndoCmd:          fmt.Sprintf("%sapt remove -y %s", pkgCheck, pkgName)}
}
