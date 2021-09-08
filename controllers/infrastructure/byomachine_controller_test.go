package controllers

import (
	"context"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/apis/infrastructure/v1alpha4"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var _ = Describe("Controllers/ByomachineController", func() {

	Context("When no BYO Hosts are available", func() {
		type testConditions struct {
			Type   clusterv1.ConditionType
			Status corev1.ConditionStatus
			Reason string
		}
		var (
			ctx               context.Context
			byoMachine        *infrastructurev1alpha4.ByoMachine
			machine           *clusterv1.Machine
			expectedCondition *testConditions
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
		})

		It("should mark BYOHostReady as False when BYOHosts are not available", func() {
			byoMachineLookupKey := types.NamespacedName{Name: byoMachine.Name, Namespace: byoMachine.Namespace}

			expectedCondition = &testConditions{
				Type:   infrastructurev1alpha4.BYOHostReady,
				Status: corev1.ConditionFalse,
				Reason: infrastructurev1alpha4.BYOHostsUnavailableReason,
			}

			Eventually(func() *testConditions {
				createdByoMachine := &infrastructurev1alpha4.ByoMachine{}

				err := k8sClient.Get(ctx, byoMachineLookupKey, createdByoMachine)
				if err != nil {
					return &testConditions{}
				}

				actualCondition := conditions.Get(createdByoMachine, infrastructurev1alpha4.BYOHostReady)
				if actualCondition != nil {
					return &testConditions{
						Type:   actualCondition.Type,
						Status: actualCondition.Status,
						Reason: actualCondition.Reason,
					}
				}
				return &testConditions{}
			}).Should(Equal(expectedCondition))
		})
		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, byoMachine)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, machine)).Should(Succeed())
		})
	})

	Context("When a single BYO Host is available", func() {
		var (
			ctx        context.Context
			byoHost    *infrastructurev1alpha4.ByoHost
			byoMachine *infrastructurev1alpha4.ByoMachine
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

			byoHost = common.NewByoHost(defaultByoHostName, defaultNamespace, nil)
			Expect(k8sClient.Create(ctx, byoHost)).Should(Succeed())

			Expect(clientFake.Create(ctx, common.NewNode(byoHost.Name, defaultNamespace))).Should(Succeed())

		})

		It("claims the first available host", func() {
			byoHostLookupKey := types.NamespacedName{Name: byoHost.Name, Namespace: byoHost.Namespace}

			By("setting cluster.Status.InfrastructureReady to True")
			ph, err := patch.NewHelper(capiCluster, k8sClient)
			Expect(err).ShouldNot(HaveOccurred())
			capiCluster.Status.InfrastructureReady = true
			Expect(ph.Patch(ctx, capiCluster, patch.WithStatusObservedGeneration{})).Should(Succeed())

			Eventually(func() bool {
				createdByoHost := &infrastructurev1alpha4.ByoHost{}
				err = k8sClient.Get(ctx, byoHostLookupKey, createdByoHost)
				if err != nil {
					return false
				}
				if createdByoHost.Status.MachineRef != nil {
					if createdByoHost.Status.MachineRef.Namespace == defaultNamespace && createdByoHost.Status.MachineRef.Name == byoMachine.Name {
						return true
					}
				}
				return false
			}).Should(BeTrue())

			byoMachineLookupkey := types.NamespacedName{Name: byoMachine.Name, Namespace: defaultNamespace}
			createdByoMachine := &infrastructurev1alpha4.ByoMachine{}

			Eventually(func() string {
				err = k8sClient.Get(ctx, byoMachineLookupkey, createdByoMachine)
				if err != nil {
					return ""
				}
				return createdByoMachine.Spec.ProviderID
			}).Should(ContainSubstring(providerIDPrefix))

			Eventually(func() bool {
				err = k8sClient.Get(ctx, byoMachineLookupkey, createdByoMachine)
				if err != nil {
					return false
				}
				return createdByoMachine.Status.Ready
			}).Should(BeTrue())

			Eventually(func() corev1.ConditionStatus {

				err = k8sClient.Get(ctx, byoMachineLookupkey, createdByoMachine)
				if err != nil {
					return corev1.ConditionFalse
				}
				readyCondition := conditions.Get(createdByoMachine, infrastructurev1alpha4.BYOHostReady)
				if readyCondition != nil {
					return readyCondition.Status
				}
				return corev1.ConditionFalse
			}).Should(Equal(corev1.ConditionTrue))

			node := corev1.Node{}
			err = clientFake.Get(ctx, types.NamespacedName{Name: byoHost.Name, Namespace: defaultNamespace}, &node)
			Expect(err).NotTo(HaveOccurred())

			Expect(node.Spec.ProviderID).To(ContainSubstring(providerIDPrefix))
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, byoMachine)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, byoHost)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, machine)).Should(Succeed())
		})

	})

	Context("When all ByoHost are attached", func() {
		type testConditions struct {
			Type   clusterv1.ConditionType
			Status corev1.ConditionStatus
			Reason string
		}
		var (
			ctx               context.Context
			machine           *clusterv1.Machine
			byoHost           *infrastructurev1alpha4.ByoHost
			byoMachine        *infrastructurev1alpha4.ByoMachine
			expectedCondition *testConditions
		)

		BeforeEach(func() {
			ctx = context.Background()
			byoHost = common.NewByoHost(defaultByoHostName, defaultNamespace, nil)
			byoHost.Labels = map[string]string{clusterv1.ClusterLabelName: capiCluster.Name}
			Expect(k8sClient.Create(ctx, byoHost)).Should(Succeed())

			ph, err := patch.NewHelper(capiCluster, k8sClient)
			Expect(err).ShouldNot(HaveOccurred())
			capiCluster.Status.InfrastructureReady = true
			Expect(ph.Patch(ctx, capiCluster, patch.WithStatusObservedGeneration{})).Should(Succeed())

			machine = common.NewMachine(defaultMachineName, defaultNamespace, defaultClusterName)
			machine.Spec.Bootstrap = clusterv1.Bootstrap{
				DataSecretName: &fakeBootstrapSecret,
			}
			Expect(k8sClient.Create(ctx, machine)).Should(Succeed())

			byoMachine = common.NewByoMachine(defaultByoMachineName, defaultNamespace, defaultClusterName, machine)
			Expect(k8sClient.Create(ctx, byoMachine)).Should(Succeed())

		})

		It("should mark BYOHostReady as False when BYOHosts is available but attached", func() {
			byoMachineLookupKey := types.NamespacedName{Name: byoMachine.Name, Namespace: byoMachine.Namespace}
			expectedCondition = &testConditions{
				Type:   infrastructurev1alpha4.BYOHostReady,
				Status: corev1.ConditionFalse,
				Reason: infrastructurev1alpha4.BYOHostsUnavailableReason,
			}
			Eventually(func() *testConditions {
				createdByoMachine := &infrastructurev1alpha4.ByoMachine{}
				err := k8sClient.Get(ctx, byoMachineLookupKey, createdByoMachine)
				if err != nil {
					return &testConditions{}
				}

				actualCondition := conditions.Get(createdByoMachine, infrastructurev1alpha4.BYOHostReady)
				if actualCondition != nil {
					return &testConditions{
						Type:   actualCondition.Type,
						Status: actualCondition.Status,
						Reason: actualCondition.Reason,
					}
				}
				return &testConditions{}
			}).Should(Equal(expectedCondition))
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, byoHost)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, byoMachine)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, machine)).Should(Succeed())
		})
	})

	Context("When multiple BYO Host are available", func() {
		var (
			ctx        context.Context
			byoHost1   *infrastructurev1alpha4.ByoHost
			byoHost2   *infrastructurev1alpha4.ByoHost
			byoMachine *infrastructurev1alpha4.ByoMachine
			machine    *clusterv1.Machine
		)

		BeforeEach(func() {
			ctx = context.Background()
			byoHost1 = common.NewByoHost(defaultByoHostName, defaultNamespace, nil)
			Expect(k8sClient.Create(ctx, byoHost1)).Should(Succeed())
			byoHost2 = common.NewByoHost(defaultByoHostName, defaultNamespace, nil)
			Expect(k8sClient.Create(ctx, byoHost2)).Should(Succeed())

			ph, err := patch.NewHelper(capiCluster, k8sClient)
			Expect(err).ShouldNot(HaveOccurred())
			capiCluster.Status.InfrastructureReady = true
			Expect(ph.Patch(ctx, capiCluster, patch.WithStatusObservedGeneration{})).Should(Succeed())

			machine = common.NewMachine(defaultMachineName, defaultNamespace, defaultClusterName)
			machine.Spec.Bootstrap = clusterv1.Bootstrap{
				DataSecretName: &fakeBootstrapSecret,
			}
			Expect(k8sClient.Create(ctx, machine)).Should(Succeed())

			Expect(clientFake.Create(ctx, common.NewNode(byoHost1.Name, defaultNamespace))).Should(Succeed())
			Expect(clientFake.Create(ctx, common.NewNode(byoHost2.Name, defaultNamespace))).Should(Succeed())

		})

		It("claims one of the available host", func() {
			byoMachine = common.NewByoMachine(defaultByoMachineName, defaultNamespace, defaultClusterName, machine)
			Expect(k8sClient.Create(ctx, byoMachine)).Should(Succeed())

			byoMachineLookupKey := types.NamespacedName{Name: byoMachine.Name, Namespace: byoMachine.Namespace}
			createdByoMachine := &infrastructurev1alpha4.ByoMachine{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, byoMachineLookupKey, createdByoMachine)
				if err != nil {
					return false
				}
				return createdByoMachine.Status.Ready
			}).Should(BeTrue())

			Eventually(func() corev1.ConditionStatus {
				err := k8sClient.Get(ctx, byoMachineLookupKey, createdByoMachine)
				if err != nil {
					return corev1.ConditionFalse
				}
				readyCondition := conditions.Get(createdByoMachine, infrastructurev1alpha4.BYOHostReady)
				if readyCondition != nil {
					return readyCondition.Status
				}
				return corev1.ConditionFalse
			}).Should(Equal(corev1.ConditionTrue))

			node1 := corev1.Node{}
			err := clientFake.Get(ctx, types.NamespacedName{Name: byoHost1.Name, Namespace: defaultNamespace}, &node1)
			Expect(err).NotTo(HaveOccurred())

			node2 := corev1.Node{}
			err = clientFake.Get(ctx, types.NamespacedName{Name: byoHost2.Name, Namespace: defaultNamespace}, &node2)
			Expect(err).NotTo(HaveOccurred())

			var nodeTagged bool
			if strings.Contains(node1.Spec.ProviderID, providerIDPrefix) || strings.Contains(node2.Spec.ProviderID, providerIDPrefix) {
				nodeTagged = true
			}
			Expect(nodeTagged).To(Equal(true))
		})

		It("does not claims the attached host", func() {
			ph, err := patch.NewHelper(byoHost2, k8sClient)
			Expect(err).ShouldNot(HaveOccurred())
			byoHost2.Labels = map[string]string{clusterv1.ClusterLabelName: capiCluster.Name}
			Expect(ph.Patch(ctx, byoHost2, patch.WithStatusObservedGeneration{})).Should(Succeed())

			byoMachine = common.NewByoMachine(defaultByoMachineName, defaultNamespace, defaultClusterName, machine)
			Expect(k8sClient.Create(ctx, byoMachine)).Should(Succeed())

			byoMachineLookupKey := types.NamespacedName{Name: byoMachine.Name, Namespace: byoMachine.Namespace}
			createdByoMachine := &infrastructurev1alpha4.ByoMachine{}
			byoHostLookupKey := types.NamespacedName{Name: byoHost1.Name, Namespace: defaultNamespace}

			Eventually(func() bool {
				createdByoHost := &infrastructurev1alpha4.ByoHost{}
				err = k8sClient.Get(ctx, byoHostLookupKey, createdByoHost)
				if err != nil {
					return false
				}
				if createdByoHost.Status.MachineRef != nil {
					if createdByoHost.Status.MachineRef.Namespace == defaultNamespace && createdByoHost.Status.MachineRef.Name == byoMachine.Name {
						return true
					}
				}
				return false
			}).Should(BeTrue())

			Eventually(func() bool {
				err = k8sClient.Get(ctx, byoMachineLookupKey, createdByoMachine)
				if err != nil {
					return false
				}
				return createdByoMachine.Status.Ready
			}).Should(BeTrue())

			Eventually(func() corev1.ConditionStatus {
				err = k8sClient.Get(ctx, byoMachineLookupKey, createdByoMachine)
				if err != nil {
					return corev1.ConditionFalse
				}
				readyCondition := conditions.Get(createdByoMachine, infrastructurev1alpha4.BYOHostReady)
				if readyCondition != nil {
					return readyCondition.Status
				}
				return corev1.ConditionFalse
			}).Should(Equal(corev1.ConditionTrue))

			node := corev1.Node{}
			err = clientFake.Get(ctx, types.NamespacedName{Name: byoHost1.Name, Namespace: defaultNamespace}, &node)
			Expect(err).NotTo(HaveOccurred())

			Expect(node.Spec.ProviderID).To(ContainSubstring(providerIDPrefix))
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, byoHost1)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, byoHost2)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, byoMachine)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, machine)).Should(Succeed())
		})
	})

	Context("Test for ByoMachine Reconcile preconditions", func() {
		type testConditions struct {
			Type   clusterv1.ConditionType
			Status corev1.ConditionStatus
			Reason string
		}
		var (
			ctx                 context.Context
			byoMachine          *infrastructurev1alpha4.ByoMachine
			machine             *clusterv1.Machine
			expectedCondition   *testConditions
			byoMachineLookupKey types.NamespacedName
		)
		BeforeEach(func() {
			ctx = context.Background()

			machine = common.NewMachine(defaultMachineName, defaultNamespace, defaultClusterName)
			Expect(k8sClient.Create(ctx, machine)).Should(Succeed())

			byoMachine = common.NewByoMachine(defaultByoMachineName, defaultNamespace, defaultClusterName, machine)
			Expect(k8sClient.Create(ctx, byoMachine)).Should(Succeed())
			byoMachineLookupKey = types.NamespacedName{Name: byoMachine.Name, Namespace: byoMachine.Namespace}

		})

		It("should add MachineFinalizer on ByoMachine", func() {
			createdByoMachine := &infrastructurev1alpha4.ByoMachine{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, byoMachineLookupKey, createdByoMachine)
				if err != nil {
					return false
				}
				return controllerutil.ContainsFinalizer(createdByoMachine, infrastructurev1alpha4.MachineFinalizer)
			}).Should(BeTrue())
		})

		It("should mark BYOHostReady as False when byomachine is paused", func() {
			ph, err := patch.NewHelper(byoMachine, k8sClient)
			Expect(err).ShouldNot(HaveOccurred())

			pauseAnnotations := map[string]string{
				clusterv1.PausedAnnotation: "paused",
			}
			annotations.AddAnnotations(byoMachine, pauseAnnotations)

			Expect(ph.Patch(ctx, byoMachine, patch.WithStatusObservedGeneration{})).Should(Succeed())

			expectedCondition = &testConditions{
				Type:   infrastructurev1alpha4.BYOHostReady,
				Status: corev1.ConditionFalse,
				Reason: infrastructurev1alpha4.ClusterOrResourcePausedReason,
			}

			Eventually(func() *testConditions {
				createdByoMachine := &infrastructurev1alpha4.ByoMachine{}
				err := k8sClient.Get(ctx, byoMachineLookupKey, createdByoMachine)
				if err != nil {
					return &testConditions{}
				}

				actualCondition := conditions.Get(createdByoMachine, infrastructurev1alpha4.BYOHostReady)
				if actualCondition != nil {
					return &testConditions{
						Type:   actualCondition.Type,
						Status: actualCondition.Status,
						Reason: actualCondition.Reason,
					}
				}
				return &testConditions{}
			}).Should(Equal(expectedCondition))
		})

		It("should mark BYOHostReady as False when cluster is paused", func() {
			pausedCluster := common.NewCluster("paused-cluster", defaultNamespace)
			pausedCluster.Spec.Paused = true
			Expect(k8sClient.Create(ctx, pausedCluster)).Should(Succeed())
			pausedMachine := common.NewMachine("paused-machine", defaultNamespace, pausedCluster.Name)
			Expect(k8sClient.Create(ctx, pausedMachine)).Should(Succeed())
			pausedByoMachine := common.NewByoMachine("paused-byo-machine", defaultNamespace, pausedCluster.Name, pausedMachine)
			Expect(k8sClient.Create(ctx, pausedByoMachine)).Should(Succeed())

			pausedByoMachineLookupKey := types.NamespacedName{Name: pausedByoMachine.Name, Namespace: pausedByoMachine.Namespace}

			expectedCondition = &testConditions{
				Type:   infrastructurev1alpha4.BYOHostReady,
				Status: corev1.ConditionFalse,
				Reason: infrastructurev1alpha4.ClusterOrResourcePausedReason,
			}

			Eventually(func() *testConditions {
				createdByoMachine := &infrastructurev1alpha4.ByoMachine{}
				err := k8sClient.Get(ctx, pausedByoMachineLookupKey, createdByoMachine)
				if err != nil {
					return &testConditions{}
				}

				actualCondition := conditions.Get(createdByoMachine, infrastructurev1alpha4.BYOHostReady)
				if actualCondition != nil {
					return &testConditions{
						Type:   actualCondition.Type,
						Status: actualCondition.Status,
						Reason: actualCondition.Reason,
					}
				}
				return &testConditions{}
			}).Should(Equal(expectedCondition))

			Expect(k8sClient.Delete(ctx, pausedCluster)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, pausedMachine)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, pausedByoMachine)).Should(Succeed())

		})

		It("should mark BYOHostReady as False when cluster.Status.InfrastructureReady is false", func() {

			expectedCondition = &testConditions{
				Type:   infrastructurev1alpha4.BYOHostReady,
				Status: corev1.ConditionFalse,
				Reason: infrastructurev1alpha4.WaitingForClusterInfrastructureReason,
			}

			By("setting cluster.Status.InfrastructureReady to False")
			Eventually(func() error {
				ph, err := patch.NewHelper(capiCluster, k8sClient)
				Expect(err).ShouldNot(HaveOccurred())
				capiCluster.Status.InfrastructureReady = false
				return ph.Patch(ctx, capiCluster, patch.WithStatusObservedGeneration{})
			}).Should(BeNil())

			Eventually(func() *testConditions {
				createdByoMachine := &infrastructurev1alpha4.ByoMachine{}
				err := k8sClient.Get(ctx, byoMachineLookupKey, createdByoMachine)
				if err != nil {
					return &testConditions{}
				}

				actualCondition := conditions.Get(createdByoMachine, infrastructurev1alpha4.BYOHostReady)
				if actualCondition != nil {
					return &testConditions{
						Type:   actualCondition.Type,
						Status: actualCondition.Status,
						Reason: actualCondition.Reason,
					}
				}
				return &testConditions{}
			}).Should(Equal(expectedCondition))

		})

		It("should mark BYOHostReady as False when machine.Spec.Bootstrap.DataSecretName is not set", func() {
			expectedCondition = &testConditions{
				Type:   infrastructurev1alpha4.BYOHostReady,
				Status: corev1.ConditionFalse,
				Reason: infrastructurev1alpha4.WaitingForBootstrapDataSecretReason,
			}

			By("setting cluster.Status.InfrastructureReady to True")
			Eventually(func() error {
				ph, err := patch.NewHelper(capiCluster, k8sClient)
				Expect(err).ShouldNot(HaveOccurred())
				capiCluster.Status.InfrastructureReady = true
				return ph.Patch(ctx, capiCluster, patch.WithStatusObservedGeneration{})
			}).Should(BeNil())

			Eventually(func() *testConditions {
				createdByoMachine := &infrastructurev1alpha4.ByoMachine{}
				err := k8sClient.Get(ctx, byoMachineLookupKey, createdByoMachine)
				if err != nil {
					return &testConditions{}
				}

				actualCondition := conditions.Get(createdByoMachine, infrastructurev1alpha4.BYOHostReady)
				if actualCondition != nil {
					return &testConditions{
						Type:   actualCondition.Type,
						Status: actualCondition.Status,
						Reason: actualCondition.Reason,
					}
				}
				return &testConditions{}
			}).Should(Equal(expectedCondition))

		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, byoMachine)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, machine)).Should(Succeed())

		})
	})

})
