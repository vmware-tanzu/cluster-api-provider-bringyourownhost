package controllers

import (
	"context"

	infrav1 "github.com/vmware-tanzu/cluster-api-provider-byoh/apis/infrastructure/v1alpha4"

	"github.com/pkg/errors"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// byoMachineScopeParams defines the input parameters used to create a new byoMachineScope.
type byoMachineScopeParams struct {
	Client     client.Client
	Cluster    *clusterv1.Cluster
	Machine    *clusterv1.Machine
	ByoMachine *infrav1.ByoMachine
	ByoHost    *infrav1.ByoHost
}

// byoMachineScope defines a scope defined around a BYOMachine and its machine, and its cluster.
type byoMachineScope struct {
	client      client.Client
	patchHelper *patch.Helper
	Cluster     *clusterv1.Cluster
	Machine     *clusterv1.Machine
	ByoMachine  *infrav1.ByoMachine
	ByoHost     *infrav1.ByoHost
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
	if params.ByoMachine == nil {
		return nil, errors.New("BYOMachine is required when creating a MachineScope")
	}

	helper, err := patch.NewHelper(params.ByoMachine, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}

	return &byoMachineScope{
		client:      params.Client,
		patchHelper: helper,
		Cluster:     params.Cluster,
		Machine:     params.Machine,
		ByoMachine:  params.ByoMachine,
		ByoHost:     params.ByoHost,
	}, nil
}

// PatchObject persists the BYOMachine spec and status.
func (m *byoMachineScope) PatchObject(ctx context.Context) error {
	conditions.SetSummary(m.ByoMachine,
		conditions.WithConditions(
			infrav1.BYOHostReady,
		),
	)

	return m.patchHelper.Patch(
		ctx,
		m.ByoMachine,
		patch.WithOwnedConditions{Conditions: []clusterv1.ConditionType{
			clusterv1.ReadyCondition,
			infrav1.BYOHostReady,
		}},
	)
}

// Close the byoMachineScope.
func (m *byoMachineScope) Close(ctx context.Context) error {
	return m.PatchObject(ctx)
}
