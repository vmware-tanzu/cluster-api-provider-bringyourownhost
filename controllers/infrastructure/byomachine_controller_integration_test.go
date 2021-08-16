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
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
)

var _ = Describe("Controllers/ByomachineController", func() {
	Context("When a BYO Host is available", func() {
		type testConditions struct {
			Type   clusterv1.ConditionType
			Status corev1.ConditionStatus
			Reason string
		}
		var (
			ctx               context.Context
			byoHost           *infrastructurev1alpha4.ByoHost
			byoMachine        *infrastructurev1alpha4.ByoMachine
			machine           *clusterv1.Machine
			expectedCondition *testConditions
		)

		BeforeEach(func() {
			ctx = context.Background()
			byoHost = common.NewByoHost(defaultByoHostName, defaultNamespace, nil)
			Expect(k8sClient.Create(ctx, byoHost)).Should(Succeed())
		})

		It("claims the first available host", func() {
			machine = common.NewMachine(defaultMachineName, defaultNamespace, defaultClusterName)
			machine.Spec.Bootstrap = clusterv1.Bootstrap{
				DataSecretName: &fakeBootstrapSecret,
			}
			Expect(k8sClient.Create(ctx, machine)).Should(Succeed())

			byoMachine = common.NewByoMachine(defaultByoMachineName, defaultNamespace, defaultClusterName, machine)
			Expect(k8sClient.Create(ctx, byoMachine)).Should(Succeed())
			byoHostLookupKey := types.NamespacedName{Name: byoHost.Name, Namespace: byoHost.Namespace}

			Eventually(func() bool {
				createdByoHost := &infrastructurev1alpha4.ByoHost{}
				err := k8sClient.Get(ctx, byoHostLookupKey, createdByoHost)
				if err != nil {
					return false
				}
				if createdByoHost.Status.MachineRef != nil {
					if createdByoHost.Status.MachineRef.Namespace == defaultNamespace && createdByoHost.Status.MachineRef.Name == defaultByoMachineName {
						return true
					}
				}
				return false
			}).Should(BeTrue())

			byoMachineLookupkey := types.NamespacedName{Name: defaultByoMachineName, Namespace: defaultNamespace}
			createdByoMachine := &infrastructurev1alpha4.ByoMachine{}

			Eventually(func() string {
				err := k8sClient.Get(ctx, byoMachineLookupkey, createdByoMachine)
				if err != nil {
					return ""
				}
				return createdByoMachine.Spec.ProviderID
			}).Should(ContainSubstring(ProviderIDPrefix))

			Eventually(func() bool {
				err := k8sClient.Get(ctx, byoMachineLookupkey, createdByoMachine)
				if err != nil {
					return false
				}
				return createdByoMachine.Status.Ready
			}).Should(BeTrue())

			Eventually(func() corev1.ConditionStatus {

				err := k8sClient.Get(ctx, byoMachineLookupkey, createdByoMachine)
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
			err := clientFake.Get(ctx, types.NamespacedName{Name: defaultNodeName, Namespace: defaultNamespace}, &node)
			Expect(err).NotTo(HaveOccurred())

			Expect(node.Spec.ProviderID).To(ContainSubstring(ProviderIDPrefix))
		})

		FIt("should mark BYOHostReady as False when cluster.Status.InfrastructureReady is false", func() {
			machine = common.NewMachine(defaultMachineName, defaultNamespace, defaultClusterName)
			Expect(k8sClient.Create(ctx, machine)).Should(Succeed())

			byoMachine = common.NewByoMachine("test-byo-machine", defaultNamespace, defaultClusterName, machine)
			Expect(k8sClient.Create(ctx, byoMachine)).Should(Succeed())

			byoMachineLookupKey := types.NamespacedName{Name: byoMachine.Name, Namespace: byoMachine.Namespace}

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

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, byoMachine)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, byoHost)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, machine)).Should(Succeed())
		})

	})
})
