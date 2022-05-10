// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// ByoClusterTemplateSpec defines the desired state of ByoClusterTemplate.
type ByoClusterTemplateSpec struct {
	Template ByoClusterTemplateResource `json:"template"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=byoclustertemplates,scope=Namespaced,categories=cluster-api,shortName=byoct
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Time duration since creation of ByoClusterTemplate"
// +k8s:defaulter-gen=true

// ByoClusterTemplate is the Schema for the byoclustertemplates API.
type ByoClusterTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ByoClusterTemplateSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// ByoClusterTemplateList contains a list of ByoClusterTemplate.
type ByoClusterTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ByoClusterTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ByoClusterTemplate{}, &ByoClusterTemplateList{})
}

// ByoClusterTemplateResource describes the data needed to create a ByoCluster from a template.
type ByoClusterTemplateResource struct {
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	ObjectMeta clusterv1.ObjectMeta `json:"metadata,omitempty"`
	Spec       ByoClusterSpec       `json:"spec"`
}
