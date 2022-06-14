// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package installer

import (
	"context"
	"strings"

	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/installer/internal/algo"
)

// K8sInstaller represent k8s installer interface
type K8sInstaller interface {
	Install() string
	Uninstall() string
}

// Error string wrapper for errors returned by the installer
type Error string

func (e Error) Error() string { return string(e) }

// BundleType is used to support various bundles
type BundleType string

const (
	// BundleTypeK8s represents a vanilla k8s bundle
	BundleTypeK8s BundleType = "k8s"
)

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

// archOldNameMap keeps the mapping of architecture new name to old name mapping
var archOldNameMap = map[string]string{
	"amd64": "x86-64",
}

// NewInstaller will return a new installer
func NewInstaller(ctx context.Context, osDist, arch, k8sVersion string, downloader BundleDownloader) (K8sInstaller, error) {
	bundleArchName := arch
	// replacing the arch name to old name to match with the bundle name
	if _, exists := archOldNameMap[arch]; exists {
		bundleArchName = archOldNameMap[arch]
	}
	// normalizing os image name and adding arch
	osArch := strings.ReplaceAll(osDist, " ", "_") + "_" + bundleArchName

	reg := GetSupportedRegistry()
	if len(reg.ListK8s(osArch)) == 0 {
		return nil, ErrOsK8sNotSupported
	}
	osbundle := reg.ResolveOsToOsBundle(osArch)
	addrs := downloader.GetBundleAddr(osbundle, k8sVersion)

	return algo.NewUbuntu20_04Installer(ctx, arch, addrs)
}
