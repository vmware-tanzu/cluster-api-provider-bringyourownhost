// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package installer

import (
	"errors"
	"fmt"
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

func (mi *mockImgpkg) Get(_, _ string) error {
	mi.callCount++
	return nil
}

func (mi *mockImgpkg) GetErrorConnectionTimedOut(_, _ string) error {
	mi.callCount++
	return errors.New("Extracting image into directory: read tcp 192.168.0.1:1->1.1.1.1:1: read: connection timed out")
}

func (mi *mockImgpkg) GetErrorNameResolution(_, _ string) error {
	mi.callCount++
	return errors.New("Fetching image: Get \"a.a/\": dial tcp: lookup a.a: Temporary failure in name resolution")
}
func (mi *mockImgpkg) GetErrorOurOfSpace(_, _ string) error {
	mi.callCount++
	return errors.New("Extracting image into directory: write /tmp/asd: no space left on device")
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
		normalizedOsVersion = "ubuntu_20.04.3_x64"
		k8sVersion = "1.22"
		repoAddr = ""
		var err error
		downloadPath, err = ioutil.TempDir(string(Separator)+"tmp", "downloaderTest")
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
	Context("When given correct arguments", func() {

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
		})
		It("Should create dir if missing and download bundle", func() {
			bd.downloadPath = fmt.Sprintf("%s%c%s%c%s%c%s", bd.downloadPath, Separator, "a", Separator, "b", Separator, "c")
			err := bd.DownloadFromRepo(
				normalizedOsVersion,
				k8sVersion,
				func(a, b string) error { return mi.Get(a, b) })
			time.Sleep(8 * time.Second)
			Expect(err).ShouldNot((HaveOccurred()))
		})
	})
	Context("When there is error during download", func() {
		It("Should return error if given bad repo", func() {
			bd.repoAddr = "a.a"
			err := bd.Download(normalizedOsVersion, k8sVersion)
			Expect(err).Should((HaveOccurred()))
			Expect(err.Error()).Should(Equal(ErrDownloadBadRepo))
		})
		It("Should return error if connection timed out", func() {
			err := bd.DownloadFromRepo(
				normalizedOsVersion,
				k8sVersion,
				func(a, b string) error { return mi.GetErrorConnectionTimedOut(a, b) })
			Expect(err).Should((HaveOccurred()))
			Expect(err.Error()).Should(Equal(ErrDownloadConnectionTimedOut))
		})
		It("Should return error if failure in name resolution", func() {
			err := bd.DownloadFromRepo(
				normalizedOsVersion,
				k8sVersion,
				func(a, b string) error { return mi.GetErrorNameResolution(a, b) })
			Expect(err).Should((HaveOccurred()))
			Expect(err.Error()).Should(Equal(ErrDownloadNameResolution))
		})
		It("Should return error if out of space", func() {
			err := bd.DownloadFromRepo(
				normalizedOsVersion,
				k8sVersion,
				func(a, b string) error { return mi.GetErrorOurOfSpace(a, b) })
			Expect(err).Should((HaveOccurred()))
			Expect(err.Error()).Should(Equal(ErrDownloadOutOfSpace))
		})

	})
})
