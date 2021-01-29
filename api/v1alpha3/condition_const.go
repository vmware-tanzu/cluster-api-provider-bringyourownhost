/*
Copyright 2020 The Kubernetes Authors.

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

const (
	// K8sComponentsInstalledCondition documents if the required Kubernetes
	// components are currently installed on the node.
	// This condition is managed by the host agent and it could be always true
	// if thehost is unmanaged; insteead, in case of managed host, it depends
	// by the nodecurrently being hosting a BYOHmachine or not.
	K8sComponentsInstalledCondition clusterv1.ConditionType = "K8sComponentsInstalled"

	// K8sNodeBootstrappedCondition documents if the Kubernetes node on the
	// node has been boostrapped.
	// This condition is managed by the host agent.
	K8sNodeBootstrappedCondition clusterv1.ConditionType = "K8sComponentsInstalled"

	// WaitingForReservationReason documents the host waiting for being selected
	// for hostinga BYOMachine before boostrapping the Kubernetes node.
	WaitingForReservationReason = "WaitingForReservation"

	// InstallingReason documents the host being installing required Kubernetes
	// components.
	InstallingReason = "Installing"

	// BootstrappingReason documents the host bootstrapping the Kubernetes Node.
	BootstrappingReason = "Bootstrapping"

	// FailedReason documents the host failng to install required Kubernetes
	// components or failing to bootstrap the Kubernetes Node.
	FailedReason = "Failed"
)
