package reconciler

import (
	"context"
	"errors"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/vmware-tanzu/cluster-api-provider-byoh/agent/cloudinit"
	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/api/v1alpha4"
	corev1 "k8s.io/api/core/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	ctrl "sigs.k8s.io/controller-runtime"
)

type HostReconciler struct {
	Client client.Client
}

func (r HostReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	byoHost := &infrastructurev1alpha4.ByoHost{}
	err := r.Client.Get(ctx, req.NamespacedName, byoHost)
	if err != nil {
		klog.Errorf("error getting ByoHost %s in namespace %s, err=%v", req.NamespacedName.Namespace, req.NamespacedName.Name, err)
		return ctrl.Result{}, err
	}

	if byoHost.Status.MachineRef == nil {
		klog.Info("Machine ref not yet set")
		return ctrl.Result{}, nil
	}

	bootstrapScript, err := r.getBootstrapScript(ctx, byoHost.Status.MachineRef.Name, byoHost.Status.MachineRef.Namespace)
	if err != nil {
		klog.Errorf("getBootstrapScript(%s, %s) return failed, failed, err=%v", byoHost.Status.MachineRef.Name, byoHost.Status.MachineRef.Namespace, err)
		return ctrl.Result{}, err
	}

	err = cloudinit.ScriptExecutor{
		WriteFilesExecutor: cloudinit.FileWriter{},
		RunCmdExecutor:     cloudinit.CmdRunner{}}.Execute(bootstrapScript)
	if err != nil {
		klog.Errorf("cloudinit.ScriptExecutor return failed, err=%v", err)
		return ctrl.Result{}, err
	}

	helper, err := patch.NewHelper(byoHost, r.Client)
	if err != nil {
		klog.Errorf("error creating path helper, err=%v", err)
		return ctrl.Result{}, err
	}
	conditions.MarkTrue(byoHost, infrastructurev1alpha4.K8sComponentsInstalledCondition)
	err = helper.Patch(ctx, byoHost)
	if err != nil {
		klog.Errorf("error in updating conditions on ByoHost, err=%v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r HostReconciler) getBootstrapScript(ctx context.Context, machineName string, namespace string) (string, error) {
	byoMachine := &infrastructurev1alpha4.ByoMachine{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: machineName, Namespace: namespace}, byoMachine)
	if err != nil {
		return "", err
	}

	machine := &clusterv1.Machine{}

	if len(byoMachine.OwnerReferences) == 0 {
		klog.Info("owner ref not yet set")
		return "", errors.New("owner ref not yet set")
	}

	//TODO: Remove this hard coding of owner reference
	err = r.Client.Get(ctx, types.NamespacedName{Name: byoMachine.OwnerReferences[0].Name, Namespace: namespace}, machine)
	if err != nil {
		return "", err
	}

	if machine.Spec.Bootstrap.DataSecretName == nil {
		klog.Info("Bootstrap secret not ready")
		return "", errors.New("Bootstrap secret not ready")
	}

	secret := &corev1.Secret{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: *machine.Spec.Bootstrap.DataSecretName, Namespace: namespace}, secret)
	if err != nil {
		return "", err
	}

	bootstrapSecret := string(secret.Data["value"])

	return string(bootstrapSecret), nil
}

func (r HostReconciler) SetupWithManager(mgr manager.Manager) error {
	return nil
}
