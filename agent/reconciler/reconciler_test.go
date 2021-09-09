package reconciler

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/agent/cloudinit/cloudinitfakes"
	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/apis/infrastructure/v1alpha4"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

var _ = Describe("Byohost Agent Tests", func() {

	var (
		ctx              = context.TODO()
		ns               = "default"
		hostName         = "test-host"
		byoHost          *infrastructurev1alpha4.ByoHost
		byoHostLookupKey types.NamespacedName
	)

	BeforeEach(func() {
		byoHost = common.NewByoHost(hostName, ns, nil)
		Expect(k8sClient.Create(ctx, byoHost)).NotTo(HaveOccurred(), "failed to create byohost")
		patchHelper, err = patch.NewHelper(byoHost, k8sClient)
		Expect(err).ShouldNot(HaveOccurred())

		byoHostLookupKey = types.NamespacedName{Name: byoHost.Name, Namespace: ns}
		fakeCommandRunner = &cloudinitfakes.FakeICmdRunner{}
		fakeFileWriter = &cloudinitfakes.FakeIFileWriter{}

		reconciler = &HostReconciler{
			Client:           k8sClient,
			CmdRunner:        fakeCommandRunner,
			FileWriter:       fakeFileWriter,
			WatchFilterValue: hostName,
		}
	})

	Context("when K8sComponentsInstallationSucceeded is False", func() {
		It("should set the Reason to ClusterOrResourcePausedReason", func() {
			annotations.AddAnnotations(byoHost, map[string]string{
				clusterv1.PausedAnnotation: "paused",
			})
			err = patchHelper.Patch(ctx, byoHost, patch.WithStatusObservedGeneration{})
			Expect(err).ToNot(HaveOccurred())

			result, reconcilerErr := reconciler.Reconcile(ctx, controllerruntime.Request{
				NamespacedName: byoHostLookupKey,
			})

			Expect(result).To(Equal(controllerruntime.Result{}))
			Expect(reconcilerErr).ToNot(HaveOccurred())

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
			result, reconcilerErr := reconciler.Reconcile(ctx, controllerruntime.Request{
				NamespacedName: byoHostLookupKey,
			})

			Expect(result).To(Equal(controllerruntime.Result{}))
			Expect(reconcilerErr).ToNot(HaveOccurred())

			updatedByoHost := &infrastructurev1alpha4.ByoHost{}
			err = k8sClient.Get(ctx, byoHostLookupKey, updatedByoHost)
			Expect(err).ToNot(HaveOccurred())

			k8sNodeBootstrapSucceeded := conditions.Get(updatedByoHost, infrastructurev1alpha4.K8sNodeBootstrapSucceeded)
			Expect(*k8sNodeBootstrapSucceeded).To(conditions.MatchCondition(clusterv1.Condition{
				Type:     infrastructurev1alpha4.K8sNodeBootstrapSucceeded,
				Status:   corev1.ConditionFalse,
				Reason:   infrastructurev1alpha4.WaitingForMachineRefReason,
				Severity: clusterv1.ConditionSeverityInfo,
			}))
		})

		It("should set the Reason to BootstrapDataSecretUnavailableReason", func() {
			byoMachine := common.NewByoMachine("test-byomachine", ns, "", nil)
			Expect(k8sClient.Create(ctx, byoMachine)).NotTo(HaveOccurred(), "failed to create byomachine")

			patchHelper, err = patch.NewHelper(byoHost, k8sClient)
			Expect(err).ShouldNot(HaveOccurred())
			byoHost.Status.MachineRef = &corev1.ObjectReference{
				Kind:       "ByoMachine",
				Namespace:  byoMachine.Namespace,
				Name:       byoMachine.Name,
				UID:        byoMachine.UID,
				APIVersion: byoHost.APIVersion,
			}
			Expect(patchHelper.Patch(ctx, byoHost, patch.WithStatusObservedGeneration{})).NotTo(HaveOccurred())

			result, reconcilerErr := reconciler.Reconcile(ctx, controllerruntime.Request{
				NamespacedName: byoHostLookupKey,
			})

			Expect(result).To(Equal(controllerruntime.Result{}))
			Expect(reconcilerErr).ToNot(HaveOccurred())

			updatedByoHost := &infrastructurev1alpha4.ByoHost{}
			err = k8sClient.Get(ctx, byoHostLookupKey, updatedByoHost)
			Expect(err).ToNot(HaveOccurred())

			byoHostRegistrationSucceeded := conditions.Get(updatedByoHost, infrastructurev1alpha4.K8sNodeBootstrapSucceeded)
			Expect(*byoHostRegistrationSucceeded).To(conditions.MatchCondition(clusterv1.Condition{
				Type:     infrastructurev1alpha4.K8sNodeBootstrapSucceeded,
				Status:   corev1.ConditionFalse,
				Reason:   infrastructurev1alpha4.BootstrapDataSecretUnavailableReason,
				Severity: clusterv1.ConditionSeverityInfo,
			}))

			Expect(k8sClient.Delete(ctx, byoMachine)).NotTo(HaveOccurred())
		})

		It("should set the Reason to CloudInitExecutionFailedReason", func() {
			//	byoHost := byoHost.DeepCopy()

			byoMachine := common.NewByoMachine("test-byomachine", ns, "", nil)
			Expect(k8sClient.Create(ctx, byoMachine)).NotTo(HaveOccurred(), "failed to create byomachine")

			By("creating the bootstrap secret")
			secret := common.NewSecret("test-secret", "test-secret-data", ns)
			Expect(k8sClient.Create(ctx, secret)).NotTo(HaveOccurred())

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

			Expect(patchHelper.Patch(ctx, byoHost, patch.WithStatusObservedGeneration{})).NotTo(HaveOccurred())

			result, reconcilerErr := reconciler.Reconcile(ctx, controllerruntime.Request{
				NamespacedName: byoHostLookupKey,
			})

			Expect(result).To(Equal(controllerruntime.Result{}))
			Expect(reconcilerErr).To(HaveOccurred())

			updatedByoHost := &infrastructurev1alpha4.ByoHost{}
			err = k8sClient.Get(ctx, byoHostLookupKey, updatedByoHost)
			Expect(err).ToNot(HaveOccurred())

			k8sNodeBootstrapSucceeded := conditions.Get(updatedByoHost, infrastructurev1alpha4.K8sNodeBootstrapSucceeded)
			Expect(*k8sNodeBootstrapSucceeded).To(conditions.MatchCondition(clusterv1.Condition{
				Type:     infrastructurev1alpha4.K8sNodeBootstrapSucceeded,
				Status:   corev1.ConditionFalse,
				Reason:   infrastructurev1alpha4.CloudInitExecutionFailedReason,
				Severity: clusterv1.ConditionSeverityError,
			}))

			Expect(k8sClient.Delete(ctx, secret)).NotTo(HaveOccurred())
			Expect(k8sClient.Delete(ctx, byoMachine)).NotTo(HaveOccurred())
		})

		It("should set the Reason to K8sNodeAbsentReason", func() {
			patchHelper, err = patch.NewHelper(byoHost, k8sClient)
			Expect(err).ShouldNot(HaveOccurred())

			if byoHost.Annotations == nil {
				byoHost.Annotations = map[string]string{}
			}
			byoHost.Annotations[hostCleanupAnnotation] = ""
			Expect(patchHelper.Patch(ctx, byoHost, patch.WithStatusObservedGeneration{})).NotTo(HaveOccurred())

			result, reconcilerErr := reconciler.Reconcile(ctx, controllerruntime.Request{
				NamespacedName: byoHostLookupKey,
			})

			Expect(result).To(Equal(controllerruntime.Result{}))
			Expect(reconcilerErr).ToNot(HaveOccurred())

			updatedByoHost := &infrastructurev1alpha4.ByoHost{}
			err = k8sClient.Get(ctx, byoHostLookupKey, updatedByoHost)
			Expect(err).ToNot(HaveOccurred())

			k8sNodeBootstrapSucceeded := conditions.Get(updatedByoHost, infrastructurev1alpha4.K8sNodeBootstrapSucceeded)
			Expect(*k8sNodeBootstrapSucceeded).To(conditions.MatchCondition(clusterv1.Condition{
				Type:     infrastructurev1alpha4.K8sNodeBootstrapSucceeded,
				Status:   corev1.ConditionFalse,
				Reason:   infrastructurev1alpha4.K8sNodeAbsentReason,
				Severity: clusterv1.ConditionSeverityInfo,
			}))

		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, byoHost)).NotTo(HaveOccurred())
		})

	})

	Context("when agent executes the bootstrap secret", func() {
		var (
			byoMachine *infrastructurev1alpha4.ByoMachine
			secret     *corev1.Secret
		)

		BeforeEach(func() {
			byoMachine = common.NewByoMachine("test-byomachine", ns, "", nil)
			Expect(k8sClient.Create(ctx, byoMachine)).NotTo(HaveOccurred(), "failed to create byomachine")

			By("creating the bootstrap secret")
			secretData := `runCmd:
- echo 'some run command'`
			secret = common.NewSecret("test-secret", secretData, ns)
			Expect(k8sClient.Create(ctx, secret)).NotTo(HaveOccurred())

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

			Expect(patchHelper.Patch(ctx, byoHost, patch.WithStatusObservedGeneration{})).NotTo(HaveOccurred())
		})

		It("should set K8sNodeBootstrapSucceeded to True", func() {
			result, reconcilerErr := reconciler.Reconcile(ctx, controllerruntime.Request{
				NamespacedName: byoHostLookupKey,
			})

			Expect(result).To(Equal(controllerruntime.Result{}))
			Expect(reconcilerErr).ToNot(HaveOccurred())

			updatedByoHost := &infrastructurev1alpha4.ByoHost{}
			err = k8sClient.Get(ctx, byoHostLookupKey, updatedByoHost)
			Expect(err).ToNot(HaveOccurred())

			k8sNodeBootstrapSucceeded := conditions.Get(updatedByoHost, infrastructurev1alpha4.K8sNodeBootstrapSucceeded)
			Expect(*k8sNodeBootstrapSucceeded).To(conditions.MatchCondition(clusterv1.Condition{
				Type:   infrastructurev1alpha4.K8sNodeBootstrapSucceeded,
				Status: corev1.ConditionTrue,
			}))
		})

		It("should execute bootstrap secret only once when K8sNodeBootstrapSucceeded is Unknown ", func() {
			result, reconcilerErr := reconciler.Reconcile(ctx, controllerruntime.Request{
				NamespacedName: byoHostLookupKey,
			})

			Expect(result).To(Equal(controllerruntime.Result{}))
			Expect(reconcilerErr).ToNot(HaveOccurred())

			result, reconcilerErr = reconciler.Reconcile(ctx, controllerruntime.Request{
				NamespacedName: byoHostLookupKey,
			})

			Expect(result).To(Equal(controllerruntime.Result{}))
			Expect(reconcilerErr).ToNot(HaveOccurred())

			Expect(fakeCommandRunner.RunCmdCallCount()).To(Equal(1))
		})

		It("should execute bootstrap secret only once when K8sNodeBootstrapSucceeded is False ", func() {
			patchHelper, err = patch.NewHelper(byoHost, k8sClient)
			Expect(err).ShouldNot(HaveOccurred())
			conditions.MarkFalse(byoHost, infrastructurev1alpha4.K8sNodeBootstrapSucceeded, infrastructurev1alpha4.CloudInitExecutionFailedReason, clusterv1.ConditionSeverityError, "")
			Expect(patchHelper.Patch(ctx, byoHost, patch.WithStatusObservedGeneration{})).NotTo(HaveOccurred())

			result, reconcilerErr := reconciler.Reconcile(ctx, controllerruntime.Request{
				NamespacedName: byoHostLookupKey,
			})

			Expect(result).To(Equal(controllerruntime.Result{}))
			Expect(reconcilerErr).ToNot(HaveOccurred())

			result, reconcilerErr = reconciler.Reconcile(ctx, controllerruntime.Request{
				NamespacedName: byoHostLookupKey,
			})

			Expect(result).To(Equal(controllerruntime.Result{}))
			Expect(reconcilerErr).ToNot(HaveOccurred())

			Expect(fakeCommandRunner.RunCmdCallCount()).To(Equal(1))
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, byoHost)).NotTo(HaveOccurred())
			Expect(k8sClient.Delete(ctx, secret)).NotTo(HaveOccurred())
			Expect(k8sClient.Delete(ctx, byoMachine)).NotTo(HaveOccurred())
		})
	})
})
