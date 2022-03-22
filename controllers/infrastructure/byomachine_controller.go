// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/cluster-api/controllers/remote"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
	infrav1 "github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/apis/infrastructure/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/annotations"
)

const (
	// ProviderIDPrefix prefix for provider id
	ProviderIDPrefix = "byoh://"
	// ProviderIDSuffixLength length of provider id suffix
	ProviderIDSuffixLength = 6
	// RequeueForbyohost requeue delay for byoh host
	RequeueForbyohost = 10 * time.Second
)

// ByoMachineReconciler reconciles a ByoMachine object
type ByoMachineReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Tracker  *remote.ClusterCacheTracker
	Recorder record.EventRecorder
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
// nolint: gocyclo, funlen
func (r *ByoMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconcile request received")

	// Fetch the ByoMachine instance
	byoMachine := &infrav1.ByoMachine{}
	err := r.Client.Get(ctx, req.NamespacedName, byoMachine)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Fetch the Machine
	machine, err := util.GetOwnerMachine(ctx, r.Client, byoMachine.ObjectMeta)
	if err != nil {
		logger.Error(err, "failed to get Owner Machine")
		return ctrl.Result{}, err
	}

	if machine == nil {
		logger.Info("Waiting for Machine Controller to set OwnerRef on ByoMachine")
		return ctrl.Result{}, nil
	}

	// Fetch the Cluster
	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, byoMachine.ObjectMeta)
	if err != nil {
		logger.Error(err, "ByoMachine owner Machine is missing cluster label or cluster does not exist")
		return ctrl.Result{}, err
	}

	if cluster == nil {
		logger.Info(fmt.Sprintf("Please associate this machine with a cluster using the label %s: <name of cluster>", clusterv1.ClusterLabelName))
		return ctrl.Result{}, nil
	}
	logger = logger.WithValues("cluster", cluster.Name)

	byoCluster := &infrav1.ByoCluster{}
	infraClusterName := client.ObjectKey{
		Namespace: byoMachine.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}

	if err = r.Client.Get(ctx, infraClusterName, byoCluster); err != nil {
		logger.Error(err, "failed to get infra cluster")
		return ctrl.Result{}, nil
	}

	helper, _ := patch.NewHelper(byoMachine, r.Client)
	defer func() {
		if err = helper.Patch(ctx, byoMachine); err != nil && reterr == nil {
			logger.Error(err, "failed to patch byomachine")
			reterr = err
		}
	}()

	// Fetch the BYOHost which is referencing this machine, if any
	refByoHost, err := r.FetchAttachedByoHost(ctx, byoMachine.Name, byoMachine.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}
	if refByoHost != nil {
		logger = logger.WithValues("BYOHost", refByoHost.Name)
	}

	// Create the machine scope
	machineScope, err := newByoMachineScope(byoMachineScopeParams{
		Client:     r.Client,
		Cluster:    cluster,
		Machine:    machine,
		ByoCluster: byoCluster,
		ByoMachine: byoMachine,
		ByoHost:    refByoHost,
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	// Return early if the object or Cluster is paused
	if annotations.IsPaused(cluster, byoMachine) {
		logger.Info("byoMachine or linked Cluster is marked as paused. Won't reconcile")
		if machineScope.ByoHost != nil {
			if err = r.setPausedConditionForByoHost(ctx, machineScope, true); err != nil {
				logger.Error(err, "cannot set paused annotation for byohost")
			}
		}
		conditions.MarkFalse(byoMachine, infrav1.BYOHostReady, infrav1.ClusterOrResourcePausedReason, clusterv1.ConditionSeverityInfo, "")
		return ctrl.Result{}, nil
	}

	// Handle deleted machines
	if !byoMachine.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, machineScope)
	}

	// Handle non-deleted machines
	return r.reconcileNormal(ctx, machineScope)
}

// FetchAttachedByoHost fetches BYOHost attached to this machine
func (r *ByoMachineReconciler) FetchAttachedByoHost(ctx context.Context, byomachineName, byomachineNamespace string) (*infrav1.ByoHost, error) {
	logger := log.FromContext(ctx)
	logger.Info("Fetching an attached ByoHost")

	selector := labels.NewSelector()
	byohostLabels, _ := labels.NewRequirement(infrav1.AttachedByoMachineLabel, selection.Equals, []string{byomachineNamespace + "." + byomachineName})
	selector = selector.Add(*byohostLabels)
	hostsList := &infrav1.ByoHostList{}
	err := r.Client.List(
		ctx,
		hostsList,
		&client.ListOptions{LabelSelector: selector},
	)
	if err != nil {
		return nil, err
	}
	var refByoHost *infrav1.ByoHost = nil
	if len(hostsList.Items) > 0 {
		refByoHost = &hostsList.Items[0]
		logger.Info("Successfully fetched an attached Byohost", "byohost", refByoHost.Name)
		if len(hostsList.Items) > 1 {
			errMsg := "more than one Byohost object attached to this Byomachine object. Only take one of it, please take care of the rest manually"
			logger.Error(errors.New(errMsg), errMsg)
		}
	}
	return refByoHost, nil
}

func (r *ByoMachineReconciler) reconcileDelete(ctx context.Context, machineScope *byoMachineScope) (reconcile.Result, error) {
	logger := log.FromContext(ctx).WithValues("cluster", machineScope.Cluster.Name)
	logger.Info("Deleting ByoMachine")
	if machineScope.ByoHost != nil {
		// Add annotation to trigger host cleanup
		logger.Info("Releasing ByoHost", "byohost", machineScope.ByoHost.Name)
		if err := r.markHostForCleanup(ctx, machineScope); err != nil {
			return ctrl.Result{}, err
		}
		r.Recorder.Eventf(machineScope.ByoHost, corev1.EventTypeNormal, "ByoHostReleaseSucceeded", "ByoHost Released by %s", machineScope.ByoMachine.Name)
		r.Recorder.Eventf(machineScope.ByoMachine, corev1.EventTypeNormal, "ByoHostReleaseSucceeded", "Released ByoHost %s", machineScope.ByoHost.Name)
	}

	controllerutil.RemoveFinalizer(machineScope.ByoMachine, infrav1.MachineFinalizer)
	return reconcile.Result{}, nil
}

func (r *ByoMachineReconciler) reconcileNormal(ctx context.Context, machineScope *byoMachineScope) (reconcile.Result, error) {
	logger := log.FromContext(ctx).WithValues("cluster", machineScope.Cluster.Name)
	logger.Info("Reconciling ByoMachine")

	controllerutil.AddFinalizer(machineScope.ByoMachine, infrav1.MachineFinalizer)

	if machineScope.ByoHost != nil {
		// if there is already byohost associated with it, make sure the paused status of byohost is false
		if err := r.setPausedConditionForByoHost(ctx, machineScope, false); err != nil {
			logger.Error(err, "Set resume flag for byohost failed")
			return ctrl.Result{}, err
		}
	}

	if !machineScope.Cluster.Status.InfrastructureReady {
		logger.Info("Cluster infrastructure is not ready yet")
		conditions.MarkFalse(machineScope.ByoMachine, infrav1.BYOHostReady, infrav1.WaitingForClusterInfrastructureReason, clusterv1.ConditionSeverityInfo, "")
		return reconcile.Result{}, nil
	}

	if machineScope.Machine.Spec.Bootstrap.DataSecretName == nil {
		logger.Info("Bootstrap Data Secret not available yet")
		conditions.MarkFalse(machineScope.ByoMachine, infrav1.BYOHostReady, infrav1.WaitingForBootstrapDataSecretReason, clusterv1.ConditionSeverityInfo, "")
		return reconcile.Result{}, nil
	}

	// If there is not yet an byoHost for this byoMachine,
	// then pick one from the host capacity pool
	if machineScope.ByoHost == nil {
		logger.Info("Attempting host reservation")
		if res, err := r.attachByoHost(ctx, machineScope); err != nil {
			return res, err
		}
		r.Recorder.Eventf(machineScope.ByoHost, corev1.EventTypeNormal, "ByoHostAttachSucceeded", "Attached to ByoMachine %s", machineScope.ByoMachine.Name)
		r.Recorder.Eventf(machineScope.ByoMachine, corev1.EventTypeNormal, "ByoHostAttachSucceeded", "Attached ByoHost %s", machineScope.ByoHost.Name)
	}

	logger.Info("Updating Node with ProviderID")
	remoteClient, err := r.getRemoteClient(ctx, machineScope.ByoMachine)
	if err != nil {
		logger.Error(err, "failed to get remote client")
		return ctrl.Result{}, err
	}

	providerID, err := r.setNodeProviderID(ctx, remoteClient, machineScope.ByoHost)
	if err != nil {
		logger.Error(err, "failed to set node providerID")
		r.Recorder.Eventf(machineScope.ByoMachine, corev1.EventTypeWarning, "SetNodeProviderFailed", "Node %s does not exist", machineScope.ByoHost.Name)
		return ctrl.Result{}, err
	}

	machineScope.ByoMachine.Spec.ProviderID = providerID
	machineScope.ByoMachine.Status.Ready = true
	conditions.MarkTrue(machineScope.ByoMachine, infrav1.BYOHostReady)
	r.Recorder.Eventf(machineScope.ByoMachine, corev1.EventTypeNormal, "NodeProvisionedSucceeded", "Provisioned Node %s", machineScope.ByoHost.Name)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ByoMachineReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	var (
		controlledType     = &infrav1.ByoMachine{}
		controlledTypeName = reflect.TypeOf(controlledType).Elem().Name()
		controlledTypeGVK  = infrav1.GroupVersion.WithKind(controlledTypeName)
	)
	logger := ctrl.LoggerFrom(ctx)
	ClusterToByoMachines := r.ClusterToByoMachines(logger)

	return ctrl.NewControllerManagedBy(mgr).
		For(controlledType).
		Watches(
			&source.Kind{Type: &infrav1.ByoHost{}},
			handler.EnqueueRequestsFromMapFunc(ByoHostToByoMachineMapFunc(controlledTypeGVK)),
		).
		// Watch the CAPI resource that owns this infrastructure resource
		Watches(
			&source.Kind{Type: &clusterv1.Machine{}},
			handler.EnqueueRequestsFromMapFunc(util.MachineToInfrastructureMapFunc(controlledTypeGVK)),
		).
		Watches(
			&source.Kind{Type: &clusterv1.Cluster{}},
			handler.EnqueueRequestsFromMapFunc(ClusterToByoMachines),
			builder.WithPredicates(predicates.ClusterUnpausedAndInfrastructureReady(ctrl.LoggerFrom(ctx))),
		).
		Complete(r)
}

// ClusterToByoMachines is a handler.ToRequestsFunc to be used to enqeue requests for reconciliation
// of ByoMachines
func (r *ByoMachineReconciler) ClusterToByoMachines(logger logr.Logger) handler.MapFunc {
	return func(o client.Object) []ctrl.Request {
		c, ok := o.(*clusterv1.Cluster)
		if !ok {
			errMsg := fmt.Sprintf("Expected a Cluster but got a %T", o)
			logger.Error(errors.New(errMsg), errMsg)
			return nil
		}

		logger = logger.WithValues("objectMapper", "ClusterToByoMachines", "namespace", c.Namespace, "Cluster", c.Name)

		// Don't handle deleted clusters
		if !c.ObjectMeta.DeletionTimestamp.IsZero() {
			logger.Info("Cluster has a deletion timestamp, skipping mapping.")
			return nil
		}

		clusterLabels := map[string]string{clusterv1.ClusterLabelName: c.Name}
		byoMachineList := &infrav1.ByoMachineList{}
		if err := r.Client.List(context.TODO(), byoMachineList, client.InNamespace(c.Namespace), client.MatchingLabels(clusterLabels)); err != nil {
			logger.Error(err, "Failed to get ByoMachine, skipping mapping.")
			return nil
		}

		result := make([]ctrl.Request, 0, len(byoMachineList.Items))
		for i := range byoMachineList.Items {
			logger.WithValues("byoMachine", byoMachineList.Items[i].Name)
			logger.Info("Adding ByoMachine to reconciliation request.")
			result = append(result, ctrl.Request{NamespacedName: client.ObjectKey{Namespace: byoMachineList.Items[i].Namespace, Name: byoMachineList.Items[i].Name}})
		}
		return result
	}
}

// setNodeProviderID patches the provider id to the node using
// client pointing to workload cluster
func (r *ByoMachineReconciler) setNodeProviderID(ctx context.Context, remoteClient client.Client, host *infrav1.ByoHost) (string, error) {
	node := &corev1.Node{}
	key := client.ObjectKey{Name: host.Name, Namespace: host.Namespace}
	err := remoteClient.Get(ctx, key, node)
	if err != nil {
		return "", err
	}

	if node.Spec.ProviderID != "" {
		var match bool
		match, err = regexp.MatchString(fmt.Sprintf("%s%s/.+", ProviderIDPrefix, host.Name), node.Spec.ProviderID)
		if err != nil {
			return "", err
		}
		if match {
			return node.Spec.ProviderID, nil
		}
		return "", errors.New("invalid format for node.Spec.ProviderID")
	}

	helper, err := patch.NewHelper(node, remoteClient)
	if err != nil {
		return "", err
	}

	node.Spec.ProviderID = fmt.Sprintf("%s%s/%s", ProviderIDPrefix, host.Name, util.RandomString(ProviderIDSuffixLength))

	return node.Spec.ProviderID, helper.Patch(ctx, node)
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

func (r *ByoMachineReconciler) setPausedConditionForByoHost(ctx context.Context, machineScope *byoMachineScope, isPaused bool) error {
	helper, err := patch.NewHelper(machineScope.ByoHost, r.Client)
	if err != nil {
		return err
	}

	if isPaused {
		desired := map[string]string{
			clusterv1.PausedAnnotation: "",
		}
		annotations.AddAnnotations(machineScope.ByoHost, desired)
	} else {
		delete(machineScope.ByoHost.Annotations, clusterv1.PausedAnnotation)
	}

	return helper.Patch(ctx, machineScope.ByoHost)
}

func (r *ByoMachineReconciler) attachByoHost(ctx context.Context, machineScope *byoMachineScope) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("cluster", machineScope.Cluster.Name)
	var selector labels.Selector
	var err error
	if machineScope.ByoHost != nil {
		return ctrl.Result{}, nil
	}

	hostsList := &infrav1.ByoHostList{}
	// LabelSelector filter for byohosts
	if machineScope.ByoMachine.Spec.Selector != nil {
		selector, err = metav1.LabelSelectorAsSelector(machineScope.ByoMachine.Spec.Selector)
		if err != nil {
			logger.Error(err, "Label Selector as selector failed")
			return ctrl.Result{}, err
		}
	} else {
		selector = labels.NewSelector()
	}

	byohostLabels, _ := labels.NewRequirement(clusterv1.ClusterLabelName, selection.DoesNotExist, nil)
	selector = selector.Add(*byohostLabels)

	err = r.Client.List(ctx, hostsList, &client.ListOptions{LabelSelector: selector})
	if err != nil {
		logger.Error(err, "failed to list byohosts")
		return ctrl.Result{RequeueAfter: RequeueForbyohost}, err
	}
	if len(hostsList.Items) == 0 {
		logger.Info("No hosts found, waiting..")
		r.Recorder.Eventf(machineScope.ByoMachine, corev1.EventTypeWarning, "ByoHostSelectionFailed", "No available ByoHost")
		conditions.MarkFalse(machineScope.ByoMachine, infrav1.BYOHostReady, infrav1.BYOHostsUnavailableReason, clusterv1.ConditionSeverityInfo, "")
		return ctrl.Result{RequeueAfter: RequeueForbyohost}, errors.New("no hosts found")
	}
	// TODO- Needs smarter logic
	host := hostsList.Items[0]

	byohostHelper, err := patch.NewHelper(&host, r.Client)
	if err != nil {
		logger.Error(err, "Creating patch helper failed")
	}

	host.Status.MachineRef = &corev1.ObjectReference{
		APIVersion: machineScope.ByoMachine.APIVersion,
		Kind:       machineScope.ByoMachine.Kind,
		Namespace:  machineScope.ByoMachine.Namespace,
		Name:       machineScope.ByoMachine.Name,
		UID:        machineScope.ByoMachine.UID,
	}
	// Set the cluster Label
	hostLabels := host.Labels
	if hostLabels == nil {
		hostLabels = make(map[string]string)
	}
	hostLabels[clusterv1.ClusterLabelName] = machineScope.ByoMachine.Labels[clusterv1.ClusterLabelName]
	hostLabels[infrav1.AttachedByoMachineLabel] = machineScope.ByoMachine.Namespace + "." + machineScope.ByoMachine.Name
	host.Labels = hostLabels

	host.Spec.BootstrapSecret = &corev1.ObjectReference{
		Kind:      "Secret",
		Namespace: machineScope.ByoMachine.Namespace,
		Name:      *machineScope.Machine.Spec.Bootstrap.DataSecretName,
	}
	if host.Annotations == nil {
		host.Annotations = make(map[string]string)
	}
	host.Annotations[infrav1.EndPointIPAnnotation] = machineScope.Cluster.Spec.ControlPlaneEndpoint.Host
	host.Annotations[infrav1.K8sVersionAnnotation] = strings.Split(*machineScope.Machine.Spec.Version, "+")[0]
	host.Annotations[infrav1.BundleLookupBaseRegistryAnnotation] = machineScope.ByoCluster.Spec.BundleLookupBaseRegistry
	host.Annotations[infrav1.BundleLookupTagAnnotation] = machineScope.ByoCluster.Spec.BundleLookupTag

	err = byohostHelper.Patch(ctx, &host)
	if err != nil {
		logger.Error(err, "failed to patch byohost")
		return ctrl.Result{}, err
	}
	logger.Info("Successfully attached Byohost", "byohost", host.Name)
	machineScope.ByoHost = &host
	return ctrl.Result{}, nil
}

// ByoHostToByoMachineMapFunc returns a handler.ToRequestsFunc that watches for
// Machine events and returns reconciliation requests for an infrastructure provider object
func ByoHostToByoMachineMapFunc(gvk schema.GroupVersionKind) handler.MapFunc {
	return func(o client.Object) []reconcile.Request {
		h, ok := o.(*infrav1.ByoHost)
		if !ok {
			return nil
		}
		if h.Status.MachineRef == nil {
			// TODO, we can enqueue byomachine which providerID is nil to get better performance than requeue
			return nil
		}

		gk := gvk.GroupKind()
		// Return early if the GroupKind doesn't match what we expect
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
	machineScope.ByoHost.Annotations[infrav1.HostCleanupAnnotation] = ""

	// Issue the patch for byohost
	return helper.Patch(ctx, machineScope.ByoHost)
}
