// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package installer

import (
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/installer/internal/algo"
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
	algoRegistry registry
	bundleDownloader
	detectedOs string
	logger     logr.Logger
}

// getSupportedRegistry returns a registry with installers for the supported OS and K8s
func getSupportedRegistry(bd *bundleDownloader, ob algo.OutputBuilder) registry {
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
		addBundleInstaller(linuxDistro, "v1.22.3", &algo.Ubuntu20_4K8s1_22{})
		/*
		 * PLACEHOLDER - ADD MORE K8S VERSIONS HERE
		 */

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

	return bd.GetBundleDirPath(k8s, tag)
}

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
func New(bundleRepo, downloadPath string, logger logr.Logger) (*installer, error) {
	if bundleRepo == "" {
		return nil, fmt.Errorf("empty bundle repo")
	}
	if downloadPath == "" {
		return nil, fmt.Errorf("empty download path")
	}

	osd := osDetector{}
	os, err := osd.Detect()
	logger.Info("Detected", "OS", os)
	if err != nil {
		return nil, ErrDetectOs
	}

	return newUnchecked(os, bundleRepo, downloadPath, logger, &logPrinter{logger})
}

// newUnchecked returns an installer bypassing os detection and checks of bundleRepo and downloadPath.
// If they are empty, returned installer will runs in preview mode, i.e.
// executes everything except the actual commands.
func newUnchecked(currentOs, bundleRepo, downloadPath string, logger logr.Logger, outputBuilder algo.OutputBuilder) (*installer, error) {
	bd := bundleDownloader{repoAddr: bundleRepo, downloadPath: downloadPath, logger: logger}

	reg := getSupportedRegistry(&bd, outputBuilder)
	if len(reg.ListK8s(currentOs)) == 0 {
		return nil, ErrOsK8sNotSupported
	}

	return &installer{
		algoRegistry:     reg,
		bundleDownloader: bd,
		detectedOs:       currentOs,
		logger:           logger}, nil
}

// Install installs the specified k8s version on the current OS
func (i *installer) Install(k8sVer, tag string) error {
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

// Uninstal uninstalls the specified k8s version on the current OS
func (i *installer) Uninstall(k8sVer, tag string) error {
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

// ListSupportedOS() returns the list of all supported OS-es. Can be invoked on a non-supported OS.
func ListSupportedOS() (osFilters, osBundles []string) {
	srd := getSupportedRegistryDescription()
	return srd.ListOS()
}

// ListSupportedK8s(os string) returns the list of supported k8s for a specific OS.
// Can be invoked on a non-supported OS
func ListSupportedK8s(os string) []string {
	srd := getSupportedRegistryDescription()
	return srd.ListK8s(os)
}

// getSupportedRegistryDescription returns a description registry of supported OS and k8s.
// It that can only by queried for OS and k8s but cannot be used for install/uninstall.
func getSupportedRegistryDescription() registry {
	return getSupportedRegistry(nil, nil)
}

// PreviewChanges describes the changes to install and uninstall K8s on OS without actually applying them.
// It returns the install and uninstall changes
// Can be invoked on a non-supported OS
func PreviewChanges(os, k8sVer string) (install, uninstall string, err error) {
	stepPreviewer := stringPrinter{msgFmt: "# %s"}
	reg := getSupportedRegistry(&bundleDownloader{}, &stepPreviewer)
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

func (lp *logPrinter) Desc(s string) { lp.logger.Info(s) }
func (lp *logPrinter) Cmd(s string)  { lp.logger.Info(s) }
func (lp *logPrinter) Out(s string)  { lp.logger.Info(s) }
func (lp *logPrinter) Err(s string)  { lp.logger.Info(s) }
func (lp *logPrinter) Msg(s string)  { lp.logger.Info(s) }

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

func (obp *stringPrinter) Desc(s string) { obp.steps = append(obp.steps, applyFmt(obp.descFmt, s)) }
func (obp *stringPrinter) Cmd(s string)  { obp.steps = append(obp.steps, applyFmt(obp.cmdFmt, s)) }
func (obp *stringPrinter) Out(s string)  { obp.steps = append(obp.steps, applyFmt(obp.outFmt, s)) }
func (obp *stringPrinter) Err(s string)  { obp.steps = append(obp.steps, applyFmt(obp.errFmt, s)) }
func (obp *stringPrinter) Msg(s string)  { obp.steps = append(obp.steps, applyFmt(obp.msgFmt, s)) }

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
