// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"context"
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	v1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var _ = Describe("ByohostWebhook/Unit", func() {
	schema := runtime.NewScheme()
	err := AddToScheme(schema)
	Expect(err).NotTo(HaveOccurred())
	decoder, _ := admission.NewDecoder(schema)
	v := &ByoHostValidator{
		decoder: decoder,
	}
	Context("When ByoHost gets a create request", func() {
		var (
			byoHost    *ByoHost
			byoHostRaw []byte
			ctx        context.Context
		)
		BeforeEach(func() {
			ctx = context.TODO()
			byoHost = &ByoHost{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ByoHost",
					APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "host1",
					Namespace: "default",
				},
				Spec: ByoHostSpec{},
			}
			byoHostRaw, err = json.Marshal(byoHost)
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("Should reject create request from invalid user", func() {
			admissionRequest := admissionv1.AdmissionRequest{
				Operation: admissionv1.Create,
				UserInfo:  v1.UserInfo{Username: "unauthorized-user"},
				Object: runtime.RawExtension{
					Raw:    byoHostRaw,
					Object: byoHost,
				},
			}
			resp := v.Handle(ctx, admission.Request{AdmissionRequest: admissionRequest})
			Expect(resp.AdmissionResponse.Allowed).To(Equal(false))
			Expect(string(resp.AdmissionResponse.Result.Reason)).To(Equal(fmt.Sprintf("%s is not a valid agent username", "unauthorized-user")))
		})
		It("Should reject request from another agent user in the group", func() {
			admissionRequest := admissionv1.AdmissionRequest{
				Operation: admissionv1.Create,
				UserInfo:  v1.UserInfo{Username: "byoh:host:host2"},
				Object: runtime.RawExtension{
					Raw:    byoHostRaw,
					Object: byoHost,
				},
			}
			resp := v.Handle(ctx, admission.Request{AdmissionRequest: admissionRequest})
			Expect(resp.AdmissionResponse.Allowed).To(Equal(false))
			Expect(string(resp.AdmissionResponse.Result.Reason)).To(Equal(fmt.Sprintf("%s cannot create/update resource %s", "byoh:host:host2", "host1")))
		})
		It("Should allow request from the valid agent user", func() {
			admissionRequest := admissionv1.AdmissionRequest{
				Operation: admissionv1.Create,
				UserInfo:  v1.UserInfo{Username: "byoh:host:host1"},
				Object: runtime.RawExtension{
					Raw:    byoHostRaw,
					Object: byoHost,
				},
			}
			resp := v.Handle(ctx, admission.Request{AdmissionRequest: admissionRequest})
			Expect(resp.AdmissionResponse.Allowed).To(Equal(true))
		})
	})

	Context("When ByoHost gets an update request", func() {
		var (
			byoHost    *ByoHost
			byoHostRaw []byte
			ctx        context.Context
		)
		BeforeEach(func() {
			ctx = context.TODO()
			byoHost = &ByoHost{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ByoHost",
					APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "host1",
					Namespace: "default",
				},
				Spec: ByoHostSpec{},
			}
			byoHostRaw, err = json.Marshal(byoHost)
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("Should reject update request from invalid user", func() {
			admissionRequest := admissionv1.AdmissionRequest{
				Operation: admissionv1.Update,
				UserInfo:  v1.UserInfo{Username: "unauthorized-user"},
				Object: runtime.RawExtension{
					Raw:    byoHostRaw,
					Object: byoHost,
				},
			}
			resp := v.Handle(ctx, admission.Request{AdmissionRequest: admissionRequest})
			Expect(resp.AdmissionResponse.Allowed).To(Equal(false))
			Expect(string(resp.AdmissionResponse.Result.Reason)).To(Equal(fmt.Sprintf("%s is not a valid agent username", "unauthorized-user")))
		})
		It("Should allow update request from manager", func() {
			admissionRequest := admissionv1.AdmissionRequest{
				Operation: admissionv1.Update,
				UserInfo:  v1.UserInfo{Username: managerServiceAccount},
				Object: runtime.RawExtension{
					Raw:    byoHostRaw,
					Object: byoHost,
				},
			}
			resp := v.Handle(ctx, admission.Request{AdmissionRequest: admissionRequest})
			Expect(resp.AdmissionResponse.Allowed).To(Equal(true))
		})
		It("Should reject request from another agent user in the group", func() {
			admissionRequest := admissionv1.AdmissionRequest{
				Operation: admissionv1.Update,
				UserInfo:  v1.UserInfo{Username: "byoh:host:host2"},
				Object: runtime.RawExtension{
					Raw:    byoHostRaw,
					Object: byoHost,
				},
			}
			resp := v.Handle(ctx, admission.Request{AdmissionRequest: admissionRequest})
			Expect(resp.AdmissionResponse.Allowed).To(Equal(false))
			Expect(string(resp.AdmissionResponse.Result.Reason)).To(Equal(fmt.Sprintf("%s cannot create/update resource %s", "byoh:host:host2", "host1")))
		})
		It("Should allow request from the valid agent user", func() {
			admissionRequest := admissionv1.AdmissionRequest{
				Operation: admissionv1.Update,
				UserInfo:  v1.UserInfo{Username: "byoh:host:host1"},
				Object: runtime.RawExtension{
					Raw:    byoHostRaw,
					Object: byoHost,
				},
			}
			resp := v.Handle(ctx, admission.Request{AdmissionRequest: admissionRequest})
			Expect(resp.AdmissionResponse.Allowed).To(Equal(true))
		})
	})
	Context("When ByoHost gets an delete request", func() {
		var (
			byoHost    *ByoHost
			byoHostRaw []byte
			ctx        context.Context
		)
		BeforeEach(func() {
			ctx = context.TODO()
			byoHost = &ByoHost{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ByoHost",
					APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "host1",
					Namespace: "default",
				},
				Spec: ByoHostSpec{},
			}
			byoHostRaw, err = json.Marshal(byoHost)
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("Should allow delete request from any user", func() {
			admissionRequest := admissionv1.AdmissionRequest{
				Operation: admissionv1.Delete,
				UserInfo:  v1.UserInfo{Username: "random-user"},
				OldObject: runtime.RawExtension{
					Raw:    byoHostRaw,
					Object: byoHost,
				},
			}
			resp := v.Handle(ctx, admission.Request{AdmissionRequest: admissionRequest})
			Expect(resp.AdmissionResponse.Allowed).To(Equal(true))
		})
		It("Should reject delete request if status.MachineRef is not nil", func() {
			byoHost.Status.MachineRef = &corev1.ObjectReference{
				Kind:       "ByoMachine",
				Namespace:  "default",
				Name:       "byomachine1",
				APIVersion: byoHost.APIVersion,
			}
			byoHostRaw, err = json.Marshal(byoHost)
			Expect(err).ShouldNot(HaveOccurred())
			admissionRequest := admissionv1.AdmissionRequest{
				Operation: admissionv1.Delete,
				UserInfo:  v1.UserInfo{Username: "random-user"},
				OldObject: runtime.RawExtension{
					Raw:    byoHostRaw,
					Object: byoHost,
				},
			}
			resp := v.Handle(ctx, admission.Request{AdmissionRequest: admissionRequest})
			Expect(resp.AdmissionResponse.Allowed).To(Equal(false))
			Expect(string(resp.AdmissionResponse.Result.Reason)).To(Equal("cannot delete ByoHost when MachineRef is assigned"))
		})
	})
})
