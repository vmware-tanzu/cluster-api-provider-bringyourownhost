// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

var _ = Describe("ByoclusterWebhook", func() {
	XContext("When ByoCluster gets a create request", func() {
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
					Name:      "byocluster-",
					Namespace: "default",
				},
				Spec: ByoClusterSpec{},
			}
			Expect(k8sClientUncached.Create(ctx, byoCluster)).Should(Succeed())
		})

		It("should fill the default value", func() {
			byoCluster.Default()
			Expect(byoCluster.Spec.BundleLookupTag).Should(Equal("v0.1.0_alpha.2"))
		})

	})
})
