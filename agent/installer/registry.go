// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package installer

import (
	"fmt"
	"regexp"
)

type osk8sInstaller interface{}
type k8sInstallerMap map[string]osk8sInstaller
type osk8sInstallerMap map[string]k8sInstallerMap
type aliasOSMap map[string]string

// Registry associates a (OS,K8sVersion) pair with an installer
type registry struct {
	osk8sInstallerMap
	aliasOSMap
}

func NewRegistry() registry {
	return registry{make(osk8sInstallerMap), make(aliasOSMap)}
}

func (r *registry) Add(os, k8sVer string, installer osk8sInstaller) {
	if _, ok := r.osk8sInstallerMap[os]; !ok {
		r.osk8sInstallerMap[os] = make(k8sInstallerMap)
	}

	if _, alreadyExist := r.osk8sInstallerMap[os][k8sVer]; alreadyExist {
		panic(fmt.Sprintf("%v %v already exists", os, k8sVer))
	}

	r.osk8sInstallerMap[os][k8sVer] = installer
}

func (r *registry) AddOSAlias(osFilter, os string) {
	if _, alreadyExist := r.aliasOSMap[osFilter]; alreadyExist {
                panic(fmt.Sprintf("alias %v already exists", osFilter))
        }

	r.aliasOSMap[osFilter] = os
}

func (r *registry) ListOS() ([]string, aliasOSMap) {
	result := make([]string, 0, len(r.osk8sInstallerMap))

	for os := range(r.osk8sInstallerMap) {
		result = append(result, os)
	}

	return result, r.aliasOSMap
}

func (r *registry) ListK8s(os string) []string {
	result := make([]string, 0, len(r.osk8sInstallerMap[os]))
	for k8s := range(r.resolveK8sInstallerMap(os)) {
		result = append(result, k8s)
	}
	return result
}

func (r *registry) GetInstaller(os, k8sVer string) osk8sInstaller {
	return r.resolveK8sInstallerMap(os)[k8sVer]
}

func (r* registry) resolveK8sInstallerMap(os string) k8sInstallerMap {
	if res, ok := r.osk8sInstallerMap[os]; ok {
		// There is direct match for the os
		return res
	}

	// Try to find os using the alias map
	for k,v := range r.aliasOSMap {
		matched, _ := regexp.MatchString(k, os)
		if matched {
			return r.osk8sInstallerMap[v]
		}
	}

	return nil
}
