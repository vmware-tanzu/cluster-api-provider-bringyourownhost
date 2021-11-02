// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package installer

import (
	"flag"
	"fmt"
	"github.com/go-logr/logr"
	"k8s.io/klog/klogr"
)

var (
	listSupportedFlag    = flag.Bool("list-supported", false, "List all supported OS and Kubernetes versions")
	listBundlesFlag      = flag.Bool("list-bundles", false, "List the BYOH Bundle names for all supported OS and Kubernetes versions")
	detectOSFlag         = flag.Bool("detect", false, "Detects the current operating system")
	installFlag          = flag.Bool("install", false, "Install a BYOH Bundle")
	uninstallFlag        = flag.Bool("uninstall", false, "Unnstall a BYOH Bundle")
	bundleRepoFlag       = flag.String("bundle-repo", "projects.registry.vmware.com", "BYOH Bundle Repository")
	cachePathFlag        = flag.String("cache-path", ".", "Path to the local bundle cache")
	k8sFlag              = flag.String("k8s", "1.22.1", "Kubernetes version")
	osFlag               = flag.String("os", "", "OS. If used with install/uninstall, override os detection")
	previewOSChangesFlag = flag.Bool("preview-os-changes", false, "Preview the install and uninstall changes for the specified OS")
)

const (
	doInstall   = true
	doUninstall = false
)

var (
	klogger logr.Logger
)

func Main() {
	klogger = klogr.New()

	flag.Parse()

	if *listSupportedFlag {
		listSupported()
		return
	}

	if *listBundlesFlag {
		listBundles()
		return
	}

	if *detectOSFlag {
		detectOS()
		return
	}

	if *installFlag {
		runInstaller(doInstall)
		return
	}

	if *uninstallFlag {
		runInstaller(doUninstall)
		return
	}

	if *previewOSChangesFlag {
		previewOSChanges()
		return
	}

	fmt.Println("No flag set. See --help")
}

func listSupported() {
	for _, os := range ListSupportedOS() {
		for _, k8s := range ListSupportedK8s(os) {
			fmt.Printf("%s %s\n", os, k8s)
		}
	}
}

func listBundles() {
	for _, os := range ListSupportedOS() {
		for _, k8s := range ListSupportedK8s(os) {
			fmt.Println(GetBundleName(os, k8s))
		}
	}
}

func detectOS() {
	osd := osDetector{}
	os, err := osd.Detect()
	if err != nil {
		klogger.Error(err, "Error detecting OS")
		return
	}

	fmt.Printf("Detected OS as: %s", os)
}

func runInstaller(install bool) {
	var i *installer
	var err error
	if *osFlag != "" {
		// Override current OS detection
		i, err = newUnchecked(*osFlag, *bundleRepoFlag, *cachePathFlag, klogger, &logPrinter{klogger})
		if err != nil {
			klogger.Error(err, "unable to create installer")
			return
		}
	} else {
		i, err = New(*bundleRepoFlag, *cachePathFlag, klogger)
		if err != nil {
			klogger.Error(err, "unable to create installer")
			return
		}
	}

	if install {
		err = i.Install(*k8sFlag)
	} else {
		err = i.Uninstall(*k8sFlag)
	}
	if err != nil {
		klogger.Error(err, "error installing/uninstalling")
	}
}

func previewOSChanges() {
	installChanges, uninstallChanges, err := PreviewChanges(*osFlag, *k8sFlag)
	if err != nil {
		klogger.Error(err, "error previewing changes for os", "os", osFlag, "k8s", *k8sFlag)
		return
	}

	fmt.Printf("Install changes:\n%s\n\n", installChanges)
	fmt.Printf("Uninstall changes:\n%s", uninstallChanges)
}
