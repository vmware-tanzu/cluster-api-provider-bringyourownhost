// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// nolint: testpackage
package algo

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Installer Algo Tests", func() {
	var (
		installer            *BaseK8sInstaller
		outputBuilderCounter OutputBuilderCounter
	)

	const (
		stepsNum = 22
	)

	BeforeEach(func() {
		outputBuilderCounter = OutputBuilderCounter{}

		ubuntu := Ubuntu20_4K8s1_22{}
		ubuntu.OutputBuilder = &outputBuilderCounter
		ubuntu.BundlePath = ""

		installer = &BaseK8sInstaller{
			K8sStepProvider: &ubuntu,
			OutputBuilder:   &outputBuilderCounter}
	})
	Context("When Installation is executed", func() {
		It("Should count each step", func() {
			err := installer.Install()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(outputBuilderCounter.LogCalledCnt).Should(Equal(stepsNum))
		})
	})
	Context("When Uninstallation is executed", func() {
		It("Should count each step", func() {
			err := installer.Uninstall()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(outputBuilderCounter.LogCalledCnt).Should(Equal(stepsNum))
		})
	})
	Context("When error occurs during installation", func() {
		var (
			mockUbuntu MockUbuntuWithError
		)
		BeforeEach(func() {
			mockUbuntu = MockUbuntuWithError{}
			mockUbuntu.errorOnStep = 5

			installer = &BaseK8sInstaller{
				K8sStepProvider: &mockUbuntu}
		})
		It("Should rollback all applied steps", func() {
			err := installer.Install()
			Expect(err).Should(HaveOccurred())
			Expect(len(mockUbuntu.doSteps)).Should(Equal(mockUbuntu.errorOnStep))
			Expect(len(mockUbuntu.undoSteps)).Should(Equal(mockUbuntu.errorOnStep))
			for i, sz := 0, len(mockUbuntu.doSteps); i < sz; i++ {
				Expect(mockUbuntu.doSteps[i]).Should(Equal(mockUbuntu.undoSteps[sz-i-1]))
			}
		})
	})
})
