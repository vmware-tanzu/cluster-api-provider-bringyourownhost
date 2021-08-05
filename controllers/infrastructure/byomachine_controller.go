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
	"fmt"
	"reflect"

	"github.com/pkg/errors"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/cluster-api/controllers/remote"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/docker/docker/daemon/logger"
	infrav1 "github.com/vmware-tanzu/cluster-api-provider-byoh/apis/infrastructure/v1alpha4"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	"sigs.k8s.io/cluster-api/util/annotations"
	clusterutilv1 "sigs.k8s.io/cluster-api/util"
)

var (
	controlledType     = &infrav1.ByoMachine{}
	controlledTypeName = reflect.TypeOf(controlledType).Elem().Name()
	controlledTypeGVK  = infrav1.GroupVersion.WithKind(controlledTypeName)
)

const (
	ProviderIDPrefix = "byoh://"
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

	byoMachine := &infrav1.ByoMachine{}
	if err := r.Client.Get(ctx, req.NamespacedName, byoMachine); err != nil {
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

	// Return early if the object or Cluster is paused.
	if annotations.IsPaused(cluster, byoMachine) {
		logger.Info("byoMachine or linked Cluster is marked as paused. Won't reconcile")
		if byoMachine.Spec.ProviderID != "" {
			if err := r.setPausedConditionForByoHost(ctx, byoMachine.Spec.ProviderID, req.Namespace, true); err != nil {
				logger.Error(err, "Set Paused flag for byohost")
				return ctrl.Result{}, nil
			}
		}
		return ctrl.Result{}, nil
	} else {
		//if there is already byhost associated with it, make sure the paused status of byohost is false
		if len(byoMachine.Spec.ProviderID) > 0 {
			if err := r.setPausedConditionForByoHost(ctx, byoMachine.Spec.ProviderID, req.Namespace, false); err != nil {
				logger.Error(err, "Set resume flag for byohost failed")
				return ctrl.Result{}, err
			}
		}
	}

	hostsList := &infrastructurev1alpha4.ByoHostList{}
	// Fetch the CAPI Machine.
	machine, err := clusterutilv1.GetOwnerMachine(ctx, r.Client, byoMachine.ObjectMeta)
	if err != nil {
		return reconcile.Result{}, err
	}
	if machine == nil {
		logger.Info("Waiting for Machine Controller to set OwnerRef on ByoMachine")
		return reconcile.Result{}, nil
	}

	// Fetch the CAPI Cluster.
	cluster, err := clusterutilv1.GetClusterFromMetadata(ctx, r.Client, machine.ObjectMeta)
	if err != nil {
		logger.Info("Machine is missing cluster label or cluster does not exist")
		return reconcile.Result{}, nil
	}
	if annotations.IsPaused(cluster, byoMachine) {
		logger.V(4).Info("ByoMachine %s/%s linked to a cluster that is paused",
			byoMachine.Namespace, byoMachine.Name)
		return reconcile.Result{}, nil
	}

	// Fetch the ByoCluster
	byoCluster := &infrav1.ByoCluster{}
	byoClusterName := client.ObjectKey{
		Namespace: byoMachine.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}
	if err := r.Client.Get(ctx, byoClusterName, byoCluster); err != nil {
		logger.Info("Waiting for VSphereCluster")
		return reconcile.Result{}, nil
	}

	// Create the patch helper.
	patchHelper, err := patch.NewHelper(byoMachine, r.Client)
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(
			err,
			"failed to init patch helper for %s %s/%s",
			byoMachine.GroupVersionKind(),
			byoMachine.Namespace,
			byoMachine.Name)
	}

	// Always issue a patch when exiting this function so changes to the
	// resource are patched back to the API server.
	defer func() {
		// always update the readyCondition.
		conditions.SetSummary(byoMachine,
			conditions.WithConditions(infrav1.HostProvisionedCondition),
		)

		err := patchHelper.Patch(
			ctx,
			byoMachine,
			patch.WithOwnedConditions{Conditions: []clusterv1.ConditionType{
			clusterv1.ReadyCondition,}}
		) 

		// Patch the VSphereMachine resource.
		if err != nil {
			if reterr == nil {
				reterr = err
			}
			logger.Error(err, "patch failed", "byomachine")
		}
	}()

	// Handle deleted machines
	if !byoMachine.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, byoMachine)
	}

	// Handle non-deleted machines
	return r.reconcileNormal(ctx, byoMachine)

}

func (r *ByoMachineReconciler) setNodeProviderID(ctx context.Context, remoteClient client.Client, host infrastructurev1alpha4.ByoHost, providerID string) error {

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
	helper.Patch(ctx, node)

	return nil
}

func (r *ByoMachineReconciler) getRemoteClient(ctx context.Context, byoMachine *infrastructurev1alpha4.ByoMachine) (client.Client, error) {
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

func (r ByoMachineReconciler) reconcileDelete(ctx context.Context, byoMachine *infrav1.ByoMachine) (reconcile.Result, error) {
	return reconcile.Result{}, nil
}

func (r ByoMachineReconciler) reconcileNormal(ctx context.Context, byoMachine *infrav1.ByoMachine) (reconcile.Result, error) {

	hostsList := &infrav1.ByoHostList{}
	err = r.Client.List(ctx, hostsList)

	if err != nil {
		logger.Error(err, "failed to list byohosts")
		return ctrl.Result{}, err
	}

	if len(hostsList.Items) == 0 {
		logger.Info("No hosts found, waiting..")
		return ctrl.Result{}, errors.New("no hosts found")
	}
	// TODO- Needs smarter logic
	host := hostsList.Items[0]

	helper, _ := patch.NewHelper(&host, r.Client)

	host.Status.MachineRef = &corev1.ObjectReference{
		APIVersion: byoMachine.APIVersion,
		Kind:       byoMachine.Kind,
		Namespace:  byoMachine.Namespace,
		Name:       byoMachine.Name,
		UID:        byoMachine.UID,
	}

	if machine.Spec.Bootstrap.DataSecretName == nil {
		logger.Info("Bootstrap secret not ready")
		return ctrl.Result{}, errors.New("bootstrap secret not ready")
	}

	host.Spec.BootstrapSecret = &corev1.ObjectReference{
		Kind:      "Secret",
		Namespace: byoMachine.Namespace,
		Name:      *machine.Spec.Bootstrap.DataSecretName,
	}

	err = helper.Patch(ctx, &host)
	if err != nil {
		logger.Error(err, "failed to patch byohost")
		return ctrl.Result{}, err
	}

	providerID := fmt.Sprintf("%s%s/%s", ProviderIDPrefix, host.Name, util.RandomString(6))
	remoteClient, err := r.getRemoteClient(ctx, byoMachine)
	if err != nil {
		logger.Error(err, "failed to get remote client")
		return ctrl.Result{}, err
	}

	err = r.setNodeProviderID(ctx, remoteClient, host, providerID)
	if err != nil {
		logger.Error(err, "failed to set node providerID")
		return ctrl.Result{}, err
	}

	helper, _ = patch.NewHelper(byoMachine, r.Client)
	byoMachine.Spec.ProviderID = &providerID
	byoMachine.Status.Ready = true

	conditions.MarkTrue(byoMachine, infrav1.HostReadyCondition)

	defer func() {
		if err := helper.Patch(ctx, byoMachine); err != nil && reterr == nil {
			logger.Error(err, "failed to patch byomachine")
			reterr = err
		}
	}()
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ByoMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(controlledType).
		Watches(
			&source.Kind{Type: &infrav1.ByoHost{}},
			&handler.EnqueueRequestForOwner{OwnerType: controlledType, IsController: false},
		).
		// Watch the CAPI resource that owns this infrastructure resource.
		Watches(
			&source.Kind{Type: &clusterv1.Machine{}},
			handler.EnqueueRequestsFromMapFunc(clusterutilv1.MachineToInfrastructureMapFunc(controlledTypeGVK)),
		).
		Complete(r)
}

func (r *ByoMachineReconciler) setNodeProviderID(ctx context.Context, remoteClient client.Client, host infrastructurev1alpha4.ByoHost, providerID string) error {

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

func (r *ByoMachineReconciler) getRemoteClient(ctx context.Context, byoMachine *infrastructurev1alpha4.ByoMachine) (client.Client, error) {
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

func (r *ByoMachineReconciler) setPausedConditionForByoHost(ctx context.Context, providerID string, nameSpace string, isPaused bool) error {

	// The format of providerID is "byoh://<byoHostName>/<RandomString(6)>
	if !strings.HasPrefix(providerID, ProviderIDPrefix) {
		return errors.New("invalid providerID prefix")
	}

	strs := strings.Split(providerID[len(ProviderIDPrefix):], "/")

	if len(strs) == 0 {
		return errors.New("invalid providerID format")
	}

	byoHostName := strs[0]

	byoHost := &infrastructurev1alpha4.ByoHost{}
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
