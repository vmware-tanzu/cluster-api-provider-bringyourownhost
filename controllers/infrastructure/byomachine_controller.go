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
	"errors"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/cluster-api/controllers/remote"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/apis/infrastructure/v1alpha4"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	"sigs.k8s.io/cluster-api/util/annotations"
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

	//Fetch the ByoMachine instance.
	byoMachine := &infrastructurev1alpha4.ByoMachine{}
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
	err = r.Client.List(ctx, hostsList)

	if err != nil {
		logger.Error(err, "list byohost failed")
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
		return ctrl.Result{}, errors.New("Bootstrap secret not ready")
	} else {
		host.Spec.BootstrapSecret = &corev1.ObjectReference{
			Kind:      "Secret",
			Namespace: byoMachine.Namespace,
			Name:      *machine.Spec.Bootstrap.DataSecretName,
		}
	}
	hostStr, _ := json.Marshal(host)
	logger.Error(errors.New(string(hostStr)), "huchen: host info")

	err = helper.Patch(ctx, &host)
	if err != nil {
		logger.Error(err, "patch byohost failed")
		return ctrl.Result{}, err
	}

	providerID := fmt.Sprintf("%s%s/%s", ProviderIDPrefix, host.Name, util.RandomString(6))
	remoteClient, err := r.getRemoteClient(ctx, byoMachine)
	if err != nil {
		logger.Error(err, "get remote client failed")
		return ctrl.Result{}, err
	}

	err = r.setNodeProviderID(ctx, remoteClient, host, providerID)
	if err != nil {
		logger.Error(err, "set node providerID failed")
		return ctrl.Result{}, err
	}

	helper, _ = patch.NewHelper(byoMachine, r.Client)
	byoMachine.Spec.ProviderID = providerID
	byoMachine.Status.Ready = true

	conditions.MarkTrue(byoMachine, infrastructurev1alpha4.HostReadyCondition)
	defer func() {
		if err := helper.Patch(ctx, byoMachine); err != nil && reterr == nil {
			logger.Error(err, "Patch byomachine failed")
			reterr = err
		}
	}()
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ByoMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1alpha4.ByoMachine{}).
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

	if err := helper.Patch(ctx, node); err != nil {
		return err
	}

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

func (r *ByoMachineReconciler) setPausedConditionForByoHost(ctx context.Context, providerID string, nameSpace string, isPaused bool) error {
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

	if isPaused == true {
		conditions.MarkTrue(byoHost, infrastructurev1alpha4.PausedCondition)
	} else {
		conditions.MarkFalse(byoHost, infrastructurev1alpha4.PausedCondition, "resume", clusterv1.ConditionSeverityInfo, "")
	}

	err = helper.Patch(ctx, byoHost)
	if err != nil {
		return err
	}

	return nil
}
