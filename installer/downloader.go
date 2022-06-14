// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package installer

import (
	"github.com/go-logr/logr"
)

// BundleDownloader represent a bundle downloader interface
type BundleDownloader interface {
	GetBundleAddr(normalizedOsVersion, k8sVersion string) string
}

// DefaultBundleDownloader implement the downloader interface
func DefaultBundleDownloader(bundleType, repoAddr, downloadPath string, logger logr.Logger) BundleDownloader {
	return NewBundleDownloader(BundleType(bundleType), repoAddr, downloadPath, logger)
}
