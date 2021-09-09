package controllers

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/apis/infrastructure/v1alpha4"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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

	Context("When cluster does not exist", func() {
		const (
			clusterName = "fakeClusterWithoutCluster"
			// a DNS-1123 subdomain must consist of lower case alphanumeric characters, '-' or '.', and must start and end with an alphanumeric character (e.g. 'example.com', regex used for validation is '[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*')
			machineName = "machine-without-cluster"
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

			byoHost = common.NewByoHost(defaultByoHostName, defaultNamespace)
			Expect(k8sClient.Create(ctx, byoHost)).Should(Succeed())
		})

		It("Should return error", func() {
			byoMachineLookupkey := types.NamespacedName{Name: byoMachine.Name, Namespace: defaultNamespace}
			request := reconcile.Request{NamespacedName: byoMachineLookupkey}

			_, err := reconciler.Reconcile(ctx, request)
			Expect(err).To(MatchError("Cluster.cluster.x-k8s.io \"" + clusterName + "\" not found"))
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, byoMachine)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, byoHost)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, machine)).Should(Succeed())
		})
	})

	Context("When node is not available", func() {
		// Reconcile assumes the node name equal to host name, or setNodeProviderID will be failed.
		// We only have one node "host-unit-test" in testEnv, not "hostWhenNodeIsNotAvailable"
		const (
			// a DNS-1123 subdomain must consist of lower case alphanumeric characters, '-' or '.', and must start and end with an alphanumeric character (e.g. 'example.com', regex used for validation is '[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*')",
			hostname    = "host-when-node-is-not-available"
			machineName = "machine-when-node-is-not-available"
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
			ph, err := patch.NewHelper(capiCluster, k8sClient)
			Expect(err).ShouldNot(HaveOccurred())
			capiCluster.Status.InfrastructureReady = true
			Expect(ph.Patch(ctx, capiCluster, patch.WithStatusObservedGeneration{})).Should(Succeed())

			byoMachine = common.NewByoMachine(defaultByoMachineName, defaultNamespace, defaultClusterName, machine)
			Expect(k8sClient.Create(ctx, byoMachine)).Should(Succeed())
			byoHost = common.NewByoHost(hostname, defaultNamespace)
			Expect(k8sClient.Create(ctx, byoHost)).Should(Succeed())

			WaitForObjectsToBePopulatedInCache(k8sClient, machine, byoMachine, byoHost)
		})

		It("Should return error", func() {
			byoMachineLookupkey := types.NamespacedName{Name: byoMachine.Name, Namespace: defaultNamespace}
			request := reconcile.Request{NamespacedName: byoMachineLookupkey}
			_, err := reconciler.Reconcile(ctx, request)
			Expect(err).To(MatchError("nodes \"" + byoHost.Name + "\" not found"))
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, byoMachine)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, byoHost)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, machine)).Should(Succeed())
		})
	})

	Context("ByoMachine delete preconditions", func() {
		var (
			ctx                 context.Context
			machineScope        *byoMachineScope
			machine             *clusterv1.Machine
			byoMachine          *infrastructurev1alpha4.ByoMachine
			byoHost             *infrastructurev1alpha4.ByoHost
			byoMachineLookupKey types.NamespacedName
			byoHostLookupKey    types.NamespacedName
		)
		BeforeEach(func() {
			ctx = context.Background()

			machine = common.NewMachine("test-machine", defaultNamespace, defaultClusterName)
			machine.Spec.Bootstrap = clusterv1.Bootstrap{
				DataSecretName: &fakeBootstrapSecret,
			}
			Expect(k8sClient.Create(ctx, machine)).NotTo(HaveOccurred(), "machine creation failed")

			byoMachine = common.NewByoMachine("test-byomachine", defaultNamespace, defaultClusterName, machine)
			Expect(k8sClient.Create(ctx, byoMachine)).NotTo(HaveOccurred(), "byoMachine creation failed")

			byoHost = common.NewByoHost("test-host", defaultNamespace)
			Expect(k8sClient.Create(ctx, byoHost)).NotTo(HaveOccurred(), "byoHost creation failed")

			ph, err := patch.NewHelper(byoHost, k8sClient)
			Expect(err).ShouldNot(HaveOccurred())
			byoHost.Status.MachineRef = &corev1.ObjectReference{
				Kind:       "ByoMachine",
				Namespace:  byoMachine.Namespace,
				Name:       byoMachine.Name,
				UID:        byoMachine.UID,
				APIVersion: byoHost.APIVersion,
			}
			Expect(ph.Patch(ctx, byoHost, patch.WithStatusObservedGeneration{})).Should(Succeed())

			byoMachineLookupKey = types.NamespacedName{Name: byoMachine.Name, Namespace: defaultNamespace}
			byoHostLookupKey = types.NamespacedName{Name: byoHost.Name, Namespace: defaultNamespace}

			ph, err = patch.NewHelper(capiCluster, k8sClient)
			Expect(err).ShouldNot(HaveOccurred())
			capiCluster.Status.InfrastructureReady = true
			Expect(ph.Patch(ctx, capiCluster, patch.WithStatusObservedGeneration{})).Should(Succeed())

			WaitForObjectsToBePopulatedInCache(k8sClient, machine, byoMachine, byoHost)
		})

		FIt("should add cleanup annotation on byohost", func() {
			ph, err := patch.NewHelper(byoMachine, k8sClient)
			Expect(err).ShouldNot(HaveOccurred())
			controllerutil.AddFinalizer(byoMachine, infrastructurev1alpha4.MachineFinalizer)
			Expect(ph.Patch(ctx, byoMachine, patch.WithStatusObservedGeneration{})).Should(Succeed())

			Expect(k8sClient.Delete(ctx, byoMachine)).Should(Succeed())
			Eventually(func() bool {
				deletedByoMachine := &infrastructurev1alpha4.ByoMachine{}
				err := k8sClient.Get(ctx, byoMachineLookupKey, deletedByoMachine)
				if err != nil {
					return false
				}
				return !deletedByoMachine.ObjectMeta.DeletionTimestamp.IsZero()
			}, "10s", "1s").Should(BeTrue())

			request := reconcile.Request{NamespacedName: byoMachineLookupKey}
			_, err = reconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())

			createdByoHost := &infrastructurev1alpha4.ByoHost{}
			Expect(k8sClient.Get(ctx, byoHostLookupKey, createdByoHost)).NotTo(HaveOccurred())

			byoHostAnnotations := createdByoHost.GetAnnotations()
			_, ok := byoHostAnnotations[hostCleanupAnnotation]
			Expect(ok).To(BeTrue())
		})

		It("should remove host reservation", func() {
			Expect(reconciler.removeHostReservation(ctx, machineScope)).NotTo(HaveOccurred(), "host reservation removal failed")

			createdByoHost := &infrastructurev1alpha4.ByoHost{}
			Expect(k8sClient.Get(ctx, byoHostLookupKey, createdByoHost)).NotTo(HaveOccurred())

			Expect(createdByoHost.Status.MachineRef).To(BeNil())

			Expect(createdByoHost.Labels[clusterv1.ClusterLabelName]).To(BeEmpty())

			byoHostAnnotations := createdByoHost.GetAnnotations()
			_, ok := byoHostAnnotations[hostCleanupAnnotation]
			Expect(ok).To(BeFalse())
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, byoMachine)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, byoHost)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, machine)).Should(Succeed())
		})
	})
})
