// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controllers_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/test/builder"
	certv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Controllers/ByoadmissionController", func() {
	var (
		err error
		CSR *certv1.CertificateSigningRequest
	)

	It("should return error for non-existent CSR", func() {
		// Call Reconcile method for a non-existing CSR
		objectKey := types.NamespacedName{Name: defaultByoHostName}
		_, err = byoAdmissionReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: objectKey})
		Expect(err).To(BeNil())
	})

	Context("When a CSR is created", func() {
		BeforeEach(func() {
			ctx = context.Background()

			// Create a CSR resource for each test
			CSR, err = builder.CertificateSigningRequest(defaultByoHostName, "test-cn", "test-org", 2048).Build()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should approve the Byoh CSR", func() {
			// Create a dummy CSR request
			_, err = clientSetFake.CertificatesV1().CertificateSigningRequests().Create(ctx, CSR, v1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Call Reconcile method
			objectKey := types.NamespacedName{Name: defaultByoHostName}
			_, err = byoAdmissionReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: objectKey})
			Expect(err).ShouldNot(HaveOccurred())

			// Fetch the updated CSR
			var updateByohCSR *certv1.CertificateSigningRequest
			updateByohCSR, err = clientSetFake.CertificatesV1().CertificateSigningRequests().Get(ctx, defaultByoHostName, v1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updateByohCSR.Status.Conditions).Should(ContainElement(certv1.CertificateSigningRequestCondition{
				Type:   certv1.CertificateApproved,
				Reason: "Approved by ByoAdmission Controller",
				Status: corev1.ConditionTrue,
			}))
		})

		It("should not approve a denied CSR", func() {
			// Create a fake denied CSR request
			CSR.Status.Conditions = append(CSR.Status.Conditions, certv1.CertificateSigningRequestCondition{
				Type: certv1.CertificateDenied,
			})

			_, err = clientSetFake.CertificatesV1().CertificateSigningRequests().Create(ctx, CSR, v1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Call Reconcile method
			objectKey := types.NamespacedName{Name: defaultByoHostName}
			_, err = byoAdmissionReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: objectKey})
			Expect(err).To(BeNil())
		})

		It("should not approve an already approved CSR", func() {
			// Create a fake approved CSR request
			CSR.Status.Conditions = append(CSR.Status.Conditions, certv1.CertificateSigningRequestCondition{
				Type: certv1.CertificateApproved,
			})

			_, err = clientSetFake.CertificatesV1().CertificateSigningRequests().Create(ctx, CSR, v1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Call Reconcile method
			objectKey := types.NamespacedName{Name: defaultByoHostName}
			_, err = byoAdmissionReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: objectKey})
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			Expect(clientSetFake.CertificatesV1().CertificateSigningRequests().Delete(ctx, defaultByoHostName, v1.DeleteOptions{})).ShouldNot(HaveOccurred())
		})

	})

})
