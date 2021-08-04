package controllers

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/apis/infrastructure/v1alpha4"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/common"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	"sigs.k8s.io/cluster-api/util/conditions"
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

		It("Should not attempt to reconcile when byomachine namespace does not exist", func() {
			byoMachineLookupkey := types.NamespacedName{Name: defaultByoMachineName, Namespace: namespace}
			request := reconcile.Request{NamespacedName: byoMachineLookupkey}
			_, err := reconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should not attempt to reconcile when byomachine name does not exist", func() {
			byoMachineLookupkey := types.NamespacedName{Name: byoMachineName, Namespace: defaultNamespace}
			request := reconcile.Request{NamespacedName: byoMachineLookupkey}
			_, err := reconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())
		})

	})

	Context("When byohost is not available", func() {
		var (
			ctx        context.Context
			byoMachine *infrastructurev1alpha4.ByoMachine
		)

		BeforeEach(func() {
			ctx = context.Background()
			byoMachine = common.NewByoMachine(defaultByoMachineName, defaultNamespace, defaultClusterName, nil)
			Expect(k8sClient.Create(ctx, byoMachine)).Should(Succeed())
		})

		It("Should return error", func() {
			byoMachineLookupkey := types.NamespacedName{Name: defaultByoMachineName, Namespace: defaultNamespace}
			request := reconcile.Request{NamespacedName: byoMachineLookupkey}
			_, err := reconciler.Reconcile(ctx, request)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("no hosts found"))
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
			byoMachine = common.NewByoMachine(defaultByoMachineName, defaultNamespace, clusterName, nil)
			Expect(k8sClient.Create(ctx, byoMachine)).Should(Succeed())
			byoHost = common.NewByoHost(defaultByoHostName, defaultNamespace, nil)
			Expect(k8sClient.Create(ctx, byoHost)).Should(Succeed())
		})

		It("Should return error", func() {
			byoMachineLookupkey := types.NamespacedName{Name: defaultByoMachineName, Namespace: defaultNamespace}
			request := reconcile.Request{NamespacedName: byoMachineLookupkey}

			_, err := reconciler.Reconcile(ctx, request)
			Expect(err).To(MatchError("clusters.cluster.x-k8s.io \"" + clusterName + "\" not found"))
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
			byoMachine = common.NewByoMachine(defaultByoMachineName, defaultNamespace, defaultClusterName, nil)
			Expect(k8sClient.Create(ctx, byoMachine)).Should(Succeed())
			byoHost = common.NewByoHost(hostname, defaultNamespace, nil)
			Expect(k8sClient.Create(ctx, byoHost)).Should(Succeed())
		})

		It("Should return error", func() {
			byoMachineLookupkey := types.NamespacedName{Name: defaultByoMachineName, Namespace: defaultNamespace}
			request := reconcile.Request{NamespacedName: byoMachineLookupkey}
			_, err := reconciler.Reconcile(ctx, request)
			Expect(err).To(MatchError("nodes \"" + hostname + "\" not found"))
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, byoMachine)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, byoHost)).Should(Succeed())
		})
	})

	Context("When cluster is paused.", func() {
		const (
			hostname = "host-unit-test-3"
			clusterName = "my-cluster-3"
		)

		var (
			ctx        context.Context
			byoMachine *infrastructurev1alpha4.ByoMachine
			byoHost    *infrastructurev1alpha4.ByoHost
			cluster    *clusterv1.Cluster
		)

		BeforeEach(func() {
			ctx = context.Background()
			cluster = common.NewCluster(clusterName, defaultNamespace)
			cluster.Spec.Paused = true
			Expect(k8sClient.Create(ctx, cluster)).Should(Succeed())
			byoMachine = common.NewByoMachine(defaultByoMachineName, defaultNamespace, clusterName, nil)
			Expect(k8sClient.Create(ctx, byoMachine)).Should(Succeed())
			byoHost = common.NewByoHost(hostname, defaultNamespace, nil)
			Expect(k8sClient.Create(ctx, byoHost)).Should(Succeed())
		})

		It("Won't reconcile", func() {
			byoMachineLookupkey := types.NamespacedName{Name: defaultByoMachineName, Namespace: defaultNamespace}
			request := reconcile.Request{NamespacedName: byoMachineLookupkey}
			_, err := reconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())

			createdByoMachine := &infrastructurev1alpha4.ByoMachine{}
			err = k8sClient.Get(context.TODO(), byoMachineLookupkey, createdByoMachine)
			Expect(err).NotTo(HaveOccurred())

			readyCondition := conditions.Get(createdByoMachine, infrastructurev1alpha4.HostReadyCondition)
			Expect(err).NotTo(HaveOccurred())
			Expect(readyCondition).To(BeNil())
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, byoMachine)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, byoHost)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, cluster)).Should(Succeed())
		})
	})

	Context("When ByoMachine is paused.", func() {
		const (
			hostname = "host-unit-test-4"
			clusterName = "my-cluster-4"
		)

		var (
			ctx        context.Context
			byoMachine *infrastructurev1alpha4.ByoMachine
			byoHost    *infrastructurev1alpha4.ByoHost
			cluster    *clusterv1.Cluster
		)

		BeforeEach(func() {
			ctx = context.Background()
			cluster = common.NewCluster(clusterName, defaultNamespace)
			Expect(k8sClient.Create(ctx, cluster)).Should(Succeed())
			byoMachine = common.NewByoMachine(defaultByoMachineName, defaultNamespace, clusterName, nil)
			byoMachine.ObjectMeta.Annotations = map[string]string{}
			byoMachine.ObjectMeta.Annotations[clusterv1.PausedAnnotation] = "paused"
			Expect(k8sClient.Create(ctx, byoMachine)).Should(Succeed())
			byoHost = common.NewByoHost(hostname, defaultNamespace, nil)
			Expect(k8sClient.Create(ctx, byoHost)).Should(Succeed())
		})

		It("Won't reconcile", func() {
			byoMachineLookupkey := types.NamespacedName{Name: defaultByoMachineName, Namespace: defaultNamespace}
			request := reconcile.Request{NamespacedName: byoMachineLookupkey}
			_, err := reconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())

			createdByoMachine := &infrastructurev1alpha4.ByoMachine{}
			err = k8sClient.Get(context.TODO(), byoMachineLookupkey, createdByoMachine)
			Expect(err).NotTo(HaveOccurred())

			readyCondition := conditions.Get(createdByoMachine, infrastructurev1alpha4.HostReadyCondition)
			Expect(err).NotTo(HaveOccurred())
			Expect(readyCondition).To(BeNil())
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, byoMachine)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, byoHost)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, cluster)).Should(Succeed())
		})
	})

})
