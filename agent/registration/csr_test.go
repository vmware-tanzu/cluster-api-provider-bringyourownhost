// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package registration_test

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/registration"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/test/utils/csr"
	certv1 "k8s.io/api/certificates/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Registration", func() {
	var (
		ctx      = context.TODO()
		byohcsr  = registration.ByohCSR{}
		ns       = "default"
		hostName = "test-host"
	)
	BeforeEach(func() {
		byohcsr = registration.ByohCSR{K8sClient: k8sClient}
	})
	Context("When csr does not already exist", func() {
		It("should create csr", func() {
			_, err := byohcsr.CreateCSR(hostName, ns)
			Expect(err).NotTo(HaveOccurred())
			ByohCSR := &certv1.CertificateSigningRequest{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: hostName, Namespace: ns}, ByohCSR)).ToNot(HaveOccurred())

			// Validate Certificate Request
			pemData, _ := pem.Decode(ByohCSR.Spec.Request)
			Expect(pemData).ToNot(Equal(nil))
			csr, err := x509.ParseCertificateRequest(pemData.Bytes)
			Expect(err).ToNot(HaveOccurred())
			Expect(csr.Subject.CommonName).To(Equal(fmt.Sprintf("byoh:host:%s", hostName)))
			Expect(csr.Subject.Organization[0]).To(Equal("byoh:hosts"))
		})
	})

	Context("When csr already exist", func() {
		It("should not create csr", func() {
			existingByohCSR, err := csr.CreateCSRResource(hostName, "test-org", ns)
			Expect(err).NotTo(HaveOccurred())
			Expect(k8sClient.Create(ctx, existingByohCSR)).NotTo(HaveOccurred())
			_, err = byohcsr.CreateCSR(hostName, ns)
			Expect(err).NotTo(HaveOccurred())

			actualByohCSRs := &certv1.CertificateSigningRequestList{}
			Expect(k8sClient.List(ctx, actualByohCSRs)).ToNot(HaveOccurred())
			Expect(len(actualByohCSRs.Items)).To(Equal(1))

			pemData, _ := pem.Decode(actualByohCSRs.Items[0].Spec.Request)
			Expect(pemData).ToNot(Equal(nil))
			csr, err := x509.ParseCertificateRequest(pemData.Bytes)
			Expect(err).ToNot(HaveOccurred())
			Expect(csr.Subject.Organization[0]).To(Equal("test-org"))
		})
	})
	AfterEach(func() {
		Expect(k8sClient.DeleteAllOf(ctx, &certv1.CertificateSigningRequest{})).ShouldNot(HaveOccurred())
	})

})
