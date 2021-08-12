package reconciler

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/cluster-api/api/v1alpha4"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/vmware-tanzu/cluster-api-provider-byoh/agent/cloudinit"
	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/apis/infrastructure/v1alpha4"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type HostReconciler struct {
	Client client.Client
}

func (r *HostReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	//Fetch the ByoHost instance.
	byoHost := &infrastructurev1alpha4.ByoHost{}
	err := r.Client.Get(ctx, req.NamespacedName, byoHost)
	if err != nil {
		klog.Errorf("error getting ByoHost %s in namespace %s, err=%v", req.NamespacedName.Namespace, req.NamespacedName.Name, err)
		return ctrl.Result{}, err
	}

	helper, _ := patch.NewHelper(byoHost, r.Client)
	defer func() {
		if err := helper.Patch(ctx, byoHost); err != nil && reterr == nil {
			klog.Errorf("failed to patch byohost, err=%v", err)
			reterr = err
		}
	}()

	if byoHost.Status.MachineRef == nil {
		klog.Info("Machine ref not yet set")
		conditions.MarkFalse(byoHost, infrastructurev1alpha4.K8sNodeBootstrapSucceeded, infrastructurev1alpha4.WaitingForMachineRefReason, v1alpha4.ConditionSeverityInfo, "")
		return ctrl.Result{}, nil
	}

	// Return early if the object or Cluster is paused.
	if annotations.HasPausedAnnotation(byoHost) {
		klog.Info("The related byoMachine or linked Cluster is marked as paused. Won't reconcile")
		return ctrl.Result{}, nil
	}

	if byoHost.Spec.BootstrapSecret == nil {
		klog.Info("BootstrapDataSecret not ready")
		conditions.MarkFalse(byoHost, infrastructurev1alpha4.K8sNodeBootstrapSucceeded, infrastructurev1alpha4.BootstrapDataSecretUnavailableReason, v1alpha4.ConditionSeverityInfo, "")
		return ctrl.Result{}, nil
	}

	bootstrapScript, err := r.getBootstrapScript(ctx, byoHost.Spec.BootstrapSecret.Name, byoHost.Spec.BootstrapSecret.Namespace)
	if err != nil {
		klog.Errorf("error getting bootstrap script, err=%v", err)
		return ctrl.Result{}, err
	}

	err = cloudinit.ScriptExecutor{
		WriteFilesExecutor: cloudinit.FileWriter{},
		RunCmdExecutor:     cloudinit.CmdRunner{}}.Execute(bootstrapScript)
	if err != nil {
		klog.Errorf("cloudinit.ScriptExecutor return failed, err=%v", err)
		return ctrl.Result{}, err
	}

	if err != nil {
		klog.Errorf("error creating path helper, err=%v", err)
		return ctrl.Result{}, err
	}

	conditions.MarkTrue(byoHost, infrastructurev1alpha4.K8sNodeBootstrapSucceeded)

	defer func() {
		if err := helper.Patch(ctx, byoHost); err != nil && reterr == nil {
			klog.Errorf("failed to patch byohost, err=%v", err)
			reterr = err
		}
	}()

	return ctrl.Result{}, nil
}

func (r *HostReconciler) getBootstrapScript(ctx context.Context, dataSecretName, namespace string) (string, error) {
	secret := &corev1.Secret{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: dataSecretName, Namespace: namespace}, secret)
	if err != nil {
		return "", err
	}

	bootstrapSecret := string(secret.Data["value"])
	return string(bootstrapSecret), nil
}

func (r *HostReconciler) SetupWithManager(mgr manager.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1alpha4.ByoHost{}).
		WithEventFilter(predicate.Funcs{
			// TODO will need to remove this and
			// will be handled with delete stories
			DeleteFunc: func(e event.DeleteEvent) bool {
				return false
			},
		}).
		Complete(r)
}
