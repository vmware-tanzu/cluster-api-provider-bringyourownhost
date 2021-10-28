// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package installer

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/cppforlife/go-cli-ui/ui"
	"github.com/k14s/imgpkg/pkg/imgpkg/cmd"
)

var (
	DownloadPathPermissions fs.FileMode = 0777
)

// bundleDownloader for downloading an OCI image.
type bundleDownloader struct {
	repoAddr     string
	downloadPath string
}

// Download is a method that downloads the bundle from repoAddr to downloadPath.
// It automatically downloads and extracts the given version for the current linux
// distribution. Creates the folder where the bundle should be saved if it does not exist.
// Download is performed in a temp directory which in case of successful download is renamed.
// If a cache for the bundle exists, nothing is downloaded.
func (bd *bundleDownloader) Download(
	normalizedOsVersion,
	k8sVersion string) error {

	return bd.DownloadFromRepo(
		normalizedOsVersion,
		k8sVersion,
		bd.downloadByImgpkg)
}

// DownloadFromRepo downloads the required bundle with the given method.
func (bd *bundleDownloader) DownloadFromRepo(
	normalizedOsVersion,
	k8sVersion string,
	downloadByTool func(string, string) error) error {

	err := ensureDirExist(bd.downloadPath)
	if err != nil {
		return err
	}

	bundleDirPath := bd.GetBundleDirPath(k8sVersion)

	// cache hit
	if checkDirExist(bundleDirPath) {
		return nil
	}

	dir, err := os.MkdirTemp(bd.downloadPath, "tempBundle")
	// It is fine if the dir path does not exist.
	defer os.RemoveAll(dir)
	if err != nil {
		return err
	}

	bundleAddr := bd.getBundleAddr(normalizedOsVersion, k8sVersion)
	err = convertError(downloadByTool(bundleAddr, dir))
	if err != nil {
		return err
	}
	return os.Rename(dir, bundleDirPath)
}

// downloadByImgpkg downloads the required bundle from the given repo using imgpkg.
func (bd *bundleDownloader) downloadByImgpkg(
	bundleAddr,
	bundleDirPath string) error {

	var confUI = ui.NewConfUI(ui.NewNoopLogger())
	defer confUI.Flush()

	imgpkgCmd := cmd.NewDefaultImgpkgCmd(confUI)
	imgpkgCmd.SetArgs([]string{"pull", "--recursive", "-b", bundleAddr, "-o", bundleDirPath})
	return imgpkgCmd.Execute()
}

// convertError returns known errors in standardized format.
func convertError(err error) error {
	downloadErrMap := map[string]Error{
		"no such host":                         ErrBundleDownload,
		"connection timed out":                 ErrBundleDownload,
		"temporary failure in name resolution": ErrBundleDownload,
		"no space left on device":              ErrBundleExtract}

	if err == nil {
		return nil
	}
	errStr := strings.ToLower(err.Error())
	for k, v := range downloadErrMap {
		if strings.HasSuffix(errStr, k) {
			return v
		}
	}
	return err
}

// GetBundleDirPath returns the path to directory containing the required bundle.
func (bd *bundleDownloader) GetBundleDirPath(k8sVersion string) string {
	return filepath.Join(bd.downloadPath, k8sVersion)
}

// GetBundleName returns the name of the bundle in normalized format.
func GetBundleName(normalizedOsVersion, k8sVersion string) string {
	return strings.ToLower(fmt.Sprintf("byoh-bundle-%s_k8s_%s", normalizedOsVersion, k8sVersion))
}

// getBundleAddr returns the exact address to the bundle in the repo.
func (bd *bundleDownloader) getBundleAddr(normalizedOsVersion, k8sVersion string) string {
	return fmt.Sprintf("%s/%s", bd.repoAddr, GetBundleName(normalizedOsVersion, k8sVersion))
}

// checkDirExist checks if a dirrectory exists.
func checkDirExist(dirPath string) bool {
	if fi, err := os.Stat(dirPath); os.IsNotExist(err) || !fi.IsDir() {
		return false
	}
	return true
}

// ensureDirExist ensures that a bundle directory already exists or creates a new one recursively.
func ensureDirExist(dirPath string) error {
	return os.MkdirAll(dirPath, DownloadPathPermissions)
}
