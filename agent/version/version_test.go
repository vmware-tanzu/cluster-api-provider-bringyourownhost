// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package version

import (
	"fmt"
	"os/exec"
	"runtime"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Agent version", func() {

	Context("When the version number and date are not set", func() {

		It("Leaves the version and date fields empty in response", func() {
			expected := Info{
				Major:     "",
				Minor:     "",
				Patch:     "",
				BuildDate: "",
				GoVersion: runtime.Version(),
				Compiler:  runtime.Compiler,
				Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
			}
			Expect(Get()).Should(Equal(expected))
		})
	})

	Context("When only date is set", func() {
		BeforeEach(func() {
			date, err := exec.Command("date").Output()
			Expect(err).NotTo(HaveOccurred())
			BuildDate = string(date)
		})

		AfterEach(func() {
			BuildDate = ""
		})

		It("Leaves version field empty in response", func() {
			expected := Info{
				Major:     "",
				Minor:     "",
				Patch:     "",
				BuildDate: BuildDate,
				GoVersion: runtime.Version(),
				Compiler:  runtime.Compiler,
				Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
			}
			Expect(Get()).Should(Equal(expected))
		})
	})

	Context("When version is set", func() {
		Context("When version is set to dev", func() {
			BeforeEach(func() {
				Version = "dev"
			})

			AfterEach(func() {
				Version = ""
			})

			It("Shows the version major to be dev", func() {
				expected := Info{
					Major:     "dev",
					Minor:     "",
					Patch:     "",
					BuildDate: "",
					GoVersion: runtime.Version(),
					Compiler:  runtime.Compiler,
					Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
				}
				Expect(Get()).Should(Equal(expected))
			})
		})

		Context("When version is set as git tag", func() {
			BeforeEach(func() {
				Version = "1.2.3"
			})

			AfterEach(func() {
				Version = ""
			})

			It("Shows the version according to the git tag passed", func() {
				expected := Info{
					Major:     "1",
					Minor:     "2",
					Patch:     "3",
					BuildDate: "",
					GoVersion: runtime.Version(),
					Compiler:  runtime.Compiler,
					Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
				}
				Expect(Get()).Should(Equal(expected))
			})
		})
	})
})
