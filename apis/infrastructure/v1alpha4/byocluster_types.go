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
)

const (
	// ClusterFinalizer allows ReconcileByoCluster to clean up Byo
	// resources associated with ByoCluster before removing it from the
	// API server.
	ClusterFinalizer = "byocluster.infrastructure.cluster.x-k8s.io"
)

// ByoClusterSpec defines the desired state of ByoCluster
type ByoClusterSpec struct {
	// ControlPlaneEndpoint represents the endpoint used to communicate with the control plane.
	// +optional
	ControlPlaneEndpoint APIEndpoint `json:"controlPlaneEndpoint"`
}

// ByoClusterStatus defines the observed state of ByoCluster
type ByoClusterStatus struct {
	// +optional
	Ready bool `json:"ready,omitempty"`

	// Conditions defines current service state of the ByoCluster.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`

	// FailureDomains is a list of failure domain objects synced from the infrastructure provider.
	FailureDomains clusterv1.FailureDomains `json:"failureDomains,omitempty"`
}

// APIEndpoint represents a reachable Kubernetes API endpoint.
type APIEndpoint struct {
	// Host is the hostname on which the API server is serving.
	Host string `json:"host"`

	// Port is the port on which the API server is serving.
	Port int32 `json:"port"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ByoCluster is the Schema for the byoclusters API
type ByoCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ByoClusterSpec   `json:"spec,omitempty"`
	Status ByoClusterStatus `json:"status,omitempty"`
}

func (c *ByoCluster) GetConditions() clusterv1.Conditions {
	return c.Status.Conditions
}

func (c *ByoCluster) SetConditions(conditions clusterv1.Conditions) {
	c.Status.Conditions = conditions
}

//+kubebuilder:object:root=true

// ByoClusterList contains a list of ByoCluster
type ByoClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ByoCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ByoCluster{}, &ByoClusterList{})
}
