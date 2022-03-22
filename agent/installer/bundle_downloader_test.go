// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// nolint: testpackage
package installer

import (
	"errors"
	"log"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type mockImgpkg struct {
	callCount int
	err       error
}

func (mi *mockImgpkg) Get(_, _ string) error {
	mi.callCount++
	return mi.err
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

	const testTag = "test-tag"

	BeforeEach(func() {
		normalizedOsVersion = "Ubuntu_20.04.3_x64"
		k8sVersion = "v1.22.5"
		repoAddr = ""
		var err error
		downloadPath, err = os.MkdirTemp("", "downloaderTest")
		if err != nil {
			log.Fatal(err)
		}
		bd = &bundleDownloader{BundleTypeK8s, repoAddr, downloadPath, logr.Discard()}
		mi = &mockImgpkg{}
	})
	AfterEach(func() {
		err := os.RemoveAll(downloadPath)
		if err != nil {
			log.Fatal(err)
		}
	})
	Context("When given correct arguments", func() {
		It("Should download bundle", func() {
			// Test download on cache missing
			err := bd.DownloadFromRepo(
				normalizedOsVersion,
				k8sVersion,
				testTag,
				mi.Get)
			Expect(err).ShouldNot(HaveOccurred())

			// Test no download on cache hit
			err = bd.DownloadFromRepo(
				normalizedOsVersion,
				k8sVersion,
				testTag,
				mi.Get)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(mi.callCount).Should(Equal(1))
		})
		It("Should create dir if missing and download bundle", func() {
			bd.downloadPath = filepath.Join(bd.downloadPath, "a", "b", "c")
			err := bd.DownloadFromRepo(
				normalizedOsVersion,
				k8sVersion,
				testTag,
				mi.Get)
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("Should create and rename dir correctly after successful download", func() {
			bd.repoAddr = "repo.ccoomm/r/"
			bd.downloadPath = filepath.Join(bd.downloadPath)
			err := bd.DownloadFromRepo(
				normalizedOsVersion,
				k8sVersion,
				testTag,
				mi.Get)
			Expect(err).ShouldNot(HaveOccurred())
			_, err = os.Stat(bd.GetBundleDirPath(k8sVersion))
			Expect(err).ShouldNot(HaveOccurred())
			notExist := os.IsNotExist(err)
			Expect(notExist).ShouldNot(BeTrue())

			_, err = os.Stat(bd.GetBundleDirPath(k8sVersion + "a"))
			Expect(err).Should(HaveOccurred())
			notExist = os.IsNotExist(err)
			Expect(notExist).Should(BeTrue())
		})
	})
	Context("When there is error during download", func() {
		It("Should return error if given bad repo", func() {
			mi.err = errors.New("fetching image: Get \"a.a.com/\": dial tcp: lookup a.a.com: no such host")
			err := bd.DownloadFromRepo(
				normalizedOsVersion,
				k8sVersion,
				testTag,
				mi.Get)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(Equal(ErrBundleDownload.Error()))
		})
		It("Should return error if connection timed out", func() {
			mi.err = errors.New("extracting image into directory: read tcp 192.168.0.1:1->1.1.1.1:1: read: connection timed out")
			err := bd.DownloadFromRepo(
				normalizedOsVersion,
				k8sVersion,
				testTag,
				mi.Get)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(Equal(ErrBundleDownload.Error()))
		})
		It("Should return error if failure in name resolution", func() {
			mi.err = errors.New("fetching image: Get \"a.a/\": dial tcp: lookup a.a: Temporary failure in name resolution")
			err := bd.DownloadFromRepo(
				normalizedOsVersion,
				k8sVersion,
				testTag,
				mi.Get)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(Equal(ErrBundleDownload.Error()))
		})
		It("Should return error if out of space", func() {
			mi.err = errors.New("extracting image into directory: write /tmp/asd: no space left on device")
			err := bd.DownloadFromRepo(
				normalizedOsVersion,
				k8sVersion,
				testTag,
				mi.Get)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(Equal(ErrBundleExtract.Error()))
		})

	})
})
