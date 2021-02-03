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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
)

const (
	// MachineFinalizer allows Reconcile to clean up resources associated with
	// a BYOMachine before removing it from the
	// API Server.
	MachineFinalizer = "byoh.infrastructure.cluster.x-k8s.io"
)

// BYOMachineSpec defines the desired state of BYOMachine
type BYOMachineSpec struct {
	// TODO: add HostSelector for allowing to restrict the list of candidate hosts.

	// ProviderID is the identification ID of the machine provided by the provider.
	// This field must match the provider ID as seen on the node object corresponding to this machine.
	// This field is required by higher level consumers of cluster-api. Example use case is cluster autoscaler
	// For Bare metal providers, this is a generated string.
	// +optional
	ProviderID *string `json:"providerID,omitempty"`
}

// BYOMachineStatus defines the observed state of BYOMachine
type BYOMachineStatus struct {
	// Ready is true when the provider resource is ready.
	// +optional
	Ready bool `json:"ready"`

	// Conditions defines current service state of the BYOMachine.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=byomachines,scope=Namespaced,categories=cluster-api
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.ready",description="True when the host used by the machine is ready"

// BYOMachine is the Schema for the BYOMachines API, allowing
// to manage in a declarative way a ClusterAPI Machine backed by a BYOHost.
type BYOMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BYOMachineSpec   `json:"spec,omitempty"`
	Status BYOMachineStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BYOMachineList contains a list of BYOMachine
type BYOMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BYOMachine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BYOMachine{}, &BYOMachineList{})
}

func (m *BYOMachine) GetConditions() clusterv1.Conditions {
	return m.Status.Conditions
}

func (m *BYOMachine) SetConditions(conditions clusterv1.Conditions) {
	m.Status.Conditions = conditions
}
