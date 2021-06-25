/*

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ByoMachineSpec defines the desired state of ByoMachine
type ByoMachineSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	ProviderID string `json:"providerID,omitempty"`
}

// ByoMachineStatus defines the observed state of ByoMachine
type ByoMachineStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// +optional
	Ready bool `json:"ready"`

	// Conditions defines current service state of the BYOMachine.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

// +kubebuilder:subresource:status
// +kubebuilder:object:root=true

// ByoMachine is the Schema for the Byomachines API
type ByoMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ByoMachineSpec   `json:"spec,omitempty"`
	Status ByoMachineStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ByoMachineList contains a list of ByoMachine
type ByoMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ByoMachine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ByoMachine{}, &ByoMachineList{})
}

func (m *ByoMachine) GetConditions() clusterv1.Conditions {
	return m.Status.Conditions
}

func (m *ByoMachine) SetConditions(conditions clusterv1.Conditions) {
	m.Status.Conditions = conditions
}
