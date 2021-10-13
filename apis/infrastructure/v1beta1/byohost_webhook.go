// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package
var byohostlog = logf.Log.WithName("byohost-resource")

func (h *ByoHost) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(h).
		Complete()
}

//+kubebuilder:webhook:path=/validate-infrastructure-cluster-x-k8s-io-v1beta1-byohost,mutating=false,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=byohosts,verbs=create;update;delete,versions=v1beta1,name=vbyohost.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &ByoHost{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (h *ByoHost) ValidateCreate() error {
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (h *ByoHost) ValidateUpdate(old runtime.Object) error {
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (h *ByoHost) ValidateDelete() error {
	byohostlog.Info("validate delete", "name", h.Name)
	groupResource := schema.GroupResource{Group: "infrastructure.cluster.x-k8s.io", Resource: "byohost"}

	if h.Status.MachineRef != nil {
		return apierrors.NewForbidden(groupResource, h.Name, errors.New("cannot delete ByoHost when MachineRef is assigned"))
	}

	return nil
}
