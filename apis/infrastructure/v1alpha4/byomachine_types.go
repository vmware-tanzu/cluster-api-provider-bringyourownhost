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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	"sigs.k8s.io/cluster-api/errors"
)

const (
	// MachineFinalizer allows ReconcileByoMachine to clean up Byo
	// resources associated with ByoMachine before removing it from the
	// API Server.
	MachineFinalizer = "byomachine.infrastructure.cluster.x-k8s.io"
)

// ByoMachineSpec defines the desired state of ByoMachine
type ByoMachineSpec struct {
	ProviderID *string `json:"providerID,omitempty"`

	// FailureDomain is the failure domain unique identifier this Machine should be attached to, as defined in Cluster API.
	// For this infrastructure provider, the name is equivalent to the name of the VSphereDeploymentZone.
	FailureDomain *string `json:"failureDomain,omitempty"`
}

// ByoMachineStatus defines the observed state of ByoMachine
type ByoMachineStatus struct {
	// +optional
	Ready bool `json:"ready"`

	// Addresses contains the VSphere instance associated addresses.
	Addresses []clusterv1.MachineAddress `json:"addresses,omitempty"`

	// Any transient errors that occur during the reconciliation of Machines
	// can be added as events to the Machine object and/or logged in the
	// controller's output.
	// +optional
	FailureReason *errors.MachineStatusError `json:"failureReason,omitempty"`

	// Any transient errors that occur during the reconciliation of Machines
	// can be added as events to the Machine object and/or logged in the
	// controller's output.
	// +optional
	FailureMessage *string `json:"failureMessage,omitempty"`

	// Conditions defines current service state of the BYOMachine.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ByoMachine is the Schema for the byomachines API
type ByoMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ByoMachineSpec   `json:"spec,omitempty"`
	Status ByoMachineStatus `json:"status,omitempty"`
}

func (m *ByoMachine) GetConditions() clusterv1.Conditions {
	return m.Status.Conditions
}

func (m *ByoMachine) SetConditions(conditions clusterv1.Conditions) {
	m.Status.Conditions = conditions
}

//+kubebuilder:object:root=true

// ByoMachineList contains a list of ByoMachine
type ByoMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ByoMachine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ByoMachine{}, &ByoMachineList{})
}
