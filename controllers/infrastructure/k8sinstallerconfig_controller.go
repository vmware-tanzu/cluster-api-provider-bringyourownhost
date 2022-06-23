// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	infrav1 "github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/apis/infrastructure/v1beta1"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/installer"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// K8sInstallerConfigReconciler reconciles a K8sInstallerConfig object
type K8sInstallerConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// k8sInstallerConfigScope defines a scope defined around a K8sInstallerConfig and its ByoMachine
type k8sInstallerConfigScope struct {
	Client     client.Client
	Logger     logr.Logger
	Cluster    *clusterv1.Cluster
	ByoMachine *infrav1.ByoMachine
	Config     *infrav1.K8sInstallerConfig
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=k8sinstallerconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=k8sinstallerconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=k8sinstallerconfigs/finalizers,verbs=update
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=byomachines,verbs=get;list;watch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=byomachines/status,verbs=get
// +kubebuilder:rbac:groups="",resources=secrets;events,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *K8sInstallerConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconcile request received")

	// Fetch the K8sInstallerConfig instance
	config := &infrav1.K8sInstallerConfig{}
	err := r.Client.Get(ctx, req.NamespacedName, config)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get K8sInstallerConfig")
		return ctrl.Result{}, err
	}

	// Create the K8sInstallerConfig scope
	scope := &k8sInstallerConfigScope{
		Client: r.Client,
		Logger: logger.WithValues("k8sinstallerconfig", config.Name),
		Config: config,
	}

	// Fetch the ByoMachine
	byoMachine, err := GetOwnerByoMachine(ctx, r.Client, &config.ObjectMeta)
	if err != nil && !apierrors.IsNotFound(err) {
		logger.Error(err, "failed to get Owner ByoMachine")
		return ctrl.Result{}, err
	}

	helper, err := patch.NewHelper(config, r.Client)
	if err != nil {
		logger.Error(err, "unable to create helper")
		return ctrl.Result{}, err
	}
	defer func() {
		if err = helper.Patch(ctx, config); err != nil && reterr == nil {
			logger.Error(err, "failed to patch K8sInstallerConfig")
			reterr = err
		}
	}()

	// Add finalizer first if not exist
	if !controllerutil.ContainsFinalizer(scope.Config, infrav1.K8sInstallerConfigFinalizer) {
		controllerutil.AddFinalizer(scope.Config, infrav1.K8sInstallerConfigFinalizer)
	}

	// Handle deleted K8sInstallerConfig
	if !config.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, scope)
	}

	if byoMachine == nil {
		logger.Info("Waiting for ByoMachine Controller to set OwnerRef on InstallerConfig")
		return ctrl.Result{}, nil
	}
	scope.ByoMachine = byoMachine
	logger = logger.WithValues("byoMachine", byoMachine.Name, "namespace", byoMachine.Namespace)
	logger.Info("byoMachine found")

	// Fetch the Cluster
	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, byoMachine.ObjectMeta)
	if err != nil {
		logger.Error(err, "ByoMachine owner Machine is missing cluster label or cluster does not exist")
		return ctrl.Result{}, err
	}
	logger = logger.WithValues("cluster", cluster.Name)
	scope.Cluster = cluster
	scope.Logger = logger

	if annotations.IsPaused(cluster, config) {
		logger.Info("Reconciliation is paused for this object")
		return ctrl.Result{}, nil
	}

	switch {
	// waiting for ByoMachine to updating it's ByoHostReady condition to false for reason InstallationSecretNotAvailableReason
	case conditions.GetReason(byoMachine, infrav1.BYOHostReady) != infrav1.InstallationSecretNotAvailableReason:
		logger.Info("ByoMachine is not waiting for InstallationSecret", "reason", conditions.GetReason(byoMachine, infrav1.BYOHostReady))
		return ctrl.Result{}, nil
	// Status is ready means a config has been generated.
	case config.Status.Ready:
		logger.Info("K8sInstallerConfig is ready")
		return ctrl.Result{}, nil
	}

	return r.reconcileNormal(ctx, scope)
}

func (r *K8sInstallerConfigReconciler) reconcileNormal(ctx context.Context, scope *k8sInstallerConfigScope) (reconcile.Result, error) {
	logger := scope.Logger
	logger.Info("Reconciling K8sInstallerConfig")

	k8sVersion := scope.Config.GetAnnotations()[infrav1.K8sVersionAnnotation]
	downloader := installer.NewBundleDownloader(scope.Config.Spec.BundleType, scope.Config.Spec.BundleRepo, "{{.BUNDLE_DOWNLOAD_PATH}}", logger)
	installerObj, err := installer.NewInstaller(ctx, scope.ByoMachine.Status.HostInfo.OSImage, scope.ByoMachine.Status.HostInfo.Architecture, k8sVersion, downloader)
	if err != nil {
		logger.Error(err, "failed to create installer instance", "osImage", scope.ByoMachine.Status.HostInfo.OSImage, "architecture", scope.ByoMachine.Status.HostInfo.Architecture, "k8sVersion", k8sVersion)
		return ctrl.Result{}, err
	}

	// creating installation secret
	if err := r.storeInstallationData(ctx, scope, installerObj.Install(), installerObj.Uninstall()); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// storeInstallationData creates a new secret with the install and unstall data passed in as input,
// sets the reference in the configuration status and ready to true.
func (r *K8sInstallerConfigReconciler) storeInstallationData(ctx context.Context, scope *k8sInstallerConfigScope, install, uninstall string) error {
	logger := scope.Logger
	logger.Info("creating installation secret")

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      scope.Config.Name,
			Namespace: scope.Config.Namespace,
			Labels: map[string]string{
				clusterv1.ClusterLabelName: scope.Cluster.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: infrav1.GroupVersion.String(),
					Kind:       scope.Config.Kind,
					Name:       scope.Config.Name,
					UID:        scope.Config.UID,
					Controller: pointer.BoolPtr(true),
				},
			},
		},
		Data: map[string][]byte{
			"install":   []byte(install),
			"uninstall": []byte(uninstall),
		},
		Type: clusterv1.ClusterSecretType,
	}

	// as secret creation and scope.Config status patch are not atomic operations
	// it is possible that secret creation happens but the config.Status patches are not applied
	if err := r.Client.Create(ctx, secret); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return errors.Wrapf(err, "failed to create installation secret for K8sInstallerConfig %s/%s", scope.Config.Namespace, scope.Config.Name)
		}
		logger.Info("installation secret for K8sInstallerConfig already exists, updating", "secret", secret.Name, "K8sInstallerConfig", scope.Config.Name)
		if err := r.Client.Update(ctx, secret); err != nil {
			return errors.Wrapf(err, "failed to update installation secret for K8sInstallerConfig %s/%s", scope.Config.Namespace, scope.Config.Name)
		}
	}
	scope.Config.Status.InstallationSecret = &corev1.ObjectReference{
		Kind:      secret.Kind,
		Namespace: secret.Namespace,
		Name:      secret.Name,
	}
	scope.Config.Status.Ready = true
	logger.Info("created installation secret")
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *K8sInstallerConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.K8sInstallerConfig{}).
		Watches(
			&source.Kind{Type: &infrav1.ByoMachine{}},
			handler.EnqueueRequestsFromMapFunc(r.ByoMachineToK8sInstallerConfigMapFunc),
		).
		Complete(r)
}

// ByoMachineToK8sInstallerConfigMapFunc is a handler.ToRequestsFunc to be used to enqeue
// request for reconciliation of K8sInstallerConfig.
func (r *K8sInstallerConfigReconciler) ByoMachineToK8sInstallerConfigMapFunc(o client.Object) []ctrl.Request {
	ctx := context.TODO()
	logger := log.FromContext(ctx)

	m, ok := o.(*infrav1.ByoMachine)
	if !ok {
		panic(fmt.Sprintf("Expected a ByoMachine but got a %T", o))
	}
	m.GetObjectKind().SetGroupVersionKind(infrav1.GroupVersion.WithKind("ByoMachine"))

	result := []ctrl.Request{}
	if m.Spec.InstallerRef != nil && m.Spec.InstallerRef.GroupVersionKind() == infrav1.GroupVersion.WithKind("K8sInstallerConfigTemplate") {
		configList := &infrav1.K8sInstallerConfigList{}
		if err := r.Client.List(ctx, configList, client.InNamespace(m.Namespace)); err != nil {
			logger.Error(err, "failed to list K8sInstallerConfig")
			return result
		}
		for idx := range configList.Items {
			config := &configList.Items[idx]
			if hasOwnerReferenceFrom(config, m) {
				name := client.ObjectKey{Namespace: config.Namespace, Name: config.Name}
				result = append(result, ctrl.Request{NamespacedName: name})
			}
		}
	}
	return result
}

func (r *K8sInstallerConfigReconciler) reconcileDelete(ctx context.Context, scope *k8sInstallerConfigScope) (reconcile.Result, error) {
	logger := scope.Logger
	logger.Info("Deleting K8sInstallerConfig")
	controllerutil.RemoveFinalizer(scope.Config, infrav1.K8sInstallerConfigFinalizer)
	return reconcile.Result{}, nil
}

// GetOwnerByoMachine returns the ByoMachine object owning the current resource.
func GetOwnerByoMachine(ctx context.Context, c client.Client, obj *metav1.ObjectMeta) (*infrav1.ByoMachine, error) {
	for _, ref := range obj.OwnerReferences {
		gv, err := schema.ParseGroupVersion(ref.APIVersion)
		if err != nil {
			return nil, err
		}
		if ref.Kind == "ByoMachine" && gv.Group == infrav1.GroupVersion.Group {
			return GetByoMachineByName(ctx, c, obj.Namespace, ref.Name)
		}
	}
	return nil, nil
}

// GetByoMachineByName finds and return a ByoMachine object using the specified params.
func GetByoMachineByName(ctx context.Context, c client.Client, namespace, name string) (*infrav1.ByoMachine, error) {
	m := &infrav1.ByoMachine{}
	key := client.ObjectKey{Name: name, Namespace: namespace}
	if err := c.Get(ctx, key, m); err != nil {
		return nil, err
	}
	return m, nil
}

// hasOwnerReferenceFrom will check if object have owner reference of the given owner
func hasOwnerReferenceFrom(obj, owner client.Object) bool {
	for _, o := range obj.GetOwnerReferences() {
		if o.Kind == owner.GetObjectKind().GroupVersionKind().Kind && o.Name == owner.GetName() {
			return true
		}
	}
	return false
}
