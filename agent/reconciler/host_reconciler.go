// Copyright 2021 VMware, Inc. All Rights Reserved.
// Copyright 2026 Platform9, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package reconciler

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/cloudinit"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/registration"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	infrastructurev1beta1 "github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/apis/infrastructure/v1beta1"
)

// HostReconciler encapsulates the data/logic needed to reconcile a ByoHost
type HostReconciler struct {
	Client              client.Client
	CmdRunner           cloudinit.ICmdRunner
	FileWriter          cloudinit.IFileWriter
	TemplateParser      cloudinit.ITemplateParser
	Recorder            record.EventRecorder
	SkipK8sInstallation bool
	DownloadPath        string
	// HeartbeatInterval is the minimum time between LastHeartbeatTime writes,
	// and the interval this reconciler self-requeues on. Wired from the
	// --heartbeat-interval flag in agent/main.go.
	HeartbeatInterval time.Duration
}

const (
	bootstrapSentinelFile = "/run/cluster-api/bootstrap-success.complete"
	// KubeadmResetCommand is the command to run to force reset/remove nodes' local file system of the files created by kubeadm
	KubeadmResetCommand = "kubeadm reset --force"
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
	if !byoHost.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, byoHost)
	}

	result, err := r.reconcileNormal(ctx, byoHost)
	if err == nil {
		result.RequeueAfter = r.HeartbeatInterval
	}
	return result, err
}

func (r *HostReconciler) reconcileNormal(ctx context.Context, byoHost *infrastructurev1beta1.ByoHost) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)
	logger = logger.WithValues("ByoHost", byoHost.Name)
	logger.Info("reconcile normal")

	// NOTE: LastHeartbeatTime is written by the agent (host clock) and evaluated
	// by the management controller (manager clock). Clock skew between the two
	// will affect perceived liveness. NTP (or equivalent) synchronization is
	// assumed on both the host and management nodes.
	now := metav1.Now()
	if byoHost.Status.LastHeartbeatTime == nil || now.Sub(byoHost.Status.LastHeartbeatTime.Time) >= r.HeartbeatInterval {
		byoHost.Status.LastHeartbeatTime = &now
	}

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
			r.Recorder.Eventf(byoHost, corev1.EventTypeWarning, "ReadBootstrapSecretFailed", "bootstrap secret %s not found", byoHost.Spec.BootstrapSecret.Name)
			return ctrl.Result{}, err
		}

		if r.SkipK8sInstallation {
			logger.Info("Skipping installation of k8s components")
		} else if !conditions.IsTrue(byoHost, infrastructurev1beta1.K8sComponentsInstallationSucceeded) {
			if byoHost.Spec.InstallationSecret == nil {
				logger.Info("InstallationSecret not ready")
				conditions.MarkFalse(byoHost, infrastructurev1beta1.K8sComponentsInstallationSucceeded, infrastructurev1beta1.K8sInstallationSecretUnavailableReason, clusterv1.ConditionSeverityInfo, "")
				return ctrl.Result{}, nil
			}
			err = r.executeInstallerController(ctx, byoHost)
			if err != nil {
				return ctrl.Result{}, err
			}
			r.Recorder.Event(byoHost, corev1.EventTypeNormal, "InstallScriptExecutionSucceeded", "install script executed")
			conditions.MarkTrue(byoHost, infrastructurev1beta1.K8sComponentsInstallationSucceeded)
		} else {
			logger.Info("install script already executed")
		}

		err = r.cleank8sdirectories(ctx)
		if err != nil {
			logger.Error(err, "error cleaning up k8s directories, please delete it manually for reconcile to proceed.")
			r.Recorder.Event(byoHost, corev1.EventTypeWarning, "CleanK8sDirectoriesFailed", "clean k8s directories failed")
			conditions.MarkFalse(byoHost, infrastructurev1beta1.K8sNodeBootstrapSucceeded, infrastructurev1beta1.CleanK8sDirectoriesFailedReason, clusterv1.ConditionSeverityError, "")
			return ctrl.Result{}, err
		}

		err = r.bootstrapK8sNode(ctx, bootstrapScript, byoHost)
		if err != nil {
			logger.Error(err, "error in bootstrapping k8s node")
			r.Recorder.Event(byoHost, corev1.EventTypeWarning, "BootstrapK8sNodeFailed", "k8s Node Bootstrap failed")
			_ = r.resetNode(ctx, byoHost)
			conditions.MarkFalse(byoHost, infrastructurev1beta1.K8sNodeBootstrapSucceeded, infrastructurev1beta1.CloudInitExecutionFailedReason, clusterv1.ConditionSeverityError, "")
			return ctrl.Result{}, err
		}
		logger.Info("k8s node successfully bootstrapped")
		r.Recorder.Event(byoHost, corev1.EventTypeNormal, "BootstrapK8sNodeSucceeded", "k8s Node Bootstraped")
		conditions.MarkTrue(byoHost, infrastructurev1beta1.K8sNodeBootstrapSucceeded)
	}

	return ctrl.Result{}, nil
}

func (r *HostReconciler) executeInstallerController(ctx context.Context, byoHost *infrastructurev1beta1.ByoHost) error {
	logger := ctrl.LoggerFrom(ctx)
	if byoHost.Spec.InstallationSecret == nil {
		return fmt.Errorf("InstallationSecret not found in ByoHost %s", byoHost.Name)
	}
	secret := &corev1.Secret{}
	err := r.Client.Get(ctx, types.NamespacedName{
		Name:      byoHost.Spec.InstallationSecret.Name,
		Namespace: byoHost.Spec.InstallationSecret.Namespace,
	}, secret)
	if err != nil {
		logger.Error(err, "error getting install script")
		r.Recorder.Eventf(byoHost, corev1.EventTypeWarning, "ReadInstallationSecretFailed", "install script %s not found", byoHost.Spec.InstallationSecret.Name)
		return err
	}
	installScriptBytes, ok := secret.Data["install"]
	if !ok {
		return fmt.Errorf("install script not found in secret %s", secret.Name)
	}
	installScript := string(installScriptBytes)
	installScript, err = r.parseScript(ctx, installScript)
	if err != nil {
		return err
	}
	logger.Info("executing install script")
	err = r.CmdRunner.RunCmd(ctx, installScript)
	if err != nil {
		logger.Error(err, "error executing installation script")
		r.Recorder.Event(byoHost, corev1.EventTypeWarning, "InstallScriptExecutionFailed", "install script execution failed")
		conditions.MarkFalse(byoHost, infrastructurev1beta1.K8sComponentsInstallationSucceeded, infrastructurev1beta1.K8sComponentsInstallationFailedReason, clusterv1.ConditionSeverityInfo, "")
		return err
	}
	return nil
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

func (r *HostReconciler) parseScript(ctx context.Context, script string) (string, error) {
	data, err := cloudinit.TemplateParser{
		Template: map[string]string{
			"BundleDownloadPath": r.DownloadPath,
		},
	}.ParseTemplate(script)
	if err != nil {
		return "", fmt.Errorf("unable to apply install parsed template to the data object")
	}
	return data, nil
}

// SetupWithManager sets up the controller with the manager
func (r *HostReconciler) SetupWithManager(ctx context.Context, mgr manager.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1beta1.ByoHost{}).
		WithEventFilter(predicates.ResourceNotPaused(ctrl.LoggerFrom(ctx))).
		Complete(r)
}

// cleanup /run/kubeadm, /etc/cni/net.d dirs to remove any stale config on the host
func (r *HostReconciler) cleank8sdirectories(ctx context.Context) error {
	logger := ctrl.LoggerFrom(ctx)

	dirs := []string{
		"/run/kubeadm/*",
		"/etc/cni/net.d/*",
	}

	errList := make([]error, 0)
	for _, dir := range dirs {
		logger.Info(fmt.Sprintf("cleaning up directory %s", dir))
		if err := common.RemoveGlob(dir); err != nil {
			logger.Error(err, fmt.Sprintf("failed to clean up directory %s", dir))
			errList = append(errList, err)
		}
	}

	if len(errList) > 0 {
		err := errList[0]               //nolint: gosec
		for _, e := range errList[1:] { //nolint: gosec
			err = fmt.Errorf("%w; %v error", err, e)
		}
		return errors.WithMessage(err, "not all k8s directories are cleaned up")
	}
	return nil
}

func (r *HostReconciler) hostCleanUp(ctx context.Context, byoHost *infrastructurev1beta1.ByoHost) error {
	logger := ctrl.LoggerFrom(ctx)
	logger.Info("cleaning up host")

	// Guard kubeadm reset behind the installation condition — only run if k8s was installed.
	// MarkFalse immediately after reset so retries skip reset and only retry the uninstall script.
	k8sComponentsInstallationSucceeded := conditions.Get(byoHost, infrastructurev1beta1.K8sComponentsInstallationSucceeded)
	if k8sComponentsInstallationSucceeded != nil && k8sComponentsInstallationSucceeded.Status == corev1.ConditionTrue {
		err := r.resetNode(ctx, byoHost)
		if err != nil {
			return err
		}
		conditions.MarkFalse(byoHost, infrastructurev1beta1.K8sComponentsInstallationSucceeded, infrastructurev1beta1.K8sNodeAbsentReason, clusterv1.ConditionSeverityInfo, "")
	} else {
		logger.Info("Skipping k8s node reset")
	}

	// Guard uninstall script behind UninstallationSecret being set — independent of the reset
	// condition so that retries after a failed uninstall still execute the script.
	if !r.SkipK8sInstallation && byoHost.Spec.UninstallationSecret != nil {
		logger.Info("Executing Uninstall script")
		secret := &corev1.Secret{}
		err := r.Client.Get(ctx, types.NamespacedName{
			Name:      byoHost.Spec.UninstallationSecret.Name,
			Namespace: byoHost.Spec.UninstallationSecret.Namespace,
		}, secret)
		if err != nil {
			logger.Error(err, "error getting uninstallation script secret")
			r.Recorder.Eventf(byoHost, corev1.EventTypeWarning, "ReadUninstallationSecretFailed", "uninstallation secret %s not found", byoHost.Spec.UninstallationSecret.Name)
			return err
		}
		uninstallScriptBytes, ok := secret.Data["uninstall"]
		if !ok {
			return fmt.Errorf("uninstall script not found in secret %s", secret.Name)
		}
		uninstallScript := string(uninstallScriptBytes)
		uninstallScript, err = r.parseScript(ctx, uninstallScript)
		if err != nil {
			logger.Error(err, "error parsing Uninstallation script")
			return err
		}
		err = r.CmdRunner.RunCmd(ctx, uninstallScript)
		if err != nil {
			logger.Error(err, "error executing Uninstallation script")
			r.Recorder.Event(byoHost, corev1.EventTypeWarning, "UninstallScriptExecutionFailed", "uninstall script execution failed")
			return err
		}
		logger.Info("host removed from the cluster and the uninstall is executed successfully")
	} else if r.SkipK8sInstallation {
		logger.Info("Skipping uninstallation of k8s components")
	}
	conditions.MarkFalse(byoHost, infrastructurev1beta1.K8sNodeBootstrapSucceeded, infrastructurev1beta1.K8sNodeAbsentReason, clusterv1.ConditionSeverityInfo, "")

	err := r.removeSentinelFile(ctx, byoHost)
	if err != nil {
		return err
	}

	err = r.deleteEndpointIP(ctx, byoHost)
	if err != nil {
		return err
	}

	byoHost.Spec.InstallationSecret = nil
	r.removeAnnotations(ctx, byoHost)
	conditions.MarkFalse(byoHost, infrastructurev1beta1.K8sNodeBootstrapSucceeded, infrastructurev1beta1.K8sNodeAbsentReason, clusterv1.ConditionSeverityInfo, "")
	return nil
}

func (r *HostReconciler) resetNode(ctx context.Context, byoHost *infrastructurev1beta1.ByoHost) error {
	logger := ctrl.LoggerFrom(ctx)
	logger.Info("Running kubeadm reset")

	err := r.CmdRunner.RunCmd(ctx, KubeadmResetCommand)
	if err != nil {
		r.Recorder.Event(byoHost, corev1.EventTypeWarning, "ResetK8sNodeFailed", "k8s Node Reset failed")
		return errors.Wrapf(err, "failed to exec kubeadm reset")
	}
	logger.Info("Kubernetes Node reset completed")
	r.Recorder.Event(byoHost, corev1.EventTypeNormal, "ResetK8sNodeSucceeded", "k8s Node Reset completed")
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

func (r *HostReconciler) removeSentinelFile(ctx context.Context, byoHost *infrastructurev1beta1.ByoHost) error {
	logger := ctrl.LoggerFrom(ctx)
	logger.Info("Removing the bootstrap sentinel file")
	if _, err := os.Stat(bootstrapSentinelFile); !os.IsNotExist(err) {
		err := os.Remove(bootstrapSentinelFile)
		if err != nil {
			return errors.Wrapf(err, "failed to delete sentinel file %s", bootstrapSentinelFile)
		}
	}
	return nil
}

func (r *HostReconciler) deleteEndpointIP(ctx context.Context, byoHost *infrastructurev1beta1.ByoHost) error {
	logger := ctrl.LoggerFrom(ctx)
	logger.Info("Removing network endpoints")
	if IP, ok := byoHost.Annotations[infrastructurev1beta1.EndPointIPAnnotation]; ok {
		return deleteIP(IP, registration.LocalHostRegistrar.ByoHostInfo.DefaultNetworkInterfaceName)
	}
	return nil
}

func (r *HostReconciler) removeAnnotations(ctx context.Context, byoHost *infrastructurev1beta1.ByoHost) {
	logger := ctrl.LoggerFrom(ctx)
	logger.Info("Removing annotations")
	// Remove host reservation
	byoHost.Status.MachineRef = nil

	// Remove BootstrapSecret
	byoHost.Spec.BootstrapSecret = nil

	// Remove UninstallationSecret reference (secret deletion is handled manager-side)
	//
	// FIXME: Currently we cleanup uninstallation secret from the management plane controller.
	// This means we have split-ownership of cleanup between the agent and the management plane controller.
	// We should ensure agent's boundary for cleanup remains within the host itself and it should not be modifying the ByoHost CR at all. The management of the ByoHost CR is the management plane's responsibility, until the host decides to "deboard". And even then it may only ask the management plane to initiate the cleanup of the CR, but not do so itself.
	// byoHost.Spec.UninstallationSecret = nil

	// Remove cluster-name label
	delete(byoHost.Labels, clusterv1.ClusterNameLabel)

	// Remove Byomachine-name label
	delete(byoHost.Labels, infrastructurev1beta1.AttachedByoMachineLabel)

	// Remove the EndPointIP annotation
	delete(byoHost.Annotations, infrastructurev1beta1.EndPointIPAnnotation)

	// Remove the cleanup annotation
	delete(byoHost.Annotations, infrastructurev1beta1.HostCleanupAnnotation)

	// Remove the cluster version annotation
	delete(byoHost.Annotations, infrastructurev1beta1.K8sVersionAnnotation)

	// Remove the bundle registry annotation
	delete(byoHost.Annotations, infrastructurev1beta1.BundleLookupBaseRegistryAnnotation)
}
