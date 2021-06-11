package reconciler

import (
	"context"
	"encoding/base64"

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
		klog.Fatal(err)
	}

	bootstrapScript, err := r.getBootstrapScript(ctx, byoHost.Status.MachineRef.Name, req.NamespacedName.Namespace)
	if err != nil {
		klog.Fatal(err)
	}

	err = cloudinit.ScriptExecutor{Executor: cloudinit.FileWriter{}}.Execute(bootstrapScript)
	if err != nil {
		klog.Fatal(err)
	}

	helper, err := patch.NewHelper(byoHost, r.Client)
	if err != nil {
		klog.Fatal(err)
	}
	conditions.MarkTrue(byoHost, infrastructurev1alpha4.K8sComponentsInstalledCondition)
	err = helper.Patch(ctx, byoHost)
	if err != nil {
		klog.Fatal(err)
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

	//TODO: Remove this hard coding of owner reference
	err = r.Client.Get(ctx, types.NamespacedName{Name: byoMachine.OwnerReferences[0].Name, Namespace: namespace}, machine)
	if err != nil {
		return "", err
	}

	secret := &corev1.Secret{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: *machine.Spec.Bootstrap.DataSecretName, Namespace: namespace}, secret)
	if err != nil {
		return "", err
	}

	encodedBootstrapSecret := string(secret.Data["value"])

	decodedScript, err := base64.StdEncoding.DecodeString(encodedBootstrapSecret)
	if err != nil {
		return "", err
	}

	return string(decodedScript), nil
}

func (r HostReconciler) SetupWithManager(mgr manager.Manager) error {
	return nil
}
