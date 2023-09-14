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
type filterOsBundlePair struct {
	osFilter string
	osBundle string
}

type filterK8sBundle struct {
	k8sFilter string
}

type filterOSBundleList []filterOsBundlePair
type filterK8sBundleList []filterK8sBundle

// Registry contains
// 1. Entries associating BYOH Bundle i.e. (OS,K8sVersion) in the Repository with Installer in Host Agent
// 2. Entries that match a concrete OS to a BYOH Bundle OS from the Repository
// 3. Entries that match a Major & Minor versions of K8s to any of their patch sub-versions (e.g.: 1.22.3 -> 1.22.*)
type registry struct {
	osk8sInstallerMap
	filterOSBundleList
	filterK8sBundleList
}

func newRegistry() registry {
	return registry{osk8sInstallerMap: make(osk8sInstallerMap)}
}

// AddBundleInstaller adds a bundle installer to the registry
func (r *registry) AddBundleInstaller(os, k8sVer string) {
	var empty interface{}

	if _, ok := r.osk8sInstallerMap[os]; !ok {
		r.osk8sInstallerMap[os] = make(k8sInstallerMap)
	}

	if _, alreadyExist := r.osk8sInstallerMap[os][k8sVer]; alreadyExist {
		panic(fmt.Sprintf("%v %v already exists", os, k8sVer))
	}

	r.osk8sInstallerMap[os][k8sVer] = empty
}

// AddOsFilter adds an OS filter to the filtered bundle list of registry
func (r *registry) AddOsFilter(osFilter, osBundle string) {
	r.filterOSBundleList = append(r.filterOSBundleList, filterOsBundlePair{osFilter: osFilter, osBundle: osBundle})
}

func (r *registry) AddK8sFilter(k8sFilter string) {
	r.filterK8sBundleList = append(r.filterK8sBundleList, filterK8sBundle{k8sFilter: k8sFilter})
}

// ListOS returns a list of OSes supported by the registry
func (r *registry) ListOS() (osFilter, osBundle []string) {
	osFilter = make([]string, 0, len(r.filterOSBundleList))
	osBundle = make([]string, 0, len(r.filterOSBundleList))

	for _, fbp := range r.filterOSBundleList {
		osFilter = append(osFilter, fbp.osFilter)
		osBundle = append(osBundle, fbp.osBundle)
	}

	return
}

// ListK8s returns a list of K8s versions supported by the registry
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
	for k8s := range r.osk8sInstallerMap[r.ResolveOsToOsBundle(osBundleHost)] {
		result = append(result, k8s)
	}

	return result
}

func (r *registry) ResolveOsToOsBundle(os string) string {
	for _, fbp := range r.filterOSBundleList {
		matched, _ := regexp.MatchString(fbp.osFilter, os)
		if matched {
			return fbp.osBundle
		}
	}

	return ""
}

// GetSupportedRegistry returns a registry with installers for the supported OS and K8s
func GetSupportedRegistry() registry {
	reg := newRegistry()

	{
		// Ubuntu

		// BYOH Bundle Repository. Associate bundle with installer
		linuxDistro := "Ubuntu_20.04.1_x86-64"
		reg.AddBundleInstaller(linuxDistro, "v1.23.*")
		reg.AddBundleInstaller(linuxDistro, "v1.24.*")
		reg.AddBundleInstaller(linuxDistro, "v1.25.*")

		/*
		 * PLACEHOLDER - ADD MORE K8S VERSIONS HERE
		 */

		// Match any patch version of the specified Major & Minor K8s version
		reg.AddK8sFilter("v1.22.*")
		reg.AddK8sFilter("v1.23.*")
		reg.AddK8sFilter("v1.24.*")
		reg.AddK8sFilter("v1.25.*")

		// Match concrete os version to repository os version
		reg.AddOsFilter("Ubuntu_20.04.*_x86-64", linuxDistro)

		/*
		 * PLACEHOLDER - POINT MORE DISTRO VERSIONS
		 */
	}

	{
		// Rocky

		// BYOH Bundle Repository. Associate bundle with installer
		linuxDistro := "Rocky_Linux_8.7_x86-64"
		reg.AddBundleInstaller(linuxDistro, "v1.23.*")
		reg.AddBundleInstaller(linuxDistro, "v1.24.*")
		reg.AddBundleInstaller(linuxDistro, "v1.25.*")

		/*
		 * PLACEHOLDER - ADD MORE K8S VERSIONS HERE
		 */

		// Match any patch version of the specified Major & Minor K8s version
		reg.AddK8sFilter("v1.22.*")
		reg.AddK8sFilter("v1.23.*")
		reg.AddK8sFilter("v1.24.*")
		reg.AddK8sFilter("v1.25.*")

		// Match concrete os version to repository os version
		reg.AddOsFilter("Rocky_Linux_8.*_x86-64", linuxDistro)

		/*
		 * PLACEHOLDER - POINT MORE DISTRO VERSIONS
		 */
	}

	/*
	 * PLACEHOLDER - ADD MORE OS HERE
	 */

	return reg
}
