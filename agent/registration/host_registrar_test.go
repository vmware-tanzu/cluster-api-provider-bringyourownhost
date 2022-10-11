// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package registration_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/registration"
	infrastructurev1beta1 "github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/apis/infrastructure/v1beta1"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/test/builder"
)

var _ = Describe("Host Registrar Tests", func() {
	var (
		hr               registration.HostRegistrar
		byoHost          *infrastructurev1beta1.ByoHost
		defaultNamespace = "default"
		ctx              = context.TODO()
	)

	BeforeEach(func() {
		hr = registration.HostRegistrar{K8sClient: k8sClient}
		byoHost = builder.ByoHost(defaultNamespace, "host").Build()
		Expect(k8sClient.Create(ctx, byoHost)).Should(Succeed())
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(ctx, byoHost)).ToNot(HaveOccurred())
	})

	Context("When a ByoHost exists and registration is done", func() {
		It("Should update the host details on the byohost successfully", func() {
			Expect(hr.UpdateHost(ctx, byoHost)).ToNot(HaveOccurred())
		})
	})
})
