package controllers

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/apis/infrastructure/v1alpha4"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/common"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Controllers/ByomachineController/Unitests", func() {
	var byoReconciler *ByoMachineReconciler

	BeforeEach(func() {
		byoReconciler = &ByoMachineReconciler{
			Client: k8sClient,
		}
	})
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
			_, err := byoReconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should not attempt to reconcile when byomachine name does not exist", func() {
			byoMachineLookupkey := types.NamespacedName{Name: byoMachineName, Namespace: defaultNamespace}
			request := reconcile.Request{NamespacedName: byoMachineLookupkey}
			_, err := byoReconciler.Reconcile(ctx, request)
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

			byoHost = common.NewByoHost(defaultByoHostName, defaultNamespace, nil)
			Expect(k8sClient.Create(ctx, byoHost)).Should(Succeed())
		})

		It("Should return error", func() {
			byoMachineLookupkey := types.NamespacedName{Name: byoMachine.Name, Namespace: defaultNamespace}
			request := reconcile.Request{NamespacedName: byoMachineLookupkey}

			_, err := byoReconciler.Reconcile(ctx, request)
			Expect(err).To(MatchError("clusters.cluster.x-k8s.io \"" + clusterName + "\" not found"))
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
			hostname = "host-when-node-is-not-available"
		)

		var (
			ctx        context.Context
			byoHost    *infrastructurev1alpha4.ByoHost
			fakeClient client.Client
		)

		BeforeEach(func() {
			ctx = context.Background()
			byoHost = common.NewByoHost(hostname, defaultNamespace, nil)
			Expect(k8sClient.Create(ctx, byoHost)).Should(Succeed())
			fakeClient = fake.NewClientBuilder().WithObjects(
				capiCluster,
			).Build()

		})

		It("Should return node not found error", func() {
			err := byoReconciler.setNodeProviderID(ctx, fakeClient, byoHost, "")
			Expect(err).To(MatchError("nodes \"" + byoHost.Name + "\" not found"))
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, byoHost)).Should(Succeed())
		})
	})

	Context("ByoMachine delete preconditions", func() {
		var (
			ctx              context.Context
			machineScope     *byoMachineScope
			machine          *clusterv1.Machine
			byoMachine       *infrastructurev1alpha4.ByoMachine
			byoHost          *infrastructurev1alpha4.ByoHost
			byoHostLookupKey types.NamespacedName
			err              error
		)
		BeforeEach(func() {
			ctx = context.Background()

			machine = common.NewMachine("test-machine", defaultNamespace, defaultClusterName)
			Expect(k8sClient.Create(ctx, machine)).NotTo(HaveOccurred(), "machine creation failed")

			byoMachine = common.NewByoMachine("test-byomachine", defaultNamespace, defaultClusterName, machine)
			Expect(k8sClient.Create(ctx, byoMachine)).NotTo(HaveOccurred(), "byoMachine creation failed")

			byoHost = common.NewByoHost("test-host", defaultNamespace, byoMachine)
			Expect(k8sClient.Create(ctx, byoHost)).NotTo(HaveOccurred(), "byoHost creation failed")

			byoHostLookupKey = types.NamespacedName{Name: byoHost.Name, Namespace: byoHost.Namespace}

			machineScope, err = newByoMachineScope(byoMachineScopeParams{
				Client:     k8sClient,
				Cluster:    capiCluster,
				Machine:    machine,
				ByoMachine: byoMachine,
				ByoHost:    byoHost,
			})
			Expect(err).NotTo(HaveOccurred(), "failed creating machineScope")

		})

		It("should add cleanup annotation on byohost", func() {

			Expect(byoReconciler.markHostForCleanup(ctx, machineScope)).NotTo(HaveOccurred(), "markHostForCleanup failed")

			createdByoHost := &infrastructurev1alpha4.ByoHost{}
			Expect(k8sClient.Get(ctx, byoHostLookupKey, createdByoHost)).NotTo(HaveOccurred())

			byoHostAnnotations := createdByoHost.GetAnnotations()
			_, ok := byoHostAnnotations[hostCleanupAnnotation]
			Expect(ok).To(BeTrue())
		})

		It("should remove host reservation", func() {
			Expect(byoReconciler.removeHostReservation(ctx, machineScope)).NotTo(HaveOccurred(), "host reservation removal failed")

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
