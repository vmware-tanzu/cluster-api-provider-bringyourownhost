// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package reconciler

import (
	"context"
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/agent/cloudinit/cloudinitfakes"
	infrastructurev1beta1 "github.com/vmware-tanzu/cluster-api-provider-byoh/apis/infrastructure/v1beta1"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/common"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/test/builder"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

var _ = Describe("Byohost Agent Tests", func() {

	var (
		ctx              = context.TODO()
		ns               = "default"
		hostName         = "test-host"
		byoHost          *infrastructurev1beta1.ByoHost
		byoMachine       *infrastructurev1beta1.ByoMachine
		byoHostLookupKey types.NamespacedName
		bootstrapSecret  *corev1.Secret
		recorder         *record.FakeRecorder
	)

	BeforeEach(func() {
		fakeCommandRunner = &cloudinitfakes.FakeICmdRunner{}
		fakeFileWriter = &cloudinitfakes.FakeIFileWriter{}
		fakeTemplateParser = &cloudinitfakes.FakeITemplateParser{}
		recorder = record.NewFakeRecorder(32)
		reconciler = &HostReconciler{
			Client:         k8sClient,
			CmdRunner:      fakeCommandRunner,
			FileWriter:     fakeFileWriter,
			TemplateParser: fakeTemplateParser,
			Recorder:       recorder,
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
			byoHost = builder.ByoHost(ns, hostName).Build()
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

			updatedByoHost := &infrastructurev1beta1.ByoHost{}
			err := k8sClient.Get(ctx, byoHostLookupKey, updatedByoHost)
			Expect(err).ToNot(HaveOccurred())
			k8sNodeBootstrapSucceeded := conditions.Get(updatedByoHost, infrastructurev1beta1.K8sNodeBootstrapSucceeded)
			Expect(*k8sNodeBootstrapSucceeded).To(conditions.MatchCondition(clusterv1.Condition{
				Type:     infrastructurev1beta1.K8sNodeBootstrapSucceeded,
				Status:   corev1.ConditionFalse,
				Reason:   infrastructurev1beta1.WaitingForMachineRefReason,
				Severity: clusterv1.ConditionSeverityInfo,
			}))
		})

		Context("When MachineRef is set", func() {
			BeforeEach(func() {
				byoMachine = builder.ByoMachine(ns, "test-byomachine").Build()
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

				updatedByoHost := &infrastructurev1beta1.ByoHost{}
				err := k8sClient.Get(ctx, byoHostLookupKey, updatedByoHost)
				Expect(err).ToNot(HaveOccurred())

				byoHostRegistrationSucceeded := conditions.Get(updatedByoHost, infrastructurev1beta1.K8sNodeBootstrapSucceeded)
				Expect(*byoHostRegistrationSucceeded).To(conditions.MatchCondition(clusterv1.Condition{
					Type:     infrastructurev1beta1.K8sNodeBootstrapSucceeded,
					Status:   corev1.ConditionFalse,
					Reason:   infrastructurev1beta1.BootstrapDataSecretUnavailableReason,
					Severity: clusterv1.ConditionSeverityInfo,
				}))
			})

			It("should return an error if we fail to load the bootstrap secret", func() {
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

				// assert events
				events := common.CollectEvents(recorder.Events)
				Expect(events).Should(ConsistOf([]string{
					fmt.Sprintf("Warning ReadBootstrapSecretFailed bootstrap secret %s not found", byoHost.Spec.BootstrapSecret.Name),
				}))
			})

			Context("When bootstrap secret is ready", func() {
				BeforeEach(func() {
					secretData := `write_files:
- path: fake/path
  content: blah
runCmd:
- echo 'some run command'`

					bootstrapSecret = builder.Secret(ns, "test-secret").
						WithData(secretData).
						Build()
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

					updatedByoHost := &infrastructurev1beta1.ByoHost{}
					err := k8sClient.Get(ctx, byoHostLookupKey, updatedByoHost)
					Expect(err).ToNot(HaveOccurred())

					k8sNodeBootstrapSucceeded := conditions.Get(updatedByoHost, infrastructurev1beta1.K8sNodeBootstrapSucceeded)
					Expect(*k8sNodeBootstrapSucceeded).To(conditions.MatchCondition(clusterv1.Condition{
						Type:     infrastructurev1beta1.K8sNodeBootstrapSucceeded,
						Status:   corev1.ConditionFalse,
						Reason:   infrastructurev1beta1.CloudInitExecutionFailedReason,
						Severity: clusterv1.ConditionSeverityError,
					}))

					// assert events
					events := common.CollectEvents(recorder.Events)
					Expect(events).Should(ConsistOf([]string{
						"Normal k8sComponentInstalled Successfully Installed K8s components",
						"Warning BootstrapK8sNodeFailed k8s Node Bootstrap failed",
						// TODO: improve test to remove this event
						"Warning ResetK8sNodeFailed k8s Node Reset failed",
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

					updatedByoHost := &infrastructurev1beta1.ByoHost{}
					err := k8sClient.Get(ctx, byoHostLookupKey, updatedByoHost)
					Expect(err).ToNot(HaveOccurred())

					k8sNodeBootstrapSucceeded := conditions.Get(updatedByoHost, infrastructurev1beta1.K8sNodeBootstrapSucceeded)
					Expect(*k8sNodeBootstrapSucceeded).To(conditions.MatchCondition(clusterv1.Condition{
						Type:   infrastructurev1beta1.K8sNodeBootstrapSucceeded,
						Status: corev1.ConditionTrue,
					}))

					// assert events
					events := common.CollectEvents(recorder.Events)
					Expect(events).Should(ConsistOf([]string{
						"Normal k8sComponentInstalled Successfully Installed K8s components",
						"Normal BootstrapK8sNodeSucceeded k8s Node Bootstraped",
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
				byoMachine = builder.ByoMachine(ns, "test-byomachine").Build()
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
					infrastructurev1beta1.HostCleanupAnnotation: "",
					infrastructurev1beta1.K8sVersionAnnotation:  "1.22",
				}
				conditions.MarkTrue(byoHost, infrastructurev1beta1.K8sNodeBootstrapSucceeded)
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
				updatedByoHost := &infrastructurev1beta1.ByoHost{}
				err := k8sClient.Get(ctx, byoHostLookupKey, updatedByoHost)
				Expect(err).ToNot(HaveOccurred())

				Expect(updatedByoHost.Labels).NotTo(HaveKey(clusterv1.ClusterLabelName))
				Expect(updatedByoHost.Status.MachineRef).To(BeNil())
				Expect(updatedByoHost.Annotations).NotTo(HaveKey(infrastructurev1beta1.HostCleanupAnnotation))
				Expect(updatedByoHost.Annotations).NotTo(HaveKey(infrastructurev1beta1.EndPointIPAnnotation))
				Expect(updatedByoHost.Annotations).NotTo(HaveKey(infrastructurev1beta1.K8sVersionAnnotation))

				k8sNodeBootstrapSucceeded := conditions.Get(updatedByoHost, infrastructurev1beta1.K8sNodeBootstrapSucceeded)
				Expect(*k8sNodeBootstrapSucceeded).To(conditions.MatchCondition(clusterv1.Condition{
					Type:     infrastructurev1beta1.K8sNodeBootstrapSucceeded,
					Status:   corev1.ConditionFalse,
					Reason:   infrastructurev1beta1.K8sNodeAbsentReason,
					Severity: clusterv1.ConditionSeverityInfo,
				}))

				// assert events
				events := common.CollectEvents(recorder.Events)
				Expect(events).Should(ConsistOf([]string{
					"Normal ResetK8sNodeSucceeded k8s Node Reset completed",
				}))
			})

			It("should return error if host cleanup failed", func() {
				fakeCommandRunner.RunCmdReturns(errors.New("failed to cleanup host"))

				result, reconcilerErr := reconciler.Reconcile(ctx, controllerruntime.Request{
					NamespacedName: byoHostLookupKey,
				})
				Expect(result).To(Equal(controllerruntime.Result{}))
				Expect(reconcilerErr.Error()).To(Equal("failed to exec kubeadm reset: failed to cleanup host"))

				updatedByoHost := &infrastructurev1beta1.ByoHost{}
				err := k8sClient.Get(ctx, byoHostLookupKey, updatedByoHost)
				Expect(err).ToNot(HaveOccurred())

				k8sNodeBootstrapSucceeded := conditions.Get(updatedByoHost, infrastructurev1beta1.K8sNodeBootstrapSucceeded)
				Expect(*k8sNodeBootstrapSucceeded).To(conditions.MatchCondition(clusterv1.Condition{
					Type:   infrastructurev1beta1.K8sNodeBootstrapSucceeded,
					Status: corev1.ConditionTrue,
				}))

				// assert events
				events := common.CollectEvents(recorder.Events)
				Expect(events).Should(ConsistOf([]string{
					"Warning ResetK8sNodeFailed k8s Node Reset failed",
				}))
			})
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, byoHost)).NotTo(HaveOccurred())
		})
	})
})
