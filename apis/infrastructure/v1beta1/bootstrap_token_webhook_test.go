// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1beta1_test

import (
	"context"
	"math/rand"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/test/builder"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("BoostrapTokenWebhook", func() {

	Context("When a boostrap token secret create request is received", func() {

		Context("When the secret name does not start with bootstrap-token-", func() {
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

			})
			It("should allow for the secret to be created", func() {
				bootstrapTokenSecret = builder.Secret("default", "i-am-a-random-secret").Build()
				err := k8sClientUncached.Create(ctx, bootstrapTokenSecret)
				Expect(err).ToNot(HaveOccurred())

				bootstrapTokenLookupKey := types.NamespacedName{Name: bootstrapTokenSecret.Name, Namespace: bootstrapTokenSecret.Namespace}
				createdBootstrapTokenSecret := &corev1.Secret{}
				err = k8sClientUncached.Get(ctx, bootstrapTokenLookupKey, createdBootstrapTokenSecret)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when the secret name starts with bootstrap-token-", func() {
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

			})

			It("should deny secret creation if namespace is other than kube-system", func() {
				bootstrapTokenSecret = builder.Secret("default", "bootstrap-token-random-secret").Build()
				err := k8sClientUncached.Create(ctx, bootstrapTokenSecret)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("admission webhook \"vsecret.kb.io\" denied the request: boostrap token secrets can only be created in kube-system namespace and not default"))
			})

			It("should deny secret creation if the token format is incorrect", func() {
				stringData := map[string]string{
					"token-id":     "abc",
					"token-secret": "xyz",
				}
				bootstrapTokenSecret = builder.Secret("kube-system", "bootstrap-token-random-secret").
					WithStringData(stringData).Build()
				err := k8sClientUncached.Create(ctx, bootstrapTokenSecret)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("admission webhook \"vsecret.kb.io\" denied the request: incorrect format for token-id and token-secret"))
			})

			It("should allow secret creation if all validations are passed", func() {
				tokenID := generateBootstrapTokenRandomValue(6)
				tokenSecret := generateBootstrapTokenRandomValue(16)
				stringData := map[string]string{
					"token-id":     tokenID,
					"token-secret": tokenSecret,
				}
				bootstrapTokenSecret = builder.Secret("kube-system", "bootstrap-token-random-secret").
					WithStringData(stringData).Build()

				err := k8sClientUncached.Create(ctx, bootstrapTokenSecret)
				Expect(err).ToNot(HaveOccurred())

				bootstrapTokenLookupKey := types.NamespacedName{Name: bootstrapTokenSecret.Name, Namespace: bootstrapTokenSecret.Namespace}
				createdBootstrapTokenSecret := &corev1.Secret{}
				err = k8sClientUncached.Get(ctx, bootstrapTokenLookupKey, createdBootstrapTokenSecret)
				Expect(err).ToNot(HaveOccurred())
			})
		})

	})
})

func generateBootstrapTokenRandomValue(length int) string {
	rand.Seed(time.Now().Unix())
	charSet := "abcdedfghijklmnopqrstuvwxyz01234556789"
	var output strings.Builder
	output.Reset()
	for i := 0; i < length; i++ {
		random := rand.Intn(len(charSet))
		randomChar := charSet[random]
		output.WriteString(string(randomChar))
	}
	return output.String()
}
