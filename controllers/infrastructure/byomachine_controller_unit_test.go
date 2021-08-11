package controllers

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/apis/infrastructure/v1alpha4"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/common"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Controllers/ByomachineController/Unitests", func() {

	Context("When byomachine is not found", func() {
		var (
			ctx context.Context
		)
		const (
			namespace      = "fakeNameSpaceWithoutByomachine"
			byoMachineName = "fakeMmachineWithoutByomachine"
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
		const (
			machineName = "machineWhenByohostIsNotAvailable"
		)
		var (
			ctx        context.Context
			byoMachine *infrastructurev1alpha4.ByoMachine
			machine    *clusterv1.Machine
		)

		BeforeEach(func() {
			ctx = context.Background()
			machine = common.NewMachine(machineName, defaultNamespace, defaultClusterName)
			Expect(k8sClient.Create(ctx, machine)).Should(Succeed())

			byoMachine = common.NewByoMachine(defaultByoMachineName, defaultNamespace, defaultClusterName, machine)
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
			Expect(k8sClient.Delete(ctx, machine)).Should(Succeed())
		})
	})

	Context("When cluster does not exist", func() {
		const (
			clusterName = "fakeClusterWithoutCluster"
			machineName = "machineWithoutCluster"
		)
		var (
			ctx        context.Context
			byoMachine *infrastructurev1alpha4.ByoMachine
			byoHost    *infrastructurev1alpha4.ByoHost
			machine    *clusterv1.Machine
		)

		BeforeEach(func() {
			ctx = context.Background()
			machine = common.NewMachine(machineName, defaultNamespace, defaultClusterName)
			Expect(k8sClient.Create(ctx, machine)).Should(Succeed())

			byoMachine = common.NewByoMachine(defaultByoMachineName, defaultNamespace, clusterName, machine)
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
			Expect(k8sClient.Delete(ctx, machine)).Should(Succeed())
		})
	})

	Context("When node is not available", func() {
		//Reconcile assumes the node name equal to host name, or setNodeProviderID will be failed.
		//We only have one node "host-unit-test" in testEnv, not "hostWhenNodeIsNotAvailable"
		const (
			hostname    = "hostWhenNodeIsNotAvailable"
			machineName = "machineWhenNodeIsNotAvailable"
		)

		var (
			ctx        context.Context
			byoMachine *infrastructurev1alpha4.ByoMachine
			byoHost    *infrastructurev1alpha4.ByoHost
			machine    *clusterv1.Machine
		)

		BeforeEach(func() {
			ctx = context.Background()
			machine = common.NewMachine(defaultMachineName, defaultNamespace, defaultClusterName)
			machine.Spec.Bootstrap = clusterv1.Bootstrap{
				DataSecretName: &fakeBootstrapSecret,
			}
			Expect(k8sClient.Create(ctx, machine)).Should(Succeed())

			byoMachine = common.NewByoMachine(defaultByoMachineName, defaultNamespace, defaultClusterName, machine)
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
			Expect(k8sClient.Delete(ctx, machine)).Should(Succeed())
		})
	})

	Context("When cluster is paused.", func() {
		const (
			hostname    = "hostWhenClusterIsPaused"
			clusterName = "clusterWhenClusterIsPaused"
			machineName = "machineWhenClusterIsPaused"
		)

		var (
			ctx        context.Context
			byoMachine *infrastructurev1alpha4.ByoMachine
			byoHost    *infrastructurev1alpha4.ByoHost
			cluster    *clusterv1.Cluster
			machine    *clusterv1.Machine
		)

		BeforeEach(func() {
			ctx = context.Background()
			cluster = common.NewCluster(clusterName, defaultNamespace)
			cluster.Spec.Paused = true
			Expect(k8sClient.Create(ctx, cluster)).Should(Succeed())

			machine = common.NewMachine(defaultMachineName, defaultNamespace, defaultClusterName)
			Expect(k8sClient.Create(ctx, machine)).Should(Succeed())

			byoMachine = common.NewByoMachine(defaultByoMachineName, defaultNamespace, clusterName, machine)
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
			Expect(k8sClient.Delete(ctx, machine)).Should(Succeed())
		})
	})

	Context("When ByoMachine is paused.", func() {
		const (
			hostname    = "hostWhenByoMachineIsPaused"
			clusterName = "clusterWhenByoMachineIsPaused"
			machineName = "machineWhenByoMachineIsPaused"
		)

		var (
			ctx        context.Context
			byoMachine *infrastructurev1alpha4.ByoMachine
			byoHost    *infrastructurev1alpha4.ByoHost
			cluster    *clusterv1.Cluster
			machine    *clusterv1.Machine
		)

		BeforeEach(func() {
			ctx = context.Background()

			cluster = common.NewCluster(clusterName, defaultNamespace)
			Expect(k8sClient.Create(ctx, cluster)).Should(Succeed())

			machine = common.NewMachine(defaultMachineName, defaultNamespace, defaultClusterName)
			Expect(k8sClient.Create(ctx, machine)).Should(Succeed())

			byoMachine = common.NewByoMachine(defaultByoMachineName, defaultNamespace, clusterName, nil)
			desired := map[string]string{
				clusterv1.PausedAnnotation: "paused",
			}
			annotations.AddAnnotations(byoMachine, desired)
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
			Expect(k8sClient.Delete(ctx, machine)).Should(Succeed())
		})
	})

})
