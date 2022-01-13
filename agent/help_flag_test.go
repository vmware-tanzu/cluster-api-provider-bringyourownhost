// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Help flag for host agent", func() {
	Context("When the help flag is provided", func() {
		var (
			expectedOptions = []string{
				"-downloadpath string",
				"-kubeconfig string",
				"-label value",
				"-metricsbindaddress string",
				"-namespace string",
				"-skip-installation",
			}
		)

		It("should output the expected option", func() {
			command := exec.Command(pathToHostAgentBinary, "--help")
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session, "5s").Should(gexec.Exit(0))

			output := string(session.Err.Contents())
			for _, line := range strings.Split(strings.TrimRight(output, "\n"), "\n") {
				line = strings.TrimSpace(line)
				if !strings.HasPrefix(line, "-") {
					continue
				}
				// Any option not belongs to expectedOptions is not allowed.
				Expect(line).To(BeElementOf(expectedOptions))
			}

		})

	})
})
