// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package installer

import (
	"errors"
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
	downloadPathNotExist = "download path does not exist"
)

// bundleDownloader for downloading an OCI image
type bundleDownloader struct {
	repoAddr     string
	downloadPath string
}

// Download is a method that downloads the bundle from repoAddr to downloadPath.
// It automatically downloads and extracts the given version for the current linux
// distribution. Creates the folder where the bundle should be saved if it does not exist
func (bd *bundleDownloader) Download(
	normalizedOsVer,
	k8sVersion string) error {

	if !bd.checkDirExist(bd.downloadPath) {
		return errors.New(downloadPathNotExist)
	}

	return bd.DownloadFromRepo(
		normalizedOsVer,
		k8sVersion,
		func(a, b string) error { return bd.downloadByImgpkg(a, b) })
}

// DownloadFromRepo downloads the required bundle with the given method.
func (bd *bundleDownloader) DownloadFromRepo(
	normalizedOsVer,
	k8sVersion string,
	downloadByTool func(string, string) error) error {

	k8sVersionDirPath := bd.getK8sDirPath(bd.downloadPath, k8sVersion)
	if bd.checkDirExist(k8sVersionDirPath) {
		return nil
	}

	dir, err := ioutil.TempDir(bd.downloadPath, "tempBundle")
	defer os.Remove(dir)
	if err != nil {
		return err
	}

	bundleAddr := fmt.Sprintf("%s/%s_k8s_%s", bd.repoAddr, normalizedOsVer, k8sVersion)
	err = downloadByTool(bundleAddr, dir)
	if err != nil {
		return err
	}
	return os.Rename(dir, k8sVersionDirPath)
}

// downloadByImgpkg downloads the required bundle from the given repo using imgpkg.
func (bd *bundleDownloader) downloadByImgpkg(
	bundleAddr,
	k8sVersionDirPath string) error {

	var confUI = ui.NewConfUI(ui.NewNoopLogger())
	defer confUI.Flush()

	imgpkgCmd := cmd.NewDefaultImgpkgCmd(confUI)

	imgpkgCmd.SetArgs([]string{"pull", "--recursive", "-b", bundleAddr, "-o", k8sVersionDirPath})
	return imgpkgCmd.Execute()
}

// getK8sDirPath returns the path to directory containing the given k8sVersion
func (bd *bundleDownloader) getK8sDirPath(downloadPath, k8sVersion string) string {
	return fmt.Sprintf("%s/%s", downloadPath, k8sVersion)
}

// checkDirExist checks if a dirrectory exists.
func (bd *bundleDownloader) checkDirExist(dirPath string) bool {
	if fi, err := os.Stat(dirPath); os.IsNotExist(err) || !fi.IsDir() {
		return false
	}
	return true
}
