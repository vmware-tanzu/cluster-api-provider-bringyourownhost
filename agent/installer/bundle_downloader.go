// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package installer

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"

	"github.com/cppforlife/go-cli-ui/ui"
	"github.com/k14s/imgpkg/pkg/imgpkg/cmd"
)

var (
	dirPermissions fs.FileMode = 0777
)

const (
	Separator = os.PathSeparator
)

// bundleDownloader for downloading an OCI image.
type bundleDownloader struct {
	repoAddr     string
	downloadPath string
}

// Download is a method that downloads the bundle from repoAddr to downloadPath.
// It automatically downloads and extracts the given version for the current linux
// distribution. Creates the folder where the bundle should be saved if it does not exist.
func (bd *bundleDownloader) Download(
	normalizedOsVersion,
	k8sVersion string) error {

	return bd.DownloadFromRepo(
		normalizedOsVersion,
		k8sVersion,
		func(a, b string) error { return bd.downloadByImgpkg(a, b) })
}

// DownloadFromRepo downloads the required bundle with the given method.
func (bd *bundleDownloader) DownloadFromRepo(
	normalizedOsVersion,
	k8sVersion string,
	downloadByTool func(string, string) error) error {

	err := bd.ensureDirExist(bd.downloadPath)
	if err != nil {
		return err
	}

	bundleDirPath := bd.GetBundleDirPath(k8sVersion)
	if bd.checkDirExist(bundleDirPath) {
		return nil
	}

	dir, err := ioutil.TempDir(bd.downloadPath, "tempBundle")
	defer os.Remove(dir)
	if err != nil {
		return err
	}

	bundleAddr := bd.getBundleAddr(normalizedOsVersion, k8sVersion)
	err = downloadByTool(bundleAddr, dir)
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

// GetBundleDirPath returns the path to directory containing the required bundle.
func (bd *bundleDownloader) GetBundleDirPath(k8sVersion string) string {
	return fmt.Sprintf("%s%c%s", bd.downloadPath, Separator, k8sVersion)
}

// getBundleAddr returns the exact address to the bundle in the repo.
func (bd *bundleDownloader) getBundleAddr(normalizedOsVersion, k8sVersion string) string {
	return fmt.Sprintf("%s/%s_k8s_%s", bd.repoAddr, normalizedOsVersion, k8sVersion)
}

// checkDirExist checks if a dirrectory exists.
func (bd *bundleDownloader) checkDirExist(dirPath string) bool {
	if fi, err := os.Stat(dirPath); os.IsNotExist(err) || !fi.IsDir() {
		return false
	}
	return true
}

// ensureDirExist ensures that a bundle directory already exists or creates a new one recursively.
func (bd *bundleDownloader) ensureDirExist(dirPath string) error {
	if !bd.checkDirExist(dirPath) {
		return os.MkdirAll(dirPath, dirPermissions)
	}
	return nil
}
