// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package installer

import (
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Byohost Installer Tests", func() {
	Context("When installer is created for unsupported OS", func() {
		It("Should return error", func() {
			_, err := New("repo", "downloadPath", logr.Discard())
			Expect(err).Should((HaveOccurred()))
		})
	})
	Context("When installer is created with empty bundle repo", func() {
		It("Should return error", func() {
			_, err := New("", "downloadPath", logr.Discard())
			Expect(err).Should((HaveOccurred()))
		})
	})
	Context("When installer is created with empty download path", func() {
		It("Should return error", func() {
			_, err := New("repo", "", logr.Discard())
			Expect(err).Should((HaveOccurred()))
		})
	})

})
