package controllers

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/api/v1alpha4"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clusterapi "sigs.k8s.io/cluster-api/api/v1alpha4"
	"sigs.k8s.io/cluster-api/util/conditions"
)

const (
	namespace = "default"
)

var _ = Describe("Controllers/ByomachineController", func() {
	Context("When a BYO Host is available", func() {
		const (
			byoMachineName = "test-machine"
			byoHostName = "test-host"
		)
		var (
			ctx        context.Context
			byoHost    *infrastructurev1alpha4.ByoHost
			byoMachine *infrastructurev1alpha4.ByoMachine
		)

		BeforeEach(func() {
			ctx = context.Background()
			byoHost = createByoHost(byoHostName, namespace)
			Expect(k8sClient.Create(ctx, byoHost)).Should(Succeed())
		})

		It("claims the first available host", func() {
			byoMachine = createByoMachine(byoMachineName, namespace)
			Expect(k8sClient.Create(ctx, byoMachine)).Should(Succeed())

			byoHostLookupKey := types.NamespacedName{Name: byoHost.Name, Namespace: byoHost.Namespace}

			Eventually(func() bool {
				createdByoHost := &infrastructurev1alpha4.ByoHost{}
				err := k8sClient.Get(ctx, byoHostLookupKey, createdByoHost)
				if err != nil {
					return false
				}
				if createdByoHost.Status.MachineRef != nil {
					if createdByoHost.Status.MachineRef.Namespace == namespace && createdByoHost.Status.MachineRef.Name == byoMachineName {
						return true
					}
				}
				return false
			}).Should(BeTrue())

			byoMachineLookupkey := types.NamespacedName{Name: byoMachineName, Namespace: namespace}
			createdByoMachine := &infrastructurev1alpha4.ByoMachine{}

			Eventually(func() string {
				err := k8sClient.Get(ctx, byoMachineLookupkey, createdByoMachine)
				if err != nil {
					return ""
				}
				return createdByoMachine.Spec.ProviderID
			}).Should(ContainSubstring("byoh://"))

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
				readyCondition := conditions.Get(createdByoMachine, infrastructurev1alpha4.HostReadyCondition)
				if readyCondition != nil {
					return readyCondition.Status
				}
				return corev1.ConditionFalse
			}).Should(Equal(corev1.ConditionTrue))

			node := corev1.Node{}
			err := clientFake.Get(ctx, types.NamespacedName{Name: "test-host", Namespace: "default"}, &node)
			Expect(err).ToNot(HaveOccurred())

			Expect(node.Spec.ProviderID).To(ContainSubstring("byoh://"))
		})

	})
})

func createByoMachine(byoMachineName string, byoMachineNamespace string) *infrastructurev1alpha4.ByoMachine {
	byoMachine := &infrastructurev1alpha4.ByoMachine{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ByoMachine",
			APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha4",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      byoMachineName,
			Namespace: byoMachineNamespace,
			Labels: map[string]string{
				clusterapi.ClusterLabelName: "test-cluster",
			},
		},
		Spec: infrastructurev1alpha4.ByoMachineSpec{},
	}
	return byoMachine
}

func createByoHost(byoHostName string, byoHostNamspace string) *infrastructurev1alpha4.ByoHost {
	byoHost := &infrastructurev1alpha4.ByoHost{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ByoHost",
			APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha4",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      byoHostName,
			Namespace: byoHostNamspace,
		},
		Spec: infrastructurev1alpha4.ByoHostSpec{},
	}
	return byoHost
}
