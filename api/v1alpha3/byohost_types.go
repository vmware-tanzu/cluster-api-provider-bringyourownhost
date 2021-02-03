/*
Copyright the Cluster API Provider BYOH contributors.

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

package v1alpha3

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
)

// BYOHostSpec defines the desired state of BYOHost.
type BYOHostSpec struct{}

// BYOHostStatus defines the observed state of BYOHost
type BYOHostStatus struct {
	// MachineRef is an optional reference to a Cluster API Machine
	// using this host.
	// +optional
	MachineRef *corev1.ObjectReference `json:"machineRef,omitempty"`

	// Conditions defines current service state of the BYOMachine.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=byohosts,scope=Cluster,categories=cluster-api
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Machine Namespace",type="string",JSONPath=".status.machineRef.namespace",description="Namespace of the machine deployed on the host"
// +kubebuilder:printcolumn:name="Machine Name",type="string",JSONPath=".status.machineRef.name",description="Name of the machine deployed on the host"

// BYOHost is the Schema for the BYOHosts API.
// A BYO Host is an host provisioned outside of Cluster API, with the entire stack
// up to the OS already configured; the Kubernetes host components like containerd,
// kubelet, kubeadm etc. might be pre-provisioned  or managed by this infrastructure provider.
type BYOHost struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BYOHostSpec   `json:"spec,omitempty"`
	Status BYOHostStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BYOHostList contains a list of BYOHost
type BYOHostList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BYOHost `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BYOHost{}, &BYOHostList{})
}

func (h *BYOHost) GetConditions() clusterv1.Conditions {
	return h.Status.Conditions
}

func (h *BYOHost) SetConditions(conditions clusterv1.Conditions) {
	h.Status.Conditions = conditions
}
