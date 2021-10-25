// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package installer

import (
	"fmt"
	"github.com/go-logr/logr"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/agent/installer/internal/algo"
)

type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrDetectOs          = Error("Error detecting OS")
	ErrOsK8sNotSupported = Error("No k8s support for OS")
	ErrBundleDownload    = Error("Error downloading bundle")
	ErrBundleExtract     = Error("Error extracting bundle")
	ErrBundleInstall     = Error("Error installing bundle")
	ErrBundleUninstall   = Error("Error uninstalling bundle")
)

type installer struct {
	bundleRepo   string
	downloadPath string
	detectedOs   string
	logger       logr.Logger
	algoRegistry registry
	bundleDownloader
}

// Preview mode executes everything escept the actual commands
var previewMode = true

// getSupportedRegistry returns a registry with installers for the supported OS and K8s
func getSupportedRegistry(downloadPath string, logger logr.Logger) registry {
	var supportedOsK8s = []struct {
		os   string
		k8s  string
		algo algo.K8sStepProvider
	}{
		{"Ubuntu_20.04.1_x86-64", "1_22", &algo.Ubuntu20_4K8s1_22{}},
		/*
		 * ADD HERE to add support for new os or k8s
		 * You may map new versions to old classes if they do the job
		 */
	}

	reg := NewRegistry()
	lp := logPrinter{logger}
	bd := bundleDownloader{downloadPath: downloadPath}
	for _, t := range supportedOsK8s {
		var bundlePath string
		if !previewMode {
			bundlePath = bd.GetBundleDirPath(t.k8s)
		}

		a := algo.BaseK8sInstaller{
			K8sStepProvider: t.algo,
			BundlePath: bundlePath,
			OutputBuilder:   &lp}
		reg.Add(t.os, t.k8s, a)
	}

	return reg
}

// New returns an installer that downloads bundles for the current OS from OCI repository with
// address bundleRepo and stores them under downloadPath. Download path is created,
// if it does not exist.
func New(bundleRepo, downloadPath string, logger logr.Logger) (*installer, error) {
	if bundleRepo == "" {
		return nil, fmt.Errorf("empty bundle repo")
	}
	if downloadPath == "" {
		return nil, fmt.Errorf("empty download path")
	}

	osDetector := osDetector{}
	os, err := osDetector.Detect()
	logger.Info("Detected", "OS", os)
	if err != nil {
		return nil, ErrDetectOs
	}

	reg := getSupportedRegistry(downloadPath, logger)
	if len(reg.ListK8s(os)) == 0 {
		return nil, ErrOsK8sNotSupported
	}

	return &installer{bundleRepo: bundleRepo,
		downloadPath: downloadPath,
		logger:       logger,
		algoRegistry: reg,
		detectedOs:   os}, nil
}

// Install installs the specified k8s version on the current OS
func (i *installer) Install(k8sVer string) error {
	algoInst, err := i.getAlgoInstallerWithBundle(k8sVer)
	if err != nil {
		return err
	}
	err = algoInst.(algo.Installer).Install()
	if err != nil {
		return ErrBundleInstall
	}

	return nil
}

// Uninstal uninstalls the specified k8s version on the current OS
func (i *installer) Uninstall(k8sVer string) error {
	algoInst, err := i.getAlgoInstallerWithBundle(k8sVer)
	if err != nil {
		return err
	}
	err = algoInst.(algo.Installer).Uninstall()
	if err != nil {
		return ErrBundleUninstall
	}

	return nil
}

// getAlgoInstallerWithBundle returns an algo.Installer instance and downloads its bundle
func (i *installer) getAlgoInstallerWithBundle(k8sVer string) (osk8sInstaller, error) {
	// This OS supports at least 1 k8s version. See New.

	algoInst := i.algoRegistry.GetInstaller(i.detectedOs, k8sVer)
	if algoInst != nil {
		return nil, ErrOsK8sNotSupported
	}

	bdErr := i.bundleDownloader.Download(i.detectedOs, k8sVer)
	if bdErr != nil {
		return nil, bdErr
	}

	return algoInst, nil
}

// ListSupportedOS() returns the list of all supported OS-es. Can be invoked on a non-supported OS.
func ListSupportedOS() []string {
	reg := getSupportedRegistry("", logr.Discard())
	return reg.ListOS()
}

// ListSupportedK8s(os string) returns the list of supported k8s for a specific OS.
// Can be invoked on a non-supported OS
func ListSupportedK8s(os string) []string {
	reg := getSupportedRegistry("", logr.Discard())
	return reg.ListK8s(os)
}

// PreviewChanges describes the changes to install and uninstall K8s on OS without actually applying them.
// It returns the install and uninstall changes
// Can be invoked on a non-supported OS
func PreviewChanges(os, k8sVer string) (install, uninstall string, err error) {
	return "", "", nil
}

// logPrinter is an adapter of OutputBilder to logr.Logger
type logPrinter struct {
	logger logr.Logger
}

func (lp *logPrinter) Desc(s string) { lp.logger.Info(s) }
func (lp *logPrinter) Cmd(s string)  { lp.logger.Info(s) }
func (lp *logPrinter) Out(s string)  { lp.logger.Info(s) }
func (lp *logPrinter) Err(s string)  { lp.logger.Info(s) }
func (lp *logPrinter) Msg(s string)  { lp.logger.Info(s) }
