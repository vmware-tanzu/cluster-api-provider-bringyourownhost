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

var _ = Describe("ByoClusterTemplateWebhook", func() {
	Context("When ByoClusterTemplate gets a create request", func() {
		var (
			byoClusterTemplate *byohv1beta1.ByoClusterTemplate
			ctx                context.Context
			k8sClientUncached  client.Client
		)
		BeforeEach(func() {
			ctx = context.Background()
			var clientErr error
			k8sClientUncached, clientErr = client.New(cfg, client.Options{Scheme: scheme.Scheme})
			Expect(clientErr).NotTo(HaveOccurred())

			byoClusterTemplate = &byohv1beta1.ByoClusterTemplate{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ByoClusterTemplate",
					APIVersion: clusterv1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "byoclustertemplate-create",
					Namespace: "default",
				},
				Spec: byohv1beta1.ByoClusterTemplateSpec{},
			}
		})

		It("should succeed when BundleLookupTag is not empty", func() {
			err := k8sClientUncached.Create(ctx, byoClusterTemplate)
			Expect(err).NotTo(HaveOccurred())
		})

	})

	Context("When ByoClusterTemplate gets an update request", func() {
		var (
			byoClusterTemplate *byohv1beta1.ByoClusterTemplate
			ctx                context.Context
			k8sClientUncached  client.Client
		)
		BeforeEach(func() {
			ctx = context.Background()
			var clientErr error
			k8sClientUncached, clientErr = client.New(cfg, client.Options{Scheme: scheme.Scheme})
			Expect(clientErr).NotTo(HaveOccurred())

			byoClusterTemplate = &byohv1beta1.ByoClusterTemplate{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ByoClusterTemplate",
					APIVersion: clusterv1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "byoclustertemplate-update",
					Namespace: "default",
				},
				Spec: byohv1beta1.ByoClusterTemplateSpec{
					Template: byohv1beta1.ByoClusterTemplateResource{
						Spec: byohv1beta1.ByoClusterSpec{},
					},
				},
			}

			Expect(k8sClientUncached.Create(ctx, byoClusterTemplate)).Should(Succeed())
		})

		AfterEach(func() {
			Expect(k8sClientUncached.Delete(ctx, byoClusterTemplate)).Should(Succeed())
		})

		It("should update the cluster with new BundleLookupTag value", func() {
			err := k8sClientUncached.Update(ctx, byoClusterTemplate)
			Expect(err).NotTo(HaveOccurred())

			updatedByoClusterTemplate := &byohv1beta1.ByoClusterTemplate{}
			byoClusterLookupKey := types.NamespacedName{Name: byoClusterTemplate.Name, Namespace: byoClusterTemplate.Namespace}
			Expect(k8sClientUncached.Get(ctx, byoClusterLookupKey, updatedByoClusterTemplate)).Should(Not(HaveOccurred()))
		})
	})
})
