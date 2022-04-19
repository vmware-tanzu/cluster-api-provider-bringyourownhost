// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// K8sInstallerConfigSpec defines the desired state of K8sInstallerConfig
type K8sInstallerConfigSpec struct {
	// BundleRepo is the OCI registry from which the carvel imgpkg bundle will be downloaded
	BundleRepo string `json:"bundleRepo"`

	// BundleType is the type of bundle (e.g. k8s) that needs to be downloaded
	BundleType string `json:"bundleType"`
}

// K8sInstallerConfigStatus defines the observed state of K8sInstallerConfig
type K8sInstallerConfigStatus struct {
	// Ready indicates the InstallationSecret field is ready to be consumed
	// +optional
	Ready bool `json:"ready,omitempty"`

	// InstallationSecret is an optional reference to a generated installation secret by K8sInstallerConfig controller
	// +optional
	InstallationSecret *corev1.ObjectReference `json:"installationSecret,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// K8sInstallerConfig is the Schema for the k8sinstallerconfigs API
type K8sInstallerConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   K8sInstallerConfigSpec   `json:"spec,omitempty"`
	Status K8sInstallerConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// K8sInstallerConfigList contains a list of K8sInstallerConfig
type K8sInstallerConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []K8sInstallerConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&K8sInstallerConfig{}, &K8sInstallerConfigList{})
}
