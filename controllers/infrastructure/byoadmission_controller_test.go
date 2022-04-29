// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controllers_test

import (
	"context"
	"fmt"

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
		err error
		CSR *certv1.CertificateSigningRequest
	)

	It("should return error for non-existent CSR", func() {
		// Start recoincilation for a non-existing CSR
		objectKey := types.NamespacedName{Name: defaultByoHostName}
		_, err = byoAdmissionReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: objectKey})
		Expect(err).To(HaveOccurred())
	})

	Context("When a CSR request is made", func() {
		BeforeEach(func() {
			ctx = context.Background()

			// Create a CSR resource for each test
			CSR, err = csr.CreateCSRResource(defaultByoHostName, "byoh:hosts")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should approve the Byoh CSRs", func() {
			// Create a dummy CSR request
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

		It("should not approve a denied CSR", func() {
			// Create a fake denied CSR request
			CSR.Status.Conditions = append(CSR.Status.Conditions, certv1.CertificateSigningRequestCondition{
				Type: certv1.CertificateDenied,
			})

			_, err = clientSetFake.CertificatesV1().CertificateSigningRequests().Create(ctx, CSR, v1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Start recoincilation
			objectKey := types.NamespacedName{Name: defaultByoHostName}
			_, err = byoAdmissionReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: objectKey})
			Expect(err).To(Equal(fmt.Errorf("CertificateSigningRequest %s is already denied", CSR.Name)))
		})

		It("should not approve an already approved CSR", func() {
			// Create a fake approved CSR request
			CSR.Status.Conditions = append(CSR.Status.Conditions, certv1.CertificateSigningRequestCondition{
				Type: certv1.CertificateApproved,
			})

			_, err = clientSetFake.CertificatesV1().CertificateSigningRequests().Create(ctx, CSR, v1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Start recoincilation
			objectKey := types.NamespacedName{Name: defaultByoHostName}
			_, err = byoAdmissionReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: objectKey})
			Expect(err).To(Equal(fmt.Errorf("CertificateSigningRequest %s is already approved", CSR.Name)))
		})

		AfterEach(func() {
			Expect(clientSetFake.CertificatesV1().CertificateSigningRequests().Delete(ctx, defaultByoHostName, v1.DeleteOptions{})).ShouldNot(HaveOccurred())
		})

	})

})
