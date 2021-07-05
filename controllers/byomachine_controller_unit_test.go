package controllers

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/api/v1alpha4"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Controllers/ByomachineController/Unitests", func() {

	Context("When byomachine is not found", func() {
		var (
			ctx context.Context
		)
		const (
			namespace      = "fake-name-space-unit-test"
			byoMachineName = "fake-machine-unit-test"
		)

		BeforeEach(func() {
			ctx = context.Background()
		})

		It("Should not error out when byomachine namespace does not exist", func() {
			byoMachineLookupkey := types.NamespacedName{Name: defaultByoMachineName, Namespace: namespace}
			request := reconcile.Request{NamespacedName: byoMachineLookupkey}
			_, err := reconciler.Reconcile(ctx, request)
			Expect(err).To(BeNil())
		})

		It("Should not error out when byomachine name does not exist", func() {
			byoMachineLookupkey := types.NamespacedName{Name: byoMachineName, Namespace: defaultNamespace}
			request := reconcile.Request{NamespacedName: byoMachineLookupkey}
			_, err := reconciler.Reconcile(ctx, request)
			Expect(err).To(BeNil())
		})

	})

	Context("When byohost is not available", func() {
		var (
			ctx        context.Context
			byoMachine *infrastructurev1alpha4.ByoMachine
		)

		BeforeEach(func() {
			ctx = context.Background()
			byoMachine = createByoMachine(defaultByoMachineName, defaultNamespace, defaultClusterName)
			Expect(k8sClient.Create(ctx, byoMachine)).Should(Succeed())
		})

		It("Should return error", func() {
			byoMachineLookupkey := types.NamespacedName{Name: defaultByoMachineName, Namespace: defaultNamespace}
			request := reconcile.Request{NamespacedName: byoMachineLookupkey}
			_, err := reconciler.Reconcile(ctx, request)
			Expect(err.Error()).To(ContainSubstring("no hosts found"))
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, byoMachine)).Should(Succeed())
		})
	})

	Context("When cluster does not exist", func() {
		const (
			clusterName = "fake-cluster-unit-test"
		)
		var (
			ctx        context.Context
			byoMachine *infrastructurev1alpha4.ByoMachine
			byoHost    *infrastructurev1alpha4.ByoHost
		)

		BeforeEach(func() {
			ctx = context.Background()
			byoMachine = createByoMachine(defaultByoMachineName, defaultNamespace, clusterName)
			Expect(k8sClient.Create(ctx, byoMachine)).Should(Succeed())
			byoHost = createByoHost(defaultByoHostName, defaultNamespace)
			Expect(k8sClient.Create(ctx, byoHost)).Should(Succeed())
		})

		It("Should return error", func() {
			byoMachineLookupkey := types.NamespacedName{Name: defaultByoMachineName, Namespace: defaultNamespace}
			request := reconcile.Request{NamespacedName: byoMachineLookupkey}

			_, err := reconciler.Reconcile(ctx, request)
			Expect(err).To(HaveOccurred())
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, byoMachine)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, byoHost)).Should(Succeed())
		})
	})

	Context("When node is not available", func() {
		//Reconcile assumes the node name equal to host name, or setNodeProviderID will be failed.
		//We only have one node "host-unit-test" in testEnv, not "host-unit-test-2"
		const (
			hostname = "host-unit-test-2"
		)

		var (
			ctx        context.Context
			byoMachine *infrastructurev1alpha4.ByoMachine
			byoHost    *infrastructurev1alpha4.ByoHost
		)

		BeforeEach(func() {
			ctx = context.Background()
			byoMachine = createByoMachine(defaultByoMachineName, defaultNamespace, defaultClusterName)
			Expect(k8sClient.Create(ctx, byoMachine)).Should(Succeed())
			byoHost = createByoHost(hostname, defaultNamespace)
			Expect(k8sClient.Create(ctx, byoHost)).Should(Succeed())
		})

		It("Should return error", func() {
			byoMachineLookupkey := types.NamespacedName{Name: defaultByoMachineName, Namespace: defaultNamespace}
			request := reconcile.Request{NamespacedName: byoMachineLookupkey}
			_, err := reconciler.Reconcile(ctx, request)
			Expect(err).To(HaveOccurred())
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, byoMachine)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, byoHost)).Should(Succeed())
		})
	})

})
