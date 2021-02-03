/*
Copyright the Cluster API Provider BYOH contributors.

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
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
)

// Common conditions and reasons

const (
	// InstallingReason documents the host is installing required Kubernetes
	// components.
	InstallingReason = "Installing"

	// BootstrappingReason documents the host is bootstrapping the Kubernetes Node.
	BootstrappingReason = "Bootstrapping"
)

// BYOHost conditions.

const (
	// K8sComponentsInstalledCondition documents if the required Kubernetes
	// components are currently installed on the node.
	// This condition is managed by the host agent and it could be always true
	// if the host is unmanaged; insteead, in case of managed host, it depends
	// by the nodecurrently being hosting a BYOHmachine or not.
	K8sComponentsInstalledCondition clusterv1.ConditionType = "K8sComponentsInstalled"

	// K8sComponentsAbsentReason documents the host waiting for being selected
	// for hosting a BYOMachine before installing the Kubernetes components.
	K8sComponentsAbsentReason = "K8sComponentsAbsent"

	// K8sNodeBootstrappedCondition documents if the Kubernetes node on the
	// node has been boostrapped.
	// This condition is managed by the host agent.
	K8sNodeBootstrappedCondition clusterv1.ConditionType = "K8sNodeBootstrapped"

	// K8sNodeAbsentReason documents the host waiting for being selected
	// for hosting a BYOMachine before boostrapping the Kubernetes node.
	K8sNodeAbsentReason = "K8sNodeAbsent"

	// K8sNodeBootstrapFailureReason documents a failure when
	// bootstrapping the Kubernetes node on the host.
	K8sNodeBootstrapFailureReason = "K8sNodeBootstrapFailure"

	// K8sNodeRemovalFailureReason documents a failure when
	// removing the Kubernetes node on the host.
	K8sNodeRemovalFailureReason = "K8sNodeRemovalFailure"
)

// BYOMachine conditions.

const (
	// HostReadyCondition reports a BYOHost operational state.
	HostReadyCondition clusterv1.ConditionType = "HostReady"

	// WaitingForClusterInfrastructureReason (Severity=Info) documents a BYOMachine waiting for the cluster
	// infrastructure to be ready before starting host reservation.
	// infrastructure.
	WaitingForClusterInfrastructureReason = "WaitingForClusterInfrastructure"

	// WaitingForBootstrapDataReason (Severity=Info) documents a BYOMachine waiting for the bootstrap
	// script to be ready before starting host reservation.
	WaitingForBootstrapDataReason = "WaitingForBootstrapData"

	// WaitingForHostReason documents the BYOMachine waiting for an
	// available host to mach selection criteria.
	WaitingForHostReason = "WaitingForHost"
)
