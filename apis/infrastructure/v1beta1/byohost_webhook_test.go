// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1beta1_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	byohv1beta1 "github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/apis/infrastructure/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("ByohostWebhook", func() {

	Context("When ByoHost gets a delete request", func() {
		var (
			byoHost *byohv1beta1.ByoHost
		)
		BeforeEach(func() {
			ctx = context.Background()
			byoHost = &byohv1beta1.ByoHost{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ByoHost",
					APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "host1",
					Namespace: "default",
				},
				Spec: byohv1beta1.ByoHostSpec{},
			}
			Expect(ValidUserK8sClient.Create(ctx, byoHost)).Should(Succeed())
		})

		It("should not reject the request", func() {
			err := ValidUserK8sClient.Delete(ctx, byoHost)
			Expect(err).To(BeNil())
		})

		Context("When ByoHost has MachineRef assigned", func() {
			var (
				byoMachine        *byohv1beta1.ByoMachine
				k8sClientUncached client.Client
			)
			BeforeEach(func() {
				var clientErr error
				k8sClientUncached, clientErr = client.New(cfg, client.Options{Scheme: scheme.Scheme})
				Expect(clientErr).NotTo(HaveOccurred())
				byoMachine = &byohv1beta1.ByoMachine{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ByoMachine",
						APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
					},
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "byomachine-",
						Namespace:    "default",
					},
					Spec: byohv1beta1.ByoMachineSpec{},
				}
				Expect(k8sClientUncached.Create(ctx, byoMachine)).Should(Succeed())

				ph, err := patch.NewHelper(byoHost, ValidUserK8sClient)
				Expect(err).ShouldNot(HaveOccurred())
				byoHost.Status.MachineRef = &corev1.ObjectReference{
					Kind:       "ByoMachine",
					Namespace:  byoMachine.Namespace,
					Name:       byoMachine.Name,
					UID:        byoMachine.UID,
					APIVersion: byoHost.APIVersion,
				}
				Expect(ph.Patch(ctx, byoHost, patch.WithStatusObservedGeneration{})).Should(Succeed())
			})

			It("should reject the request", func() {
				err := ValidUserK8sClient.Delete(ctx, byoHost)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("admission webhook \"vbyohost.kb.io\" denied the request: cannot delete ByoHost when MachineRef is assigned"))
			})

			AfterEach(func() {
				// delete the byohost resource
				ph, err := patch.NewHelper(byoHost, ValidUserK8sClient)
				Expect(err).ShouldNot(HaveOccurred())
				byoHost.Status.MachineRef = nil
				Expect(ph.Patch(ctx, byoHost, patch.WithStatusObservedGeneration{})).Should(Succeed())
				Expect(ValidUserK8sClient.Delete(ctx, byoHost)).Should(Succeed())
			})
		})
	})
	Context("When ByoHost gets a create request", func() {
		var (
			byoHost *byohv1beta1.ByoHost
		)
		BeforeEach(func() {
			ctx = context.Background()
			byoHost = &byohv1beta1.ByoHost{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ByoHost",
					APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "host1",
					Namespace: "default",
				},
				Spec: byohv1beta1.ByoHostSpec{},
			}
		})

		It("should allow the request from a valid user", func() {
			Expect(ValidUserK8sClient.Create(ctx, byoHost)).Should(Succeed())
			// cleanup
			Expect(ValidUserK8sClient.Delete(ctx, byoHost)).Should(Succeed())
		})

		It("should reject the request from an invalid user", func() {
			err := InvalidUserK8sClient.Create(ctx, byoHost)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("is not a valid agent username"))
		})
	})
	Context("When ByoHost gets a update request", func() {
		var (
			byoHost *byohv1beta1.ByoHost
		)
		BeforeEach(func() {
			ctx = context.Background()
			byoHost = &byohv1beta1.ByoHost{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ByoHost",
					APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "host1",
					Namespace: "default",
				},
				Spec: byohv1beta1.ByoHostSpec{},
			}
			Expect(ValidUserK8sClient.Create(ctx, byoHost)).Should(Succeed())
		})
		It("should allow the request from a valid user", func() {
			arch := "amd64"
			byoHost.Status.HostDetails.Architecture = arch
			Expect(ValidUserK8sClient.Update(ctx, byoHost)).Should(Succeed())
			Eventually(func() (done bool) {
				updatedByoHost := &byohv1beta1.ByoHost{}
				err := ValidUserK8sClient.Get(ctx, types.NamespacedName{Namespace: "default", Name: "host1"}, updatedByoHost)
				Expect(err).ShouldNot(HaveOccurred())
				return updatedByoHost.Status.HostDetails.Architecture == arch
			})
		})
		It("should reject the request from an invalid user", func() {
			err := InvalidUserK8sClient.Update(ctx, byoHost)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("is not a valid agent username"))
		})
		AfterEach(func() {
			Expect(ValidUserK8sClient.Delete(ctx, byoHost)).Should(Succeed())
		})
	})
})
