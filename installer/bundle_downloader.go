// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package installer

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/go-logr/logr"
)

var (
	// DownloadPathPermissions file mode permissions for download path
	DownloadPathPermissions fs.FileMode = 0777
)

// bundleDownloader for downloading an OCI image.
type bundleDownloader struct {
	bundleType   BundleType
	repoAddr     string
	downloadPath string
	logger       logr.Logger
}

// NewBundleDownloader will return a new bundle downloader instance
func NewBundleDownloader(bundleType, repoAddr, downloadPath string, logger logr.Logger) *bundleDownloader {
	return &bundleDownloader{
		bundleType:   BundleType(bundleType),
		repoAddr:     repoAddr,
		downloadPath: downloadPath,
		logger:       logger,
	}
}

// convertError returns known errors in standardized format.
// func convertError(err error) error {
// 	downloadErrMap := map[string]Error{
// 		"no such host":                         ErrBundleDownload,
// 		"connection timed out":                 ErrBundleDownload,
// 		"temporary failure in name resolution": ErrBundleDownload,
// 		"no space left on device":              ErrBundleExtract}

// 	if err == nil {
// 		return nil
// 	}
// 	errStr := strings.ToLower(err.Error())
// 	for k, v := range downloadErrMap {
// 		if strings.HasSuffix(errStr, k) {
// 			return v
// 		}
// 	}
// 	return err
// }

// GetBundleDirPath returns the path to directory containing the required bundle.
func (bd *bundleDownloader) GetBundleDirPath(k8sVersion string) string {
	// Not storing tag as a subdir of k8s because we can't atomically move
	// the temp bundle dir to a non-existing dir.
	// Using "-" instead of ":" because Windows doesn't like the latter
	return fmt.Sprintf("%s-%s", filepath.Join(bd.getBundlePathWithRepo(), string(bd.bundleType)), k8sVersion)
}

// GetBundleName returns the name of the bundle in normalized format.
func GetBundleName(normalizedOsVersion string) string {
	return strings.ToLower(fmt.Sprintf("byoh-bundle-%s_k8s", normalizedOsVersion))
}

// getBundlePathWithRepo returns the path
func (bd *bundleDownloader) getBundlePathWithRepo() string {
	return filepath.Join(bd.downloadPath, strings.ReplaceAll(bd.repoAddr, "/", "."))
}

// GetBundleAddr returns the exact address to the bundle in the repo.
func (bd *bundleDownloader) GetBundleAddr(normalizedOsVersion, k8sVersion string) string {
	return fmt.Sprintf("%s/%s:%s", bd.repoAddr, GetBundleName(normalizedOsVersion), k8sVersion)
}

// checkDirExist checks if a dirrectory exists.
// func checkDirExist(dirPath string) bool {
// 	if fi, err := os.Stat(dirPath); os.IsNotExist(err) || !fi.IsDir() {
// 		return false
// 	}
// 	return true
// }

// ensureDirExist ensures that a bundle directory already exists or creates a new one recursively.
// func ensureDirExist(dirPath string) error {
// 	return os.MkdirAll(dirPath, DownloadPathPermissions)
// }
