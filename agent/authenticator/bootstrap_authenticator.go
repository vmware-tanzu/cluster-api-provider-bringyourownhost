// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package authenticator

import (
	"context"
	"crypto/rsa"

	certv1 "k8s.io/api/certificates/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// BootstrapAuthenticator encapsulates the data/logic needed to reconcile a hostCSR
type BootstrapAuthenticator struct {
	Client     client.Client
	HostName   string
	PrivateKey *rsa.PrivateKey
}

// Reconcile handles events for the host CSR that is registered by this agent process
func (r *BootstrapAuthenticator) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	logger := ctrl.LoggerFrom(ctx)
	logger.Info("Reconcile request received", "resource", req.NamespacedName)

	// Fetch the host CSR instance
	hostCSR := &certv1.CertificateSigningRequest{}
	err := r.Client.Get(ctx, req.NamespacedName, hostCSR)
	if err != nil {
		logger.Error(err, "error getting host CSR")
		return ctrl.Result{}, err
	}

	// TODO: workflow for approved CSR

	// TODO: workflow for rejected CSR

	// return if not approved or denied
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the manager
func (r *BootstrapAuthenticator) SetupWithManager(ctx context.Context, mgr manager.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&certv1.CertificateSigningRequest{}).WithEventFilter(
		// watch only own created CSR
		predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				return e.Object.GetName() == r.HostName
			},
			UpdateFunc: func(e event.UpdateEvent) bool {
				return e.ObjectOld.GetName() == r.HostName
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				return e.Object.GetName() == r.HostName
			}}).
		Complete(r)
}
