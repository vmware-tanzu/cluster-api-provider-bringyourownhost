// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1beta1_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	byohv1beta1 "github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/apis/infrastructure/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubectl/pkg/scheme"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("ByoclusterWebhook", func() {
	Context("When ByoCluster gets a create request", func() {
		var (
			byoCluster        *byohv1beta1.ByoCluster
			ctx               context.Context
			k8sClientUncached client.Client
		)
		BeforeEach(func() {
			ctx = context.Background()
			var clientErr error
			k8sClientUncached, clientErr = client.New(cfg, client.Options{Scheme: scheme.Scheme})
			Expect(clientErr).NotTo(HaveOccurred())

			byoCluster = &byohv1beta1.ByoCluster{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ByoCluster",
					APIVersion: clusterv1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "byocluster-create",
					Namespace: "default",
				},
				Spec: byohv1beta1.ByoClusterSpec{},
			}
		})

		It("should reject the request when BundleLookupTag is empty", func() {
			err := k8sClientUncached.Create(ctx, byoCluster)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("admission webhook \"vbyocluster.kb.io\" denied the request: ByoCluster.infrastructure.cluster.x-k8s.io \"" + byoCluster.Name + "\" is invalid: <nil>: Internal error: cannot create ByoCluster without Spec.BundleLookupTag"))
		})

		It("should success when BundleLookupTag is not empty", func() {
			byoCluster.Spec.BundleLookupTag = "v0.1.0_alpha.2"
			err := k8sClientUncached.Create(ctx, byoCluster)
			Expect(err).NotTo(HaveOccurred())
		})

	})

	Context("When ByoCluster gets an update request", func() {
		var (
			byoCluster        *byohv1beta1.ByoCluster
			ctx               context.Context
			k8sClientUncached client.Client
		)
		BeforeEach(func() {
			ctx = context.Background()
			var clientErr error
			k8sClientUncached, clientErr = client.New(cfg, client.Options{Scheme: scheme.Scheme})
			Expect(clientErr).NotTo(HaveOccurred())

			byoCluster = &byohv1beta1.ByoCluster{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ByoCluster",
					APIVersion: clusterv1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "byocluster-update",
					Namespace: "default",
				},
				Spec: byohv1beta1.ByoClusterSpec{
					BundleLookupTag: "v0.1.0_alpha.2",
				},
			}

			Expect(k8sClientUncached.Create(ctx, byoCluster)).Should(Succeed())
		})

		AfterEach(func() {
			Expect(k8sClientUncached.Delete(ctx, byoCluster)).Should(Succeed())
		})

		It("should reject the request when BundleLookupTag is empty", func() {
			byoCluster.Spec.BundleLookupTag = ""
			err := k8sClientUncached.Update(ctx, byoCluster)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("admission webhook \"vbyocluster.kb.io\" denied the request: ByoCluster.infrastructure.cluster.x-k8s.io \"" + byoCluster.Name + "\" is invalid: <nil>: Internal error: cannot update ByoCluster with empty Spec.BundleLookupTag"))
		})

		It("should update the cluster with new BundleLookupTag value", func() {
			newBundleLookupTag := "new_tag"

			byoCluster.Spec.BundleLookupTag = newBundleLookupTag
			err := k8sClientUncached.Update(ctx, byoCluster)
			Expect(err).NotTo(HaveOccurred())

			updatedByoCluster := &byohv1beta1.ByoCluster{}
			byoCLusterLookupKey := types.NamespacedName{Name: byoCluster.Name, Namespace: byoCluster.Namespace}
			Expect(k8sClientUncached.Get(ctx, byoCLusterLookupKey, updatedByoCluster)).Should(Not(HaveOccurred()))
			Expect(updatedByoCluster.Spec.BundleLookupTag).To(Equal(newBundleLookupTag))
		})
	})
})
