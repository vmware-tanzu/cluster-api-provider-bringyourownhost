// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controllers_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/test/utils/csr"
	certv1 "k8s.io/api/certificates/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = FDescribe("Controllers/ByoadmissionController", func() {
	var (
		ctx context.Context
	)
	It("should approve the Byoh CSRs", func() {
		ctx = context.TODO()

		CSR, err := csr.CreateCSRResource(defaultByoHostName, "byoh:hosts")
		Expect(err).NotTo(HaveOccurred())
		_, err = clientSetFake.CertificatesV1().CertificateSigningRequests().Create(ctx, CSR, v1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		objectKey := types.NamespacedName{Namespace: defaultNamespace, Name: defaultByoHostName}
		_, err = byoAdmissionReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: objectKey})
		Expect(err).ShouldNot(HaveOccurred())

		updateByohCSR, err := clientSetFake.CertificatesV1().CertificateSigningRequests().Get(ctx, defaultByoHostName, v1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(updateByohCSR.Status.Conditions).Should(ContainElement(certv1.CertificateSigningRequestCondition{
			Type:   certv1.CertificateApproved,
			Reason: "Approved by ByoAdmission Controller",
		}))
	})
})
