// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package algo

import (
	"fmt"
	"path/filepath"
	"strings"
)

func NewAptStep(k *BaseK8sInstaller, aptPkg string) Step {
	pkgName := strings.Split(aptPkg, ".")[0]
	pkgAbsolutePath := filepath.Join(k.BundlePath, aptPkg)

	return &ShellStep{
		BaseK8sInstaller: k,
		Desc:             pkgName,
		DoCmd:            fmt.Sprintf("apt install -y '%s'", pkgAbsolutePath),
		UndoCmd:          fmt.Sprintf("apt remove -y %s", pkgName)}
}
