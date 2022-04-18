// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"context"
	"fmt"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/validate-v1-boootstrap-token,mutating=false,failurePolicy=fail,groups="",sideEffects=none,admissionReviewVersions=v1,resources=secrets,verbs=create,versions=v1,name=vsecret.kb.io

// bootstrapTokenValidator validates Pods
type BootstrapTokenValidator struct {
	Client  client.Client
	decoder *admission.Decoder
}

// BootstrapTokenValidator admits a secret if it is of a specific format and namespace.
func (v *BootstrapTokenValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	secret := &corev1.Secret{}

	err := v.decoder.Decode(req, secret)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if secret.Namespace != "kube-system" {
		return admission.Denied(fmt.Sprintf("boostrap secrets can only be created in kube-system namespace and not %s", secret.Namespace))
	}

	return admission.Allowed("")
}

// BootstrapTokenValidator implements admission.DecoderInjector.
// A decoder will be automatically injected.

// InjectDecoder injects the decoder.
func (v *BootstrapTokenValidator) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}
