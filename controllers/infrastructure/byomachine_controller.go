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
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"

	"github.com/pkg/errors"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/cluster-api/controllers/remote"

	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	infrav1 "github.com/vmware-tanzu/cluster-api-provider-byoh/apis/infrastructure/v1alpha4"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	clusterutilv1 "sigs.k8s.io/cluster-api/util"
)

const (
	// ProviderIDPrefix is the string data prefixed to a BIOS UUID in order
	// to build a provider ID.
	ProviderIDPrefix = "byph://"

	// ProviderIDPattern is a regex pattern and is used by ConvertProviderIDToUUID
	// to convert a providerID into a UUID string.
	ProviderIDPattern = `(?i)^` + ProviderIDPrefix + `([a-f\d]{8}-[a-f\d]{4}-[a-f\d]{4}-[a-f\d]{4}-[a-f\d]{12})$`

	// UUIDPattern is a regex pattern and is used by ConvertUUIDToProviderID
	// to convert a UUID into a providerID string.
	UUIDPattern = `(?i)^[a-f\d]{8}-[a-f\d]{4}-[a-f\d]{4}-[a-f\d]{4}-[a-f\d]{12}$`
)

func ConvertUUIDToProviderID(uuid string) string {
	if uuid == "" {
		return ""
	}
	pattern := regexp.MustCompile(UUIDPattern)
	if !pattern.MatchString(uuid) {
		return ""
	}
	return ProviderIDPrefix + uuid
}

var (
	controlledType     = &infrav1.ByoMachine{}
	controlledTypeName = reflect.TypeOf(controlledType).Elem().Name()
	controlledTypeGVK  = infrav1.GroupVersion.WithKind(controlledTypeName)
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
		logger.V(4).Info("ByoMachine %s linked to a cluster that is paused",
			byoMachine)
		return reconcile.Result{}, nil
	}

	// Fetch the ByoCluster
	byoCluster := &infrav1.ByoCluster{}
	byoClusterName := client.ObjectKey{
		Namespace: byoMachine.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}
	if err := r.Client.Get(ctx, byoClusterName, byoCluster); err != nil {
		logger.Info("Waiting for ByoCluster")
		return reconcile.Result{}, nil
	}

	// Create the patch helper.
	patchHelper, err := patch.NewHelper(byoMachine, r.Client)
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(
			err,
			"failed to init patch helper for %s",
			byoMachine)
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
			patch.WithOwnedConditions{
				Conditions: []clusterv1.ConditionType{clusterv1.ReadyCondition},
			},
		)

		// Patch the VSphereMachine resource.
		if err != nil {
			if reterr == nil {
				reterr = err
			}
			logger.Error(err, "patch failed", "byomachine", byoMachine.Namespace+"/"+byoMachine.Name)
		}
	}()

	// Handle deleted machines
	if !byoMachine.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, byoMachine)
	}

	// Handle non-deleted machines
	return r.reconcileNormal(ctx, byoMachine, cluster)

}

func (r ByoMachineReconciler) reconcileDelete(ctx context.Context, byoMachine *infrav1.ByoMachine) (reconcile.Result, error) {
	hostsList := &infrav1.ByoHostList{}

	err := r.Client.List(ctx, hostsList)
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err,
			"unable to list ByoHost part of ByoMachine %s/%s", byoMachine.Namespace, byoMachine.Name)
	}
	for _, host := range hostsList.Items {
		if host.Status.MachineRef.UID == byoMachine.UID {
			host.Status.MachineRef = nil
			host.Labels[infrav1.ByoHostProvisionLabel] = "false"
			err := r.Client.Status().Update(ctx, &host)
			if err != nil {
				return reconcile.Result{}, errors.Wrapf(err,
					"unable to update ByoHost status %s/%s", host.Namespace, host.Name)
			}
		}
	}

	controllerutil.RemoveFinalizer(byoMachine, infrav1.MachineFinalizer)
	return reconcile.Result{}, nil
}

func (r *ByoMachineReconciler) reconcileByohost(ctx context.Context, byoMachine *infrav1.ByoMachine, computeObj *unstructured.Unstructured) (bool, error) {
	selector, err := metav1.LabelSelectorAsSelector(byoMachine.Spec.Selector)
	if err != nil {
		return false, err
	}
	// Find the current byohost match the selector
	hostsList := &infrav1.ByoHostList{}
	err = r.Client.List(ctx, hostsList, client.InNamespace(byoMachine.Namespace), client.MatchingLabels{infrav1.LabelByoMachineOwn: byoMachine.Name})
	if err != nil {
		return false, err
	}

	// The hostsList should have one at most. To handle corner case we should remove MachineRef
	for _, host := range hostsList.Items {
		if computeObj == nil {
			Data, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&host)
			if selector.Matches(labels.Set(host.Labels)) && err == nil {
				computeObj = &unstructured.Unstructured{Object: Data}
				computeObj.SetGroupVersionKind(host.GetObjectKind().GroupVersionKind())
				computeObj.SetAPIVersion(host.GetObjectKind().GroupVersionKind().GroupVersion().String())
				computeObj.SetKind(host.GetObjectKind().GroupVersionKind().Kind)
				continue
			}
		}

		if host.Status.MachineRef.UID == byoMachine.UID {
			host.Status.MachineRef = nil
			delete(host.Labels, infrav1.LabelByoMachineOwn)
			r.Client.Update(ctx, &host)
		}
	}

	if computeObj != nil {
		return true, nil
	}

	err = r.Client.List(ctx, hostsList, client.InNamespace(byoMachine.Namespace), client.MatchingLabelsSelector{Selector: selector})
	if err != nil {
		return false, err
	}
	for _, host := range hostsList.Items {
		host.Labels[infrav1.LabelByoMachineOwn] = byoMachine.Name
		host.Status.MachineRef = &corev1.ObjectReference{
			APIVersion: byoMachine.APIVersion,
			Kind:       byoMachine.Kind,
			Namespace:  byoMachine.Namespace,
			Name:       byoMachine.Name,
			UID:        byoMachine.UID,
		}
		Data, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&host)
		if err == nil && r.Client.Update(ctx, &host) == nil {
			computeObj = &unstructured.Unstructured{Object: Data}
			computeObj.SetGroupVersionKind(host.GetObjectKind().GroupVersionKind())
			computeObj.SetAPIVersion(host.GetObjectKind().GroupVersionKind().GroupVersion().String())
			computeObj.SetKind(host.GetObjectKind().GroupVersionKind().Kind)
			return true, nil
		}
	}

	return false, fmt.Errorf("no availabe compute resouce find for %s", byoMachine)
}

func (r ByoMachineReconciler) waitReadyState(ctx context.Context, byoMachine *infrav1.ByoMachine, computeObj *unstructured.Unstructured) (bool, error) {
	logger := log.FromContext(ctx)

	ready, ok, err := unstructured.NestedBool(computeObj.Object, "status", "ready")
	if !ok {
		if err != nil {
			return false, errors.Wrapf(err,
				"unexpected error when getting status.ready from %s %s/%s for %s",
				computeObj.GroupVersionKind(),
				computeObj.GetNamespace(),
				computeObj.GetName(),
				byoMachine)
		}
		logger.Info("status.ready not found",
			"computeGVK", computeObj.GroupVersionKind().String(),
			"computeNamespace", computeObj.GetNamespace(),
			"computeName", computeObj.GetName())
		return false, nil
	}
	if !ready {
		logger.Info("status.ready is false",
			"computeGVK", computeObj.GroupVersionKind().String(),
			"computeNamespace", computeObj.GetNamespace(),
			"computeName", computeObj.GetName())
		return false, nil
	}

	return true, nil
}

func (r *ByoMachineReconciler) reconcileProviderID(ctx context.Context, byoMachine *infrav1.ByoMachine, computeObj *unstructured.Unstructured) (bool, error) {
	logger := log.FromContext(ctx)

	biosUUID, ok, err := unstructured.NestedString(computeObj.Object, "spec", "biosUUID")
	if !ok {
		if err != nil {
			return false, errors.Wrapf(err,
				"unexpected error when getting spec.biosUUID from %s %s/%s for %s",
				computeObj.GroupVersionKind(),
				computeObj.GetNamespace(),
				computeObj.GetName(),
				byoMachine)
		}
		logger.Info("spec.biosUUID not found",
			"computeGVK", computeObj.GroupVersionKind().String(),
			"computeNamespace", computeObj.GetNamespace(),
			"computeName", computeObj.GetName())
		return false, nil
	}
	if biosUUID == "" {
		logger.Info("spec.biosUUID is empty",
			"computeGVK", computeObj.GroupVersionKind().String(),
			"computeNamespace", computeObj.GetNamespace(),
			"computeName", computeObj.GetName())
		return false, nil
	}

	providerID := ConvertUUIDToProviderID(biosUUID)
	if providerID == "" {
		return false, errors.Errorf("invalid BIOS UUID %s from %s %s/%s for %s",
			biosUUID,
			computeObj.GroupVersionKind(),
			computeObj.GetNamespace(),
			computeObj.GetName(),
			byoMachine)
	}
	if byoMachine.Spec.ProviderID == nil || *byoMachine.Spec.ProviderID != providerID {
		byoMachine.Spec.ProviderID = &providerID
		logger.Info("updated provider ID", "provider-id", providerID)
	}

	return true, nil
}

func (r *ByoMachineReconciler) reconcileNetwork(ctx context.Context, byoMachine *infrav1.ByoMachine, computeObj *unstructured.Unstructured) (bool, error) {
	logger := log.FromContext(ctx)

	var errs []error
	if networkStatusListOfIfaces, ok, _ := unstructured.NestedSlice(computeObj.Object, "status", "network"); ok {
		networkStatusList := []infrav1.NetworkStatus{}
		for i, networkStatusListMemberIface := range networkStatusListOfIfaces {
			if buf, err := json.Marshal(networkStatusListMemberIface); err != nil {
				logger.Error(err,
					"unsupported data for member of status.network list",
					"index", i)
				errs = append(errs, err)
			} else {
				var networkStatus infrav1.NetworkStatus
				err := json.Unmarshal(buf, &networkStatus)
				if err == nil && networkStatus.MACAddr == "" {
					err = errors.New("macAddr is required")
					errs = append(errs, err)
				}
				if err != nil {
					logger.Error(err,
						"unsupported data for member of status.network list",
						"index", i, "data", string(buf))
					errs = append(errs, err)
				} else {
					networkStatusList = append(networkStatusList, networkStatus)
				}
			}
		}
		byoMachine.Status.Network = networkStatusList
	}

	if addresses, ok, _ := unstructured.NestedStringSlice(computeObj.Object, "status", "addresses"); ok {
		var machineAddresses []clusterv1.MachineAddress
		for _, addr := range addresses {
			machineAddresses = append(machineAddresses, clusterv1.MachineAddress{
				Type:    clusterv1.MachineExternalIP,
				Address: addr,
			})
		}
		byoMachine.Status.Addresses = machineAddresses
	}

	if len(byoMachine.Status.Addresses) == 0 {
		logger.Info("waiting on IP addresses")
		return false, kerrors.NewAggregate(errs)
	}

	return true, nil
}

func (r ByoMachineReconciler) reconcileNormal(ctx context.Context, byoMachine *infrav1.ByoMachine, cluster *clusterv1.Cluster) (reconcile.Result, error) {
	logger := log.FromContext(ctx)
	var computeObj *unstructured.Unstructured

	controllerutil.AddFinalizer(byoMachine, infrav1.MachineFinalizer)

	if !cluster.Status.InfrastructureReady {
		logger.Info("Cluster infrastructure is not ready yet")
		conditions.MarkFalse(byoMachine, infrav1.HostProvisionedCondition, infrav1.WaitingForClusterInfrastructureReason, clusterv1.ConditionSeverityInfo, "")
		return reconcile.Result{}, nil
	}

	if ok, err := r.reconcileByohost(ctx, byoMachine, computeObj); !ok {
		if err != nil {
			return reconcile.Result{}, errors.Wrapf(err,
				"unexpected error while reconciling compute resouse for %s", byoMachine)
		}
		logger.Info("No suitable compute resouse find")
		return reconcile.Result{}, nil
	}

	if ok, err := r.waitReadyState(ctx, byoMachine, computeObj); !ok {
		if err != nil {
			return reconcile.Result{}, errors.Wrapf(err,
				"unexpected error while reconciling ready state for %s", byoMachine)
		}
		logger.Info("waiting for ready state")
		// ByoMachine wraps a compute(Baremetal or VM), so we are mirroring status from the underlying compute
		// in order to provide evidences about machine provisioning while provisioning is actually happening.
		conditions.SetMirror(byoMachine, infrav1.HostProvisionedCondition, conditions.UnstructuredGetter(computeObj))
		return reconcile.Result{}, nil
	}

	if ok, err := r.reconcileProviderID(ctx, byoMachine, computeObj); !ok {
		if err != nil {
			return reconcile.Result{}, errors.Wrapf(err,
				"unexpected error while reconciling provider ID for %s", byoMachine)
		}
		logger.Info("provider ID is not reconciled")
		return reconcile.Result{}, nil
	}

	if ok, err := r.reconcileNetwork(ctx, byoMachine, computeObj); !ok {
		if err != nil {
			return reconcile.Result{}, errors.Wrapf(err,
				"unexpected error while reconciling network for %s", byoMachine)
		}
		logger.Info("network is not reconciled")
		conditions.MarkFalse(byoMachine, infrav1.HostProvisionedCondition, infrav1.WaitingForNetworkAddressesReason, clusterv1.ConditionSeverityInfo, "")
		return reconcile.Result{}, nil
	}

	byoMachine.Status.Ready = true
	conditions.MarkTrue(byoMachine, infrav1.HostProvisionedCondition)

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
