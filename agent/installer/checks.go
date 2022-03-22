// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package installer

import (
	"fmt"
	"runtime"

	"github.com/go-logr/logr"
	utilsexec "k8s.io/utils/exec"
)

func checkPreRequsitePackages() error {
	if runtime.GOOS == "linux" {
		unavailablePackages := make([]string, 0)
		execr := utilsexec.New()
		for _, pkgName := range preRequisitePackages {
			_, err := execr.LookPath(pkgName)
			if err != nil {
				unavailablePackages = append(unavailablePackages, pkgName)
			}
		}
		if len(unavailablePackages) != 0 {
			return fmt.Errorf("required package(s): %s not found", unavailablePackages)
		}
		return nil
	}
	return nil
}

func runPrechecks(logger logr.Logger, os string) bool {
	precheckSuccessful := true

	// verify that packages are available when user has chosen to install kubernetes binaries
	err := checkPreRequsitePackages()
	if err != nil {
		logger.Error(err, "Failed pre-requisite packages precheck")
		precheckSuccessful = false
	}
	return precheckSuccessful
}
