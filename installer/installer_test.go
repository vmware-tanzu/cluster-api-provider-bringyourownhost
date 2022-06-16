// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package installer_test

import (
	"context"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/installer"
)

var _ = Describe("Byohost Installer Tests", func() {
	var (
		os         string
		arch       string
		k8sversion string
		downloader = installer.NewBundleDownloader("k8s", "repoAddr", "downloadPath", logr.Discard())
	)

	BeforeEach(func() {
		os = "Ubuntu 20.04"
		arch = "amd64"
		k8sversion = "1.22.9"
	})

	Context("When installer object is created for valid OS and arch", func() {
		It("should create the object successfully", func() {
			_, err := installer.NewInstaller(context.TODO(), os, arch, k8sversion, downloader)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})

	Context("When installer object is created for invalid arch", func() {
		It("should fail create the object", func() {
			arch = "arm64"
			_, err := installer.NewInstaller(context.TODO(), os, arch, k8sversion, downloader)
			Expect(err).To(MatchError(installer.ErrOsK8sNotSupported))
		})
	})

	Context("When installer object is created for invalid OS", func() {
		It("should fail create the object", func() {
			os = "rhel"
			_, err := installer.NewInstaller(context.TODO(), os, arch, k8sversion, downloader)
			Expect(err).To(MatchError(installer.ErrOsK8sNotSupported))
		})
	})
})
