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
type filterBundlePair struct {
	osFilter string
	osBundle string
}
type filterBundleList []filterBundlePair

// Registry contains
// 1. Entries associating BYOH Bundle i.e. (OS,K8sVersion) in the Repository with Installer in Host Agent
// 2. Entries that match a concrete OS to a BYOH Bundle OS from the Repository
type registry struct {
	osk8sInstallerMap
	filterBundleList
}

func newRegistry() registry {
	return registry{osk8sInstallerMap: make(osk8sInstallerMap)}
}

func (r *registry) AddBundleInstaller(os, k8sVer string, installer osk8sInstaller) {
	if _, ok := r.osk8sInstallerMap[os]; !ok {
		r.osk8sInstallerMap[os] = make(k8sInstallerMap)
	}

	if _, alreadyExist := r.osk8sInstallerMap[os][k8sVer]; alreadyExist {
		panic(fmt.Sprintf("%v %v already exists", os, k8sVer))
	}

	r.osk8sInstallerMap[os][k8sVer] = installer
}

func (r *registry) AddOsFilter(osFilter, osBundle string) {
	r.filterBundleList = append(r.filterBundleList, filterBundlePair{osFilter: osFilter, osBundle: osBundle})
}

func (r *registry) ListOS() (osFilter, osBundle []string) {
	osFilter = make([]string, 0, len(r.filterBundleList))
	osBundle = make([]string, 0, len(r.filterBundleList))

	for _, fbp := range r.filterBundleList {
		osFilter = append(osFilter, fbp.osFilter)
		osBundle = append(osBundle, fbp.osBundle)
	}

	return
}

func (r *registry) ListK8s(osBundleHost string) []string {
	var result []string

	// os bundle
	if k8sMap, ok := r.osk8sInstallerMap[osBundleHost]; ok {
		for k8s := range k8sMap {
			result = append(result, k8s)
		}

		return result
	}

	// os host
	for k8s := range r.osk8sInstallerMap[r.resolveOsToOsBundle(osBundleHost)] {
		result = append(result, k8s)
	}

	return result
}

func (r *registry) GetInstaller(osHost, k8sVer string) (osk8si osk8sInstaller, osBundle string) {
	osBundle = r.resolveOsToOsBundle(osHost)
	osk8si = r.osk8sInstallerMap[osBundle][k8sVer]
	return
}

func (r *registry) resolveOsToOsBundle(os string) string {
	for _, fbp := range r.filterBundleList {
		matched, _ := regexp.MatchString(fbp.osFilter, os)
		if matched {
			return fbp.osBundle
		}
	}

	return ""
}
