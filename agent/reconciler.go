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
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"vmware-tanzu/cluster-api-provider-byoh/agent/cloudinit"
	infrav1 "vmware-tanzu/cluster-api-provider-byoh/api/v1alpha3"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/conditions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type HostReconciler struct {
	Client client.Client
	Log    logr.Logger

	Host *infrav1.BYOHost
}

func (r *HostReconciler) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.BYOHost{}).
		// Ensures we are considering events for this host only.
		WithEventFilter(r.isThisHost()).
		WithOptions(options).
		Complete(r)
}

func (r *HostReconciler) isThisHost() predicate.Funcs {
	var f = func(obj runtime.Object, meta metav1.Object) bool {
		h, ok := obj.(*infrav1.BYOHost)
		if !ok {
			return true
		}
		return h.Name == r.Host.Name
	}

	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return f(e.ObjectNew, e.MetaNew)
		},
		CreateFunc: func(e event.CreateEvent) bool {
			return f(e.Object, e.Meta)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return f(e.Object, e.Meta)
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return f(e.Object, e.Meta)
		},
	}
}

// Reconcile ensures the back-end state reflects the Kubernetes resource state intent.
func (r *HostReconciler) Reconcile(req ctrl.Request) (_ ctrl.Result, reterr error) {
	ctx := context.TODO()
	logger := r.Log.WithValues("BYOHost", req.Name)

	// Fetch the BYOHost instance.
	byoHost := &infrav1.BYOHost{}
	err := r.Client.Get(ctx, req.NamespacedName, byoHost)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Fetch the BYOMachine and the Machine instance, if any.
	byoMachine := &infrav1.BYOMachine{}
	machine := &clusterv1.Machine{}
	if byoHost.Status.MachineRef != nil {
		key := types.NamespacedName{Namespace: byoHost.Status.MachineRef.Namespace, Name: byoHost.Status.MachineRef.Name}
		err := r.Client.Get(ctx, key, byoMachine)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, err
		}
		logger = logger.WithValues("BYOMachine", key)

		// Fetch the Machine.
		machine, err = util.GetOwnerMachine(ctx, r.Client, byoMachine.ObjectMeta)
		if err != nil {
			return ctrl.Result{}, err
		}
		if machine == nil {
			logger.Info("Machine Controller has not yet set OwnerRef")
			return ctrl.Result{}, nil
		}

		logger = logger.WithValues("machine", machine.Name)
	}

	// Create the host scope
	hostScope, err := newBYOHostScope(byoHostScopeParams{
		Client:     r.Client,
		BYOHost:    byoHost,
		BYOMachine: byoMachine,
		Machine:    machine,
	})
	if err != nil {
		return ctrl.Result{}, errors.Errorf("failed to create scope: %+v", err)
	}

	// Always close the scope when exiting this function so we can persist any BYOHost changes.
	defer func() {
		if err := hostScope.Close(ctx); err != nil && reterr == nil {
			reterr = err
		}
	}()

	// Handle deleted machines
	if !byoHost.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, logger, hostScope)
	}
	return r.reconcileNormal(ctx, logger, hostScope)
}

func (r *HostReconciler) reconcileNormal(ctx context.Context, logger logr.Logger, scope *byoHostScope) (ctrl.Result, error) {

	// If the host is managed and the K8sComponents are provided by the site admin,
	// surface this using conditions.
	if !scope.ShouldManageK8sComponents() {
		conditions.MarkTrue(scope.BYOHost, infrav1.K8sComponentsInstalledCondition)

		// TODO: consider if to reconcile Kubernetes version label (is "the site admin changes the kunernets version while the agent is running" a use case to be supported?)
	}

	// If the Host is assigned to a Machine.
	if scope.IsAssignedToMachine() {
		// Ensure Kubernetes components are installed on the host.
		if scope.ShouldManageK8sComponents() && !scope.HasKubernetesComponents() {
			logger.Info("Intalling Kubernetes Components...")

			// TODO:installing Kubernetes components.

			conditions.MarkTrue(scope.BYOHost, infrav1.K8sComponentsInstalledCondition)
		}

		// Ensure the Kubernetes node is bootstrapped.
		if !scope.IsKubernetesNodeBootstrapped() {
			logger.Info("Bootstrapping Kubernetes Node...")

			if err := r.bootstrapNode(ctx, logger, scope); err != nil {
				conditions.MarkFalse(scope.BYOHost, infrav1.K8sNodeBootstrappedCondition, infrav1.K8sNodeBootstrapFailureReason, clusterv1.ConditionSeverityError, err.Error())
				return ctrl.Result{}, err
			}
		}
		conditions.MarkTrue(scope.BYOHost, infrav1.K8sNodeBootstrappedCondition)

		return ctrl.Result{}, nil
	}

	// Otherwise the host is spare capacity, so ensure it is in a clean state.

	// Ensure the Kubernetes node are removed.
	if scope.IsKubernetesNodeBootstrapped() {
		logger.Info("Removing Kubernetes Node...")
		err := r.removeNode(ctx, logger)
		if err != nil {
			conditions.MarkFalse(scope.BYOHost, infrav1.K8sNodeBootstrappedCondition, infrav1.K8sNodeRemovalFailureReason, clusterv1.ConditionSeverityError, err.Error())
		}
	}
	conditions.MarkFalse(scope.BYOHost, infrav1.K8sNodeBootstrappedCondition, infrav1.K8sNodeAbsentReason, clusterv1.ConditionSeverityInfo, "K8s node will be installed after the host is reserved by a BYOMachine")

	// If the host Should manage Kubernetes components, ensure it is removed.
	if scope.ShouldManageK8sComponents() {
		if scope.HasKubernetesComponents() {
			logger.Info("Removing Kubernetes Components...")
			// TODO: removing Kubernetes components.
		}
		conditions.MarkFalse(scope.BYOHost, infrav1.K8sComponentsInstalledCondition, infrav1.K8sComponentsAbsentReason, clusterv1.ConditionSeverityInfo, "K8s components will be installed after the host is reserved by a BYOMachine")
	}

	return ctrl.Result{}, nil
}

func (r *HostReconciler) reconcileDelete(ctx context.Context, logger logr.Logger, scope *byoHostScope) (ctrl.Result, error) {
	// TODO: return error if deleting with machine ref (this should be blocked via webhhooks)
	//  consider if to add a finalizer so to give another chance to cleanup

	return ctrl.Result{}, nil
}

func (r *HostReconciler) bootstrapNode(ctx context.Context, logger logr.Logger, scope *byoHostScope) error {
	logger.Info("Running node init script...")

	// Fetching cloud init scripts generated by the kubeadm bootstrap provider and run it.
	// NOTE: The init script targets cloud init, but fo BYOHost we are using a custom
	// cloud init adapter that supports only what is required for getting the node started.
	bootstrapData, err := r.getBootstrapData(ctx, logger, scope.Machine)
	if err != nil {
		return errors.Wrap(err, "failed to get machine's bootstrap data")
	}

	if err := cloudinit.Run(ctx, logger, bootstrapData); err != nil {
		return errors.Wrap(err, "failed to run machine's bootstrap script")
	}

	logger.Info("Creating the bootstrap sentinel file...")

	dir := filepath.Dir(bootstrapSentinelFile)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.Mkdir(dir, 0644); err != nil {
			return errors.Wrapf(err, "failed to create sentinel file %s", bootstrapSentinelFile)
		}
	}

	if _, err := os.Stat(bootstrapSentinelFile); os.IsNotExist(err) {
		f, err := os.Create(bootstrapSentinelFile)
		if err != nil {
			return errors.Wrapf(err, "failed to create sentinel file %s", bootstrapSentinelFile)
		}
		defer f.Close()
	}

	logger.Info("Kubernetes Node boostrapped")
	return nil
}

func (r *HostReconciler) getBootstrapData(ctx context.Context, _ logr.Logger, machine *clusterv1.Machine) ([]byte, error) {
	// TODO: we should probably reconsider this (node agent reads secrets)
	// and make the bootstrap secret to flow down in a different way.

	if machine.Spec.Bootstrap.DataSecretName == nil {
		return nil, errors.New("machine's bootstrap.dataSecretName is nil")
	}

	s := &corev1.Secret{}
	key := client.ObjectKey{Namespace: machine.Namespace, Name: *machine.Spec.Bootstrap.DataSecretName}
	if err := r.Client.Get(ctx, key, s); err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve bootstrap data secret %s/%s", key.Namespace, key.Name)
	}

	value, ok := s.Data["value"]
	if !ok {
		return nil, errors.New("error retrieving bootstrap data: secret value key is missing")
	}

	return value, nil
}

func (r *HostReconciler) removeNode(_ context.Context, logger logr.Logger) error {
	logger.Info("Running kubeadm reset...")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := exec.Command("kubeadm", []string{"reset", "--force", "--ignore-preflight-errors=all"}...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		logger.Info("kubeadm reset::", "err", err, "stdout", stdout, "stderr", stderr)
		return errors.Wrapf(err, "failed to exec kubeadm reset")
	}

	logger.Info("Removing the bootstrap sentinel file...")

	if _, err := os.Stat(bootstrapSentinelFile); !os.IsNotExist(err) {
		err := os.Remove(bootstrapSentinelFile)
		if err != nil {
			return errors.Wrapf(err, "failed to delete sentinel file %s", bootstrapSentinelFile)
		}
	}

	logger.Info("Kubernetes Node removed")
	return nil
}
