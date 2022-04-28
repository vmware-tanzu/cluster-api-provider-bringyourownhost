// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controllers_test

import (
	"context"
	"crypto/x509"
	"encoding/pem"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/test/utils/csr"
	certv1 "k8s.io/api/certificates/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Controllers/ByoadmissionController", func() {
	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
	})
	It("should approve the Byoh CSRs", func() {
		// Create a dummy CSR request
		CSR, err := csr.CreateCSRResource(defaultByoHostName, "byoh:hosts")
		Expect(err).NotTo(HaveOccurred())
		_, err = clientSetFake.CertificatesV1().CertificateSigningRequests().Create(ctx, CSR, v1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		// Start recoincilation
		objectKey := types.NamespacedName{Name: defaultByoHostName}
		_, err = byoAdmissionReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: objectKey})
		Expect(err).ShouldNot(HaveOccurred())

		// Fetch the updated CSR
		updateByohCSR, err := clientSetFake.CertificatesV1().CertificateSigningRequests().Get(ctx, defaultByoHostName, v1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(updateByohCSR.Status.Conditions).Should(ContainElement(certv1.CertificateSigningRequestCondition{
			Type:   certv1.CertificateApproved,
			Reason: "Approved by ByoAdmission Controller",
		}))
	})

	It("CSR should have a proper name", func() {
		CSR, err := csr.CreateCSRResource("byoh-csr-host1", "byoh:hosts")
		Expect(err).NotTo(HaveOccurred())
		_, err = clientSetFake.CertificatesV1().CertificateSigningRequests().Create(ctx, CSR, v1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		// Validate Certificate Request and Certificate details
		updateByohCSR, err := clientSetFake.CertificatesV1().CertificateSigningRequests().Get(ctx, "byoh-csr-host1", v1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(updateByohCSR.Name).To(Equal("byoh-csr-host1"))
		pemData, _ := pem.Decode(updateByohCSR.Spec.Request)
		Expect(pemData).ToNot(Equal(nil))
		csr, err := x509.ParseCertificateRequest(pemData.Bytes)
		Expect(err).ToNot(HaveOccurred())
		Expect(csr.Subject.Organization[0]).To(Equal("byoh:hosts"))
	})
})
