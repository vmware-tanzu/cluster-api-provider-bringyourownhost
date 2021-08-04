package reconciler

import (
	"context"
	"errors"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/vmware-tanzu/cluster-api-provider-byoh/agent/cloudinit"
	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/apis/infrastructure/v1alpha4"
	corev1 "k8s.io/api/core/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	ctrl "sigs.k8s.io/controller-runtime"
)

type HostReconciler struct {
	Client client.Client
}

func (r HostReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	//Fetch the ByoHost instance.
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

	//Fetch the ByoMachine instance.
	byoMachine := &infrastructurev1alpha4.ByoMachine{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: byoHost.Status.MachineRef.Name, Namespace: byoHost.Status.MachineRef.Namespace}, byoMachine)
	if err != nil {
		klog.Errorf("ByoMachine owner Machine is missing cluster label or cluster does not exist, err=%v", err)
		return ctrl.Result{}, err
	}

	// Fetch the Cluster.
	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, byoMachine.ObjectMeta)
	if err != nil {
		klog.Errorf("ByoMachine owner Machine is missing cluster label or cluster does not exist, err=%v", err)
		return ctrl.Result{}, err
	}

	if cluster == nil {
		klog.Infof("Please associate this machine with a cluster using the label %s: <name of cluster>", clusterv1.ClusterLabelName)
		return ctrl.Result{}, nil
	}

	// Return early if the object or Cluster is paused.
	if annotations.IsPaused(cluster, byoHost) {
		klog.Info("byoMachine or linked Cluster is marked as paused. Won't reconcile")
		return ctrl.Result{}, nil
	}

	// Fetch the Machine.
	machine, err := util.GetOwnerMachine(ctx, r.Client, byoMachine.ObjectMeta)
	if err != nil {
		klog.Errorf("ByoMachine is missing owner Machine, err=%v", err)
		return ctrl.Result{}, err
	}

	if machine == nil {
		klog.Info("Waiting for Machine Controller to set OwnerRef on ByoMachine")
		return ctrl.Result{}, nil
	}

	bootstrapScript, err := r.getBootstrapScript(ctx, machine, byoHost.Status.MachineRef.Namespace)
	if err != nil {
		klog.Errorf("error getting bootstrap script for machine %s in namespace %s, err=%v", byoHost.Status.MachineRef.Name, byoHost.Status.MachineRef.Namespace, err)
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

func (r HostReconciler) getBootstrapScript(ctx context.Context, machine *clusterv1.Machine, namespace string) (string, error) {
	if machine.Spec.Bootstrap.DataSecretName == nil {
		klog.Info("Bootstrap secret not ready")
		return "", errors.New("bootstrap secret not ready")
	}

	secret := &corev1.Secret{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: *machine.Spec.Bootstrap.DataSecretName, Namespace: namespace}, secret)
	if err != nil {
		return "", err
	}

	bootstrapSecret := string(secret.Data["value"])

	return string(bootstrapSecret), nil
}

func (r HostReconciler) SetupWithManager(mgr manager.Manager) error {
	return nil
}
