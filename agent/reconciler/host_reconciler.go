// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package reconciler

import (
	"context"
	"os"

	"github.com/pkg/errors"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/agent/cloudinit"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/agent/installer"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/agent/registration"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/kube-vip/kube-vip/pkg/vip"
	infrastructurev1beta1 "github.com/vmware-tanzu/cluster-api-provider-byoh/apis/infrastructure/v1beta1"
)

type HostReconciler struct {
	Client         client.Client
	CmdRunner      cloudinit.ICmdRunner
	FileWriter     cloudinit.IFileWriter
	TemplateParser cloudinit.ITemplateParser
}

const (
	bootstrapSentinelFile = "/run/cluster-api/bootstrap-success.complete"
	KubeadmResetCommand   = "kubeadm reset --force"
)

// Reconcile handles events for the ByoHost that is registered by this agent process
func (r *HostReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	logger := ctrl.LoggerFrom(ctx)
	logger.Info("Reconcile request received")

	// Fetch the ByoHost instance
	byoHost := &infrastructurev1beta1.ByoHost{}
	err := r.Client.Get(ctx, req.NamespacedName, byoHost)
	if err != nil {
		logger.Error(err, "error getting ByoHost")
		return ctrl.Result{}, err
	}
	helper, _ := patch.NewHelper(byoHost, r.Client)
	defer func() {
		err = helper.Patch(ctx, byoHost)
		if err != nil && reterr == nil {
			logger.Error(err, "failed to patch byohost")
			reterr = err
		}
	}()

	// Check for host cleanup annotation
	hostAnnotations := byoHost.GetAnnotations()
	_, ok := hostAnnotations[infrastructurev1beta1.HostCleanupAnnotation]
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

func (r *HostReconciler) reconcileNormal(ctx context.Context, byoHost *infrastructurev1beta1.ByoHost) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)
	if byoHost.Status.MachineRef == nil {
		logger.Info("Machine ref not yet set")
		conditions.MarkFalse(byoHost, infrastructurev1beta1.K8sNodeBootstrapSucceeded, infrastructurev1beta1.WaitingForMachineRefReason, clusterv1.ConditionSeverityInfo, "")
		return ctrl.Result{}, nil
	}

	if byoHost.Spec.BootstrapSecret == nil {
		logger.Info("BootstrapDataSecret not ready")
		conditions.MarkFalse(byoHost, infrastructurev1beta1.K8sNodeBootstrapSucceeded, infrastructurev1beta1.BootstrapDataSecretUnavailableReason, clusterv1.ConditionSeverityInfo, "")
		return ctrl.Result{}, nil
	}

	if !conditions.IsTrue(byoHost, infrastructurev1beta1.K8sNodeBootstrapSucceeded) {
		bootstrapScript, err := r.getBootstrapScript(ctx, byoHost.Spec.BootstrapSecret.Name, byoHost.Spec.BootstrapSecret.Namespace)
		if err != nil {
			logger.Error(err, "error getting bootstrap script")
			return ctrl.Result{}, err
		}

		err = r.installK8sComponents(ctx, byoHost)
		if err != nil {
			logger.Error(err, "error in installing k8s components")
			conditions.MarkFalse(byoHost, infrastructurev1beta1.K8sComponentsInstallationSucceeded, infrastructurev1beta1.K8sComponentsInstallationFailedReason, clusterv1.ConditionSeverityInfo, "")
			return ctrl.Result{}, err
		}

		err = r.kubeadmDirCleanup(ctx)
		if err != nil {
			logger.Error(err, "error cleaning up kubeadm directory, please delete it manually for reconcile to proceed.")
			return ctrl.Result{}, err
		}

		err = r.bootstrapK8sNode(ctx, bootstrapScript, byoHost)
		if err != nil {
			logger.Error(err, "error in bootstrapping k8s node")
			_ = r.resetNode(ctx)
			conditions.MarkFalse(byoHost, infrastructurev1beta1.K8sNodeBootstrapSucceeded, infrastructurev1beta1.CloudInitExecutionFailedReason, clusterv1.ConditionSeverityError, "")
			return ctrl.Result{}, err
		}
		logger.Info("k8s node successfully bootstrapped")

		conditions.MarkTrue(byoHost, infrastructurev1beta1.K8sNodeBootstrapSucceeded)
	}

	return ctrl.Result{}, nil
}

func (r *HostReconciler) reconcileDelete(ctx context.Context, byoHost *infrastructurev1beta1.ByoHost) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (r *HostReconciler) getBootstrapScript(ctx context.Context, dataSecretName, namespace string) (string, error) {
	secret := &corev1.Secret{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: dataSecretName, Namespace: namespace}, secret)
	if err != nil {
		return "", err
	}

	bootstrapSecret := string(secret.Data["value"])
	return bootstrapSecret, nil
}

func (r *HostReconciler) SetupWithManager(ctx context.Context, mgr manager.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1beta1.ByoHost{}).
		WithEventFilter(predicates.ResourceNotPaused(ctrl.LoggerFrom(ctx))).
		Complete(r)
}

// cleanup kubeadm dir to remove any stale config on the host
func (r HostReconciler) kubeadmDirCleanup(ctx context.Context) error {
	logger := ctrl.LoggerFrom(ctx)
	logger.Info("cleaning up kubeadm directory")
	const kubeadmDir = "/run/kubeadm"
	return os.RemoveAll(kubeadmDir)
}

func (r HostReconciler) hostCleanUp(ctx context.Context, byoHost *infrastructurev1beta1.ByoHost) error {
	logger := ctrl.LoggerFrom(ctx)
	logger.Info("cleaning up host")
	err := r.resetNode(ctx)
	if err != nil {
		return err
	}

	k8sInstaller, _ := installer.New("", "", logger)
	k8sVersion := byoHost.GetAnnotations()[infrastructurev1beta1.K8sVersionAnnotation]
	err = k8sInstaller.Uninstall(k8sVersion)
	if err != nil {
		return err
	}

	logger.Info("Removing the bootstrap sentinel file...")
	if _, err := os.Stat(bootstrapSentinelFile); !os.IsNotExist(err) {
		err := os.Remove(bootstrapSentinelFile)
		if err != nil {
			return errors.Wrapf(err, "failed to delete sentinel file %s", bootstrapSentinelFile)
		}
	}
	if IP, ok := byoHost.Annotations[infrastructurev1beta1.EndPointIPAnnotation]; ok {
		network, err := vip.NewConfig(IP, registration.LocalHostRegistrar.ByoHostInfo.DefaultNetworkInterfaceName, false)
		if err == nil {
			err := network.DeleteIP()
			if err != nil {
				return err
			}
		}
	}
	// Remove host reservation
	byoHost.Status.MachineRef = nil

	// Remove cluster-name label
	delete(byoHost.Labels, clusterv1.ClusterLabelName)

	// Remove the EndPointIP annotation
	delete(byoHost.Annotations, infrastructurev1beta1.EndPointIPAnnotation)

	// Remove the cleanup annotation
	delete(byoHost.Annotations, infrastructurev1beta1.HostCleanupAnnotation)

	// Remove the cluster version annotation
	delete(byoHost.Annotations, infrastructurev1beta1.K8sVersionAnnotation)

	conditions.MarkFalse(byoHost, infrastructurev1beta1.K8sNodeBootstrapSucceeded, infrastructurev1beta1.K8sNodeAbsentReason, clusterv1.ConditionSeverityInfo, "")
	return nil
}

func (r *HostReconciler) resetNode(ctx context.Context) error {
	logger := ctrl.LoggerFrom(ctx)
	logger.Info("Running kubeadm reset")

	err := r.CmdRunner.RunCmd(KubeadmResetCommand)
	if err != nil {
		return errors.Wrapf(err, "failed to exec kubeadm reset")
	}

	logger.Info("Kubernetes Node reset completed")
	return nil
}

func (r *HostReconciler) bootstrapK8sNode(ctx context.Context, bootstrapScript string, byoHost *infrastructurev1beta1.ByoHost) error {
	logger := ctrl.LoggerFrom(ctx)
	logger.Info("Bootstraping k8s Node")
	return cloudinit.ScriptExecutor{
		WriteFilesExecutor:    r.FileWriter,
		RunCmdExecutor:        r.CmdRunner,
		ParseTemplateExecutor: r.TemplateParser}.Execute(bootstrapScript)
}

func (r *HostReconciler) installK8sComponents(ctx context.Context, byoHost *infrastructurev1beta1.ByoHost) error {
	logger := ctrl.LoggerFrom(ctx)
	logger.Info("Installing K8s")
	conditions.MarkFalse(byoHost, infrastructurev1beta1.K8sComponentsInstallationSucceeded, infrastructurev1beta1.K8sComponentsInstallingReason, clusterv1.ConditionSeverityInfo, "")

	k8sInstaller, _ := installer.New("", "", logger)
	k8sVersion := byoHost.GetAnnotations()[infrastructurev1beta1.K8sVersionAnnotation]
	err := k8sInstaller.Install(k8sVersion)
	if err != nil {
		return err
	}
	conditions.MarkTrue(byoHost, infrastructurev1beta1.K8sComponentsInstallationSucceeded)
	return nil
}
