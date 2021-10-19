// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package installer

import (
	"io/ioutil"
	"log"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type mockImgpkg struct {
	callCount int
}

func (mi *mockImgpkg) Get(_, k8sVerDirPath string) error {
	mi.callCount++
	return nil
}

var _ = Describe("Byohost Installer Tests", func() {

	var (
		bd                  *bundleDownloader
		mi                  *mockImgpkg
		repoAddr            string
		downloadPath        string
		normalizedOsVersion string
		k8sVersion          string
	)

	BeforeEach(func() {
		normalizedOsVersion = "Ubuntu_20.04.3_x64"
		k8sVersion = "1.22"
	})

	Context("When given correct arguments", func() {
		BeforeEach(func() {
			repoAddr = ""
			var err error
			downloadPath, err = ioutil.TempDir("/tmp", "downloaderTest")
			if err != nil {
				log.Fatal(err)
			}
			bd = &bundleDownloader{repoAddr, downloadPath}
			mi = &mockImgpkg{}
		})
		AfterEach(func() {
			err := os.RemoveAll(downloadPath)
			if err != nil {
				log.Fatal(err)
			}
		})
		It("Should download bundle", func() {
			// Test download on cache missing
			err := bd.DownloadFromRepo(
				normalizedOsVersion,
				k8sVersion,
				func(a, b string) error { return mi.Get(a, b) })
			Expect(err).ShouldNot((HaveOccurred()))

			// Test no download on cache hit
			err = bd.DownloadFromRepo(
				normalizedOsVersion,
				k8sVersion,
				func(a, b string) error { return mi.Get(a, b) })
			Expect(err).ShouldNot((HaveOccurred()))
			Expect(mi.callCount).Should(Equal(1))

			// Making linter happy
			err = bd.Download(
				normalizedOsVersion,
				k8sVersion)
			Expect(err).ShouldNot((HaveOccurred()))
		})
		It("Should create dir if missing and download bundle", func() {
			bd.downloadPath = bd.downloadPath + "/a/b/c/d"
			err := bd.DownloadFromRepo(
				normalizedOsVersion,
				k8sVersion,
				func(a, b string) error { return mi.Get(a, b) })
			time.Sleep(8 * time.Second)
			Expect(err).ShouldNot((HaveOccurred()))
		})
	})
})
