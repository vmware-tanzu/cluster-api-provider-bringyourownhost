// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package registration

import (
	"crypto/rand"
	"crypto/rsa"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Registration", func() {
	Context("When generateCSR is called", func() {
		var (
			hostName = "test-host"
		)
		It("should return error if Private Key is not valid", func() {
			certData, err := generateCSR(hostName, &rsa.PrivateKey{})
			Expect(err).Should(HaveOccurred())
			Expect(certData).To(BeNil())
		})
		It("should return csrData with the correct arguments", func() {
			privateKeyData, err := rsa.GenerateKey(rand.Reader, 2048)
			Expect(err).Should(Not(HaveOccurred()))
			certData, err := generateCSR(hostName, privateKeyData)
			Expect(err).Should(Not(HaveOccurred()))
			Expect(certData).ToNot(BeNil())
		})
	})
})
