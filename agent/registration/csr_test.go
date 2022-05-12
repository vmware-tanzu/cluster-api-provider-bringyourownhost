// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package registration_test

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/registration"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/test/builder"
	certv1 "k8s.io/api/certificates/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("CSR Registration", func() {
	var (
		ctx      = context.TODO()
		hostName = "test-host"
	)
	Context("When bootstrap kubeconfig is provided", func() {

		AfterEach(func() {
			csrList, err := clientSetFake.CertificatesV1().CertificateSigningRequests().List(ctx, v1.ListOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			for _, csr := range csrList.Items {
				err = clientSetFake.CertificatesV1().CertificateSigningRequests().Delete(ctx, csr.Name, v1.DeleteOptions{})
				Expect(err).ShouldNot(HaveOccurred())
			}
		})
		fileDir, err := ioutil.TempDir("", "bootstrap")
		Expect(err).ShouldNot(HaveOccurred())
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
		testDatabootstrapInvalid := []byte(`
apiVersion: v1
kind: Config
clusters:
- cluster:
    certificate-authority: ca.crt
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
		It("should return error if hostname is invalid", func() {
			CSRRegistrar := registration.ByohCSR{BootstrapClient: clientSetFake}
			_, _, err := CSRRegistrar.RequestBYOHClientCert("")
			Expect(err).To(MatchError("hostname is not valid"))
		})
		It("should return client config if bootstrap kubeconfig is valid", func() {
			fileboot, err := ioutil.TempFile(fileDir, "bootstrapkubeconfig")
			Expect(err).ShouldNot(HaveOccurred())
			err = os.WriteFile(fileboot.Name(), testDatabootstrapValid, os.FileMode(0755))
			Expect(err).ShouldNot(HaveOccurred())
			restConfig, err := registration.LoadRESTClientConfig(fileboot.Name())
			Expect(err).ShouldNot(HaveOccurred())
			Expect(restConfig).ToNot(BeNil())
			Expect(restConfig.Host).To(Equal("https://cluster-a.com"))
		})
		It("should return error if bootstrap kubeconfig is invalid", func() {
			fileboot, err := ioutil.TempFile(fileDir, "bootstrapkubeconfig")
			Expect(err).ShouldNot(HaveOccurred())
			err = os.WriteFile(fileboot.Name(), testDatabootstrapInvalid, os.FileMode(0755))
			Expect(err).ShouldNot(HaveOccurred())
			restConfig, err := registration.LoadRESTClientConfig(fileboot.Name())
			Expect(err).Should(HaveOccurred())
			Expect(restConfig).To(BeNil())
		})
		It("should create csr if bootstrap kubeconfig is valid", func() {
			CSRRegistrar := registration.ByohCSR{BootstrapClient: clientSetFake}
			_, _, err := CSRRegistrar.RequestBYOHClientCert(hostName)
			Expect(err).NotTo(HaveOccurred())
			ByohCSR, err := clientSetFake.CertificatesV1().CertificateSigningRequests().Get(ctx, fmt.Sprintf(registration.ByohCSRNameFormat, hostName), v1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			// Validate k8s CSR resource
			Expect(ByohCSR.Spec.SignerName).Should(Equal(certv1.KubeAPIServerClientSignerName))
			Expect(ByohCSR.Spec.Usages).Should(Equal([]certv1.KeyUsage{certv1.UsageClientAuth}))
			Expect(*ByohCSR.Spec.ExpirationSeconds).Should(Equal(int32(registration.ExpirationSeconds)))
			// Validate Certificate Request
			pemData, _ := pem.Decode(ByohCSR.Spec.Request)
			Expect(pemData).ToNot(Equal(nil))
			csr, err := x509.ParseCertificateRequest(pemData.Bytes)
			Expect(err).ToNot(HaveOccurred())
			Expect(csr.Subject.CommonName).To(Equal(fmt.Sprintf(registration.ByohCSRCNFormat, hostName)))
			Expect(csr.Subject.Organization[0]).To(Equal("byoh:hosts"))

			Expect(os.Remove(registration.TmpPrivateKey)).ShouldNot(HaveOccurred())
		})
		It("should write kubeconfig if bootstrap kubeconfig is valid", func() {
			fileboot, err := ioutil.TempFile(fileDir, "boostrapkubeconfig")
			Expect(err).ShouldNot(HaveOccurred())
			filekubeconfig, err := ioutil.TempFile(fileDir, "kubeconfig")
			Expect(err).ShouldNot(HaveOccurred())
			err = os.WriteFile(fileboot.Name(), testDatabootstrapValid, os.FileMode(0755))
			Expect(err).ShouldNot(HaveOccurred())
			restConfig, err := registration.LoadRESTClientConfig(fileboot.Name())
			Expect(err).ShouldNot(HaveOccurred())
			err = registration.WriteKubeconfigFromBootstrapping(restConfig, filekubeconfig.Name(), "cert-data", "key-data")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(filekubeconfig.Name()).To(BeARegularFile())
			content, err := os.ReadFile(filekubeconfig.Name())
			Expect(err).ShouldNot(HaveOccurred())
			Expect(content).ShouldNot(BeEmpty())
		})
		It("should fail creating CSR if the private key got changed", func() {
			byohCSR, err := builder.CertificateSigningRequest(
				fmt.Sprintf(registration.ByohCSRNameFormat, hostName),
				fmt.Sprintf(registration.ByohCSRCNFormat, hostName),
				"byoh:hosts", 2048).Build()
			Expect(err).NotTo(HaveOccurred())
			_, err = clientSetFake.CertificatesV1().CertificateSigningRequests().Create(ctx, byohCSR, v1.CreateOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			CSRRegistrar := registration.ByohCSR{BootstrapClient: clientSetFake}
			_, _, err = CSRRegistrar.RequestBYOHClientCert(hostName)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("retrieved csr is not compatible"))

			Expect(os.Remove(registration.TmpPrivateKey)).ShouldNot(HaveOccurred())
		})
	})
})
