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
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	controllerruntime "sigs.k8s.io/controller-runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ = Describe("Byohost Agent Tests", func() {
	Context("when K8sComponentsInstallationSucceeded is False", func() {
		type testConditions struct {
			Type   clusterv1.ConditionType
			Status corev1.ConditionStatus
			Reason string
		}

		var (
			ctx               = context.TODO()
			ns                = "default"
			hostName          = "test-host"
			byoHost           *infrastructurev1alpha4.ByoHost
			expectedCondition *testConditions
			byoHostLookupKey  types.NamespacedName
		)

		BeforeEach(func() {
			expectedCondition = &testConditions{
				Type:   infrastructurev1alpha4.K8sNodeBootstrapSucceeded,
				Status: corev1.ConditionFalse,
			}

			byoHost = common.NewByoHost(hostName, ns, nil)
			Expect(k8sClient.Create(ctx, byoHost)).NotTo(HaveOccurred(), "failed to create byohost")
			patchHelper, err = patch.NewHelper(byoHost, k8sClient)
			Expect(err).ShouldNot(HaveOccurred())

			byoHostLookupKey = types.NamespacedName{Name: hostName, Namespace: ns}
		})

		It("should set the Reason to ClusterOrResourcePausedReason", func() {
			annotations.AddAnnotations(byoHost, map[string]string{
				clusterv1.PausedAnnotation: "paused",
			})
			patchHelper.Patch(ctx, byoHost, patch.WithStatusObservedGeneration{})

			result, err := reconciler.Reconcile(ctx, controllerruntime.Request{
				NamespacedName: byoHostLookupKey,
			})

			Expect(result).To(Equal(ctrl.Result{}))
			Expect(err).ToNot(HaveOccurred())

			updatedByoHost := &infrastructurev1alpha4.ByoHost{}
			err = k8sClient.Get(ctx, byoHostLookupKey, updatedByoHost)
			Expect(err).ToNot(HaveOccurred())
			bootstrapSucceededCondition := conditions.Get(updatedByoHost, infrastructurev1alpha4.K8sNodeBootstrapSucceeded)

			Expect(*bootstrapSucceededCondition).To(conditions.MatchCondition(clusterv1.Condition{
				Type:     infrastructurev1alpha4.K8sNodeBootstrapSucceeded,
				Status:   corev1.ConditionFalse,
				Reason:   infrastructurev1alpha4.ClusterOrResourcePausedReason,
				Severity: clusterv1.ConditionSeverityInfo,
			}))
		})

		It("should set the Reason to WaitingForMachineRefReason", func() {
			expectedCondition.Reason = infrastructurev1alpha4.WaitingForMachineRefReason
			byoHostLookupKey := types.NamespacedName{Name: hostName, Namespace: ns}
			Eventually(func() *testConditions {
				createdByoHost := &infrastructurev1alpha4.ByoHost{}
				err = k8sClient.Get(ctx, byoHostLookupKey, createdByoHost)
				if err != nil {
					return &testConditions{}
				}
				actualCondition := conditions.Get(createdByoHost, infrastructurev1alpha4.K8sNodeBootstrapSucceeded)
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

		It("should set the Reason to BootstrapDataSecretUnavailableReason", func() {
			expectedCondition.Reason = infrastructurev1alpha4.BootstrapDataSecretUnavailableReason
			byoMachine := common.NewByoMachine("test-byomachine", ns, "", nil)

			byoHostLookupKey := types.NamespacedName{Name: hostName, Namespace: ns}

			By("patching machineRef to ByoHost")
			Eventually(func() error {
				patchHelper, err = patch.NewHelper(byoHost, k8sClient)
				Expect(err).ShouldNot(HaveOccurred())
				byoHost.Status.MachineRef = &corev1.ObjectReference{
					Kind:       "ByoMachine",
					Namespace:  byoMachine.Namespace,
					Name:       byoMachine.Name,
					UID:        byoMachine.UID,
					APIVersion: byoHost.APIVersion,
				}
				return patchHelper.Patch(ctx, byoHost, patch.WithStatusObservedGeneration{})
			}).Should(BeNil())

			Eventually(func() *testConditions {
				createdByoHost := &infrastructurev1alpha4.ByoHost{}
				err = k8sClient.Get(ctx, byoHostLookupKey, createdByoHost)
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

		It("should set the Reason to CloudInitExecutionFailedReason", func() {
			expectedCondition.Reason = infrastructurev1alpha4.CloudInitExecutionFailedReason
			byoMachine := common.NewByoMachine("test-byomachine", ns, "", nil)
			byoHostLookupKey := types.NamespacedName{Name: hostName, Namespace: ns}

			By("creating the bootstrap secret")
			secret := common.NewSecret("test-secret", "test-secret-data", ns)
			Expect(k8sClient.Create(ctx, secret)).NotTo(HaveOccurred())

			By("patching the machineref and bootstrap secret")
			Eventually(func() error {
				patchHelper, err = patch.NewHelper(byoHost, k8sClient)
				Expect(err).ShouldNot(HaveOccurred())
				byoHost.Status.MachineRef = &corev1.ObjectReference{
					Kind:       "ByoMachine",
					Namespace:  byoMachine.Namespace,
					Name:       byoMachine.Name,
					UID:        byoMachine.UID,
					APIVersion: byoHost.APIVersion,
				}
				byoHost.Spec.BootstrapSecret = &corev1.ObjectReference{
					Kind:      "Secret",
					Namespace: secret.Namespace,
					Name:      secret.Name,
				}
				return patchHelper.Patch(ctx, byoHost, patch.WithStatusObservedGeneration{})
			}).Should(BeNil())

			Eventually(func() *testConditions {
				createdByoHost := &infrastructurev1alpha4.ByoHost{}
				err = k8sClient.Get(ctx, byoHostLookupKey, createdByoHost)
				if err != nil {
					return &testConditions{}
				}
				actualCondition := conditions.Get(createdByoHost, infrastructurev1alpha4.K8sNodeBootstrapSucceeded)
				if actualCondition != nil {
					return &testConditions{
						Type:   actualCondition.Type,
						Status: actualCondition.Status,
						Reason: actualCondition.Reason}
				}
				return &testConditions{}
			}).Should(Equal(expectedCondition))

			// Delete the secret
			Expect(k8sClient.Delete(ctx, secret)).NotTo(HaveOccurred())
		})
		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, byoHost)).NotTo(HaveOccurred())
		})

	})
})
