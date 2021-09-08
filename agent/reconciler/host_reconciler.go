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
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/pkg/errors"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/agent/cloudinit"
	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/apis/infrastructure/v1alpha4"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type HostReconciler struct {
	Client     client.Client
	CmdRunner  cloudinit.ICmdRunner
	FileWriter cloudinit.IFileWriter
}

const (
	hostCleanupAnnotation = "byoh.infrastructure.cluster.x-k8s.io/unregistering"
	KubeadmResetCommand   = "kubeadm reset --force"
)

func (r HostReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	// Fetch the ByoHost instance.
	byoHost := &infrastructurev1alpha4.ByoHost{}
	err := r.Client.Get(ctx, req.NamespacedName, byoHost)
	if err != nil {
		klog.Errorf("error getting ByoHost %s in namespace %s, err=%v", req.NamespacedName.Namespace, req.NamespacedName.Name, err)
		return ctrl.Result{}, err
	}

	helper, _ := patch.NewHelper(byoHost, r.Client)
	defer func() {
		if err = helper.Patch(ctx, byoHost); err != nil && reterr == nil {
			klog.Errorf("failed to patch byohost, err=%v", err)
			reterr = err
		}
	}()

	// Return early if the object is paused.
	if annotations.HasPausedAnnotation(byoHost) {
		klog.Info("The related byoMachine or linked Cluster is marked as paused. Won't reconcile")
		conditions.MarkFalse(byoHost, infrastructurev1alpha4.K8sNodeBootstrapSucceeded, infrastructurev1alpha4.ClusterOrResourcePausedReason, v1alpha4.ConditionSeverityInfo, "")
		return ctrl.Result{}, nil
	}

	// Check for host cleanup annotation
	hostAnnotations := byoHost.GetAnnotations()
	_, ok := hostAnnotations[hostCleanupAnnotation]
	if ok {
		err = r.hostCleanUp(ctx, byoHost)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Handle deleted machines
	if !byoHost.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, byoHost)
	}
	return r.reconcileNormal(ctx, byoHost)
}

func (r *HostReconciler) reconcileNormal(ctx context.Context, byoHost *infrastructurev1alpha4.ByoHost) (ctrl.Result, error) {
	if byoHost.Status.MachineRef == nil {
		klog.Info("Machine ref not yet set")
		conditions.MarkFalse(byoHost, infrastructurev1alpha4.K8sNodeBootstrapSucceeded, infrastructurev1alpha4.WaitingForMachineRefReason, v1alpha4.ConditionSeverityInfo, "")
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

	if conditions.IsUnknown(byoHost, infrastructurev1alpha4.K8sNodeBootstrapSucceeded) || conditions.IsFalse(byoHost, infrastructurev1alpha4.K8sNodeBootstrapSucceeded) {
		err = r.bootstrapK8sNode(bootstrapScript, byoHost)
		if err != nil {
			klog.Errorf("error in bootstrapping k8s node, err=%v", err)
			conditions.MarkFalse(byoHost, infrastructurev1alpha4.K8sNodeBootstrapSucceeded, infrastructurev1alpha4.CloudInitExecutionFailedReason, v1alpha4.ConditionSeverityError, "")
			return ctrl.Result{}, err
		}
		klog.Info("k8s node successfully bootstrapped")
	}

	conditions.MarkTrue(byoHost, infrastructurev1alpha4.K8sNodeBootstrapSucceeded)

	return ctrl.Result{}, nil
}

func (r *HostReconciler) reconcileDelete(ctx context.Context, byoHost *infrastructurev1alpha4.ByoHost) (ctrl.Result, error) {
	// TODO: add logic when this host has MachineRef assigned

	return ctrl.Result{}, nil
}

func (r HostReconciler) getBootstrapScript(ctx context.Context, dataSecretName, namespace string) (string, error) {
	secret := &corev1.Secret{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: dataSecretName, Namespace: namespace}, secret)
	if err != nil {
		return "", err
	}

	bootstrapSecret := string(secret.Data["value"])
	return bootstrapSecret, nil
}

func (r HostReconciler) SetupWithManager(mgr manager.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1alpha4.ByoHost{}).
		Complete(r)
}

func (r HostReconciler) hostCleanUp(ctx context.Context, byoHost *infrastructurev1alpha4.ByoHost) error {
	err := r.resetNode()
	if err != nil {
		return err
	}

	conditions.MarkFalse(byoHost, infrastructurev1alpha4.K8sNodeBootstrapSucceeded, infrastructurev1alpha4.K8sNodeAbsentReason, v1alpha4.ConditionSeverityInfo, "")
	return nil
}

func (r *HostReconciler) resetNode() error {
	klog.Info("Running kubeadm reset...")

	err := r.CmdRunner.RunCmd(KubeadmResetCommand)
	if err != nil {
		return errors.Wrapf(err, "failed to exec kubeadm reset")
	}

	klog.Info("Kubernetes Node reset")
	return nil
}

func (r HostReconciler) bootstrapK8sNode(bootstrapScript string, byoHost *infrastructurev1alpha4.ByoHost) error {
	return cloudinit.ScriptExecutor{
		WriteFilesExecutor: r.FileWriter,
		RunCmdExecutor:     r.CmdRunner}.Execute(bootstrapScript)
}
