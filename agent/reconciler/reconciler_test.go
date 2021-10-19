// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package reconciler_test

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/agent/cloudinit/cloudinitfakes"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/agent/reconciler"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/agent/reconciler/reconcilerfakes"
	infrastructurev1beta1 "github.com/vmware-tanzu/cluster-api-provider-byoh/apis/infrastructure/v1beta1"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/test/builder"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
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
	)

	BeforeEach(func() {
		fakeCommandRunner = &cloudinitfakes.FakeICmdRunner{}
		fakeFileWriter = &cloudinitfakes.FakeIFileWriter{}
		fakeTemplateParser = &cloudinitfakes.FakeITemplateParser{}
		fakeInstaller = &reconcilerfakes.FakeInstaller{}

		hostReconciler = &reconciler.HostReconciler{
			Client:         k8sClient,
			CmdRunner:      fakeCommandRunner,
			FileWriter:     fakeFileWriter,
			TemplateParser: fakeTemplateParser,
			K8sInstaller:   fakeInstaller,
		}
	})

	It("should return an error if ByoHost is not found", func() {
		_, err := hostReconciler.Reconcile(ctx, controllerruntime.Request{
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
			result, reconcilerErr := hostReconciler.Reconcile(ctx, controllerruntime.Request{
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
				result, reconcilerErr := hostReconciler.Reconcile(ctx, controllerruntime.Request{
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

				result, reconcilerErr := hostReconciler.Reconcile(ctx, controllerruntime.Request{
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

				It("should install k8s components", func() {
					result, reconcilerErr := hostReconciler.Reconcile(ctx, controllerruntime.Request{
						NamespacedName: byoHostLookupKey,
					})
					Expect(result).To(Equal(controllerruntime.Result{}))
					Expect(reconcilerErr).ToNot(HaveOccurred())
				})

				It("should set K8sComponentsInstallationSucceeded to false with Reason K8sComponentsInstallationFailedReason if Install fails", func() {
					fakeInstaller.InstallReturns(errors.New("k8s components install failed"))
					result, reconcilerErr := hostReconciler.Reconcile(ctx, controllerruntime.Request{
						NamespacedName: byoHostLookupKey,
					})
					Expect(result).To(Equal(controllerruntime.Result{}))
					Expect(reconcilerErr).To(HaveOccurred())

					updatedByoHost := &infrastructurev1beta1.ByoHost{}
					err := k8sClient.Get(ctx, byoHostLookupKey, updatedByoHost)
					Expect(err).ToNot(HaveOccurred())

					k8sComponentsInstallationSucceeded := conditions.Get(updatedByoHost, infrastructurev1beta1.K8sComponentsInstallationSucceeded)
					Expect(*k8sComponentsInstallationSucceeded).To(conditions.MatchCondition(clusterv1.Condition{
						Type:     infrastructurev1beta1.K8sComponentsInstallationSucceeded,
						Status:   corev1.ConditionFalse,
						Reason:   infrastructurev1beta1.K8sComponentsInstallationFailedReason,
						Severity: clusterv1.ConditionSeverityInfo,
					}))
				})

				It("should set K8sNodeBootstrapSucceeded to false with Reason CloudInitExecutionFailedReason if the bootstrap execution fails", func() {
					fakeCommandRunner.RunCmdReturns(errors.New("I failed"))

					result, reconcilerErr := hostReconciler.Reconcile(ctx, controllerruntime.Request{
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
				})

				It("should set K8sNodeBootstrapSucceeded to True if the boostrap execution succeeds", func() {
					result, reconcilerErr := hostReconciler.Reconcile(ctx, controllerruntime.Request{
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
				})

				It("should execute bootstrap secret only once ", func() {
					_, reconcilerErr := hostReconciler.Reconcile(ctx, controllerruntime.Request{
						NamespacedName: byoHostLookupKey,
					})
					Expect(reconcilerErr).ToNot(HaveOccurred())

					_, reconcilerErr = hostReconciler.Reconcile(ctx, controllerruntime.Request{
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
				k8sVersion := byoHost.Annotations[infrastructurev1beta1.K8sVersionAnnotation]
				result, reconcilerErr := hostReconciler.Reconcile(ctx, controllerruntime.Request{
					NamespacedName: byoHostLookupKey,
				})
				Expect(result).To(Equal(controllerruntime.Result{}))
				Expect(reconcilerErr).ToNot(HaveOccurred())

				// assert kubeadm reset is called
				Expect(fakeCommandRunner.RunCmdCallCount()).To(Equal(1))
				Expect(fakeCommandRunner.RunCmdArgsForCall(0)).To(Equal(reconciler.KubeadmResetCommand))

				updatedByoHost := &infrastructurev1beta1.ByoHost{}
				err := k8sClient.Get(ctx, byoHostLookupKey, updatedByoHost)
				Expect(err).ToNot(HaveOccurred())

				// assert installer.Uninstall is called
				Expect(fakeInstaller.UninstallCallCount()).To(Equal(1))
				Expect(fakeInstaller.UninstallArgsForCall(0)).To(Equal(k8sVersion))

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
			})

			It("should return error if host cleanup failed", func() {
				fakeCommandRunner.RunCmdReturns(errors.New("failed to cleanup host"))

				result, reconcilerErr := hostReconciler.Reconcile(ctx, controllerruntime.Request{
					NamespacedName: byoHostLookupKey,
				})
				Expect(result).To(Equal(controllerruntime.Result{}))
				Expect(reconcilerErr.Error()).To(Equal("failed to exec kubeadm reset: failed to cleanup host"))

				updatedByoHost := &infrastructurev1beta1.ByoHost{}
				err := k8sClient.Get(ctx, byoHostLookupKey, updatedByoHost)
				Expect(err).ToNot(HaveOccurred())

				// assert if k8sNodeBootstrapSucceeded is still True
				k8sNodeBootstrapSucceeded := conditions.Get(updatedByoHost, infrastructurev1beta1.K8sNodeBootstrapSucceeded)
				Expect(*k8sNodeBootstrapSucceeded).To(conditions.MatchCondition(clusterv1.Condition{
					Type:   infrastructurev1beta1.K8sNodeBootstrapSucceeded,
					Status: corev1.ConditionTrue,
				}))
			})

			It("should return error if uninstall failed", func() {
				fakeInstaller.UninstallReturns(errors.New("k8s components uninstall failed"))

				result, reconcilerErr := hostReconciler.Reconcile(ctx, controllerruntime.Request{
					NamespacedName: byoHostLookupKey,
				})
				Expect(result).To(Equal(controllerruntime.Result{}))
				Expect(reconcilerErr.Error()).To(Equal("k8s components uninstall failed"))

				updatedByoHost := &infrastructurev1beta1.ByoHost{}
				err := k8sClient.Get(ctx, byoHostLookupKey, updatedByoHost)
				Expect(err).ToNot(HaveOccurred())

				// assert if k8sNodeBootstrapSucceeded is still True
				k8sNodeBootstrapSucceeded := conditions.Get(updatedByoHost, infrastructurev1beta1.K8sNodeBootstrapSucceeded)
				Expect(*k8sNodeBootstrapSucceeded).To(conditions.MatchCondition(clusterv1.Condition{
					Type:   infrastructurev1beta1.K8sNodeBootstrapSucceeded,
					Status: corev1.ConditionTrue,
				}))
			})
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, byoHost)).NotTo(HaveOccurred())
		})
	})
})
