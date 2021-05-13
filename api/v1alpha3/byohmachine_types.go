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

package v1alpha3

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ByohMachineSpec defines the desired state of ByohMachine
type ByohMachineSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of ByohMachine. Edit ByohMachine_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// ByohMachineStatus defines the observed state of ByohMachine
type ByohMachineStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:subresource:status
// +kubebuilder:object:root=true

// ByohMachine is the Schema for the byohmachines API
type ByohMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ByohMachineSpec   `json:"spec,omitempty"`
	Status ByohMachineStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ByohMachineList contains a list of ByohMachine
type ByohMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ByohMachine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ByohMachine{}, &ByohMachineList{})
}
