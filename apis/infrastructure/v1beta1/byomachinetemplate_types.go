// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ByoMachineTemplateSpec defines the desired state of ByoMachineTemplate
type ByoMachineTemplateSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Template ByoMachineTemplateResource `json:"template"`
}

// ByoMachineTemplateStatus defines the observed state of ByoMachineTemplate
type ByoMachineTemplateStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ByoMachineTemplate is the Schema for the byomachinetemplates API
type ByoMachineTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ByoMachineTemplateSpec   `json:"spec,omitempty"`
	Status ByoMachineTemplateStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ByoMachineTemplateList contains a list of ByoMachineTemplate
type ByoMachineTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ByoMachineTemplate `json:"items"`
}

// ByoMachineTemplateResource defines the desired state of ByoMachineTemplateResource
type ByoMachineTemplateResource struct {
	// Spec is the specification of the desired behavior of the machine.
	Spec ByoMachineSpec `json:"spec"`
}

func init() {
	SchemeBuilder.Register(&ByoMachineTemplate{}, &ByoMachineTemplateList{})
}
