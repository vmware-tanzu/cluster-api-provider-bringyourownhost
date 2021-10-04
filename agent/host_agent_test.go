package main

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Main", func() {

	Context("When the label flag is provided", func() {
		var (
			labels labelFlags
		)
		BeforeEach(func() {
			labels = make(labelFlags)
		})
		It("Should accept the single label flag", func() {
			expLabels := labelFlags{"k1": "v1"}
			Expect(labels.Set("k1=v1")).NotTo(HaveOccurred())
			Expect(labels).Should(Equal(expLabels))
		})

		It("Should accept the single label flag with comma separated kv pairs", func() {
			expLabels := labelFlags{"k1": "v1", "k2": "v2"}
			Expect(labels.Set("k1=v1,k2=v2")).NotTo(HaveOccurred())
			Expect(labels).Should(Equal(expLabels))
		})

		It("Should accept the single label flag with comma separated kv pairs with trailing comma", func() {
			expLabels := labelFlags{"k1": "v1", "k2": "v2"}
			Expect(labels.Set("k1=v1,k2=v2,")).NotTo(HaveOccurred())
			Expect(labels).Should(Equal(expLabels))
		})

		It("Should accept the multiple label flags", func() {
			expLabels := labelFlags{"k1": "v1", "k2": "v2"}
			Expect(labels.Set("k1=v1")).NotTo(HaveOccurred())
			Expect(labels.Set("k2=v2")).NotTo(HaveOccurred())
			Expect(labels).Should(Equal(expLabels))
		})

		It("Should accept the multiple label flags with comma separated kv pairs", func() {
			expLabels := labelFlags{"k1": "v1", "k2": "v2", "k3": "v3", "k4": "v4"}
			Expect(labels.Set("k1=v1,k2=v2")).NotTo(HaveOccurred())
			Expect(labels.Set("k3=v3,k4=v4")).NotTo(HaveOccurred())
			Expect(labels).Should(Equal(expLabels))
		})

		It("Should accept the multiple label flags with a mix of comma separated kv pairs and a single kv pair", func() {
			expLabels := labelFlags{"k1": "v1", "k2": "v2", "k3": "v3"}
			Expect(labels.Set("k1=v1,k2=v2")).NotTo(HaveOccurred())
			Expect(labels.Set("k3=v3")).NotTo(HaveOccurred())
			Expect(labels).Should(Equal(expLabels))
		})

		It("Should not accept the single label flag with only a key", func() {
			Expect(labels.Set("k1")).To(MatchError("invalid argument value. expect key=value, got k1"))
		})

		It("Should not accept the single label flag with comma separated errorneous input", func() {
			Expect(labels.Set("k1=v1,k2")).To(MatchError("invalid argument value. expect key=value, got k1=v1,k2"))
		})
	})
})
