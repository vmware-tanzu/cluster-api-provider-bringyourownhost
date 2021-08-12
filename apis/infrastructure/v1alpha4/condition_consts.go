package v1alpha4

import clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"

const (
	// K8sComponentsInstalledCondition documents if the required Kubernetes
	// components are currently installed on the node.
	// This condition is managed by the host agent and it could be always true
	// if the host is unmanaged; insteead, in case of managed host, it depends
	// by the nodecurrently being hosting a BYOHmachine or not.
	K8sComponentsInstalledCondition clusterv1.ConditionType = "K8sComponentsInstalled"
	HostReadyCondition              clusterv1.ConditionType = "HostReady"

	// VMProvisionedCondition documents the status of the provisioning of a VSphereMachine and its underlying VSphereVM.
	HostProvisionedCondition clusterv1.ConditionType = "HostProvisioned"
)

const (
	// WaitingForClusterInfrastructureReason (Severity=Info) documents a ByoMachine waiting for the cluster
	// infrastructure to be ready before starting the provisioning process.
	WaitingForClusterInfrastructureReason = "WaitingForClusterInfrastructure"

	WaitingForNetworkAddressesReason = "WaitingForNetworkAddresses"
)
