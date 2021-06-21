/*

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

	"github.com/go-logr/logr"
	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/api/v1alpha4"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/controllers/remote"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ByoMachineReconciler reconciles a ByoMachine object
type ByoMachineReconciler struct {
	client.Client
	Log     logr.Logger
	Scheme  *runtime.Scheme
	Tracker *remote.ClusterCacheTracker
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=byomachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=byomachines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;machines,verbs=get;list;watch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines;machines/status,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch

func (r *ByoMachineReconciler) Reconcile(ctx context.Context, req reconcile.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("Byomachine", req.NamespacedName)

	byoMachine := &infrastructurev1alpha4.ByoMachine{}
	r.Client.Get(ctx, req.NamespacedName, byoMachine)
	//TODO - TDD this error check
	//if err != nil {
	//	if apierrors.IsNotFound(err) {
	//		return ctrl.Result{}, nil
	//	}
	//	return ctrl.Result{}, err
	//}

	hostsList := &infrastructurev1alpha4.ByoHostList{}
	r.Client.List(ctx, hostsList)

	// TODO - TDD this check
	// if err := ; err != nil {
	// 	return ctrl.Result{}, err
	// }

	if len(hostsList.Items) == 0 {
		r.Log.Info("No hosts found, waiting..")
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

	helper.Patch(ctx, &host)

	providerID := fmt.Sprintf("byoh://%s/%s", host.Name, util.RandomString(6))
	remoteClient, _ := r.getRemoteClient(ctx, byoMachine)
	// if err != nil {
	// 	return ctrl.Result{}, err
	// }
	r.setNodeProviderID(ctx, remoteClient, host, providerID)
	// if err != nil {
	// 	return ctrl.Result{}, err
	// }

	helper, _ = patch.NewHelper(byoMachine, r.Client)
	byoMachine.Spec.ProviderID = providerID
	byoMachine.Status.Ready = true

	conditions.MarkTrue(byoMachine, infrastructurev1alpha4.HostReadyCondition)
	//fmt.Println(byoMachine.Status)
	err := helper.Patch(ctx, byoMachine)
	if err != nil {
		fmt.Printf("err: %s", err.Error())
	}
	return ctrl.Result{}, nil
}

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
	helper, _ := patch.NewHelper(node, remoteClient)

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
