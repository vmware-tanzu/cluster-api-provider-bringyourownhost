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
	"reflect"

	infrav1 "vmware-tanzu/cluster-api-provider-byoh/api/v1alpha3"

	"github.com/go-logr/logr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	clusterutilv1 "sigs.k8s.io/cluster-api/util"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=byomachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=byomachines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines;machines/status,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch

type MachineReconciler struct {
	Client client.Client
	Log    logr.Logger
}

func (r *MachineReconciler) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	var (
		controlledType     = &infrav1.BYOMachine{}
		controlledTypeName = reflect.TypeOf(controlledType).Elem().Name()
		controlledTypeGVK  = infrav1.GroupVersion.WithKind(controlledTypeName)
	)

	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.BYOMachine{}).
		// Watch the CAPI resource that owns this infrastructure resource.
		Watches(
			&source.Kind{Type: &clusterv1.Machine{}},
			&handler.EnqueueRequestsFromMapFunc{
				ToRequests: clusterutilv1.MachineToInfrastructureMapFunc(controlledTypeGVK),
			},
		).
		WithOptions(options).
		Complete(r)
}

// Reconcile ensures the back-end state reflects the Kubernetes resource state intent.
func (r MachineReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	r.Log.Info("Processing", "req", req)

	return reconcile.Result{}, nil
}
