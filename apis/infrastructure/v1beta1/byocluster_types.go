// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
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

	// BundleLookupBaseRegistry is the base Registry URL that is used for pulling byoh bundle images,
	// if not set, the default will be set to https://projects.registry.vmware.com/cluster_api_provider_bringyourownhost
	// +optional
	BundleLookupBaseRegistry string `json:"bundleLookupBaseRegistry,omitempty"`

	// BundleLookupTag is the tag of the BYOH bundle to be used
	BundleLookupTag string `json:"bundleLookupTag,omitempty"`
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
//+kubebuilder:resource:path=byoclusters,scope=Namespaced,shortName=byoc
//+kubebuilder:subresource:status

// ByoCluster is the Schema for the byoclusters API
type ByoCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ByoClusterSpec   `json:"spec,omitempty"`
	Status ByoClusterStatus `json:"status,omitempty"`
}

// GetConditions gets the condition for the ByoCluster status
func (byoCluster *ByoCluster) GetConditions() clusterv1.Conditions {
	return byoCluster.Status.Conditions
}

// SetConditions sets the conditions for the ByoCluster status
func (byoCluster *ByoCluster) SetConditions(conditions clusterv1.Conditions) {
	byoCluster.Status.Conditions = conditions
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
