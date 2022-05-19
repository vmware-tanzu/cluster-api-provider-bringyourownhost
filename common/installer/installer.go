// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package installer

import (
	"context"
	"strings"

	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/installer"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/common/installer/internal/algo"
)

// K8sInstaller represent k8s installer interface
type K8sInstaller interface {
	Install() string
	Uninstall() string
}

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

	reg := installer.GetSupportedRegistry(nil)
	if len(reg.ListK8s(osArch)) == 0 {
		return nil, installer.ErrOsK8sNotSupported
	}
	_, osbundle := reg.GetInstaller(osArch, k8sVersion)
	addrs := downloader.GetBundleAddr(osbundle, k8sVersion, k8sVersion)

	return algo.NewUbuntu20_04Installer(ctx, arch, addrs)
}
