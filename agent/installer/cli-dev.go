// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package installer

import (
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/go-logr/logr"
	"k8s.io/klog/v2/klogr"
)

var (
	listSupportedFlag    *bool
	detectOSFlag         *bool
	installFlag          *bool
	uninstallFlag        *bool
	bundleRepoFlag       *string
	cachePathFlag        *string
	k8sFlag              *string
	osFlag               *string
	tagFlag              *string
	previewOSChangesFlag *bool
)

const (
	doInstall   = true
	doUninstall = false
)

var (
	klogger logr.Logger
)

// Main entry point for the installer dev/test CLI
func Main() {
	klogger = klogr.New()

	listSupportedFlag = flag.Bool("list-supported", false, "List all supported OS, Kubernetes versions and BYOH Bundle names")
	detectOSFlag = flag.Bool("detect", false, "Detects the current operating system")
	installFlag = flag.Bool("install", false, "Install a BYOH Bundle")
	uninstallFlag = flag.Bool("uninstall", false, "Unnstall a BYOH Bundle")
	bundleRepoFlag = flag.String("bundle-repo", "projects.registry.vmware.com", "BYOH Bundle Repository")
	cachePathFlag = flag.String("cache-path", ".", "Path to the local bundle cache")
	k8sFlag = flag.String("k8s", "1.22.1", "Kubernetes version")
	osFlag = flag.String("os", "", "OS. If used with install/uninstall, override os detection")
	tagFlag = flag.String("tag", "", "BYOH Bundle tag")
	previewOSChangesFlag = flag.Bool("preview-os-changes", false, "Preview the install and uninstall changes for the specified OS")

	flag.Parse()

	if *listSupportedFlag {
		listSupported()
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
	w := new(tabwriter.Writer)
	const (
		minwidth = 8
		tabwidth = 8
		padding  = 0
		flags    = 0
	)
	w.Init(os.Stdout, minwidth, tabwidth, padding, '\t', flags)
	defer func() {
		err := w.Flush()
		if err != nil {
			klogger.Error(err, "Failed to flush the tabwriter")
		}
	}()
	_, err := fmt.Fprintf(w, "The corresponding bundles (particular to a patch version) should be pushed to the OCI registry of choice\n"+
		"By default, BYOH uses projects.registry.vmware.com\n\n"+
		"Note: It may happen that a specific patch version of a k8s minor release is not available in the OCI registry\n\n")
	if err != nil {
		klogger.Error(err, "Failed to write to tabwriter")
	}
	_, err = fmt.Fprintf(w, "%s\t%s\t%s\n", "OS", "K8S Version", "BYOH Bundle Name")
	if err != nil {
		klogger.Error(err, "Failed to write to tabwriter")
	}
	_, err = fmt.Fprintf(w, "%s\t%s\t%s\n", "---", "-----------", "----------------")
	if err != nil {
		klogger.Error(err, "Failed to write to tabwriter")
	}
	osFilters, osBundles := ListSupportedOS()
	for i := range osFilters {
		for _, k8s := range ListSupportedK8s(osBundles[i]) {
			_, err = fmt.Fprintf(w, "%s\t %s\t%s:%s\n", osFilters[i], k8s, GetBundleName(osBundles[i]), k8s)
			if err != nil {
				klogger.Error(err, "Failed to write to tabwriter")
			}
		}
	}
}

func detectOS() {
	osd := osDetector{}
	detectedOs, err := osd.Detect()
	if err != nil {
		klogger.Error(err, "Error detecting OS")
		return
	}

	fmt.Printf("Detected OS as: %s", detectedOs)
}

func runInstaller(install bool) {
	var i *installer
	var err error
	if *osFlag != "" {
		// Override current OS detection
		i, err = newUnchecked(*osFlag, BundleTypeK8s, *cachePathFlag, klogger, &logPrinter{klogger})
		if err != nil {
			klogger.Error(err, "unable to create installer")
			return
		}
	} else {
		i, err = New(*cachePathFlag, BundleTypeK8s, klogger)
		if err != nil {
			klogger.Error(err, "unable to create installer")
			return
		}
	}
	if install {
		err = i.Install(*bundleRepoFlag, *k8sFlag, *tagFlag)
	} else {
		err = i.Uninstall(*bundleRepoFlag, *k8sFlag, *tagFlag)
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
