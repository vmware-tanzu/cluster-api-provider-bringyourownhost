/*
Copyright 2021.

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
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/cluster-api/controllers/remote"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
	infrav1 "github.com/vmware-tanzu/cluster-api-provider-byoh/apis/infrastructure/v1alpha4"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	"sigs.k8s.io/cluster-api/util/annotations"
)

const (
	providerIDPrefix       = "byoh://"
	providerIDSuffixLength = 6
	hostCleanupAnnotation  = "byoh.infrastructure.cluster.x-k8s.io/unregistering"
)

// ByoMachineReconciler reconciles a ByoMachine object
type ByoMachineReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	Tracker *remote.ClusterCacheTracker
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=byomachines,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=byomachines/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=byomachines/finalizers,verbs=update
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=byohosts,verbs=get;list;watch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=byohosts/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;machines,verbs=get;list;watch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines;machines/status,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ByoMachine object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile

// Reconcile handles ByoMachine events
func (r *ByoMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	logger := log.FromContext(ctx).WithValues("namespace", req.Namespace, "BYOMachine", req.Name)

	// Fetch the ByoMachine instance.
	byoMachine := &infrav1.ByoMachine{}
	err := r.Client.Get(ctx, req.NamespacedName, byoMachine)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Fetch the Machine.
	machine, err := util.GetOwnerMachine(ctx, r.Client, byoMachine.ObjectMeta)
	if err != nil {
		logger.Error(err, "failed to get Owner Machine")
		return ctrl.Result{}, err
	}

	if machine == nil {
		logger.Info("Waiting for Machine Controller to set OwnerRef on ByoMachine")
		return ctrl.Result{}, nil
	}

	// Fetch the Cluster.
	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, byoMachine.ObjectMeta)
	if err != nil {
		logger.Error(err, "ByoMachine owner Machine is missing cluster label or cluster does not exist")
		return ctrl.Result{}, err
	}

	if cluster == nil {
		logger.Info(fmt.Sprintf("Please associate this machine with a cluster using the label %s: <name of cluster>", clusterv1.ClusterLabelName))
		return ctrl.Result{}, nil
	}

	helper, _ := patch.NewHelper(byoMachine, r.Client)
	defer func() {
		if err = helper.Patch(ctx, byoMachine); err != nil && reterr == nil {
			logger.Error(err, "failed to patch byomachine")
			reterr = err
		}
	}()

	// Return early if the object or Cluster is paused.
	if annotations.IsPaused(cluster, byoMachine) {
		logger.Info("byoMachine or linked Cluster is marked as paused. Won't reconcile")
		if byoMachine.Spec.ProviderID != "" {
			if err = r.setPausedConditionForByoHost(ctx, byoMachine.Spec.ProviderID, req.Namespace, true); err != nil {
				logger.Error(err, "Set Paused flag for byohost")
			}
		}
		conditions.MarkFalse(byoMachine, infrav1.BYOHostReady, infrav1.ClusterOrResourcePausedReason, clusterv1.ConditionSeverityInfo, "")
		return ctrl.Result{}, nil
	}

	// TODO: Till we do not have index on host.Status.MachineRef
	allHosts := &infrav1.ByoHostList{}
	err = r.Client.List(ctx, allHosts)
	if err != nil {
		return ctrl.Result{}, err
	}

	var refByoHost *infrav1.ByoHost
	for _, byoHost := range allHosts.Items {
		if byoHost.Status.MachineRef != nil && (byoHost.Status.MachineRef.Name == byoMachine.Name && byoHost.Status.MachineRef.Namespace == byoMachine.Namespace) {
			refByoHost = &byoHost
		}
	}

	// Create the machine scope
	machineScope, err := newByoMachineScope(byoMachineScopeParams{
		Client:     r.Client,
		Cluster:    cluster,
		Machine:    machine,
		ByoMachine: byoMachine,
		ByoHost:    refByoHost,
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	// Handle deleted machines
	if !byoMachine.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, machineScope)
	}

	// Handle non-deleted machines
	return r.reconcileNormal(ctx, machineScope)
}

func (r *ByoMachineReconciler) reconcileDelete(ctx context.Context, machineScope *byoMachineScope) (reconcile.Result, error) {
	if machineScope.ByoHost != nil {
		// Add annotation to trigger host cleanup
		if err := r.markHostForCleanup(ctx, machineScope); err != nil {
			return ctrl.Result{}, err
		}

		if !(conditions.IsFalse(machineScope.ByoHost, infrav1.K8sNodeBootstrapSucceeded) && conditions.GetReason(machineScope.ByoHost, infrav1.K8sNodeBootstrapSucceeded) == infrav1.K8sNodeAbsentReason) {
			conditions.MarkFalse(machineScope.ByoMachine, infrav1.BYOHostReady, clusterv1.DeletingReason, clusterv1.ConditionSeverityInfo, "Removing the Kubernetes node...")
			return ctrl.Result{}, nil
		}

		if err := r.removeHostReservation(ctx, machineScope); err != nil {
			return ctrl.Result{}, err
		}
	}
	controllerutil.RemoveFinalizer(machineScope.ByoMachine, infrav1.MachineFinalizer)
	return reconcile.Result{}, nil
}

func (r *ByoMachineReconciler) reconcileNormal(ctx context.Context, machineScope *byoMachineScope) (reconcile.Result, error) {
	logger := log.FromContext(ctx).WithValues("namespace", machineScope.ByoMachine.Namespace, "BYOMachine", machineScope.ByoMachine.Name)
	// TODO: Uncomment below line when we have tests for byomachine delete
	controllerutil.AddFinalizer(machineScope.ByoMachine, infrav1.MachineFinalizer)

	// TODO: Remove the below check after refactoring setting of Pause annotation on byoHost
	if len(machineScope.ByoMachine.Spec.ProviderID) > 0 {
		// if there is already byohost associated with it, make sure the paused status of byohost is false
		if err := r.setPausedConditionForByoHost(ctx, machineScope.ByoMachine.Spec.ProviderID, machineScope.ByoMachine.Namespace, false); err != nil {
			logger.Error(err, "Set resume flag for byohost failed")
			return ctrl.Result{}, err
		}
	}

	if !machineScope.Cluster.Status.InfrastructureReady {
		logger.Info("Cluster infrastructure is not ready yet")
		conditions.MarkFalse(machineScope.ByoMachine, infrav1.BYOHostReady, infrav1.WaitingForClusterInfrastructureReason, clusterv1.ConditionSeverityInfo, "")
		return reconcile.Result{}, errors.New("cluster infrastructure is not ready yet")
	}

	if machineScope.Machine.Spec.Bootstrap.DataSecretName == nil {
		logger.Info("Bootstrap Data Secret not available yet")
		conditions.MarkFalse(machineScope.ByoMachine, infrav1.BYOHostReady, infrav1.WaitingForBootstrapDataSecretReason, clusterv1.ConditionSeverityInfo, "")
		return reconcile.Result{}, errors.New("bootstrap data secret not available yet")
	}

	// If there is not yet an byoHost for this byoMachine,
	// then pick one from the host capacity pool.
	if machineScope.ByoHost == nil {
		logger.Info("Attempting host reservation")
		if res, err := r.attachByoHost(ctx, logger, machineScope.Machine, machineScope.ByoMachine); err != nil || !res.IsZero() {
			return res, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ByoMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	var (
		controlledType     = &infrav1.ByoMachine{}
		controlledTypeName = reflect.TypeOf(controlledType).Elem().Name()
		controlledTypeGVK  = infrav1.GroupVersion.WithKind(controlledTypeName)
	)

	return ctrl.NewControllerManagedBy(mgr).
		For(controlledType).
		Watches(
			&source.Kind{Type: &infrav1.ByoHost{}},
			handler.EnqueueRequestsFromMapFunc(ByoHostToByoMachineMapFunc(controlledTypeGVK)),
		).
		// Watch the CAPI resource that owns this infrastructure resource.
		Watches(
			&source.Kind{Type: &clusterv1.Machine{}},
			handler.EnqueueRequestsFromMapFunc(util.MachineToInfrastructureMapFunc(controlledTypeGVK)),
		).
		Complete(r)
}

// setNodeProviderID patches the provider id to the node using
// client pointing to workload cluster
func (r *ByoMachineReconciler) setNodeProviderID(ctx context.Context, remoteClient client.Client, host *infrav1.ByoHost, providerID string) error {
	node := &corev1.Node{}
	key := client.ObjectKey{Name: host.Name, Namespace: host.Namespace}
	err := remoteClient.Get(ctx, key, node)

	if err != nil {
		return err
	}
	helper, err := patch.NewHelper(node, remoteClient)
	if err != nil {
		return err
	}

	node.Spec.ProviderID = providerID

	return helper.Patch(ctx, node)
}

func (r *ByoMachineReconciler) getRemoteClient(ctx context.Context, byoMachine *infrav1.ByoMachine) (client.Client, error) {
	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, byoMachine.ObjectMeta)
	if err != nil {
		return nil, err
	}
	remoteClient, err := r.Tracker.GetClient(ctx, util.ObjectKey(cluster))
	if err != nil {
		return nil, err
	}

	return remoteClient, nil
}

func (r *ByoMachineReconciler) setPausedConditionForByoHost(ctx context.Context, providerID, nameSpace string, isPaused bool) error {
	// The format of providerID is "byoh://<byoHostName>/<RandomString(6)>
	if !strings.HasPrefix(providerID, providerIDPrefix) {
		return errors.New("invalid providerID prefix")
	}

	strs := strings.Split(providerID[len(providerIDPrefix):], "/")

	if len(strs) == 0 {
		return errors.New("invalid providerID format")
	}

	byoHostName := strs[0]

	byoHost := &infrav1.ByoHost{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: byoHostName, Namespace: nameSpace}, byoHost)
	if err != nil {
		return err
	}

	helper, err := patch.NewHelper(byoHost, r.Client)
	if err != nil {
		return err
	}

	if isPaused {
		desired := map[string]string{
			clusterv1.PausedAnnotation: "paused",
		}
		annotations.AddAnnotations(byoHost, desired)
	} else {
		_, ok := byoHost.Annotations[clusterv1.PausedAnnotation]
		if ok {
			delete(byoHost.Annotations, clusterv1.PausedAnnotation)
		}
	}

	return helper.Patch(ctx, byoHost)
}

func (r *ByoMachineReconciler) attachByoHost(ctx context.Context, logger logr.Logger, machine *clusterv1.Machine, byoMachine *infrav1.ByoMachine) (ctrl.Result, error) {
	hostsList := &infrav1.ByoHostList{}
	// LabelSelector filter for byohosts
	byohostLabels, _ := labels.NewRequirement(clusterv1.ClusterLabelName, selection.DoesNotExist, nil)
	selector := labels.NewSelector().Add(*byohostLabels)
	err := r.Client.List(ctx, hostsList, &client.ListOptions{LabelSelector: selector})
	if err != nil {
		logger.Error(err, "failed to list byohosts")
		return ctrl.Result{}, err
	}
	if len(hostsList.Items) == 0 {
		logger.Info("No hosts found, waiting..")
		conditions.MarkFalse(byoMachine, infrav1.BYOHostReady, infrav1.BYOHostsUnavailableReason, clusterv1.ConditionSeverityInfo, "")
		return ctrl.Result{}, errors.New("no hosts found")
	}
	// TODO- Needs smarter logic
	host := hostsList.Items[0]

	byohostHelper, err := patch.NewHelper(&host, r.Client)
	if err != nil {
		logger.Error(err, "Creating patch helper failed")
	}

	host.Status.MachineRef = &corev1.ObjectReference{
		APIVersion: byoMachine.APIVersion,
		Kind:       byoMachine.Kind,
		Namespace:  byoMachine.Namespace,
		Name:       byoMachine.Name,
		UID:        byoMachine.UID,
	}
	// Set the cluster Label
	hostLabels := host.Labels
	if hostLabels == nil {
		hostLabels = make(map[string]string)
	}
	hostLabels[clusterv1.ClusterLabelName] = byoMachine.Labels[clusterv1.ClusterLabelName]
	host.Labels = hostLabels

	if machine.Spec.Bootstrap.DataSecretName == nil {
		logger.Info("Bootstrap secret not ready")
		return ctrl.Result{}, errors.New("bootstrap secret not ready")
	}

	host.Spec.BootstrapSecret = &corev1.ObjectReference{
		Kind:      "Secret",
		Namespace: byoMachine.Namespace,
		Name:      *machine.Spec.Bootstrap.DataSecretName,
	}

	err = byohostHelper.Patch(ctx, &host)
	if err != nil {
		logger.Error(err, "failed to patch byohost")
		return ctrl.Result{}, err
	}
	providerID := fmt.Sprintf("%s%s/%s", providerIDPrefix, host.Name, util.RandomString(providerIDSuffixLength))
	remoteClient, err := r.getRemoteClient(ctx, byoMachine)
	if err != nil {
		logger.Error(err, "failed to get remote client")
		return ctrl.Result{}, err
	}

	err = r.setNodeProviderID(ctx, remoteClient, &host, providerID)
	if err != nil {
		logger.Error(err, "failed to set node providerID")
		return ctrl.Result{RequeueAfter: time.Second * 10}, err
	}

	byoMachine.Spec.ProviderID = providerID
	byoMachine.Status.Ready = true

	conditions.MarkTrue(byoMachine, infrav1.BYOHostReady)
	return ctrl.Result{}, err
}

// MachineToInfrastructureMapFunc returns a handler.ToRequestsFunc that watches for
// Machine events and returns reconciliation requests for an infrastructure provider object.
func ByoHostToByoMachineMapFunc(gvk schema.GroupVersionKind) handler.MapFunc {
	return func(o client.Object) []reconcile.Request {
		h, ok := o.(*infrav1.ByoHost)
		if !ok {
			return nil
		}
		if h.Status.MachineRef == nil {
			// TODO, we can enqueue byomachine which provideID is nil to get better performance than requeue
			return nil
		}

		gk := gvk.GroupKind()
		// Return early if the GroupKind doesn't match what we expect.
		byomachineGK := h.Status.MachineRef.GroupVersionKind().GroupKind()
		if gk != byomachineGK {
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

func (r *ByoMachineReconciler) markHostForCleanup(ctx context.Context, machineScope *byoMachineScope) error {
	helper, _ := patch.NewHelper(machineScope.ByoHost, r.Client)

	if machineScope.ByoHost.Annotations == nil {
		machineScope.ByoHost.Annotations = map[string]string{}
	}
	machineScope.ByoHost.Annotations[hostCleanupAnnotation] = ""

	// Issue the patch.
	return helper.Patch(ctx, machineScope.ByoHost)
}

func (r *ByoMachineReconciler) removeHostReservation(ctx context.Context, machineScope *byoMachineScope) error {
	helper, _ := patch.NewHelper(machineScope.ByoHost, r.Client)

	// Remove host reservation.
	machineScope.ByoHost.Status.MachineRef = nil

	// TODO: Remove cluster-label on byohost

	// Remove the cleanup annotation
	delete(machineScope.ByoHost.Annotations, hostCleanupAnnotation)

	// Issue the patch.
	return helper.Patch(ctx, machineScope.ByoHost)
}
