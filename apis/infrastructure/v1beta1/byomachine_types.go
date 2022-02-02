// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

const (
	// MachineFinalizer allows ReconcileByoMachine to clean up Byo
	// resources associated with ByoMachine before removing it from the
	// API Server.
	MachineFinalizer = "byomachine.infrastructure.cluster.x-k8s.io"
)

// ByoMachineSpec defines the desired state of ByoMachine
type ByoMachineSpec struct {
	// Label Selector to choose the byohost
	Selector *metav1.LabelSelector `json:"selector,omitempty"`

	ProviderID string `json:"providerID,omitempty"`
}

// NetworkStatus provides information about one of a VM's networks.
type NetworkStatus struct {
	// Connected is a flag that indicates whether this network is currently
	// connected to the VM.
	Connected bool `json:"connected,omitempty"`

	// IPAddrs is one or more IP addresses reported by vm-tools.
	// +optional
	IPAddrs []string `json:"ipAddrs,omitempty"`

	// MACAddr is the MAC address of the network device.
	MACAddr string `json:"macAddr"`

	// NetworkInterfaceName is the name of the network interface.
	// +optional
	NetworkInterfaceName string `json:"networkInterfaceName,omitempty"`

	// IsDefault is a flag that indicates whether this interface name is where
	// the default gateway sit on.
	IsDefault bool `json:"isDefault,omitempty"`
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

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=byomachines,scope=Namespaced,shortName=byom
//+kubebuilder:subresource:status

// ByoMachine is the Schema for the byomachines API
type ByoMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ByoMachineSpec   `json:"spec,omitempty"`
	Status ByoMachineStatus `json:"status,omitempty"`
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

// GetConditions returns the conditions of ByoMachine status
func (byoMachine *ByoMachine) GetConditions() clusterv1.Conditions {
	return byoMachine.Status.Conditions
}

// SetConditions sets the conditions of ByoMachine status
func (byoMachine *ByoMachine) SetConditions(conditions clusterv1.Conditions) {
	byoMachine.Status.Conditions = conditions
}
