// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package algo

import (
	"log"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Installer Algo Tests", func() {
	var (
		installer            *BaseK8sInstaller
		outputBuilderCounter OutputBuilderCounter
	)

	const (
		stepsNum = 24
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
			Expect(err).ShouldNot((HaveOccurred()))
			Expect(outputBuilderCounter.LogCalledCnt).Should(Equal(stepsNum))
		})
	})
	Context("When Uninstallation is executed", func() {
		It("Should count each step", func() {
			err := installer.Uninstall()
			Expect(err).ShouldNot((HaveOccurred()))
			Expect(outputBuilderCounter.LogCalledCnt).Should(Equal(stepsNum))
		})
	})
	Context("When error occurs during installation", func() {
		var (
			err        error
			mockUbuntu MockUbuntuWithError
		)
		const (
			stepWhenError    = 5
			obcExpectedTotal = stepWhenError*4 + 2
		)
		BeforeEach(func() {
			mockUbuntu = MockUbuntuWithError{}
			mockUbuntu.OutputBuilder = &outputBuilderCounter
			mockUbuntu.BundlePath, err = os.MkdirTemp("", "algoErrTest")
			if err != nil {
				log.Fatal(err)
			}

			installer = &BaseK8sInstaller{
				BundlePath:      mockUbuntu.BundlePath,
				K8sStepProvider: &mockUbuntu,
				OutputBuilder:   &outputBuilderCounter}
		})
		AfterEach(func() {
			err = os.RemoveAll(mockUbuntu.BundlePath)
			if err != nil {
				log.Fatal(err)
			}
		})
		It("Should rollback all applied steps", func() {
			err = installer.Install()
			Expect(err).Should((HaveOccurred()))
			Expect(outputBuilderCounter.LogCalledCnt).Should(Equal(obcExpectedTotal))
		})
	})
})
