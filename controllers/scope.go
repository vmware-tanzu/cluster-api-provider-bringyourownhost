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

package controllers

import (
	"context"

	infrav1 "vmware-tanzu/cluster-api-provider-byoh/api/v1alpha3"

	"github.com/pkg/errors"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// byoMachineScopeParams defines the input parameters used to create a new byoMachineScope.
type byoMachineScopeParams struct {
	Client client.Client

	Cluster    *clusterv1.Cluster
	Machine    *clusterv1.Machine
	BYOMachine *infrav1.BYOMachine
	BYOHost    *infrav1.BYOHost
}

// newBYOMachineScope creates a new MachineScope from the supplied parameters.
// This is meant to be called for each reconcile iteration.
func newBYOMachineScope(params byoMachineScopeParams) (*byoMachineScope, error) {
	if params.Client == nil {
		return nil, errors.New("Client is required when creating a MachineScope")
	}
	if params.Cluster == nil {
		return nil, errors.New("Cluster is required when creating a MachineScope")
	}
	if params.Machine == nil {
		return nil, errors.New("Machine is required when creating a MachineScope")
	}
	if params.BYOMachine == nil {
		return nil, errors.New("BYOMachine is required when creating a MachineScope")
	}

	helper, err := patch.NewHelper(params.BYOMachine, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}

	return &byoMachineScope{
		client:      params.Client,
		patchHelper: helper,

		Cluster:    params.Cluster,
		Machine:    params.Machine,
		BYOMachine: params.BYOMachine,
		BYOHost:    params.BYOHost,
	}, nil
}

// byoMachineScope defines a scope defined around a BYOMachine and its machine, and its cluster.
type byoMachineScope struct {
	client      client.Client
	patchHelper *patch.Helper

	Cluster    *clusterv1.Cluster
	Machine    *clusterv1.Machine
	BYOMachine *infrav1.BYOMachine
	BYOHost    *infrav1.BYOHost
}

// EnsureClusterLabel ensures the cluster label is applied to the BYOMachine.
func (m *byoMachineScope) EnsureClusterLabel() {
	if m.BYOMachine.Labels == nil {
		m.BYOMachine.Labels = make(map[string]string)
	}
	m.BYOMachine.Labels[clusterv1.ClusterLabelName] = m.Cluster.Name
}

func (m *byoMachineScope) HasFinalizer() bool {
	return controllerutil.ContainsFinalizer(m.BYOMachine, infrav1.MachineFinalizer)
}

func (m *byoMachineScope) AddFinalizer() {
	controllerutil.AddFinalizer(m.BYOMachine, infrav1.MachineFinalizer)
}

func (m *byoMachineScope) RemoveFinalizer() {
	controllerutil.RemoveFinalizer(m.BYOMachine, infrav1.MachineFinalizer)
}

func (m *byoMachineScope) ClusterInfrastructureReady() bool {
	return m.Cluster.Status.InfrastructureReady
}

func (m *byoMachineScope) BootstrapDataSecretCreated() bool {
	return m.Machine.Spec.Bootstrap.DataSecretName != nil
}

func (m *byoMachineScope) HasHost() bool {
	return m.BYOHost != nil
}

func (m *byoMachineScope) HostShouldManageK8sComponents() bool {
	// TODO: this should be derived from an host field/label/annotation
	return false
}

func (m *byoMachineScope) IsK8sComponentsInstalled() bool {
	if m.BYOHost != nil {
		return conditions.IsTrue(m.BYOHost, infrav1.K8sComponentsInstalledCondition)
	}
	return false
}

func (m *byoMachineScope) IsK8sComponentsDeleted() bool {
	if m.BYOHost != nil {
		return conditions.IsFalse(m.BYOHost, infrav1.K8sComponentsInstalledCondition) &&
			conditions.GetReason(m.BYOHost, infrav1.K8sComponentsInstalledCondition) == infrav1.K8sComponentsAbsentReason
	}
	return true
}

func (m *byoMachineScope) IsK8sNodeBootsrapped() bool {
	if m.BYOHost != nil {
		return conditions.IsTrue(m.BYOHost, infrav1.K8sNodeBootstrappedCondition)
	}
	return false
}

func (m *byoMachineScope) IsK8sNodeDeleted() bool {
	if m.BYOHost != nil {
		return conditions.IsFalse(m.BYOHost, infrav1.K8sNodeBootstrappedCondition) &&
			conditions.GetReason(m.BYOHost, infrav1.K8sNodeBootstrappedCondition) == infrav1.K8sNodeAbsentReason
	}
	return true
}

// PatchObject persists the BYOMachine spec and status.
func (m *byoMachineScope) PatchObject(ctx context.Context) error {
	conditions.SetSummary(m.BYOMachine,
		conditions.WithConditions(
			infrav1.HostReadyCondition,
		),
	)

	return m.patchHelper.Patch(
		ctx,
		m.BYOMachine,
		patch.WithOwnedConditions{Conditions: []clusterv1.ConditionType{
			clusterv1.ReadyCondition,
			infrav1.HostReadyCondition,
		}},
	)
}

// Close the byoMachineScope.
func (m *byoMachineScope) Close(ctx context.Context) error {
	return m.PatchObject(ctx)
}
