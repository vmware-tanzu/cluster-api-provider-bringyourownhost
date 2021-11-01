// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package installer

import (
	"flag"
	"fmt"
	"k8s.io/klog"
	"k8s.io/klog/klogr"
)

var (
	listFlag             = flag.Bool("list", false, "List all supported OS and Kubernetes versions")
	listBundlesFlag      = flag.Bool("listBundles", false, "List the BYOH Bundle names for all supported OS and Kubernetes versions")
	detectOSFlag         = flag.Bool("detect", false, "Detects the current operating system")
	installFlag          = flag.Bool("install", false, "Install a BYOH Bundle")
	uninstallFlag        = flag.Bool("uninstall", false, "Unnstall a BYOH Bundle")
	bundleRepoFlag       = flag.String("bundleRepo", "projects.registry.vmware.com", "BYOH Bundle Repository")
	k8sFlag              = flag.String("k8s", "1.22.1", "Kubernetes version")
	osFlag               = flag.String("os", "", "OS. If used with install/uninstall, skip os detection")
	previewOSChangesFlag = flag.Bool("previewOSChanges", false, "Preview the install and uninstall changes for the specified OS")
)

const (
	doInstall   = true
	doUninstall = false
)

func Main() {
	flag.Parse()

	if *listFlag {
		list()
	}

	if *listBundlesFlag {
		listBundles()
	}

	if *detectOSFlag {
		detectOS()
	}

	if *installFlag {
		runInstaller(doInstall)
	}

	if *uninstallFlag {
		runInstaller(doUninstall)
	}

	if *previewOSChangesFlag {
		previewOSChanges()
	}
}

func list() {
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
		fmt.Printf("Error detecting OS %s", err)
		return
	}

	fmt.Printf("Detected OS as: %s", os)
}

func runInstaller(install bool) {
	klog.InitFlags(nil)
	klogger := klogr.New()

	if *bundleRepoFlag == "" {
		bd := bundleDownloader{repoAddr : "", downloadPath : "."}
		fmt.Printf("Bundle repo not specified. Provide bundle contents in %s\n", bd.GetBundleDirPath(*k8sFlag))
	}

	var i *installer
	var err error
	if *osFlag != "" {
		// Override current OS detection
		i, err = newUnchecked(*osFlag, *bundleRepoFlag, ".", klogger, &logPrinter{klogger})
		if err != nil {
			fmt.Println(err)
			return
		}
	} else {
		i, err = New("norepo", ".", klogger)
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	if install {
		err = i.Install(*k8sFlag)
	} else {
		err = i.Uninstall(*k8sFlag)
	}
	if err != nil {
		fmt.Println(err)
	}
}

func previewOSChanges() {
	installChanges, uninstallChanges, err := PreviewChanges(*osFlag, *k8sFlag)
	if err != nil {
		fmt.Printf("Error previewing changes for os '%s' k8s '%s' %s", *osFlag, *k8sFlag, err)
		return
	}

	fmt.Printf("Install changes:\n%s\n\n", installChanges)
	fmt.Printf("Uninstall changes:\n%s", uninstallChanges)
}
