// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

const (
	// HostCleanupAnnotation annotation used to mark a host for cleanup
	HostCleanupAnnotation = "byoh.infrastructure.cluster.x-k8s.io/unregistering"
	// EndPointIPAnnotation annotation used to store the IP address of the endpoint
	EndPointIPAnnotation = "byoh.infrastructure.cluster.x-k8s.io/endpointip"
	// K8sVersionAnnotation annotation used to store the k8s version
	K8sVersionAnnotation = "byoh.infrastructure.cluster.x-k8s.io/k8sversion"
	// AttachedByoMachineLabel label used to mark a node name attached to a byo host
	AttachedByoMachineLabel = "byoh.infrastructure.cluster.x-k8s.io/byomachine-name"
	// BundleLookupBaseRegistryAnnotation annotation used to store the base registry for the bundle lookup
	BundleLookupBaseRegistryAnnotation = "byoh.infrastructure.cluster.x-k8s.io/bundle-registry"
	// BundleLookupTagAnnotation annotation used to store the bundle tag
	BundleLookupTagAnnotation = "byoh.infrastructure.cluster.x-k8s.io/bundle-tag"
)

// ByoHostSpec defines the desired state of ByoHost
type ByoHostSpec struct {
	// BootstrapSecret is an optional reference to a Cluster API Secret
	// for bootstrap purpose
	// +optional
	BootstrapSecret *corev1.ObjectReference `json:"bootstrapSecret,omitempty"`
}

// HostInfo is a set of details about the host platform.
type HostInfo struct {
	// The Operating System reported by the host.
	OSName string `json:"osname,omitempty"`

	// OS Image reported by the host.
	OSImage string `json:"osimage,omitempty"`

	// The Architecture reported by the host.
	Architecture string `json:"architecture,omitempty"`
}

// ByoHostStatus defines the observed state of ByoHost
type ByoHostStatus struct {
	// MachineRef is an optional reference to a Cluster API Machine
	// using this host.
	// +optional
	MachineRef *corev1.ObjectReference `json:"machineRef,omitempty"`

	// Conditions defines current service state of the BYOMachine.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`

	// HostDetails returns the platform details of the host.
	// +optional
	HostDetails HostInfo `json:"hostinfo,omitempty"`

	// Network returns the network status for each of the host's configured
	// network interfaces.
	// +optional
	Network []NetworkStatus `json:"network,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=byohosts,scope=Namespaced,shortName=byoh
//+kubebuilder:subresource:status

// ByoHost is the Schema for the byohosts API
type ByoHost struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ByoHostSpec   `json:"spec,omitempty"`
	Status ByoHostStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ByoHostList contains a list of ByoHost
type ByoHostList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ByoHost `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ByoHost{}, &ByoHostList{})
}

// GetConditions gets the ByoHost status conditions
func (byoHost *ByoHost) GetConditions() clusterv1.Conditions {
	return byoHost.Status.Conditions
}

// SetConditions sets the ByoHost status conditions
func (byoHost *ByoHost) SetConditions(conditions clusterv1.Conditions) {
	byoHost.Status.Conditions = conditions
}
