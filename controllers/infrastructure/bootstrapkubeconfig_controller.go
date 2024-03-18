// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	b64 "encoding/base64"
	"time"

	infrastructurev1beta1 "github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/apis/infrastructure/v1beta1"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/common/bootstraptoken"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	clientcmdlatest "k8s.io/client-go/tools/clientcmd/api/latest"
	bootstraputil "k8s.io/cluster-bootstrap/token/util"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// BootstrapKubeconfigReconciler reconciles a BootstrapKubeconfig object
type BootstrapKubeconfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	// ttl is the time to live for the generated bootstrap token
	ttl = time.Hour * 12
)

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=bootstrapkubeconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=bootstrapkubeconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=bootstrapkubeconfigs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *BootstrapKubeconfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconcile request received")

	// Fetch the BootstrapKubeconfig instance
	bootstrapKubeconfig := &infrastructurev1beta1.BootstrapKubeconfig{}
	err := r.Client.Get(ctx, req.NamespacedName, bootstrapKubeconfig)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// There already is bootstrap-kubeconfig data associated with this object
	// Do not create secrets again
	if bootstrapKubeconfig.Status.BootstrapKubeconfigData != nil {
		return ctrl.Result{}, nil
	}

	tokenStr, err := bootstraputil.GenerateBootstrapToken()
	if err != nil {
		return ctrl.Result{}, err
	}

	bootstrapKubeconfigSecret, err := bootstraptoken.GenerateSecretFromBootstrapToken(tokenStr, ttl)
	if err != nil {
		return ctrl.Result{}, err
	}

	// create secret
	err = r.Client.Create(ctx, bootstrapKubeconfigSecret)
	if err != nil {
		return ctrl.Result{}, err
	}

	bootstrapKubeconfigData, err := bootstraptoken.GenerateBootstrapKubeconfigFromBootstrapToken(tokenStr, bootstrapKubeconfig)
	if err != nil {
		return ctrl.Result{}, err
	}

	caData := bootstrapKubeconfigData.Clusters[infrastructurev1beta1.DefaultClusterName].CertificateAuthorityData
	decodedCAData, err := b64.StdEncoding.DecodeString(string(caData))
	if err != nil {
		return ctrl.Result{}, err
	}

	bootstrapKubeconfigData.Clusters[infrastructurev1beta1.DefaultClusterName].CertificateAuthorityData = decodedCAData
	runtimeEncodedBootstrapKubeConfig, err := runtime.Encode(clientcmdlatest.Codec, bootstrapKubeconfigData)
	if err != nil {
		return ctrl.Result{}, err
	}

	helper, err := patch.NewHelper(bootstrapKubeconfig, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}

	bootstrapKubeconfigDataStr := string(runtimeEncodedBootstrapKubeConfig)
	bootstrapKubeconfig.Status.BootstrapKubeconfigData = &bootstrapKubeconfigDataStr

	return ctrl.Result{}, helper.Patch(ctx, bootstrapKubeconfig)
}

// SetupWithManager sets up the controller with the Manager.
func (r *BootstrapKubeconfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1beta1.BootstrapKubeconfig{}).
		Complete(r)
}
