// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1beta1_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/scheme"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("BoostrapTokenWebhook", func() {

	FContext("When a boostrap token secret create request is received", func() {
		var (
			bootstrapTokenSecret *corev1.Secret
			ctx                  context.Context
			k8sClientUncached    client.Client
		)
		BeforeEach(func() {
			ctx = context.Background()
			var clientErr error
			k8sClientUncached, clientErr = client.New(cfg, client.Options{Scheme: scheme.Scheme})
			Expect(clientErr).NotTo(HaveOccurred())

			bootstrapTokenSecret = &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: clusterv1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "byocluster-create",
					Namespace: "default",
				},
			}

		})
		It("should deny secret creation is namespace is other than kube-system", func() {
			err := k8sClientUncached.Create(ctx, bootstrapTokenSecret)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("admission webhook \"vsecret.kb.io\" denied the request: boostrap secrets can only be created in kube-system namespace and not default"))
		})
	})
})
