// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"context"
	"net/http"

	v1 "k8s.io/api/admission/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

//+kubebuilder:webhook:path=/validate-infrastructure-cluster-x-k8s-io-v1beta1-byohost,mutating=false,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=byohosts,verbs=create;update;delete,versions=v1beta1,name=vbyohost.kb.io,admissionReviewVersions={v1,v1beta1}

// +k8s:deepcopy-gen=false
// ByoHostValidator validates ByoHosts
type ByoHostValidator struct {
	//	Client  client.Client
	decoder *admission.Decoder
}

// nolint: gocritic
// Handle handles all the requests for ByoHost resource
func (v *ByoHostValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	if req.Operation == v1.Delete {
		byoHost := &ByoHost{}
		err := v.decoder.DecodeRaw(req.OldObject, byoHost)
		if err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		if byoHost.Status.MachineRef != nil {
			return admission.Denied("cannot delete ByoHost when MachineRef is assigned")
		}
	}

	// TODO: verify if req.UserInfo.Username has rbac permission to update the byohost

	return admission.Allowed("")
}

// InjectDecoder injects the decoder.
func (v *ByoHostValidator) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}
