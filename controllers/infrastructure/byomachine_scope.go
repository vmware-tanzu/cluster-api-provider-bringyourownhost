package controllers

import (
	"github.com/pkg/errors"
	infrav1 "github.com/vmware-tanzu/cluster-api-provider-byoh/apis/infrastructure/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
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

// byoMachineScope defines a scope defined around a ByoMachine and its machine, and its cluster.
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
func newByoMachineScope(params byoMachineScopeParams) (*byoMachineScope, error) {
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
