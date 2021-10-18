// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package installer

import (
	"errors"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type mockImgpkg struct {
	callCount int
	bd        *bundleDownloader
}

func (mi *mockImgpkg) Get(_, k8sVerDirPath string) error {
	mi.callCount++
	return nil
}

func RemoveDir(dir string) error {
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
		repoAddr = ""
		downloadPath = "/tmp/downloaderTest" + strconv.Itoa(rand.Intn(100000))
		bd = &bundleDownloader{repoAddr, downloadPath}
		mi = &mockImgpkg{0, bd}
		normalizedOsVer = "Ubuntu_20.04.3_x64"
		k8sVer = "1.22"
	})

	Context("When given correct arguments", func() {
		BeforeEach(func() {
			mi.callCount = 0
			err := RemoveDir(downloadPath)
			if err != nil {
				err = os.Mkdir(downloadPath, dirPermissions)
				if err != nil {
					log.Fatal(err)
				}
			}
		})
		AfterEach(func() {
			err := RemoveDir(downloadPath)
			if err != nil {
				log.Fatal(err)
			}
		})
		It("Should download bundle", func() {
			err := bd.DownloadFromRepo(
				normalizedOsVer,
				k8sVer,
				func(a, b string) error { return mi.Get(a, b) })
			Expect(err).ShouldNot((HaveOccurred()))

			err = bd.DownloadFromRepo(
				normalizedOsVer,
				k8sVer,
				func(a, b string) error { return mi.Get(a, b) })
			Expect(err).ShouldNot((HaveOccurred()))
			Expect(mi.callCount).Should(Equal(1))

			err = bd.Download(
				normalizedOsVer,
				k8sVer)
			Expect(err).ShouldNot((HaveOccurred()))
		})
	})
	Context("When given bad arguments", func() {
		It("Should error when given bad repo address", func() {
			repoAddr = "a$s!d.a*s-d.a-s-d"
			bd = &bundleDownloader{repoAddr, downloadPath}
			err := bd.Download(
				normalizedOsVer,
				k8sVer)
			Expect(err).Should(HaveOccurred())
			Expect(err).ShouldNot(Equal(downloadPathNotExist))
		})
		It("Should error when given bad download path", func() {
			downloadPath = "./a$s!d.a*s-d.a-s-d"
			bd = &bundleDownloader{repoAddr, downloadPath}
			err := bd.Download(
				normalizedOsVer,
				k8sVer)
			if !Expect(err).Should(Equal(errors.New(downloadPathNotExist))) {
				err = RemoveDir(downloadPath)
				Expect(err).ShouldNot(HaveOccurred())
			}
		})
	})

})
