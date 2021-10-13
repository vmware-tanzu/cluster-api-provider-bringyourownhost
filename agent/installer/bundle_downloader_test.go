// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package installer

import (
	"errors"
	"log"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type mockImgpkg struct {
	callCount int
	bd        *bundleDownloader
}

func (mi *mockImgpkg) Get(_, k8sVerDirPath string) error {
	mi.callCount++
	return mi.bd.makeK8sVerDir(k8sVerDirPath, k8sVerDirPath+"/testDownloader")
}

func RemoveContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	err = os.Remove(dir)
	if err != nil {
		return err
	}
	return nil
}

var _ = Describe("Byohost Installer Tests", func() {

	var (
		bd              *bundleDownloader
		mi              *mockImgpkg
		repoAddr        string
		downloadPath    string
		normalizedOsVer string
		k8sVer          string
	)

	BeforeEach(func() {
		bd = &bundleDownloader{}
		mi = &mockImgpkg{0, bd}
		repoAddr = ""
		downloadPath = "./testFolder"
		normalizedOsVer = "Ubuntu_20.04.3_x64"
		k8sVer = "1.22"
	})

	Context("When given correct arguments", func() {
		BeforeEach(func() {
			mi.callCount = 0
			err := RemoveContents(downloadPath)
			if err != nil {
				err = os.Mkdir(downloadPath, dirPermissions)
				if err != nil {
					log.Fatal(err)
				}
			}
		})
		AfterEach(func() {
			err := RemoveContents(downloadPath)
			if err != nil {
				log.Fatal(err)
			}
		})
		It("Should download bundle", func() {
			err := bd.DownloadFromRepo(repoAddr,
				downloadPath,
				normalizedOsVer,
				k8sVer,
				func(a, b string) error { return mi.Get(a, b) })
			Expect(err).ShouldNot((HaveOccurred()))

			err = bd.DownloadFromRepo(repoAddr,
				downloadPath,
				normalizedOsVer,
				k8sVer,
				func(a, b string) error { return mi.Get(a, b) })
			Expect(err).Should(Equal(errors.New(k8sVersionAlreadyDownloaded)))
			Expect(mi.callCount).Should(Equal(1))

			err = bd.Download(repoAddr,
				downloadPath,
				normalizedOsVer,
				k8sVer)
			Expect(err).Should(Equal(errors.New(k8sVersionAlreadyDownloaded)))
		})
	})
	Context("When given bad arguments", func() {
		It("Should error when given bad repo address", func() {
			repoAddr = "a$s!d.a*s-d.a-s-d"
			err := bd.Download(repoAddr,
				downloadPath,
				normalizedOsVer,
				k8sVer)
			Expect(err).Should(HaveOccurred())
			Expect(err).ShouldNot(Equal(downloadPathNotExist))
		})
		It("Should error when given bad download path", func() {
			downloadPath = "./a$s!d.a*s-d.a-s-d"
			err := bd.Download(repoAddr,
				downloadPath,
				normalizedOsVer,
				k8sVer)
			if !Expect(err).Should(Equal(errors.New(downloadPathNotExist))) {
				err = RemoveContents(downloadPath)
				Expect(err).ShouldNot(HaveOccurred())
			}
		})
	})

})
