// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package reconciler

import (
	"context"
	"fmt"
	"os"

	"github.com/cohesity/cluster-api-provider-bringyourownhost/agent/cloudinit"
	"github.com/cohesity/cluster-api-provider-bringyourownhost/agent/registration"
	"github.com/cohesity/cluster-api-provider-bringyourownhost/common"
	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	infrastructurev1beta1 "github.com/cohesity/cluster-api-provider-bringyourownhost/api/infrastructure/v1beta1"
	byohruntime "github.com/cohesity/cluster-api-provider-bringyourownhost/util/runtime"
	"github.com/kube-vip/kube-vip/pkg/vip"
)

// HostReconciler encapsulates the data/logic needed to reconcile a ByoHost
type HostReconciler struct {
	Client              client.Client
	CmdRunner           cloudinit.ICmdRunner
	FileWriter          cloudinit.IFileWriter
	TemplateParser      cloudinit.ITemplateParser
	Recorder            record.EventRecorder
	ContainerRuntime    byohruntime.ContainerRuntime
	DownloadPath        string
	SkipK8sInstallation bool
}

const (
	bootstrapSentinelFile = "/run/cluster-api/bootstrap-success.complete"
	// KubeadmResetCommand is the command to run to force reset/remove nodes' local file system of the files created by kubeadm
	KubeadmResetCommand = "kubeadm reset --force --v=5"

	// Kubelet cleanup commands
	removeKubeletBinaryCmd = "rm -f /usr/bin/kubelet"
	stopKubeletServiceCmd  = "systemctl stop kubelet || true"
)

// errContainerRuntimeNil is returned when container runtime is nil
var errContainerRuntimeNil = errors.New("container runtime is nil")

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
	logger = logger.WithValues("ByoHost", byoHost.Name)
	logger.Info("reconcile normal")
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

		err = r.bootstrapK8sNode(ctx, bootstrapScript)
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
	secret := &corev1.Secret{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: byoHost.Spec.InstallationSecret.Name, Namespace: byoHost.Spec.InstallationSecret.Namespace}, secret)
	if err != nil {
		logger.Error(err, "error getting install and uninstall script")
		r.Recorder.Eventf(byoHost, corev1.EventTypeWarning, "ReadInstallationSecretFailed", "install and uninstall script %s not found", byoHost.Spec.InstallationSecret.Name)
		return err
	}
	installScript := string(secret.Data["install"])
	uninstallScript := string(secret.Data["uninstall"])

	byoHost.Spec.UninstallationScript = &uninstallScript
	installScript, err = r.parseScript(ctx, installScript)
	if err != nil {
		return err
	}
	logger.Info("executing install script on byohost", "name", byoHost.Name)
	err = r.CmdRunner.RunCmd(ctx, installScript)
	if err != nil {
		logger.Error(err, "error executing installation script")
		r.Recorder.Event(byoHost, corev1.EventTypeWarning, "InstallScriptExecutionFailed", "install script execution failed")
		conditions.MarkFalse(byoHost, infrastructurev1beta1.K8sComponentsInstallationSucceeded, infrastructurev1beta1.K8sComponentsInstallationFailedReason, clusterv1.ConditionSeverityInfo, "")
		return err
	}
	logger.Info("Successfully executed install script on byohost", "name", byoHost.Name)
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
		WithEventFilter(predicates.ResourceNotPaused(mgr.GetScheme(), ctrl.LoggerFrom(ctx))).
		Complete(r)
}

// cleanup /run/kubeadm dirs to remove any stale config on the host
func (r *HostReconciler) cleank8sdirectories(ctx context.Context) error {
	logger := ctrl.LoggerFrom(ctx)

	dirs := []string{
		"/run/kubeadm/*",
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

	k8sComponentsInstallationSucceeded := conditions.Get(byoHost, infrastructurev1beta1.K8sComponentsInstallationSucceeded)
	if k8sComponentsInstallationSucceeded != nil && k8sComponentsInstallationSucceeded.Status == corev1.ConditionTrue {
		err := r.resetNode(ctx, byoHost)
		if err != nil {
			return err
		}
		if r.SkipK8sInstallation {
			logger.Info("Skipping uninstallation of k8s components")
		} else {
			if byoHost.Spec.UninstallationScript == nil {
				errC := r.checkAndPopulateUninstallScriptFromInstallSecret(ctx, byoHost)
				if errC != nil {
					return fmt.Errorf("UninstallationScript not found in Byohost %s: %w", byoHost.Name, errC)
				}
			}
			logger.Info("Executing Uninstall script")
			uninstallScript := *byoHost.Spec.UninstallationScript
			uninstallScript, err = r.parseScript(ctx, uninstallScript)
			if err != nil {
				logger.Error(err, "error parsing Uninstallation script")
				return err
			}
			err = r.CmdRunner.RunCmd(ctx, uninstallScript)
			if err != nil {
				logger.Error(err, "error execting Uninstallation script")
				r.Recorder.Event(byoHost, corev1.EventTypeWarning, "UninstallScriptExecutionFailed", "uninstall script execution failed")
				return err
			}
		}
		conditions.MarkFalse(byoHost, infrastructurev1beta1.K8sComponentsInstallationSucceeded, infrastructurev1beta1.K8sNodeAbsentReason, clusterv1.ConditionSeverityInfo, "")
		logger.Info("host removed from the cluster and the uninstall is executed successfully")
	} else {
		logger.Info("Skipping k8s node reset and k8s component uninstallation")
	}
	conditions.MarkFalse(byoHost, infrastructurev1beta1.K8sNodeBootstrapSucceeded, infrastructurev1beta1.K8sNodeAbsentReason, clusterv1.ConditionSeverityInfo, "")

	err := r.removeSentinelFile(ctx)
	if err != nil {
		return err
	}

	err = r.deleteEndpointIP(ctx, byoHost)
	if err != nil {
		return err
	}

	byoHost.Spec.InstallationSecret = nil
	byoHost.Spec.UninstallationScript = nil
	r.removeAnnotations(ctx, byoHost)
	conditions.MarkFalse(byoHost, infrastructurev1beta1.K8sNodeBootstrapSucceeded, infrastructurev1beta1.K8sNodeAbsentReason, clusterv1.ConditionSeverityInfo, "")
	return nil
}

func (r *HostReconciler) resetNode(ctx context.Context, byoHost *infrastructurev1beta1.ByoHost) error {
	logger := ctrl.LoggerFrom(ctx)

	logger.Info("Removing containers")
	err := r.removeContainers(ctx)
	if err != nil {
		r.Recorder.Event(byoHost, corev1.EventTypeWarning, "RemoveContainersFailed", "failed to remove containers using crictl")
		return errors.Wrapf(err, "failed to remove containers using crictl")
	}

	logger.Info("Removed containers successfully")

	logger.Info("Running kubeadm reset")

	err = r.CmdRunner.RunCmd(ctx, KubeadmResetCommand)
	if err != nil {
		r.Recorder.Event(byoHost, corev1.EventTypeWarning, "ResetK8sNodeFailed", "k8s Node Reset failed")
		return errors.Wrapf(err, "failed to exec kubeadm reset")
	}
	logger.Info("Kubernetes Node reset completed")
	r.Recorder.Event(byoHost, corev1.EventTypeNormal, "ResetK8sNodeSucceeded", "k8s Node Reset completed")
	return nil
}

func (r *HostReconciler) bootstrapK8sNode(ctx context.Context, bootstrapScript string) error {
	logger := ctrl.LoggerFrom(ctx)
	logger.Info("Bootstraping k8s Node")
	err := cloudinit.ScriptExecutor{
		WriteFilesExecutor:    r.FileWriter,
		RunCmdExecutor:        r.CmdRunner,
		ParseTemplateExecutor: r.TemplateParser,
	}.Execute(bootstrapScript)
	if err != nil {
		return fmt.Errorf("failed to execute bootstrap script: %w", err)
	}
	return nil
}

func (r *HostReconciler) removeContainers(ctx context.Context) error {
	// Cleanup kubelet before removing containers
	err := r.cleanupKubelet(ctx)
	if err != nil {
		return fmt.Errorf("failed to cleanup kubelet: %w", err)
	}

	var containerRuntime byohruntime.ContainerRuntime
	if r.ContainerRuntime != nil {
		// Use injected container runtime (for testing)
		containerRuntime = r.ContainerRuntime
	} else {
		// Create container runtime
		criSocketPath, detectErr := byohruntime.DetectCRISocket()
		if detectErr != nil {
			return fmt.Errorf("failed to detect cri socket path: %w", detectErr)
		}

		containerRuntime = byohruntime.NewContainerRuntime(criSocketPath)
	}

	if containerRuntime == nil {
		return fmt.Errorf("container runtime is nil after initialization: %w", errContainerRuntimeNil)
	}

	// Connect to the container runtime
	if err = containerRuntime.Connect(); err != nil {
		return fmt.Errorf("failed to connect to container runtime: %w", err)
	}

	// List containers
	containers, err := containerRuntime.ListKubeContainers()
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	// Remove containers
	err = containerRuntime.RemoveContainers(containers)
	if err != nil {
		return fmt.Errorf("failed to remove containers: %w", err)
	}
	return nil
}

func (r *HostReconciler) removeSentinelFile(ctx context.Context) error {
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
		networks, err := vip.NewConfig(IP, registration.LocalHostRegistrar.ByoHostInfo.DefaultNetworkInterfaceName, false, "", false, 0, unix.RTN_UNICAST, unix.RTN_UNICAST, "", "", "", false, nil)
		if err == nil {
			for _, network := range networks {
				_, err := network.DeleteIP()
				if err != nil {
					return fmt.Errorf("failed to delete endpoint IP: %w", err)
				}
			}
		}
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

// checkAndPopulateUninstallScriptFromInstallSecret populates the uninstall script on
// byohost object if the install secret contains an uninstall script.
func (r *HostReconciler) checkAndPopulateUninstallScriptFromInstallSecret(ctx context.Context, byoHost *infrastructurev1beta1.ByoHost) error {
	logger := ctrl.LoggerFrom(ctx)
	logger.Info("Populating uninstall script from install secret")

	if byoHost.Spec.InstallationSecret == nil {
		return errors.New("installation secret reference is nil")
	}

	secret := &corev1.Secret{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: byoHost.Spec.InstallationSecret.Name, Namespace: byoHost.Spec.InstallationSecret.Namespace}, secret)
	if err != nil {
		r.Recorder.Eventf(byoHost, corev1.EventTypeWarning, "ReadInstallationSecretFailed", "install and uninstall script %s not found", byoHost.Spec.InstallationSecret.Name)
		return fmt.Errorf("failed to get installation secret %s: %w", byoHost.Spec.InstallationSecret.Name, err)
	}

	uninstallScript := string(secret.Data["uninstall"])
	if uninstallScript == "" {
		r.Recorder.Eventf(byoHost, corev1.EventTypeWarning, "UninstallScriptNotFound", "uninstall script not found in install secret %s", byoHost.Spec.InstallationSecret.Name)
		return errors.New("uninstall script not found in installation secret")
	}

	byoHost.Spec.UninstallationScript = &uninstallScript

	return nil
}

// CleanupKubelet removes the kubelet binary and stops the kubelet service
func (r *HostReconciler) cleanupKubelet(ctx context.Context) error {
	logger := ctrl.LoggerFrom(ctx)

	// First remove kubelet binary
	logger.Info("Removing kubelet binary")
	if err := r.CmdRunner.RunCmd(ctx, removeKubeletBinaryCmd); err != nil {
		return fmt.Errorf("failed to remove kubelet binary: %w", err)
	}

	// Then stop kubelet service
	logger.Info("Stopping kubelet service")
	if err := r.CmdRunner.RunCmd(ctx, stopKubeletServiceCmd); err != nil {
		return fmt.Errorf("failed to stop kubelet service: %w", err)
	}

	return nil
}
