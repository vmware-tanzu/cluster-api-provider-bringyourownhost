// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"strings"

	certv1 "k8s.io/api/certificates/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ByoAdmissionReconciler reconciles a ByoAdmission object
type ByoAdmissionReconciler struct {
	ClientSet clientset.Interface
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=byohosts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=byohosts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=byohosts/finalizers,verbs=update
//+kubebuilder:rbac:groups=certificates.k8s.io,resources=certificatesigningrequests,verbs=create;get;watch

// Reconcile continuosuly checks for CSRs and approves them
func (r *ByoAdmissionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var err error
	logger := log.FromContext(ctx)
	logger.Info("Reconcile request received", "object", req.NamespacedName)

	// Fetch the CSR from the api-server.
	csr, err := r.ClientSet.CertificatesV1().CertificateSigningRequests().Get(ctx, req.NamespacedName.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Error(err, "CertificateSigningRequest not found, won't reconcile")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Update the CSR.
	csr.Status.Conditions = append(csr.Status.Conditions, certv1.CertificateSigningRequestCondition{
		Type:   certv1.CertificateApproved,
		Reason: "Approved by ByoAdmission Controller",
	})

	// Approver the CSR.
	logger.Info("Approving CSR", "object", req.NamespacedName)
	_, err = r.ClientSet.CertificatesV1().CertificateSigningRequests().UpdateApproval(ctx, csr.Name, csr, metav1.UpdateOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}

	logger.Info("CSR Approved", "object", req.NamespacedName)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ByoAdmissionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&certv1.CertificateSigningRequest{}).WithEventFilter(
		// watch only BYOH created CSRs
		predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				return strings.HasPrefix(e.Object.GetName(), "byoh-csr")
			},
			UpdateFunc: func(e event.UpdateEvent) bool {
				return strings.HasPrefix(e.ObjectOld.GetName(), "byoh-csr")
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				return strings.HasPrefix(e.Object.GetName(), "byoh-csr")
			}}).
		Complete(r)
}
