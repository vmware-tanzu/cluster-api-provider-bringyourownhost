// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package registration

import (
	"crypto/rand"
	"crypto/rsa"
	"os"

	. "github.com/onsi/ginkgo/v2"
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
		It("should write kubeconfig if bootstrap kubeconfig is valid", func() {
			testDatabootstrapValid := []byte(`
apiVersion: v1
kind: Config
clusters:
- cluster:
    certificate-authority-data: VGVzdA==
    server: https://cluster-a.com
  name: cluster-a
contexts:
- context:
    cluster: cluster-a
    namespace: ns-a
    user: user-a
  name: context-a
current-context: context-a
users:
- name: user-a
  user:
    token: mytoken-a
`)
			fileDir, err := os.MkdirTemp("", "bootstrap")
			Expect(err).ShouldNot(HaveOccurred())
			fileboot, err := os.CreateTemp(fileDir, "boostrapkubeconfig")
			Expect(err).ShouldNot(HaveOccurred())
			filekubeconfig, err := os.CreateTemp(fileDir, "kubeconfig")
			Expect(err).ShouldNot(HaveOccurred())
			err = os.WriteFile(fileboot.Name(), testDatabootstrapValid, os.FileMode(0755))
			Expect(err).ShouldNot(HaveOccurred())
			restConfig, err := LoadRESTClientConfig(fileboot.Name())
			Expect(err).ShouldNot(HaveOccurred())
			err = writeKubeconfigFromBootstrapping(restConfig, filekubeconfig.Name(), []byte("cert-data"), []byte("key-data"))
			Expect(err).ShouldNot(HaveOccurred())
			Expect(filekubeconfig.Name()).To(BeARegularFile())
			content, err := os.ReadFile(filekubeconfig.Name())
			Expect(err).ShouldNot(HaveOccurred())
			Expect(content).ShouldNot(BeEmpty())
			err = os.RemoveAll(fileDir)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
