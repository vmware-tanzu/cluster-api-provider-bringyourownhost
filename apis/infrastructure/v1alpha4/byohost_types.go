/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha4

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
)

const (
	HostCleanupAnnotation = "byoh.infrastructure.cluster.x-k8s.io/unregistering"
	EndPointIPAnnotation  = "byoh.infrastructure.cluster.x-k8s.io/endpointip"
	K8sVersionAnnotation  = "byoh.infrastructure.cluster.x-k8s.io/k8sversion"
)

// ByoHostSpec defines the desired state of ByoHost
type ByoHostSpec struct {
	// BootstrapSecret is an optional reference to a Cluster API Secret
	// for bootstrap purpose
	// +optional
	BootstrapSecret *corev1.ObjectReference `json:"bootstrapSecret,omitempty"`
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

func (h *ByoHost) GetConditions() clusterv1.Conditions {
	return h.Status.Conditions
}

func (h *ByoHost) SetConditions(conditions clusterv1.Conditions) {
	h.Status.Conditions = conditions
}
