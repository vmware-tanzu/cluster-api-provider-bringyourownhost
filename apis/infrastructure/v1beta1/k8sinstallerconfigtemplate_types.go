// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// K8sInstallerConfigTemplateSpec defines the desired state of K8sInstallerConfigTemplate
type K8sInstallerConfigTemplateSpec struct {
	Template K8sInstallerConfigTemplateResource `json:"template"`
}

// K8sInstallerConfigTemplateStatus defines the observed state of K8sInstallerConfigTemplate
type K8sInstallerConfigTemplateStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// K8sInstallerConfigTemplate is the Schema for the k8sinstallerconfigtemplates API
type K8sInstallerConfigTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   K8sInstallerConfigTemplateSpec   `json:"spec,omitempty"`
	Status K8sInstallerConfigTemplateStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// K8sInstallerConfigTemplateList contains a list of K8sInstallerConfigTemplate
type K8sInstallerConfigTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []K8sInstallerConfigTemplate `json:"items"`
}

type K8sInstallerConfigTemplateResource struct {
	// Spec is the specification of the desired behavior of the installer config.
	Spec K8sInstallerConfigSpec `json:"spec"`
}

func init() {
	SchemeBuilder.Register(&K8sInstallerConfigTemplate{}, &K8sInstallerConfigTemplateList{})
}
