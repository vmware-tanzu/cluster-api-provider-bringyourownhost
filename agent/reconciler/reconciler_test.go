package reconciler

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/agent/cloudinit/cloudinitfakes"
	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/apis/infrastructure/v1alpha4"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
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
		byoMachine       *infrastructurev1alpha4.ByoMachine
		byoHostLookupKey types.NamespacedName
		bootstrapSecret  *corev1.Secret
	)

	BeforeEach(func() {
		fakeCommandRunner = &cloudinitfakes.FakeICmdRunner{}
		fakeFileWriter = &cloudinitfakes.FakeIFileWriter{}
		fakeTemplateParser = &cloudinitfakes.FakeITemplateParser{}

		reconciler = &HostReconciler{
			Client:         k8sClient,
			CmdRunner:      fakeCommandRunner,
			FileWriter:     fakeFileWriter,
			TemplateParser: fakeTemplateParser,
		}
	})

	It("should return an error if ByoHost is not found", func() {
		_, err := reconciler.Reconcile(ctx, controllerruntime.Request{
			NamespacedName: types.NamespacedName{
				Name:      "non-existent-host",
				Namespace: ns},
		})
		Expect(err).To(MatchError("byohosts.infrastructure.cluster.x-k8s.io \"non-existent-host\" not found"))
	})

	Context("When ByoHost exists", func() {
		BeforeEach(func() {
			byoHost = common.NewByoHost(hostName, ns)
			Expect(k8sClient.Create(ctx, byoHost)).NotTo(HaveOccurred(), "failed to create byohost")
			var err error
			patchHelper, err = patch.NewHelper(byoHost, k8sClient)
			Expect(err).ShouldNot(HaveOccurred())

			byoHostLookupKey = types.NamespacedName{Name: byoHost.Name, Namespace: ns}
		})

		It("should set the Reason to WaitingForMachineRefReason if MachineRef isn't found", func() {
			result, reconcilerErr := reconciler.Reconcile(ctx, controllerruntime.Request{
				NamespacedName: byoHostLookupKey,
			})

			Expect(result).To(Equal(controllerruntime.Result{}))
			Expect(reconcilerErr).ToNot(HaveOccurred())

			updatedByoHost := &infrastructurev1alpha4.ByoHost{}
			err := k8sClient.Get(ctx, byoHostLookupKey, updatedByoHost)
			Expect(err).ToNot(HaveOccurred())

			k8sNodeBootstrapSucceeded := conditions.Get(updatedByoHost, infrastructurev1alpha4.K8sNodeBootstrapSucceeded)
			Expect(*k8sNodeBootstrapSucceeded).To(conditions.MatchCondition(clusterv1.Condition{
				Type:     infrastructurev1alpha4.K8sNodeBootstrapSucceeded,
				Status:   corev1.ConditionFalse,
				Reason:   infrastructurev1alpha4.WaitingForMachineRefReason,
				Severity: clusterv1.ConditionSeverityInfo,
			}))
		})

		Context("When MachineRef is set", func() {
			BeforeEach(func() {
				byoMachine = common.NewByoMachine("test-byomachine", ns, "", nil)
				Expect(k8sClient.Create(ctx, byoMachine)).NotTo(HaveOccurred(), "failed to create byomachine")
				byoHost.Status.MachineRef = &corev1.ObjectReference{
					Kind:       "ByoMachine",
					Namespace:  byoMachine.Namespace,
					Name:       byoMachine.Name,
					UID:        byoMachine.UID,
					APIVersion: byoHost.APIVersion,
				}
				Expect(patchHelper.Patch(ctx, byoHost, patch.WithStatusObservedGeneration{})).NotTo(HaveOccurred())
			})

			It("should set the Reason to BootstrapDataSecretUnavailableReason", func() {
				result, reconcilerErr := reconciler.Reconcile(ctx, controllerruntime.Request{
					NamespacedName: byoHostLookupKey,
				})
				Expect(result).To(Equal(controllerruntime.Result{}))
				Expect(reconcilerErr).ToNot(HaveOccurred())

				updatedByoHost := &infrastructurev1alpha4.ByoHost{}
				err := k8sClient.Get(ctx, byoHostLookupKey, updatedByoHost)
				Expect(err).ToNot(HaveOccurred())

				byoHostRegistrationSucceeded := conditions.Get(updatedByoHost, infrastructurev1alpha4.K8sNodeBootstrapSucceeded)
				Expect(*byoHostRegistrationSucceeded).To(conditions.MatchCondition(clusterv1.Condition{
					Type:     infrastructurev1alpha4.K8sNodeBootstrapSucceeded,
					Status:   corev1.ConditionFalse,
					Reason:   infrastructurev1alpha4.BootstrapDataSecretUnavailableReason,
					Severity: clusterv1.ConditionSeverityInfo,
				}))
			})

			It("return an error if we fail to load the bootstrap secret", func() {
				byoHost.Spec.BootstrapSecret = &corev1.ObjectReference{
					Kind:      "Secret",
					Namespace: "non-existent",
					Name:      "non-existent",
				}
				Expect(patchHelper.Patch(ctx, byoHost, patch.WithStatusObservedGeneration{})).NotTo(HaveOccurred())

				result, reconcilerErr := reconciler.Reconcile(ctx, controllerruntime.Request{
					NamespacedName: byoHostLookupKey,
				})
				Expect(result).To(Equal(controllerruntime.Result{}))
				Expect(reconcilerErr).To(MatchError("secrets \"non-existent\" not found"))
			})

			Context("When bootstrap secret is ready", func() {
				BeforeEach(func() {
					secretData := `write_files:
- path: fake/path
  content: blah
runCmd:
- echo 'some run command'`
					bootstrapSecret = common.NewSecret("test-secret", secretData, ns)
					Expect(k8sClient.Create(ctx, bootstrapSecret)).NotTo(HaveOccurred())

					byoHost.Spec.BootstrapSecret = &corev1.ObjectReference{
						Kind:      "Secret",
						Namespace: bootstrapSecret.Namespace,
						Name:      bootstrapSecret.Name,
					}

					Expect(patchHelper.Patch(ctx, byoHost, patch.WithStatusObservedGeneration{})).NotTo(HaveOccurred())
				})

				It("should set the Reason to CloudInitExecutionFailedReason if the boostrap execution fails", func() {
					fakeCommandRunner.RunCmdReturns(errors.New("I failed"))

					result, reconcilerErr := reconciler.Reconcile(ctx, controllerruntime.Request{
						NamespacedName: byoHostLookupKey,
					})

					Expect(result).To(Equal(controllerruntime.Result{}))
					Expect(reconcilerErr).To(HaveOccurred())

					updatedByoHost := &infrastructurev1alpha4.ByoHost{}
					err := k8sClient.Get(ctx, byoHostLookupKey, updatedByoHost)
					Expect(err).ToNot(HaveOccurred())

					k8sNodeBootstrapSucceeded := conditions.Get(updatedByoHost, infrastructurev1alpha4.K8sNodeBootstrapSucceeded)
					Expect(*k8sNodeBootstrapSucceeded).To(conditions.MatchCondition(clusterv1.Condition{
						Type:     infrastructurev1alpha4.K8sNodeBootstrapSucceeded,
						Status:   corev1.ConditionFalse,
						Reason:   infrastructurev1alpha4.CloudInitExecutionFailedReason,
						Severity: clusterv1.ConditionSeverityError,
					}))
				})

				It("should set K8sNodeBootstrapSucceeded to True if the boostrap execution succeeds", func() {
					result, reconcilerErr := reconciler.Reconcile(ctx, controllerruntime.Request{
						NamespacedName: byoHostLookupKey,
					})
					Expect(result).To(Equal(controllerruntime.Result{}))
					Expect(reconcilerErr).ToNot(HaveOccurred())

					Expect(fakeCommandRunner.RunCmdCallCount()).To(Equal(1))
					Expect(fakeFileWriter.WriteToFileCallCount()).To(Equal(1))

					updatedByoHost := &infrastructurev1alpha4.ByoHost{}
					err := k8sClient.Get(ctx, byoHostLookupKey, updatedByoHost)
					Expect(err).ToNot(HaveOccurred())

					k8sNodeBootstrapSucceeded := conditions.Get(updatedByoHost, infrastructurev1alpha4.K8sNodeBootstrapSucceeded)
					Expect(*k8sNodeBootstrapSucceeded).To(conditions.MatchCondition(clusterv1.Condition{
						Type:   infrastructurev1alpha4.K8sNodeBootstrapSucceeded,
						Status: corev1.ConditionTrue,
					}))
				})

				It("should execute bootstrap secret only once ", func() {
					_, reconcilerErr := reconciler.Reconcile(ctx, controllerruntime.Request{
						NamespacedName: byoHostLookupKey,
					})
					Expect(reconcilerErr).ToNot(HaveOccurred())

					_, reconcilerErr = reconciler.Reconcile(ctx, controllerruntime.Request{
						NamespacedName: byoHostLookupKey,
					})
					Expect(reconcilerErr).ToNot(HaveOccurred())

					Expect(fakeCommandRunner.RunCmdCallCount()).To(Equal(1))
					Expect(fakeFileWriter.WriteToFileCallCount()).To(Equal(1))
				})

				AfterEach(func() {
					Expect(k8sClient.Delete(ctx, bootstrapSecret)).NotTo(HaveOccurred())
				})
			})

			AfterEach(func() {
				Expect(k8sClient.Delete(ctx, byoMachine)).NotTo(HaveOccurred())
			})
		})

		Context("When the ByoHost is marked for cleanup", func() {
			BeforeEach(func() {
				byoMachine = common.NewByoMachine("test-byomachine", ns, "", nil)
				Expect(k8sClient.Create(ctx, byoMachine)).NotTo(HaveOccurred(), "failed to create byomachine")
				byoHost.Status.MachineRef = &corev1.ObjectReference{
					Kind:       "ByoMachine",
					Namespace:  byoMachine.Namespace,
					Name:       byoMachine.Name,
					UID:        byoMachine.UID,
					APIVersion: byoHost.APIVersion,
				}
				byoHost.Labels = map[string]string{clusterv1.ClusterLabelName: "test-cluster"}
				byoHost.Annotations = map[string]string{
					infrastructurev1alpha4.HostCleanupAnnotation:    "",
					infrastructurev1alpha4.ClusterVersionAnnotation: "1.22",
				}
				conditions.MarkTrue(byoHost, infrastructurev1alpha4.K8sNodeBootstrapSucceeded)
				Expect(patchHelper.Patch(ctx, byoHost, patch.WithStatusObservedGeneration{})).NotTo(HaveOccurred())
			})

			It("should reset the node and set the Reason to K8sNodeAbsentReason", func() {
				result, reconcilerErr := reconciler.Reconcile(ctx, controllerruntime.Request{
					NamespacedName: byoHostLookupKey,
				})
				Expect(result).To(Equal(controllerruntime.Result{}))
				Expect(reconcilerErr).ToNot(HaveOccurred())

				Expect(fakeCommandRunner.RunCmdCallCount()).To(Equal(1))
				Expect(fakeCommandRunner.RunCmdArgsForCall(0)).To(Equal(KubeadmResetCommand))

				updatedByoHost := &infrastructurev1alpha4.ByoHost{}
				err := k8sClient.Get(ctx, byoHostLookupKey, updatedByoHost)
				Expect(err).ToNot(HaveOccurred())

				Expect(updatedByoHost.Labels).NotTo(HaveKey(clusterv1.ClusterLabelName))
				Expect(updatedByoHost.Status.MachineRef).To(BeNil())
				Expect(updatedByoHost.Annotations).NotTo(HaveKey(infrastructurev1alpha4.HostCleanupAnnotation))
				Expect(updatedByoHost.Annotations).NotTo(HaveKey(infrastructurev1alpha4.EndPointIPAnnotation))
				Expect(updatedByoHost.Annotations).NotTo(HaveKey(infrastructurev1alpha4.ClusterVersionAnnotation))

				k8sNodeBootstrapSucceeded := conditions.Get(updatedByoHost, infrastructurev1alpha4.K8sNodeBootstrapSucceeded)
				Expect(*k8sNodeBootstrapSucceeded).To(conditions.MatchCondition(clusterv1.Condition{
					Type:     infrastructurev1alpha4.K8sNodeBootstrapSucceeded,
					Status:   corev1.ConditionFalse,
					Reason:   infrastructurev1alpha4.K8sNodeAbsentReason,
					Severity: clusterv1.ConditionSeverityInfo,
				}))
			})

			It("should return error if host cleanup failed", func() {
				fakeCommandRunner.RunCmdReturns(errors.New("failed to cleanup host"))

				result, reconcilerErr := reconciler.Reconcile(ctx, controllerruntime.Request{
					NamespacedName: byoHostLookupKey,
				})
				Expect(result).To(Equal(controllerruntime.Result{}))
				Expect(reconcilerErr.Error()).To(Equal("failed to exec kubeadm reset: failed to cleanup host"))

				updatedByoHost := &infrastructurev1alpha4.ByoHost{}
				err := k8sClient.Get(ctx, byoHostLookupKey, updatedByoHost)
				Expect(err).ToNot(HaveOccurred())

				k8sNodeBootstrapSucceeded := conditions.Get(updatedByoHost, infrastructurev1alpha4.K8sNodeBootstrapSucceeded)
				Expect(*k8sNodeBootstrapSucceeded).To(conditions.MatchCondition(clusterv1.Condition{
					Type:   infrastructurev1alpha4.K8sNodeBootstrapSucceeded,
					Status: corev1.ConditionTrue,
				}))
			})
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, byoHost)).NotTo(HaveOccurred())
		})
	})
})
