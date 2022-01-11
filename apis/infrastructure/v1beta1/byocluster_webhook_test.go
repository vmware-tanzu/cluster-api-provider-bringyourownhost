// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/scheme"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("ByoclusterWebhook", func() {
	Context("When ByoCluster gets a create request", func() {
		var (
			byoCluster        *ByoCluster
			ctx               context.Context
			k8sClientUncached client.Client
		)
		BeforeEach(func() {
			ctx = context.Background()
			var clientErr error
			k8sClientUncached, clientErr = client.New(cfg, client.Options{Scheme: scheme.Scheme})
			Expect(clientErr).NotTo(HaveOccurred())

			byoCluster = &ByoCluster{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ByoCluster",
					APIVersion: clusterv1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "byocluster-create",
					Namespace: "default",
				},
				Spec: ByoClusterSpec{},
			}
		})

		It("should reject the request when BundleLookupTag is empty", func() {
			err := k8sClientUncached.Create(ctx, byoCluster)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot create ByoCluster without Spec.BundleLookupTag"))
		})

	})

	Context("When ByoCluster gets a update request", func() {
		var (
			byoCluster        *ByoCluster
			ctx               context.Context
			k8sClientUncached client.Client
		)
		BeforeEach(func() {
			ctx = context.Background()
			var clientErr error
			k8sClientUncached, clientErr = client.New(cfg, client.Options{Scheme: scheme.Scheme})
			Expect(clientErr).NotTo(HaveOccurred())

			byoCluster = &ByoCluster{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ByoCluster",
					APIVersion: clusterv1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "byocluster-update",
					Namespace: "default",
				},
				Spec: ByoClusterSpec{
					BundleLookupTag: "v0.1.0_alpha.2",
				},
			}

			Expect(k8sClientUncached.Create(ctx, byoCluster)).Should(Succeed())
		})

		It("should reject the request when BundleLookupTag is empty", func() {
			byoCluster.Spec.BundleLookupTag = ""
			err := k8sClientUncached.Update(ctx, byoCluster)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot update ByoCluster with empty Spec.BundleLookupTag"))
		})

	})
})
