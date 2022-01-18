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
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("ByohostWebhook", func() {

	Context("When ByoHost gets a delete request", func() {
		var (
			byoHost           *byohv1beta1.ByoHost
			ctx               context.Context
			k8sClientUncached client.Client
		)
		BeforeEach(func() {
			ctx = context.Background()
			var clientErr error

			k8sClientUncached, clientErr = client.New(cfg, client.Options{Scheme: scheme.Scheme})
			Expect(clientErr).NotTo(HaveOccurred())

			byoHost = &byohv1beta1.ByoHost{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ByoHost",
					APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "byohost-",
					Namespace:    "default",
				},
				Spec: byohv1beta1.ByoHostSpec{},
			}
			Expect(k8sClientUncached.Create(ctx, byoHost)).Should(Succeed())
		})

		It("should not reject the request", func() {
			err := byoHost.ValidateDelete()
			Expect(err).To(BeNil())
		})

		Context("When ByoHost has MachineRef assigned", func() {
			var (
				byoMachine *byohv1beta1.ByoMachine
			)
			BeforeEach(func() {
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

				ph, err := patch.NewHelper(byoHost, k8sClientUncached)
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
				err := byoHost.ValidateDelete()
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("byohost.infrastructure.cluster.x-k8s.io \"" + byoHost.Name + "\" is forbidden: cannot delete ByoHost when MachineRef is assigned"))
			})
		})
	})

})
