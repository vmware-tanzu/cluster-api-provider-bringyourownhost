// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package authenticator_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/authenticator"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/test/builder"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Bootstrap Authenticator", func() {

	Context("When CSR is submitted", func() {
		var (
			ctx               context.Context
			k8sClientUncached client.Client
		)
		It("should return if the created CSR is not approved or denied", func() {
			ctx = context.Background()
			var clientErr error
			k8sClientUncached, clientErr = client.New(cfg, client.Options{Scheme: scheme.Scheme})
			Expect(clientErr).NotTo(HaveOccurred())

			ByohCSR, err := builder.CertificateSigningRequest(fmt.Sprintf(authenticator.ByohCSRNameFormat, hostName), fmt.Sprintf("byoh:host:%s", hostName), "byoh:hosts", 2048).Build()
			Expect(err).NotTo(HaveOccurred())
			Expect(k8sClientUncached.Create(ctx, ByohCSR)).NotTo(HaveOccurred())
			WaitForObjectsToBePopulatedInCache(ByohCSR)

			_, err = bootstrapAuthenticator.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: fmt.Sprintf(authenticator.ByohCSRNameFormat, hostName)}})
			Expect(err).NotTo(HaveOccurred())
		})
		It("should return error if CSR does not exist", func() {
			ctx = context.Background()
			_, err := bootstrapAuthenticator.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: fmt.Sprintf(authenticator.ByohCSRNameFormat, "fake-host")}})
			Expect(err.Error()).To(ContainSubstring("CertificateSigningRequest.certificates.k8s.io \"byoh-csr-fake-host\" not found"))
		})
	})
})
