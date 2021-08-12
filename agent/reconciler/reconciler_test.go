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
)

var _ = Describe("Reconciler", func() {
	FContext("test reconciler", func() {
		var (
			//ns *corev1.Namespace
			ns       string
			byoHost  *infrastructurev1alpha4.ByoHost
			hostName string
		)
		BeforeEach(func() {
			hostName = "test-host"
			ns = "default"
			// ns = common.NewNamespace("default")
			// Expect(k8sClient.Create(context.TODO(), ns)).NotTo(HaveOccurred(), "failed to create test namespace")

			byoHost = common.NewByoHost(hostName, ns, nil)
			Expect(k8sClient.Create(context.TODO(), byoHost)).NotTo(HaveOccurred(), "failed to create byohost")

		})

		FIt("should set the K8sComponentsInstallationSucceeded status to false with WaitingForMachineRefReason reason", func() {
			byoHostLookupKey := types.NamespacedName{Name: hostName, Namespace: ns}
			type testConditions struct {
				Type   clusterv1.ConditionType
				Status corev1.ConditionStatus
				Reason string
			}
			expectedCondition := &testConditions{
				Type:   infrastructurev1alpha4.K8sNodeBootstrapSucceeded,
				Status: corev1.ConditionFalse,
				Reason: infrastructurev1alpha4.WaitingForMachineRefReason,
			}
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

		AfterEach(func() {
			err = k8sClient.Delete(context.TODO(), byoHost)
			Expect(err).NotTo(HaveOccurred())
		})

	})
})
