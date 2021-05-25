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
	"github.com/go-logr/logr"
	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/api/v1alpha4"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ByoMachineReconciler reconciles a ByoMachine object
type ByoMachineReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=Byomachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=Byomachines/status,verbs=get;update;patch

func (r *ByoMachineReconciler) Reconcile(req reconcile.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = r.Log.WithValues("Byomachine", req.NamespacedName)

	// your logic here
	byoMachine := &infrastructurev1alpha4.ByoMachine{}
	r.Client.Get(ctx, req.NamespacedName, byoMachine)
	//if err != nil {
	//	if apierrors.IsNotFound(err) {
	//		return ctrl.Result{}, nil
	//	}
	//	return ctrl.Result{}, err
	//}

	hostsList := &infrastructurev1alpha4.ByoHostList{}
	if err := r.Client.List(
		ctx,
		hostsList,
	); err != nil {
		return ctrl.Result{}, err
	}
	host := hostsList.Items[0]

	hostBeforePatch := client.MergeFromWithOptions(host.DeepCopyObject(), client.MergeFromWithOptimisticLock{})

	host.Status.MachineRef = &corev1.ObjectReference{
		APIVersion: byoMachine.APIVersion,
		Kind:       byoMachine.Kind,
		Namespace:  byoMachine.Namespace,
		Name:       byoMachine.Name,
		UID:        byoMachine.UID,
	}

	if err := r.Client.Status().Patch(ctx, &host, hostBeforePatch); err != nil {

		if apierrors.IsConflict(err) {
			//logger.Info("Conflict with attempting host reservation, requeue for retry")
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ByoMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1alpha4.ByoMachine{}).
		Complete(r)
}
