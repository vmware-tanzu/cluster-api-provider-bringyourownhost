// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package registration_test

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/registration"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/test/builder"
	certv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2/klogr"
)

var _ = Describe("CSR Registration", func() {
	var (
		ctx                = context.TODO()
		hostName           = "test-host"
		fileDir            string
		certExpiryDuration = int64((time.Hour * 24).Seconds())
	)
	Context("When bootstrap kubeconfig is provided", func() {
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
		var testCert = `
-----BEGIN CERTIFICATE-----
MIIBvzCCAWWgAwIBAgIRAMd7Mz3fPrLm1aFUn02lLHowCgYIKoZIzj0EAwIwIzEh
MB8GA1UEAwwYazNzLWNsaWVudC1jYUAxNjE2NDMxOTU2MB4XDTIxMDQxOTIxNTMz
MFoXDTIyMDQxOTIxNTMzMFowMjEVMBMGA1UEChMMc3lzdGVtOm5vZGVzMRkwFwYD
VQQDExBzeXN0ZW06bm9kZTp0ZXN0MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE
Xd9aZm6nftepZpUwof9RSUZqZDgu7dplIiDt8nnhO5Bquy2jn7/AVx20xb0Xz0d2
XLn3nn5M+lR2p3NlZmqWHaNrMGkwDgYDVR0PAQH/BAQDAgWgMBMGA1UdJQQMMAoG
CCsGAQUFBwMBMAwGA1UdEwEB/wQCMAAwHwYDVR0jBBgwFoAU/fZa5enijRDB25DF
NT1/vPUy/hMwEwYDVR0RBAwwCoIIRE5TOnRlc3QwCgYIKoZIzj0EAwIDSAAwRQIg
b3JL5+Q3zgwFrciwfdgtrKv8MudlA0nu6EDQO7eaJbwCIQDegFyC4tjGPp/5JKqQ
kovW9X7Ook/tTW0HyX6D6HRciA==
-----END CERTIFICATE-----
						`
		BeforeEach(func() {
			csrList, err := k8sClientSet.CertificatesV1().CertificateSigningRequests().List(ctx, metav1.ListOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			for _, csr := range csrList.Items {
				err = k8sClientSet.CertificatesV1().CertificateSigningRequests().Delete(ctx, csr.Name, metav1.DeleteOptions{})
				Expect(err).ShouldNot(HaveOccurred())
			}
			registration.ConfigPath = "config"
			fileDir, err = os.MkdirTemp("", "bootstrap")
			Expect(err).ShouldNot(HaveOccurred())

		})

		AfterEach(func() {
			err := os.RemoveAll(fileDir)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return error if hostname is invalid", func() {
			CSRRegistrar, err := registration.NewByohCSR(cfg, logr.Discard(), certExpiryDuration)
			Expect(err).ShouldNot(HaveOccurred())
			_, _, err = CSRRegistrar.RequestBYOHClientCert("")
			Expect(err).To(MatchError("hostname is not valid"))
		})
		It("should return client config if bootstrap kubeconfig is valid", func() {
			fileboot, err := os.CreateTemp(fileDir, "bootstrapkubeconfig")
			Expect(err).ShouldNot(HaveOccurred())
			err = os.WriteFile(fileboot.Name(), testDatabootstrapValid, os.FileMode(0755))
			Expect(err).ShouldNot(HaveOccurred())
			restConfig, err := registration.LoadRESTClientConfig(fileboot.Name())
			Expect(err).ShouldNot(HaveOccurred())
			Expect(restConfig).ToNot(BeNil())
			Expect(restConfig.Host).To(Equal("https://cluster-a.com"))
		})
		It("should return error if bootstrap kubeconfig is invalid", func() {
			fileboot, err := os.CreateTemp(fileDir, "bootstrapkubeconfig")
			Expect(err).ShouldNot(HaveOccurred())
			err = os.WriteFile(fileboot.Name(), testDatabootstrapInvalid, os.FileMode(0755))
			Expect(err).ShouldNot(HaveOccurred())
			restConfig, err := registration.LoadRESTClientConfig(fileboot.Name())
			Expect(err).Should(HaveOccurred())
			Expect(restConfig).To(BeNil())
		})
		It("should create csr if bootstrap kubeconfig is valid", func() {
			CSRRegistrar, err := registration.NewByohCSR(cfg, logr.Discard(), certExpiryDuration)
			Expect(err).ShouldNot(HaveOccurred())
			_, _, err = CSRRegistrar.RequestBYOHClientCert(hostName)
			Expect(err).NotTo(HaveOccurred())
			ByohCSR, err := k8sClientSet.CertificatesV1().CertificateSigningRequests().Get(ctx, fmt.Sprintf(registration.ByohCSRNameFormat, hostName), metav1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			// Validate k8s CSR resource
			Expect(ByohCSR.Spec.SignerName).Should(Equal(certv1.KubeAPIServerClientSignerName))
			Expect(ByohCSR.Spec.Usages).Should(Equal([]certv1.KeyUsage{certv1.UsageClientAuth}))
			Expect(*ByohCSR.Spec.ExpirationSeconds).Should(Equal(int32((time.Hour * 24).Seconds())))
			// Validate Certificate Request
			pemData, _ := pem.Decode(ByohCSR.Spec.Request)
			Expect(pemData).ToNot(Equal(nil))
			csr, err := x509.ParseCertificateRequest(pemData.Bytes)
			Expect(err).ToNot(HaveOccurred())
			Expect(csr.Subject.CommonName).To(Equal(fmt.Sprintf(registration.ByohCSRCNFormat, hostName)))
			Expect(csr.Subject.Organization[0]).To(Equal("byoh:hosts"))

			Expect(os.Remove(registration.TmpPrivateKey)).ShouldNot(HaveOccurred())
		})
		It("should fail creating CSR if the private key got changed", func() {
			byohCSR, err := builder.CertificateSigningRequest(
				fmt.Sprintf(registration.ByohCSRNameFormat, hostName),
				fmt.Sprintf(registration.ByohCSRCNFormat, hostName),
				"byoh:hosts", 2048).Build()
			Expect(err).NotTo(HaveOccurred())
			_, err = k8sClientSet.CertificatesV1().CertificateSigningRequests().Create(ctx, byohCSR, metav1.CreateOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			CSRRegistrar, err := registration.NewByohCSR(cfg, klogr.New(), certExpiryDuration)
			Expect(err).ShouldNot(HaveOccurred())
			_, _, err = CSRRegistrar.RequestBYOHClientCert(hostName)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("retrieved csr is not compatible"))

			Expect(os.Remove(registration.TmpPrivateKey)).ShouldNot(HaveOccurred())
		})
		It("should timeout if the CSR is not approved", func() {
			registration.CSRApprovalTimeout = time.Second * 5
			CSRRegistrar, err := registration.NewByohCSR(cfg, klogr.New(), certExpiryDuration)
			Expect(err).ShouldNot(HaveOccurred())
			err = CSRRegistrar.BootstrapKubeconfig(hostName)
			Expect(err).Should(HaveOccurred())
			Expect(err).To(MatchError("timed out waiting for the condition"))
			Expect(os.Remove(registration.TmpPrivateKey)).ShouldNot(HaveOccurred())
		})
		It("should return error if not able to write kubeconfig", func() {
			// Simulate ByoAdmission Controller
			go func() {
				for {
					time.Sleep(time.Millisecond * 100)
					byohCSR, err := k8sClientSet.CertificatesV1().CertificateSigningRequests().Get(ctx, fmt.Sprintf(registration.ByohCSRNameFormat, hostName), metav1.GetOptions{})
					if err != nil {
						continue
					}
					byohCSR.Status.Conditions = append(byohCSR.Status.Conditions, certv1.CertificateSigningRequestCondition{
						Type:    certv1.CertificateApproved,
						Reason:  "approved",
						Message: "approved",
						Status:  corev1.ConditionTrue,
					})
					_, err = k8sClientSet.CertificatesV1().CertificateSigningRequests().UpdateApproval(ctx, fmt.Sprintf(registration.ByohCSRNameFormat, hostName), byohCSR, metav1.UpdateOptions{})
					Expect(err).ShouldNot(HaveOccurred())
					byohCSR, err = k8sClientSet.CertificatesV1().CertificateSigningRequests().Get(ctx, fmt.Sprintf(registration.ByohCSRNameFormat, hostName), metav1.GetOptions{})
					Expect(err).ShouldNot(HaveOccurred())
					byohCSR.Status.Certificate = []byte(testCert)
					_, err = k8sClientSet.CertificatesV1().CertificateSigningRequests().UpdateStatus(ctx, byohCSR, metav1.UpdateOptions{})
					Expect(err).ShouldNot(HaveOccurred())
					return
				}
			}()
			registration.ConfigPath = "/non-existent-mount/config"
			CSRRegistrar, err := registration.NewByohCSR(cfg, klogr.New(), certExpiryDuration)
			Expect(err).ShouldNot(HaveOccurred())
			err = CSRRegistrar.BootstrapKubeconfig(hostName)
			Expect(err).Should(HaveOccurred())
			Expect(err).To(MatchError("mkdir /non-existent-mount: permission denied"))
			Expect(os.Remove(registration.TmpPrivateKey)).ShouldNot(HaveOccurred())
		})
		It("should create kubeconfig if csr is approved", func() {
			// Simulate ByoAdmission Controller
			go func() {
				for {
					time.Sleep(time.Millisecond * 100)
					byohCSR, err := k8sClientSet.CertificatesV1().CertificateSigningRequests().Get(ctx, fmt.Sprintf(registration.ByohCSRNameFormat, hostName), metav1.GetOptions{})
					if err != nil {
						continue
					}
					byohCSR.Status.Conditions = append(byohCSR.Status.Conditions, certv1.CertificateSigningRequestCondition{
						Type:    certv1.CertificateApproved,
						Reason:  "approved",
						Message: "approved",
						Status:  corev1.ConditionTrue,
					})
					_, err = k8sClientSet.CertificatesV1().CertificateSigningRequests().UpdateApproval(ctx, fmt.Sprintf(registration.ByohCSRNameFormat, hostName), byohCSR, metav1.UpdateOptions{})
					Expect(err).ShouldNot(HaveOccurred())
					byohCSR, err = k8sClientSet.CertificatesV1().CertificateSigningRequests().Get(ctx, fmt.Sprintf(registration.ByohCSRNameFormat, hostName), metav1.GetOptions{})
					Expect(err).ShouldNot(HaveOccurred())
					byohCSR.Status.Certificate = []byte(testCert)
					_, err = k8sClientSet.CertificatesV1().CertificateSigningRequests().UpdateStatus(ctx, byohCSR, metav1.UpdateOptions{})
					Expect(err).ShouldNot(HaveOccurred())
					return
				}
			}()
			CSRRegistrar, err := registration.NewByohCSR(cfg, klogr.New(), certExpiryDuration)
			Expect(err).ShouldNot(HaveOccurred())
			err = CSRRegistrar.BootstrapKubeconfig(hostName)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(registration.ConfigPath).To(BeARegularFile())
			Expect(os.Remove(registration.ConfigPath)).ShouldNot(HaveOccurred())
		})
	})
	Context("When GetBYOHConfigPath is called", func() {
		homePath := os.Getenv("HOME")
		BeforeEach(func() {
			registration.ConfigPath = ""
		})
		AfterEach(func() {
			os.Setenv("HOME", homePath)
		})
		It("should return ConfigPath if set", func() {
			registration.ConfigPath = "/tmp/config"
			actualPath := registration.GetBYOHConfigPath()
			Expect(actualPath).To(Equal(registration.ConfigPath))
		})
		It("should return default path if homedir is not set", func() {
			os.Setenv("HOME", "")
			actualPath := registration.GetBYOHConfigPath()
			Expect(actualPath).To(Equal(".byoh/config"))
		})
		It("should return path under user home dir if ConfigPath is not set", func() {
			homeDir, err := os.UserHomeDir()
			Expect(err).ShouldNot(HaveOccurred())
			expectedPath := filepath.Join(homeDir, registration.DefaultConfigPath)
			actualPath := registration.GetBYOHConfigPath()
			Expect(actualPath).To(Equal(expectedPath))
		})
	})
})
