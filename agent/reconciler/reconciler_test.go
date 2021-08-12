package reconciler

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

var _ = Describe("Reconciler", func() {
	Context("when K8sComponentsInstallationSucceeded is False", func() {
		type testConditions struct {
			Type   clusterv1.ConditionType
			Status corev1.ConditionStatus
			Reason string
		}

		var (
			//ns *corev1.Namespace
			ns                string
			byoHost           *infrastructurev1alpha4.ByoHost
			hostName          string
			expectedCondition *testConditions
		)

		BeforeEach(func() {
			hostName = "test-host"
			ns = "default"
			expectedCondition = &testConditions{
				Type:   infrastructurev1alpha4.K8sNodeBootstrapSucceeded,
				Status: corev1.ConditionFalse,
			}

			byoHost = common.NewByoHost(hostName, ns, nil)
			Expect(k8sClient.Create(context.TODO(), byoHost)).NotTo(HaveOccurred(), "failed to create byohost")

		})

		It("should set the Reason to WaitingForMachineRefReason", func() {
			expectedCondition.Reason = infrastructurev1alpha4.WaitingForMachineRefReason
			byoHostLookupKey := types.NamespacedName{Name: hostName, Namespace: ns}
			Eventually(func() *testConditions {
				createdByoHost := &infrastructurev1alpha4.ByoHost{}
				err := k8sClient.Get(context.TODO(), byoHostLookupKey, createdByoHost)
				if err != nil {
					return &testConditions{}
				}
				byoHostRegistrationSucceeded := conditions.Get(createdByoHost, infrastructurev1alpha4.K8sNodeBootstrapSucceeded)
				if byoHostRegistrationSucceeded != nil {
					return &testConditions{
						Type:   byoHostRegistrationSucceeded.Type,
						Status: byoHostRegistrationSucceeded.Status,
						Reason: byoHostRegistrationSucceeded.Reason,
					}
				}
				return &testConditions{}
			}).Should(Equal(expectedCondition))
		})

		FIt("should set the Reason to BootstrapDataSecretUnavailableReason", func() {
			expectedCondition.Reason = infrastructurev1alpha4.BootstrapDataSecretUnavailableReason
			byoMachine := common.NewByoMachine("test-byomachine", ns, "", nil)

			byoHostLookupKey := types.NamespacedName{Name: hostName, Namespace: ns}

			Eventually(func() error {
				ph, err := patch.NewHelper(byoHost, k8sClient)
				Expect(err).ShouldNot(HaveOccurred())
				byoHost.Status.MachineRef = &corev1.ObjectReference{
					Kind:       "ByoMachine",
					Namespace:  byoMachine.Namespace,
					Name:       byoMachine.Name,
					UID:        byoMachine.UID,
					APIVersion: byoHost.APIVersion,
				}
				return ph.Patch(context.TODO(), byoHost, patch.WithStatusObservedGeneration{})
			}).Should(BeNil())

			Eventually(func() *testConditions {
				createdByoHost := &infrastructurev1alpha4.ByoHost{}
				err = k8sClient.Get(context.TODO(), byoHostLookupKey, createdByoHost)
				if err != nil {
					return &testConditions{}
				}

				byoHostRegistrationSucceeded := conditions.Get(createdByoHost, infrastructurev1alpha4.K8sNodeBootstrapSucceeded)
				if byoHostRegistrationSucceeded != nil {
					return &testConditions{
						Type:   byoHostRegistrationSucceeded.Type,
						Status: byoHostRegistrationSucceeded.Status,
						Reason: byoHostRegistrationSucceeded.Reason,
					}
				}
				return &testConditions{}
			}).Should(Equal(expectedCondition))
		})

		AfterEach(func() {
			err = k8sClient.Delete(context.TODO(), byoHost)
			Expect(err).NotTo(HaveOccurred())
		})

	})
})
