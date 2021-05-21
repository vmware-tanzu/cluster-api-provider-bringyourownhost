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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ByoHostSpec defines the desired state of ByoHost
type ByoHostSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of ByoHost. Edit byohost_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// ByoHostStatus defines the observed state of ByoHost
type ByoHostStatus struct {
	// MachineRef is an optional reference to a Cluster API Machine
	// using this host.
	// +optional
	MachineRef *corev1.ObjectReference `json:"machineRef,omitempty"`
}

//+kubebuilder:object:root=true
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
