// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package installer

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/installer/internal/algo"
)

// Error string wrapper for errors returned by the installer
type Error string

func (e Error) Error() string { return string(e) }

const (
	// ErrDetectOs error type when supported OS could not be detected
	ErrDetectOs = Error("Error detecting OS")
	// ErrOsK8sNotSupported error type when the OS is not supported by the k8s installer
	ErrOsK8sNotSupported = Error("No k8s support for OS")
	// ErrBundleDownload error type when the bundle download fails
	ErrBundleDownload = Error("Error downloading bundle")
	// ErrBundleExtract error type when the bundle extraction fails
	ErrBundleExtract = Error("Error extracting bundle")
	// ErrBundleInstall error type when the bundle installation fails
	ErrBundleInstall = Error("Error installing bundle")
	// ErrBundleUninstall error type when the bundle uninstallation fails
	ErrBundleUninstall = Error("Error uninstalling bundle")
)

// BundleType is used to support various bundles
type bundleType string

const (
	// BundleTypeK8s represents a vanilla k8s bundle
	BundleTypeK8s bundleType = "k8s"
)

var preRequisitePackages = []string{"socat", "ebtables", "ethtool", "conntrack"}

type installer struct {
	algoRegistry registry
	bundleDownloader
	detectedOs string
	logger     logr.Logger
}

// getSupportedRegistry returns a registry with installers for the supported OS and K8s
func getSupportedRegistry(ob algo.OutputBuilder) registry {
	reg := newRegistry()

	addBundleInstaller := func(osBundle, k8sBundle string, stepProvider algo.K8sStepProvider) {
		a := algo.BaseK8sInstaller{
			K8sStepProvider: stepProvider,
			/* BundlePath: will be set when tag is known */
			OutputBuilder: ob}

		reg.AddBundleInstaller(osBundle, k8sBundle, &a)
	}

	{
		// Ubuntu

		// BYOH Bundle Repository. Associate bundle with installer
		linuxDistro := "Ubuntu_20.04.1_x86-64"
		addBundleInstaller(linuxDistro, "v1.21.*", &algo.Ubuntu20_4K8s1_22{})
		addBundleInstaller(linuxDistro, "v1.22.*", &algo.Ubuntu20_4K8s1_22{})
		addBundleInstaller(linuxDistro, "v1.23.*", &algo.Ubuntu20_4K8s1_22{})

		/*
		 * PLACEHOLDER - ADD MORE K8S VERSIONS HERE
		 */

		// Match any patch version of the specified Major & Minor K8s version
		reg.AddK8sFilter("v1.21.*")
		reg.AddK8sFilter("v1.22.*")
		reg.AddK8sFilter("v1.23.*")

		// Match concrete os version to repository os version
		reg.AddOsFilter("Ubuntu_20.04.*_x86-64", linuxDistro)

		/*
		 * PLACEHOLDER - POINT MORE DISTRO VERSIONS
		 */
	}

	/*
	 * PLACEHOLDER - ADD MORE OS HERE
	 */

	return reg
}

func (bd *bundleDownloader) getBundlePathDirOrPreview(k8s, tag string) string {
	if bd == nil || bd.downloadPath == "" {
		return ""
	}

	return bd.GetBundleDirPath(k8s)
}

// DownloadOrPreview downloads the bundle if bundleDownloader is configured with a download path else runs in preview mode without downloading
func (bd *bundleDownloader) DownloadOrPreview(os, k8s, tag string) error {
	if bd == nil || bd.downloadPath == "" {
		bd.logger.Info("Running in preview mode, skip bundle download")
		return nil
	}

	return bd.Download(os, k8s, tag)
}

// New returns an installer that downloads bundles for the current OS from OCI repository with
// address bundleRepo and stores them under downloadPath. Download path is created,
// if it does not exist.
func New(downloadPath string, bundleType bundleType, logger logr.Logger) (*installer, error) {
	if downloadPath == "" {
		return nil, fmt.Errorf("empty download path")
	}

	osd := osDetector{}
	os, err := osd.Detect()
	if err != nil {
		return nil, ErrDetectOs
	}
	logger.Info("Detected", "OS", os)

	precheckSuccessful := runPrechecks(logger, os)
	if !precheckSuccessful {
		return nil, errors.New("precheck failed")
	}

	return newUnchecked(os, bundleType, downloadPath, logger, &logPrinter{logger})
}

// newUnchecked returns an installer bypassing os detection and checks of downloadPath.
// If it is empty, returned installer will run in preview mode, i.e.
// executes everything except the actual commands.
func newUnchecked(currentOs string, bundleType bundleType, downloadPath string, logger logr.Logger, outputBuilder algo.OutputBuilder) (*installer, error) {
	bd := bundleDownloader{repoAddr: "", bundleType: bundleType, downloadPath: downloadPath, logger: logger}

	reg := getSupportedRegistry(outputBuilder)
	if len(reg.ListK8s(currentOs)) == 0 {
		return nil, ErrOsK8sNotSupported
	}

	return &installer{
		algoRegistry:     reg,
		bundleDownloader: bd,
		detectedOs:       currentOs,
		logger:           logger}, nil
}

// setBundleRepo sets the repo from which the bundle will be downloaded.
func (i *installer) setBundleRepo(bundleRepo string) {
	i.bundleDownloader.repoAddr = bundleRepo
}

// Install installs the specified k8s version on the current OS
func (i *installer) Install(bundleRepo, k8sVer, tag string) error {
	i.setBundleRepo(bundleRepo)
	algoInst, err := i.getAlgoInstallerWithBundle(k8sVer, tag)
	if err != nil {
		return err
	}
	err = algoInst.(algo.Installer).Install()
	if err != nil {
		return ErrBundleInstall
	}

	return nil
}

// Uninstall uninstalls the specified k8s version on the current OS
func (i *installer) Uninstall(bundleRepo, k8sVer, tag string) error {
	i.setBundleRepo(bundleRepo)
	algoInst, err := i.getAlgoInstallerWithBundle(k8sVer, tag)
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
func (i *installer) getAlgoInstallerWithBundle(k8sVer, tag string) (osk8sInstaller, error) {
	// This OS supports at least 1 k8s version. See New.

	algoInst, osBundle := i.algoRegistry.GetInstaller(i.detectedOs, k8sVer)
	if algoInst == nil {
		return nil, ErrOsK8sNotSupported
	}
	i.logger.Info("Current OS will be handled as", "OS", osBundle)

	// copy installer from registry and set BundlePath including tag
	// empty means preview mode
	algoInstCopy := *algoInst.(*algo.BaseK8sInstaller)
	algoInstCopy.BundlePath = i.bundleDownloader.getBundlePathDirOrPreview(k8sVer, tag)

	bdErr := i.bundleDownloader.DownloadOrPreview(osBundle, k8sVer, tag)
	if bdErr != nil {
		return nil, bdErr
	}

	return &algoInstCopy, nil
}

// ListSupportedOS returns the list of all supported OS-es. Can be invoked on a non-supported OS.
func ListSupportedOS() (osFilters, osBundles []string) {
	srd := getSupportedRegistryDescription()
	return srd.ListOS()
}

// ListSupportedK8s returns the list of supported k8s for a specific OS.
// Can be invoked on a non-supported OS
func ListSupportedK8s(os string) []string {
	srd := getSupportedRegistryDescription()
	return srd.ListK8s(os)
}

// getSupportedRegistryDescription returns a description registry of supported OS and k8s.
// It that can only be queried for OS and k8s but cannot be used for install/uninstall.
func getSupportedRegistryDescription() registry {
	return getSupportedRegistry(nil)
}

// PreviewChanges describes the changes to install and uninstall K8s on OS without actually applying them.
// It returns install and uninstall changes
// Can be invoked on a non-supported OS
func PreviewChanges(os, k8sVer string) (install, uninstall string, err error) {
	stepPreviewer := stringPrinter{msgFmt: "# %s"}
	reg := getSupportedRegistry(&stepPreviewer)
	installer, _ := reg.GetInstaller(os, k8sVer)

	if installer == nil {
		err = ErrOsK8sNotSupported
		return
	}

	err = installer.(algo.Installer).Install()
	if err != nil {
		return
	}
	install = stepPreviewer.String()
	stepPreviewer.steps = nil
	err = installer.(algo.Installer).Uninstall()
	if err != nil {
		return
	}
	uninstall = stepPreviewer.String()
	return
}

// logPrinter is an adapter of OutputBilder to logr.Logger
type logPrinter struct {
	logger logr.Logger
}

// Desc logPrinter implementation of OutputBuilder Desc method
func (lp *logPrinter) Desc(s string) { lp.logger.Info(s) }

// Cmd logPrinter implementation of OutputBuilder Cmd method
func (lp *logPrinter) Cmd(s string) { lp.logger.Info(s) }

// Out logPrinter implementation of OutputBuilder Out method
func (lp *logPrinter) Out(s string) { lp.logger.Info(s) }

// Err logPrinter implementation of OutputBuilder Err method
func (lp *logPrinter) Err(s string) { lp.logger.Info(s) }

// Msg logPrinter implementation of OutputBuilder Msg method
func (lp *logPrinter) Msg(s string) { lp.logger.Info(s) }

// stringPrinter is an adapter of OutputBuilder to string
type stringPrinter struct {
	steps      []string
	descFmt    string
	cmdFmt     string
	outFmt     string
	errFmt     string
	msgFmt     string
	strDivider string
}

// Desc stringPrinter implementation of description output
func (obp *stringPrinter) Desc(s string) { obp.steps = append(obp.steps, applyFmt(obp.descFmt, s)) }

// Cmd stringPrinter implementation of command output
func (obp *stringPrinter) Cmd(s string) { obp.steps = append(obp.steps, applyFmt(obp.cmdFmt, s)) }

// Out stringPrinter implementation of info/content output
func (obp *stringPrinter) Out(s string) { obp.steps = append(obp.steps, applyFmt(obp.outFmt, s)) }

// Err stringPrinter implementation of error output
func (obp *stringPrinter) Err(s string) { obp.steps = append(obp.steps, applyFmt(obp.errFmt, s)) }

// Msg stringPrinter implementation of message output
func (obp *stringPrinter) Msg(s string) { obp.steps = append(obp.steps, applyFmt(obp.msgFmt, s)) }

// String implements the Stringer interface
// It joins the string array by adding new lines between the strings and returns it as a single string
func (obp *stringPrinter) String() string {
	if obp.strDivider == "" {
		obp.strDivider = "\n"
	}
	return strings.Join(obp.steps, obp.strDivider)
}

// applyFmt applies a given format to a string or returns the string if format is empty
func applyFmt(stepFmt, s string) string {
	if stepFmt == "" {
		stepFmt = "%s"
	}
	return fmt.Sprintf(stepFmt, s)
}
