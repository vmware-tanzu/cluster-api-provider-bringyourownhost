// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package installer

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/cppforlife/go-cli-ui/ui"
	"github.com/k14s/imgpkg/pkg/imgpkg/cmd"
)

var (
	dirPermissions fs.FileMode = 0777
)

const (
	downloadPathNotExist        = "download path does not exist"
	k8sVersionAlreadyDownloaded = "k8s version already downloaded"
)

// bundleDownloader for downloading an OCI image
type bundleDownloader struct {
}

// Download is a method that downloads the bundle from repoAddr to downloadPath.
// It automatically downloads and extracts the given version for the current linux
// distribution by using helper methods to gather all required  information. If
// the folder where the bundle should be saved does exist the bundle is downloaded.
// Finally the method returns whether the download was successful.
func (bd *bundleDownloader) Download(
	repoAddr,
	downloadPath,
	normalizedOsVer,
	k8sVer string) error {

	return bd.DownloadFromRepo(
		repoAddr,
		downloadPath,
		normalizedOsVer,
		k8sVer,
		func(a, b string) error { return bd.downloadByImgpkg(a, b) })
}

// DownloadFromRepo downloads the required bundle with the given method.
func (bd *bundleDownloader) DownloadFromRepo(
	repoAddr,
	downloadPath,
	normalizedOsVer,
	k8sVer string,
	f func(string, string) error) error {

	k8sVerDirPath := bd.getK8sDirPath(downloadPath, k8sVer)
	err := bd.makeK8sVerDir(downloadPath, k8sVerDirPath)
	if err != nil {
		return err
	}
	bundleAddr := fmt.Sprintf("%s/%s_k8s_%s", repoAddr, normalizedOsVer, k8sVer)
	return f(bundleAddr, k8sVerDirPath)
}

// downloadByImgpkg downloads the required bundle from the given repo using imgpkg.
func (bd *bundleDownloader) downloadByImgpkg(
	bundleAddr,
	k8sVerDirPath string) error {

	var confUI = ui.NewConfUI(ui.NewNoopLogger())
	defer confUI.Flush()

	imgpkgCmd := cmd.NewDefaultImgpkgCmd(confUI)

	imgpkgCmd.SetArgs([]string{"pull", "--recursive", "-b", bundleAddr, "-o", k8sVerDirPath})
	err := imgpkgCmd.Execute()
	return err
}

// getK8sDirPath returns the path to directory containing the given k8sVer
func (bd *bundleDownloader) getK8sDirPath(downloadPath, k8sVer string) string {
	return fmt.Sprintf("%s/%s", downloadPath, k8sVer)
}

// makeK8sVerDir checks if the path exists and creates a directory for the required k8s version.
func (bd *bundleDownloader) makeK8sVerDir(downloadPath, k8sVerDirPath string) error {
	if !bd.checkDirExist(downloadPath) {
		return errors.New(downloadPathNotExist)
	}
	if bd.checkDirExist(k8sVerDirPath) {
		isDirEmpty, err := bd.isEmpty(k8sVerDirPath)
		if err != nil {
			return err
		}
		if !isDirEmpty {
			return errors.New(k8sVersionAlreadyDownloaded)
		}
	} else {
		err := os.Mkdir(k8sVerDirPath, dirPermissions)
		if err != nil {
			return err
		}
	}
	return nil
}

// checkDirExist checks if a dirrectory exists.
func (bd *bundleDownloader) checkDirExist(dirPath string) bool {
	if fi, err := os.Stat(dirPath); os.IsNotExist(err) || !fi.IsDir() {
		return false
	}
	return true
}

// isEmpty checks if a directory is empty
func (bd *bundleDownloader) isEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}
