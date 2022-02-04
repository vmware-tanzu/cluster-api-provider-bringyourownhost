// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package version_test

import (
	"fmt"
	"os/exec"
	"runtime"

	"sigs.k8s.io/cluster-api-provider-bringyourownhost/agent/version"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Agent version", func() {

	Context("When the version number and date are not set", func() {

		It("Leaves the version and date fields empty in response", func() {
			expected := version.Info{
				Major:     "",
				Minor:     "",
				Patch:     "",
				BuildDate: "",
				GoVersion: runtime.Version(),
				Compiler:  runtime.Compiler,
				Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
			}
			Expect(version.Get()).Should(Equal(expected))
		})
	})

	Context("When only date is set", func() {
		BeforeEach(func() {
			date, err := exec.Command("date").Output()
			Expect(err).NotTo(HaveOccurred())
			version.BuildDate = string(date)
		})

		AfterEach(func() {
			version.BuildDate = ""
		})

		It("Leaves version field empty in response", func() {
			expected := version.Info{
				Major:     "",
				Minor:     "",
				Patch:     "",
				BuildDate: version.BuildDate,
				GoVersion: runtime.Version(),
				Compiler:  runtime.Compiler,
				Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
			}
			Expect(version.Get()).Should(Equal(expected))
		})
	})

	Context("When version is set", func() {
		Context("When version is set to dev", func() {
			BeforeEach(func() {
				version.Version = version.Dev
			})

			AfterEach(func() {
				version.Version = ""
			})

			It("Shows the version major to be dev", func() {
				expected := version.Info{
					Major:     version.Dev,
					Minor:     "",
					Patch:     "",
					BuildDate: "",
					GoVersion: runtime.Version(),
					Compiler:  runtime.Compiler,
					Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
				}
				Expect(version.Get()).Should(Equal(expected))
			})
		})

		Context("When version is set as git tag", func() {
			BeforeEach(func() {
				version.Version = "v1.2.3"
			})

			AfterEach(func() {
				version.Version = ""
			})

			It("Shows the version according to the git tag passed", func() {
				expected := version.Info{
					Major:     "1",
					Minor:     "2",
					Patch:     "3",
					BuildDate: "",
					GoVersion: runtime.Version(),
					Compiler:  runtime.Compiler,
					Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
				}
				Expect(version.Get()).Should(Equal(expected))
			})
		})

		Context("When version is set as invalid git tag", func() {
			BeforeEach(func() {
				version.Version = "1.2.3"
			})

			AfterEach(func() {
				version.Version = ""
			})

			It("Leaves the version fields empty", func() {
				expected := version.Info{
					Major:     "",
					Minor:     "",
					Patch:     "",
					BuildDate: "",
					GoVersion: runtime.Version(),
					Compiler:  runtime.Compiler,
					Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
				}
				Expect(version.Get()).Should(Equal(expected))
			})
		})
	})
})
