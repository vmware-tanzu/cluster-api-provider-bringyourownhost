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
	"github.com/go-logr/logr"
	"github.com/k14s/imgpkg/pkg/imgpkg/cmd"
)

var (
	// DownloadPathPermissions file mode permissions for download path
	DownloadPathPermissions fs.FileMode = 0777
)

// bundleDownloader for downloading an OCI image.
type bundleDownloader struct {
	bundleType   bundleType
	repoAddr     string
	downloadPath string
	logger       logr.Logger
}

// Download is a method that downloads the bundle from repoAddr to downloadPath.
// It automatically downloads and extracts the given version for the current linux
// distribution. Creates the folder where the bundle should be saved if it does not exist.
// Download is performed in a temp directory which in case of successful download is renamed.
// If a cache for the bundle exists, nothing is downloaded.
func (bd *bundleDownloader) Download(
	normalizedOsVersion,
	k8sVersion string,
	tag string) error {

	return bd.DownloadFromRepo(
		normalizedOsVersion,
		k8sVersion,
		tag,
		bd.downloadByImgpkg)
}

// DownloadFromRepo downloads the required bundle with the given method.
func (bd *bundleDownloader) DownloadFromRepo(
	normalizedOsVersion,
	k8sVersion string,
	tag string,
	downloadByTool func(string, string) error) error {

	downloadPathWithRepo := bd.getBundlePathWithRepo()

	err := ensureDirExist(downloadPathWithRepo)
	defer func(name string) {
		err = os.Remove(name)
		if err != nil {
			bd.logger.Error(err, "Failed to remove directory", "path", name)
		}
	}(downloadPathWithRepo)

	if err != nil {
		return err
	}

	bundleDirPath := bd.GetBundleDirPath(k8sVersion)

	// cache hit
	if checkDirExist(bundleDirPath) {
		bd.logger.Info("Cache hit", "path", bundleDirPath)
		return nil
	}

	bd.logger.Info("Cache miss", "path", bundleDirPath)

	dir, err := os.MkdirTemp(downloadPathWithRepo, "tempBundle")
	// It is fine if the dir path does not exist.
	defer func() {
		err = os.RemoveAll(dir)
		if err != nil {
			bd.logger.Error(err, "Failed to remove temp bundle dir", "path", dir)
		}
	}()
	if err != nil {
		return err
	}
	bundleAddr := bd.getBundleAddr(normalizedOsVersion, k8sVersion, tag)
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

	bd.logger.Info("Downloading bundle", "from", bundleAddr)

	var confUI = ui.NewConfUI(ui.NewNoopLogger())
	defer confUI.Flush()

	imgpkgCmd := cmd.NewDefaultImgpkgCmd(confUI)
	imgpkgCmd.SetArgs([]string{"pull", "--recursive", "-i", bundleAddr, "-o", bundleDirPath})
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

// getBundleAddr returns the exact address to the bundle in the repo.
func (bd *bundleDownloader) getBundleAddr(normalizedOsVersion, k8sVersion, tag string) string {
	return fmt.Sprintf("%s/%s:%s", bd.repoAddr, GetBundleName(normalizedOsVersion), tag)
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
