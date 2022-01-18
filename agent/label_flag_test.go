// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// nolint: testpackage
package main

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Label flag for host agent", func() {

	Context("When the label flag is provided", func() {
		var (
			labels labelFlags
		)
		BeforeEach(func() {
			labels = make(labelFlags)
		})
		It("Should accept the single label flag", func() {
			expectedLabels := labelFlags{"k1": "v1"}
			Expect(labels.Set("k1=v1")).NotTo(HaveOccurred())
			Expect(labels).Should(Equal(expectedLabels))
		})

		It("Should accept the single label flag with comma separated kv pairs", func() {
			expectedLabels := labelFlags{"k1": "v1", "k2": "v2"}
			Expect(labels.Set("k1=v1,k2=v2")).NotTo(HaveOccurred())
			Expect(labels).Should(Equal(expectedLabels))
		})

		It("Should accept the single label flag with comma separated kv pairs with trailing comma", func() {
			expectedLabels := labelFlags{"k1": "v1", "k2": "v2"}
			Expect(labels.Set("k1=v1,k2=v2,")).NotTo(HaveOccurred())
			Expect(labels).Should(Equal(expectedLabels))
		})

		It("Should accept the multiple label flags", func() {
			expectedLabels := labelFlags{"k1": "v1", "k2": "v2"}
			Expect(labels.Set("k1=v1")).NotTo(HaveOccurred())
			Expect(labels.Set("k2=v2")).NotTo(HaveOccurred())
			Expect(labels).Should(Equal(expectedLabels))
		})

		It("Should accept the multiple label flags with comma separated kv pairs", func() {
			expectedLabels := labelFlags{"k1": "v1", "k2": "v2", "k3": "v3", "k4": "v4"}
			Expect(labels.Set("k1=v1,k2=v2")).NotTo(HaveOccurred())
			Expect(labels.Set("k3=v3,k4=v4")).NotTo(HaveOccurred())
			Expect(labels).Should(Equal(expectedLabels))
		})

		It("Should accept the multiple label flags with a mix of comma separated kv pairs and a single kv pair", func() {
			expectedLabels := labelFlags{"k1": "v1", "k2": "v2", "k3": "v3"}
			Expect(labels.Set("k1=v1,k2=v2")).NotTo(HaveOccurred())
			Expect(labels.Set("k3=v3")).NotTo(HaveOccurred())
			Expect(labels).Should(Equal(expectedLabels))
		})

		It("Should not accept the single label flag with only a key", func() {
			Expect(labels.Set("k1")).To(MatchError("invalid argument value. expect key=value, got k1"))
		})

		It("Should not accept the single label flag with comma separated errorneous input", func() {
			Expect(labels.Set("k1=v1,k2")).To(MatchError("invalid argument value. expect key=value, got k1=v1,k2"))
		})
	})
})
