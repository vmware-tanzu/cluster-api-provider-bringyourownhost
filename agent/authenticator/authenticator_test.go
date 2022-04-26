// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package authenticator_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/test/utils/csr"
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

			ByohCSR, err := csr.CreateCSRResource(hostName, "byoh:hosts", ns)
			Expect(err).NotTo(HaveOccurred())
			Expect(k8sClientUncached.Create(ctx, ByohCSR)).NotTo(HaveOccurred())
			WaitForObjectsToBePopulatedInCache(ByohCSR)

			_, err = bootstrapAuthenticator.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      hostName,
					Namespace: ns}})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
