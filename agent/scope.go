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

package main

import (
	"context"
	"os"

	infrav1 "vmware-tanzu/cluster-api-provider-byoh/api/v1alpha3"

	"github.com/pkg/errors"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// Note: this sentinel file has been introduced only in CAPI v1alpha4,
	// but we are anticipating is usage because it makes easier to track
	// Kubernetes node bootstrap completed.
	bootstrapSentinelFile = "/run/cluster-api/bootstrap-success.complete"
)

// byoHostScopeParams defines the input parameters used to create a new byoHostScope.
type byoHostScopeParams struct {
	Client client.Client

	BYOHost    *infrav1.BYOHost
	BYOMachine *infrav1.BYOMachine
	Machine    *clusterv1.Machine
}

// newBYOHostScope creates a new HostScope from the supplied parameters.
// This is meant to be called for each reconcile iteration.
func newBYOHostScope(params byoHostScopeParams) (*byoHostScope, error) {
	if params.BYOHost == nil {
		return nil, errors.New("BYOHost is required when creating a HostScope")
	}

	helper, err := patch.NewHelper(params.BYOHost, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}

	return &byoHostScope{
		client:      params.Client,
		patchHelper: helper,

		BYOHost:    params.BYOHost,
		BYOMachine: params.BYOMachine,
		Machine:    params.Machine,
	}, nil
}

// byoHostScope defines a scope defined around a BYOHost.
type byoHostScope struct {
	client      client.Client
	patchHelper *patch.Helper

	BYOHost    *infrav1.BYOHost
	BYOMachine *infrav1.BYOMachine
	Machine    *clusterv1.Machine
}

func (m *byoHostScope) IsAssignedToMachine() bool {
	// NB. a deleting machine should trigger host cleanup
	return m.BYOHost.Status.MachineRef != nil && m.BYOMachine.ObjectMeta.DeletionTimestamp.IsZero()
}

func (m *byoHostScope) ShouldManageK8sComponents() bool {
	// TODO: this should be derived from an host field/label/annotation
	// to be set when registering the host
	return false
}

func (m *byoHostScope) HasKubernetesComponents() bool {
	// TODO: This is temporary; might be we want to use something different, e.g. kubelet --version
	info, err := os.Stat("/run/cluster-api/install-success.complete")
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func (m *byoHostScope) IsKubernetesNodeBootstrapped() bool {
	info, err := os.Stat(bootstrapSentinelFile)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// PatchObject persists the BYOHost spec and status.
func (m *byoHostScope) PatchObject(ctx context.Context) error {
	conditions.SetSummary(m.BYOHost,
		conditions.WithConditions(
			infrav1.K8sComponentsInstalledCondition,
			infrav1.K8sNodeBootstrappedCondition,
		),
	)

	return m.patchHelper.Patch(
		ctx,
		m.BYOHost,
		patch.WithOwnedConditions{Conditions: []clusterv1.ConditionType{
			clusterv1.ReadyCondition,
			infrav1.K8sComponentsInstalledCondition,
			infrav1.K8sNodeBootstrappedCondition,
		}},
	)
}

// Close the BYOMachineScope.
func (m *byoHostScope) Close(ctx context.Context) error {
	return m.PatchObject(ctx)
}
