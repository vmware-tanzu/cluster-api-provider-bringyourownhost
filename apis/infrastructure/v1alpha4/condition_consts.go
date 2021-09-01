package v1alpha4

import clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"

// Conditions and Reasons defined on BYOHost
const (
	// K8sNodeBootstrapSucceeded documents if the required Kubernetes
	// components are currently installed on the node.
	// This condition is managed by the host agent and it could be always true
	// if the host is unmanaged; insteead, in case of managed host, it depends
	// by the node currently being hosting a BYOHmachine or not.
	K8sNodeBootstrapSucceeded clusterv1.ConditionType = "K8sNodeBootstrapSucceeded"

	// K8sNodeBootstrapSucceeded is False
	WaitingForMachineRefReason           = "WaitingForMachineRefToBeAssigned"
	BootstrapDataSecretUnavailableReason = "BootstrapDataSecretUnavailable"
	CloudInitExecutionFailedReason       = "CloudInitExecutionFailed"
	K8sNodeAbsentReason                  = "K8sNodeAbsent"
)

// Conditions and Reasons defined on BYOMachine
const (
	BYOHostReady clusterv1.ConditionType = "BYOHostReady"

	// BYOHostReady is False
	WaitingForClusterInfrastructureReason = "WaitingForClusterInfrastructure"
	WaitingForBootstrapDataSecretReason   = "WaitingForBootstrapDataSecret"
	BYOHostsUnavailableReason             = "BYOHostsUnavailable"
)

// Reasons common to all Byo Resources
const (
	ClusterOrResourcePausedReason = "ClusterOrResourcePaused"
)
