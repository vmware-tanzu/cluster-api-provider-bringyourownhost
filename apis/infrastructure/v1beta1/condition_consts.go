// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

// Conditions and Reasons defined on BYOHost
const (
	// K8sNodeBootstrapSucceeded documents if the node is successfully bootstrapped by kubeadm
	// This condition is managed by the host agent. and it could be always true
	// if the host is unmanaged; instead, in case of managed host, it depends
	// on the node currently being hosting a ByoMachine or not.
	K8sNodeBootstrapSucceeded clusterv1.ConditionType = "K8sNodeBootstrapSucceeded"

	// K8sComponentsInstallationSucceeded documents if the required Kubernetes
	// components are currently installed on the node.
	K8sComponentsInstallationSucceeded clusterv1.ConditionType = "K8sComponentsInstallationSucceeded"

	// WaitingForMachineRefReason indicates when a ByoHost is registered into a capacity pool and
	// waiting for a byohost.Status.MachineRef to be assigned
	WaitingForMachineRefReason = "WaitingForMachineRefToBeAssigned"

	// BootstrapDataSecretUnavailableReason indicates that the bootstrap provider is yet to provide the
	// secret that contains bootstrap information
	// This secret is available on byohost.Spec.BootstrapSecret field
	BootstrapDataSecretUnavailableReason = "BootstrapDataSecretUnavailable"

	// CleanK8sDirectoriesFailedReason indicates that clean k8s directories failed for some reason, please
	// delete it manually for reconcile to proceed.
	// The cleaned directories are /run/kubeadm and /etc/cni/net.d
	CleanK8sDirectoriesFailedReason = "CleanK8sDirectoriesFailed"

	// CloudInitExecutionFailedReason indicates that cloudinit failed to parse and execute the directives
	// that are part of the cloud-config file
	CloudInitExecutionFailedReason = "CloudInitExecutionFailed"

	// K8sNodeAbsentReason indicates that the node is not a Kubernetes node
	// This is usually set after executing kubeadm reset on the node
	K8sNodeAbsentReason = "K8sNodeAbsent"

	// K8sComponentsInstallingReason indicates that the k8s components are being
	// downloaded and installed
	// TODO unused, remove it
	K8sComponentsInstallingReason = "K8sComponentsInstalling"

	// K8sComponentsInstallationFailedReason indicates that the installer failed to install all the
	// k8s components on this host
	K8sComponentsInstallationFailedReason = "K8sComponentsInstallationFailed"
)

// Conditions and Reasons defined on BYOMachine
const (

	// BYOHostReady documents the k8s node is ready and can take on workloads
	BYOHostReady clusterv1.ConditionType = "BYOHostReady"

	// WaitingForClusterInfrastructureReason indicates the cluster that the ByoMachine belongs to
	// is waiting to be owned by the corresponding CAPI Cluster
	WaitingForClusterInfrastructureReason = "WaitingForClusterInfrastructure"

	// WaitingForBootstrapDataSecretReason indicates that the bootstrap provider is yet to provide the
	// secret that contains bootstrap information
	// This secret is available on Machine.Spec.Bootstrap.DataSecretName
	WaitingForBootstrapDataSecretReason = "WaitingForBootstrapDataSecret"

	// BYOHostsUnavailableReason indicates that no byohosts are available in the capacity pool
	BYOHostsUnavailableReason = "BYOHostsUnavailable"
)

// Reasons common to all Byo Resources
const (

	// ClusterOrResourcePausedReason indicates that either
	// Spec.Paused field on the cluster is set to true
	// or the resource is marked with Paused annotation
	ClusterOrResourcePausedReason = "ClusterOrResourcePaused"
)
