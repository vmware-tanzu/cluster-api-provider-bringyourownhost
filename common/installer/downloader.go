// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package installer

import (
	"github.com/go-logr/logr"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/installer"
)

// BundleDownloader represent a bundle downloader interface
type BundleDownloader interface {
	GetBundleAddr(normalizedOsVersion, k8sVersion, tag string) string
}

// DefaultBundleDownloader implement the downloader interface
func DefaultBundleDownloader(bundleType, repoAddr, downloadPath string, logger logr.Logger) BundleDownloader {
	return installer.NewBundleDownloader(installer.BundleType(bundleType), repoAddr, downloadPath, logger)
}
