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
	"fmt"
	"reflect"
	"time"

	infrav1 "vmware-tanzu/cluster-api-provider-byoh/api/v1alpha3"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/controllers/remote"
	"sigs.k8s.io/cluster-api/util"
	clusterutilv1 "sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/conditions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	hostMachineRefIndex = "status.machineref"
)

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=byomachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=byomachines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=byohosts,verbs=get;list;watch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=byohosts/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;machines,verbs=get;list;watch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines;machines/status,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch

type MachineReconciler struct {
	Client  client.Client
	Log     logr.Logger
	Tracker *remote.ClusterCacheTracker
}

func (r *MachineReconciler) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	var (
		controlledType     = &infrav1.BYOMachine{}
		controlledTypeName = reflect.TypeOf(controlledType).Elem().Name()
		controlledTypeGVK  = infrav1.GroupVersion.WithKind(controlledTypeName)
	)

	// Add index to BYOHost for listing by Machine reference.
	if err := mgr.GetCache().IndexField(&infrav1.BYOHost{},
		hostMachineRefIndex,
		r.indexBYOHostByMachineRef,
	); err != nil {
		return errors.Wrapf(err, "error creating %s index", hostMachineRefIndex)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.BYOMachine{}).
		// Watch the CAPI machine that owns this infrastructure resource.
		Watches(
			&source.Kind{Type: &clusterv1.Machine{}},
			&handler.EnqueueRequestsFromMapFunc{
				ToRequests: clusterutilv1.MachineToInfrastructureMapFunc(controlledTypeGVK),
			},
		).
		// Watch the BYOHost that is hosting this infrastructure resource.
		Watches(
			&source.Kind{Type: &infrav1.BYOHost{}},
			&handler.EnqueueRequestsFromMapFunc{
				ToRequests: BYOHostToInfrastructureMapFunc(),
			},
		).
		WithOptions(options).
		Complete(r)
}

// BYOHostToInfrastructureMapFunc handles BYOHost events and returns reconciliation requests for an infrastructure provider object.
func BYOHostToInfrastructureMapFunc() handler.ToRequestsFunc {
	return func(o handler.MapObject) []reconcile.Request {
		h, ok := o.Object.(*infrav1.BYOHost)
		if !ok {
			return nil
		}
		if h.Status.MachineRef == nil {
			return nil
		}

		return []reconcile.Request{
			{
				NamespacedName: client.ObjectKey{
					Namespace: h.Status.MachineRef.Namespace,
					Name:      h.Status.MachineRef.Name,
				},
			},
		}
	}
}

func (r *MachineReconciler) indexBYOHostByMachineRef(o runtime.Object) []string {
	host, ok := o.(*infrav1.BYOHost)
	if !ok {
		r.Log.Error(errors.New("incorrect type"), "expected a BYOHost", "type", fmt.Sprintf("%T", o))
		return nil
	}

	if host.Status.MachineRef != nil {
		return []string{fmt.Sprintf("%s/%s", host.Status.MachineRef.Namespace, host.Status.MachineRef.Name)}
	}
	return nil
}

// Reconcile ensures the back-end state reflects the Kubernetes resource state intent.
func (r MachineReconciler) Reconcile(req ctrl.Request) (_ ctrl.Result, reterr error) {
	ctx := context.TODO()
	logger := r.Log.WithValues("namespace", req.Namespace, "BYOMachine", req.Name)

	// Fetch the BYOMachine instance.
	byoMachine := &infrav1.BYOMachine{}
	err := r.Client.Get(ctx, req.NamespacedName, byoMachine)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Fetch the BYOHost which is referencing this machine, if any
	hostsList := &infrav1.BYOHostList{}
	if err := r.Client.List(
		ctx,
		hostsList,
		client.MatchingFields{hostMachineRefIndex: fmt.Sprintf("%s/%s", byoMachine.Namespace, byoMachine.Name)},
	); err != nil {
		return ctrl.Result{}, err
	}
	var byoHost *infrav1.BYOHost
	if len(hostsList.Items) == 1 {
		byoHost = &hostsList.Items[0]
		logger = logger.WithValues("BYOHost", byoHost.Name)
	}

	// Fetch the Machine.
	machine, err := util.GetOwnerMachine(ctx, r.Client, byoMachine.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}
	if machine == nil {
		logger.Info("Machine Controller has not yet set OwnerRef")
		return ctrl.Result{}, nil
	}

	logger = logger.WithValues("machine", machine.Name)

	// Fetch the Cluster.
	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, machine.ObjectMeta)
	if err != nil {
		logger.Info("Machine is missing cluster label or cluster does not exist")
		return ctrl.Result{}, nil
	}

	if util.IsPaused(cluster, byoMachine) {
		logger.Info("BYOMachine or linked Cluster is marked as paused. Won't reconcile")
		return ctrl.Result{}, nil
	}

	logger = logger.WithValues("cluster", cluster.Name)

	// Create the machine scope
	machineScope, err := newBYOMachineScope(byoMachineScopeParams{
		Client:     r.Client,
		Cluster:    cluster,
		Machine:    machine,
		BYOMachine: byoMachine,
		BYOHost:    byoHost,
	})
	if err != nil {
		return ctrl.Result{}, errors.Errorf("failed to create scope: %+v", err)
	}

	// Always close the scope when exiting this function so we can persist any BYOMachine changes.
	defer func() {
		if err := machineScope.Close(ctx); err != nil && reterr == nil {
			reterr = err
		}
	}()

	// Addsthe cluster label if missing.
	machineScope.EnsureClusterLabel()

	// Add finalizer first if not exist to avoid the race condition between create and delete
	if !machineScope.HasFinalizer() {
		machineScope.AddFinalizer()
		return ctrl.Result{}, nil
	}

	// Handle deleted machines
	if !byoMachine.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, logger, machineScope)
	}
	return r.reconcileNormal(ctx, logger, machineScope)
}

func (r MachineReconciler) reconcileNormal(ctx context.Context, logger logr.Logger, scope *byoMachineScope) (ctrl.Result, error) {

	// Make sure cluster infrastructure is available and populated.
	if !scope.ClusterInfrastructureReady() {
		logger.Info("Cluster infrastructure is not ready yet")
		conditions.MarkFalse(scope.BYOMachine, infrav1.HostReadyCondition, infrav1.WaitingForClusterInfrastructureReason, clusterv1.ConditionSeverityInfo, "")
		return ctrl.Result{}, nil
	}

	// Make sure bootstrap data is available and populated.
	if !scope.BootstrapDataSecretCreated() {
		logger.Info("Bootstrap data secret reference is not yet available")
		conditions.MarkFalse(scope.BYOMachine, infrav1.HostReadyCondition, infrav1.WaitingForBootstrapDataReason, clusterv1.ConditionSeverityInfo, "")
		return ctrl.Result{}, nil
	}

	// If there is not yet an host for this infrastructure machine,
	// then pick one of the host capacity pool.
	if !scope.HasHost() {
		logger.Info("Attempting host reservation")
		if res, err := r.attemptHostReservation(ctx, logger, scope); err != nil || !res.IsZero() {
			return res, err
		}
	}

	// TODO: Find a solution for rolling up host errors.

	// If the Kubernetes components on the host are not yet installed,
	// then wait for the node agent to report installation completed.
	if !scope.IsK8sComponentsInstalled() {
		logger.Info("Kubernetes Components on the host are not ready yet")
		// TODO: might be we want to use something different than installing for the managed use case (and treat this as error)
		conditions.MarkFalse(scope.BYOMachine, infrav1.HostReadyCondition, infrav1.InstallingReason, clusterv1.ConditionSeverityInfo, "")
		return ctrl.Result{}, nil
	}

	// If the Kubernetes node on the host are not yet bootstrapped,
	// then wait for the node agent to report bootstrap completed.
	if !scope.IsK8sNodeBootsrapped() {
		logger.Info("Kubernetes node on the host is not ready yet")
		conditions.MarkFalse(scope.BYOMachine, infrav1.HostReadyCondition, infrav1.BootstrappingReason, clusterv1.ConditionSeverityInfo, "")
		return ctrl.Result{}, nil
	}

	// Set the ProviderID
	// This field must match the provider ID as seen on the node object corresponding to this machine.
	// This field is required by higher level consumers of cluster-api. Example use case is cluster autoscaler.
	if scope.BYOMachine.Spec.ProviderID == nil {
		// TODO: handle the use case where BYOHost runs on a cloud provider (e.g. vSphere, AWS, Azure)

		logger.Info("Reconciling Node.ProviderID")
		if res, err := r.reconcileBareMetalProviderID(ctx, logger, scope); err != nil || !res.IsZero() {
			return res, err
		}
	}

	// The Kubernetes node on the host is bootstrapped, the provider ID
	// is reported for the CAPI Machine to use, so the infrastructure machine
	// can be considered ready
	scope.BYOMachine.Status.Ready = true
	conditions.MarkTrue(scope.BYOMachine, infrav1.HostReadyCondition)

	return ctrl.Result{}, nil
}

func (r MachineReconciler) reconcileDelete(ctx context.Context, logger logr.Logger, scope *byoMachineScope) (ctrl.Result, error) {

	// If the is still an host reserved by this machine, remove the host reservation.
	if scope.HasHost() {
		// Add an annotation to the host to trigger cleanup without waiting for the sync loop.
		if err := r.markHostForCleanup(ctx, logger, scope); err != nil {
			return ctrl.Result{}, err
		}

		// if the Kubernetes node on the host is not yet deleted,
		// wait for the node agent to report node deleted.
		if !scope.IsK8sNodeDeleted() {
			logger.Info("Kubernetes Node on the host is not removed yet")
			// TODO: might be we should use somothing different that Deleting...
			conditions.MarkFalse(scope.BYOMachine, infrav1.HostReadyCondition, clusterv1.DeletingReason, clusterv1.ConditionSeverityInfo, "Removing the Kubernetes node...")
			return ctrl.Result{}, nil
		}

		// If the Kubernetes components on the node are not yet delete,
		// wait for the node agent to report installation completed.
		if scope.HostShouldManageK8sComponents() && !scope.IsK8sComponentsDeleted() {
			logger.Info("Kubernetes components on the host are not removed yet")
			// TODO: might be we should use somothing different that Deleting...
			conditions.MarkFalse(scope.BYOMachine, infrav1.HostReadyCondition, clusterv1.DeletingReason, clusterv1.ConditionSeverityInfo, "Removing the Kubernetes components...")
			return ctrl.Result{}, nil
		}

		logger.Info("Removing host reservation")
		if err := r.removeHostReservation(ctx, logger, scope); err != nil {
			return ctrl.Result{}, err
		}
	}

	// The underlying BYOHost is now returned to the host capacity pool,
	// so it is possible to remove the finalizer so the infrastructure
	// machine can be finally deleted.
	scope.RemoveFinalizer()
	return ctrl.Result{}, nil
}

func (r MachineReconciler) attemptHostReservation(ctx context.Context, logger logr.Logger, scope *byoMachineScope) (ctrl.Result, error) {
	hostsList := &infrav1.BYOHostList{}
	if err := r.Client.List(
		ctx,
		hostsList,
		// TODO: apply BYOMachine selectors
	); err != nil {
		return ctrl.Result{}, err
	}

	// TODO: filter by BYOPlan selectors

	if len(hostsList.Items) == 0 {
		logger.Info("Waiting for available host matching the request")
		conditions.MarkFalse(scope.BYOMachine, infrav1.HostReadyCondition, infrav1.WaitingForHostReason, clusterv1.ConditionSeverityWarning, "Waiting for an available host matching the request")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// TODO: make the host selection logic smarter:
	// - it should account for the Kubernetes version for managed machines
	// - it should account for failure domains
	// - it should account for previews assignations (an host, recently used by a MD/KCP, if possible should be re-used by the the same MD/KCP)
	// - it should be deterministic  (e.g candidate host ordered alphabetically, pick the first)

	host := &hostsList.Items[0]

	// TODO: pass the KubernetesVersion defined at machine level (for the managed scenario only)

	// Create the host patch before setting the machine ref.
	hostPatch := client.MergeFromWithOptions(host.DeepCopyObject(), client.MergeFromWithOptimisticLock{})

	// Set the host reservation.
	host.Status.MachineRef = &corev1.ObjectReference{
		APIVersion: scope.BYOMachine.APIVersion,
		Kind:       scope.BYOMachine.Kind,
		Namespace:  scope.BYOMachine.Namespace,
		Name:       scope.BYOMachine.Name,
		UID:        scope.BYOMachine.UID,
	}

	// Issue the patch.
	if err := r.Client.Status().Patch(ctx, host, hostPatch); err != nil {
		if apierrors.IsConflict(err) {
			logger.Info("Conflict with attempting host reservation, requeue for retry")
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}

	// Update the scope with the reserved host.
	scope.BYOHost = host
	logger.Info("Host reserved", "Name", host.Name)
	return ctrl.Result{}, nil
}

func (r MachineReconciler) reconcileBareMetalProviderID(ctx context.Context, logger logr.Logger, scope *byoMachineScope) (ctrl.Result, error) {
	// Getting remote client
	remoteClient, err := r.Tracker.GetClient(ctx, util.ObjectKey(scope.Cluster))
	if err != nil {
		return ctrl.Result{}, err
	}

	// Gets the node.
	// NOTE: This works under the assumption that the host.name == node.name (kubeadm default)
	node := &corev1.Node{}
	key := client.ObjectKey{Name: scope.BYOHost.Name}
	if err := remoteClient.Get(ctx, key, node); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Waiting for the Node to be registered, requeue")
			conditions.MarkFalse(scope.BYOMachine, infrav1.HostReadyCondition, infrav1.BootstrappingReason, clusterv1.ConditionSeverityInfo, "Waiting for the Node to be registered")
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}
		return ctrl.Result{}, err
	}

	providerID := fmt.Sprintf("byon://%s/%s", scope.BYOHost.Name, util.RandomString(6))
	logger.Info("Patching the node", "node", node.Name, "ProviderID", providerID)

	// Create the node patch before setting the provider ID.
	nodePatch := client.MergeFromWithOptions(node.DeepCopyObject())

	// Issue the patch.
	node.Spec.ProviderID = providerID
	if err := remoteClient.Patch(ctx, node, nodePatch); err != nil {
		return ctrl.Result{}, err
	}

	// Update the BYOMachine with the provider ID.
	scope.BYOMachine.Spec.ProviderID = &providerID
	return ctrl.Result{}, nil
}

const hostCleanupAnnotation = "byoh.infrastructure.cluster.x-k8s.io/unregistering"

func (r MachineReconciler) markHostForCleanup(ctx context.Context, _ logr.Logger, scope *byoMachineScope) error {
	// Create the host patch before setting the machine ref.
	hostPatch := client.MergeFromWithOptions(scope.BYOHost.DeepCopyObject())

	// Remove the host reservation.
	if scope.BYOHost.Annotations == nil {
		scope.BYOHost.Annotations = map[string]string{}
	}
	scope.BYOHost.Annotations[hostCleanupAnnotation] = ""

	// Issue the patch.
	if err := r.Client.Status().Patch(ctx, scope.BYOHost, hostPatch); err != nil {
		return err
	}
	return nil
}

func (r MachineReconciler) removeHostReservation(ctx context.Context, logger logr.Logger, scope *byoMachineScope) error {
	// Create the host patch before setting the machine ref.
	hostPatch := client.MergeFromWithOptions(scope.BYOHost.DeepCopyObject())

	// Remove the host reservation.
	scope.BYOHost.Status.MachineRef = nil

	// Remove the cleanup annotation
	delete(scope.BYOHost.Annotations, hostCleanupAnnotation)

	// Issue the patch.
	if err := r.Client.Status().Patch(ctx, scope.BYOHost, hostPatch); err != nil {
		return err
	}

	logger.Info("Host %s returned to available capacity", scope.BYOHost.Name)
	return nil
}
