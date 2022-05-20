// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controllers_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	infrav1 "github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/apis/infrastructure/v1beta1"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/test/builder"
	eventutils "github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/test/utils/events"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Controllers/K8sInstallerConfigController", func() {
	var (
		ctx                         context.Context
		byoMachine                  *infrav1.ByoMachine
		k8sinstallerConfig          *infrav1.K8sInstallerConfig
		k8sinstallerConfigTemplate  *infrav1.K8sInstallerConfigTemplate
		machine                     *clusterv1.Machine
		k8sClientUncached           client.Client
		byoMachineLookupKey         types.NamespacedName
		k8sInstallerConfigLookupKey types.NamespacedName
		installerSecretLookupKey    types.NamespacedName
		testClusterVersion          = "v1.22.1_xyz"
		testBundleRepo              = "test-repo"
		testBundleType              = "k8s"
	)

	BeforeEach(func() {
		ctx = context.Background()

		var clientErr error
		k8sClientUncached, clientErr = client.New(cfg, client.Options{Scheme: scheme.Scheme})
		Expect(clientErr).NotTo(HaveOccurred())

		machine = builder.Machine(defaultNamespace, defaultMachineName).
			WithClusterName(defaultClusterName).
			WithClusterVersion(testClusterVersion).
			WithBootstrapDataSecret(fakeBootstrapSecret).
			Build()
		Expect(k8sClientUncached.Create(ctx, machine)).Should(Succeed())

		byoMachine = builder.ByoMachine(defaultNamespace, defaultByoMachineName).
			WithClusterLabel(defaultClusterName).
			WithOwnerMachine(machine).
			Build()
		Expect(k8sClientUncached.Create(ctx, byoMachine)).Should(Succeed())

		k8sinstallerConfigTemplate = builder.K8sInstallerConfigTemplate(defaultNamespace, defaultK8sInstallerConfigName).
			WithBundleRepo(testBundleRepo).
			WithBundleType(testBundleType).
			Build()
		Expect(k8sClientUncached.Create(ctx, k8sinstallerConfigTemplate)).Should(Succeed())

		k8sinstallerConfig = builder.K8sInstallerConfig(defaultNamespace, defaultK8sInstallerConfigName).
			WithClusterLabel(defaultClusterName).
			WithOwnerByoMachine(byoMachine).
			WithBundleRepo(testBundleRepo).
			WithBundleType(testBundleType).
			Build()
		Expect(k8sClientUncached.Create(ctx, k8sinstallerConfig)).Should(Succeed())

		WaitForObjectsToBePopulatedInCache(machine, byoMachine, k8sinstallerConfig, k8sinstallerConfigTemplate)

		byoMachineLookupKey = types.NamespacedName{Name: byoMachine.Name, Namespace: byoMachine.Namespace}
		k8sInstallerConfigLookupKey = types.NamespacedName{Name: k8sinstallerConfig.Name, Namespace: k8sinstallerConfig.Namespace}
		installerSecretLookupKey = types.NamespacedName{Name: k8sinstallerConfig.Name, Namespace: k8sinstallerConfig.Namespace}
	})

	AfterEach(func() {
		eventutils.DrainEvents(recorder.Events)
	})

	It("should ignore k8sinstallerconfig if it is not found", func() {
		_, err := k8sInstallerConfigReconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      "non-existent-k8sinstallerconfig",
				Namespace: "non-existent-k8sinstallerconfig"}})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should ignore when owner is not set", func() {

		k8sinstallerconfigWithNoOwner := builder.K8sInstallerConfig(defaultNamespace, defaultK8sInstallerConfigName).
			WithClusterLabel(defaultClusterName).
			Build()
		Expect(k8sClientUncached.Create(ctx, k8sinstallerconfigWithNoOwner)).Should(Succeed())

		WaitForObjectsToBePopulatedInCache(k8sinstallerconfigWithNoOwner)

		_, err := k8sInstallerConfigReconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      k8sinstallerconfigWithNoOwner.Name,
				Namespace: k8sinstallerconfigWithNoOwner.Namespace}})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should return error when byomachine does not contain cluster name", func() {

		byoMachineWithNonExistingCluster := builder.ByoMachine(defaultNamespace, defaultByoMachineName).
			WithOwnerMachine(machine).
			Build()
		Expect(k8sClientUncached.Create(ctx, byoMachineWithNonExistingCluster)).Should(Succeed())

		k8sinstallerconfigWithNonExistingCluster := builder.K8sInstallerConfig(defaultNamespace, defaultK8sInstallerConfigName).
			WithClusterLabel("non-existent-cluster").
			WithOwnerByoMachine(byoMachineWithNonExistingCluster).
			Build()
		Expect(k8sClientUncached.Create(ctx, k8sinstallerconfigWithNonExistingCluster)).Should(Succeed())

		WaitForObjectsToBePopulatedInCache(byoMachineWithNonExistingCluster, k8sinstallerconfigWithNonExistingCluster)

		_, err := k8sInstallerConfigReconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      k8sinstallerconfigWithNonExistingCluster.Name,
				Namespace: k8sinstallerconfigWithNonExistingCluster.Namespace}})
		Expect(err).To(MatchError(util.ErrNoCluster))
	})

	It("should return error when cluster does not exist", func() {

		byoMachineWithNonExistingCluster := builder.ByoMachine(defaultNamespace, defaultByoMachineName).
			WithClusterLabel("non-existent-cluster").
			WithOwnerMachine(machine).
			Build()
		Expect(k8sClientUncached.Create(ctx, byoMachineWithNonExistingCluster)).Should(Succeed())

		k8sinstallerconfigWithNonExistingCluster := builder.K8sInstallerConfig(defaultNamespace, defaultK8sInstallerConfigName).
			WithClusterLabel("non-existent-cluster").
			WithOwnerByoMachine(byoMachineWithNonExistingCluster).
			Build()
		Expect(k8sClientUncached.Create(ctx, k8sinstallerconfigWithNonExistingCluster)).Should(Succeed())

		WaitForObjectsToBePopulatedInCache(byoMachineWithNonExistingCluster, k8sinstallerconfigWithNonExistingCluster)

		_, err := k8sInstallerConfigReconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      k8sinstallerconfigWithNonExistingCluster.Name,
				Namespace: k8sinstallerconfigWithNonExistingCluster.Namespace}})
		Expect(err).To(MatchError("failed to get Cluster/non-existent-cluster: Cluster.cluster.x-k8s.io \"non-existent-cluster\" not found"))
	})

	It("should ignore when k8sinstallerconfig is paused", func() {

		ph, err := patch.NewHelper(k8sinstallerConfig, k8sClientUncached)
		Expect(err).ShouldNot(HaveOccurred())
		pauseAnnotations := map[string]string{
			clusterv1.PausedAnnotation: "paused",
		}
		annotations.AddAnnotations(k8sinstallerConfig, pauseAnnotations)
		Expect(ph.Patch(ctx, k8sinstallerConfig, patch.WithStatusObservedGeneration{})).Should(Succeed())
		WaitForObjectToBeUpdatedInCache(k8sinstallerConfig, func(object client.Object) bool {
			return annotations.HasPausedAnnotation(object.(*infrav1.K8sInstallerConfig))
		})

		_, err = k8sInstallerConfigReconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      k8sinstallerConfig.Name,
				Namespace: k8sinstallerConfig.Namespace}})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should ignore when byomachine is not waiting for InstallationSecret", func() {
		_, err := k8sInstallerConfigReconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      k8sinstallerConfig.Name,
				Namespace: k8sinstallerConfig.Namespace}})
		Expect(err).NotTo(HaveOccurred())
	})

	Context("When ByoMachine wait for InstallerSecret", func() {

		BeforeEach(func() {
			ph, err := patch.NewHelper(byoMachine, k8sClientUncached)
			Expect(err).ShouldNot(HaveOccurred())
			byoMachine.Status.HostInfo = infrav1.HostInfo{
				Architecture: "amd64",
				OSName:       "linux",
				OSImage:      "Ubuntu 20.04.1 LTS",
			}
			conditions.Set(byoMachine, &clusterv1.Condition{
				Type:   infrav1.BYOHostReady,
				Reason: infrav1.InstallationSecretNotAvailableReason,
			})
			Expect(ph.Patch(ctx, byoMachine, patch.WithStatusObservedGeneration{})).Should(Succeed())
			WaitForObjectToBeUpdatedInCache(byoMachine, func(object client.Object) bool {
				return object.(*infrav1.ByoMachine).Status.HostInfo.Architecture == "amd64"
			})
		})

		It("should add K8sInstallerConfigFinalizer on K8sInstallerConfig", func() {
			_, err := k8sInstallerConfigReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      k8sinstallerConfig.Name,
					Namespace: k8sinstallerConfig.Namespace}})
			Expect(err).NotTo(HaveOccurred())

			updatedConfig := &infrav1.K8sInstallerConfig{}
			err = k8sClientUncached.Get(ctx, k8sInstallerConfigLookupKey, updatedConfig)
			Expect(err).ToNot(HaveOccurred())
			Expect(controllerutil.ContainsFinalizer(updatedConfig, infrav1.K8sInstallerConfigFinalizer)).To(BeTrue())
		})

		It("should ignore when K8sInstallerConfig status is ready", func() {

			ph, err := patch.NewHelper(k8sinstallerConfig, k8sClientUncached)
			Expect(err).ShouldNot(HaveOccurred())
			k8sinstallerConfig.Status.Ready = true
			Expect(ph.Patch(ctx, k8sinstallerConfig, patch.WithStatusObservedGeneration{})).Should(Succeed())
			WaitForObjectToBeUpdatedInCache(k8sinstallerConfig, func(object client.Object) bool {
				return object.(*infrav1.K8sInstallerConfig).Status.Ready
			})

			_, err = k8sInstallerConfigReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      k8sinstallerConfig.Name,
					Namespace: k8sinstallerConfig.Namespace}})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should throw error if os distribution is not supported", func() {
			ph, err := patch.NewHelper(byoMachine, k8sClientUncached)
			Expect(err).ShouldNot(HaveOccurred())
			unsupportedOsDist := "unsupportedOsDist"
			byoMachine.Status.HostInfo.OSImage = unsupportedOsDist
			Expect(ph.Patch(ctx, byoMachine, patch.WithStatusObservedGeneration{})).Should(Succeed())
			WaitForObjectToBeUpdatedInCache(byoMachine, func(object client.Object) bool {
				return object.(*infrav1.ByoMachine).Status.HostInfo.OSImage == unsupportedOsDist
			})

			_, err = k8sInstallerConfigReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      k8sinstallerConfig.Name,
					Namespace: k8sinstallerConfig.Namespace}})
			Expect(err).Should(MatchError("No k8s support for OS"))
		})

		It("should throw error if architecture is not supported", func() {
			ph, err := patch.NewHelper(byoMachine, k8sClientUncached)
			Expect(err).ShouldNot(HaveOccurred())
			unsupportedArch := "unsupportedArch"
			byoMachine.Status.HostInfo.Architecture = unsupportedArch
			Expect(ph.Patch(ctx, byoMachine, patch.WithStatusObservedGeneration{})).Should(Succeed())
			WaitForObjectToBeUpdatedInCache(byoMachine, func(object client.Object) bool {
				return object.(*infrav1.ByoMachine).Status.HostInfo.Architecture == unsupportedArch
			})

			_, err = k8sInstallerConfigReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      k8sinstallerConfig.Name,
					Namespace: k8sinstallerConfig.Namespace}})
			Expect(err).Should(MatchError("No k8s support for OS"))
		})

		It("should create secret of same name as of K8sInstallerConfig", func() {
			_, err := k8sInstallerConfigReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      k8sinstallerConfig.Name,
					Namespace: k8sinstallerConfig.Namespace}})
			Expect(err).NotTo(HaveOccurred())

			createdSecret := &corev1.Secret{}
			err = k8sClientUncached.Get(ctx, installerSecretLookupKey, createdSecret)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should create secret with data fields install and uninstall", func() {
			_, err := k8sInstallerConfigReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      k8sinstallerConfig.Name,
					Namespace: k8sinstallerConfig.Namespace}})
			Expect(err).NotTo(HaveOccurred())

			createdSecret := &corev1.Secret{}
			err = k8sClientUncached.Get(ctx, installerSecretLookupKey, createdSecret)
			Expect(err).ToNot(HaveOccurred())
			_, exists := createdSecret.Data["install"]
			Expect(exists).To(BeTrue())
			_, exists = createdSecret.Data["uninstall"]
			Expect(exists).To(BeTrue())
		})

		It("should be add secret reference to K8sInstallerConfig", func() {
			_, err := k8sInstallerConfigReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      k8sinstallerConfig.Name,
					Namespace: k8sinstallerConfig.Namespace}})
			Expect(err).NotTo(HaveOccurred())

			updatedConfig := &infrav1.K8sInstallerConfig{}
			err = k8sClientUncached.Get(ctx, k8sInstallerConfigLookupKey, updatedConfig)
			Expect(err).ToNot(HaveOccurred())

			createdSecret := &corev1.Secret{}
			err = k8sClientUncached.Get(ctx, installerSecretLookupKey, createdSecret)
			Expect(err).ToNot(HaveOccurred())

			Expect(updatedConfig.Status.InstallationSecret.Name).Should(Equal(createdSecret.Name))
			Expect(updatedConfig.Status.InstallationSecret.Namespace).Should(Equal(createdSecret.Namespace))
		})

		It("should be add secret reference to K8sInstallerConfig even if secret already exists", func() {

			createdSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      k8sinstallerConfig.Name,
					Namespace: k8sinstallerConfig.Namespace,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: infrav1.GroupVersion.String(),
							Kind:       "K8sInstallerConfig",
							Name:       k8sinstallerConfig.Name,
							UID:        k8sinstallerConfig.UID,
							Controller: pointer.BoolPtr(true),
						},
					},
				},
				Data: map[string][]byte{
					"install":   []byte("dummy install"),
					"uninstall": []byte("dummy uninstall"),
				},
				Type: clusterv1.ClusterSecretType,
			}
			err := k8sClientUncached.Create(ctx, createdSecret)
			Expect(err).NotTo(HaveOccurred())

			_, err = k8sInstallerConfigReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      k8sinstallerConfig.Name,
					Namespace: k8sinstallerConfig.Namespace}})
			Expect(err).NotTo(HaveOccurred())

			updatedConfig := &infrav1.K8sInstallerConfig{}
			err = k8sClientUncached.Get(ctx, k8sInstallerConfigLookupKey, updatedConfig)
			Expect(err).ToNot(HaveOccurred())

			Expect(updatedConfig.Status.InstallationSecret.Name).Should(Equal(createdSecret.Name))
			Expect(updatedConfig.Status.InstallationSecret.Namespace).Should(Equal(createdSecret.Namespace))
		})

		It("should be make K8sInstallerConfig ready after secret creation", func() {
			_, err := k8sInstallerConfigReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      k8sinstallerConfig.Name,
					Namespace: k8sinstallerConfig.Namespace}})
			Expect(err).NotTo(HaveOccurred())

			updatedConfig := &infrav1.K8sInstallerConfig{}
			err = k8sClientUncached.Get(ctx, k8sInstallerConfigLookupKey, updatedConfig)
			Expect(err).ToNot(HaveOccurred())

			Expect(updatedConfig.Status.Ready).To(BeTrue())
		})

		Context("When K8sInstallerConfig is deleted", func() {
			BeforeEach(func() {
				_, err := k8sInstallerConfigReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      k8sinstallerConfig.Name,
						Namespace: k8sinstallerConfig.Namespace}})
				Expect(err).NotTo(HaveOccurred())

				Expect(k8sClientUncached.Delete(ctx, k8sinstallerConfig)).Should(Succeed())

				WaitForObjectToBeUpdatedInCache(k8sinstallerConfig, func(object client.Object) bool {
					return !object.(*infrav1.K8sInstallerConfig).ObjectMeta.DeletionTimestamp.IsZero()
				})
			})

			It("should delete the k8sInstallerConfig object", func() {
				deletedConfig := &infrav1.K8sInstallerConfig{}
				// assert K8sInstallerConfig Exists before reconcile
				Expect(k8sClientUncached.Get(ctx, k8sInstallerConfigLookupKey, deletedConfig)).Should(Not(HaveOccurred()))
				_, err := k8sInstallerConfigReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      k8sinstallerConfig.Name,
						Namespace: k8sinstallerConfig.Namespace}})
				Expect(err).NotTo(HaveOccurred())

				// assert K8sInstallerConfig does not exists
				err = k8sClientUncached.Get(ctx, k8sInstallerConfigLookupKey, deletedConfig)
				Expect(err).To(MatchError(fmt.Sprintf("k8sinstallerconfigs.infrastructure.cluster.x-k8s.io %q not found", k8sInstallerConfigLookupKey.Name)))
			})

			Context("When owner ByoMachine not found", func() {
				BeforeEach(func() {
					ph, err := patch.NewHelper(byoMachine, k8sClientUncached)
					Expect(err).ShouldNot(HaveOccurred())
					controllerutil.AddFinalizer(byoMachine, infrav1.MachineFinalizer)
					Expect(ph.Patch(ctx, byoMachine, patch.WithStatusObservedGeneration{})).Should(Succeed())

					Expect(k8sClientUncached.Delete(ctx, byoMachine)).Should(Succeed())
					WaitForObjectToBeUpdatedInCache(byoMachine, func(object client.Object) bool {
						return !object.(*infrav1.ByoMachine).ObjectMeta.DeletionTimestamp.IsZero()
					})
					_, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: byoMachineLookupKey})
					Expect(err).ToNot(HaveOccurred())
				})

				It("should delete the k8sInstallerConfig object", func() {
					deletedConfig := &infrav1.K8sInstallerConfig{}
					// assert K8sInstallerConfig Exists before reconcile
					Expect(k8sClientUncached.Get(ctx, k8sInstallerConfigLookupKey, deletedConfig)).Should(Not(HaveOccurred()))
					_, err := k8sInstallerConfigReconciler.Reconcile(ctx, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Name:      k8sinstallerConfig.Name,
							Namespace: k8sinstallerConfig.Namespace}})
					Expect(err).NotTo(HaveOccurred())

					// assert K8sInstallerConfig does not exists
					err = k8sClientUncached.Get(ctx, k8sInstallerConfigLookupKey, deletedConfig)
					Expect(err).To(MatchError(fmt.Sprintf("k8sinstallerconfigs.infrastructure.cluster.x-k8s.io %q not found", k8sInstallerConfigLookupKey.Name)))
				})
			})
		})
	})

	Context("ByoMachine to K8sInstallerConfig reconcile request", func() {
		It("should not return reconcile request if ByoMachine InstallerRef doesn't exists", func() {
			result := k8sInstallerConfigReconciler.ByoMachineToK8sInstallerConfigMapFunc(byoMachine)
			Expect(len(result)).To(BeZero())
		})

		It("should not return reconcile request if ByoMachine InstallerRef doesn't refer to K8sInstallerConfitTemplate", func() {
			ph, err := patch.NewHelper(byoMachine, k8sClientUncached)
			Expect(err).ShouldNot(HaveOccurred())
			byoMachine.Spec.InstallerRef = &corev1.ObjectReference{
				Kind:      "RandomInstallerTemplate",
				Name:      k8sinstallerConfig.Name,
				Namespace: k8sinstallerConfig.Namespace,
			}
			Expect(ph.Patch(ctx, byoMachine, patch.WithStatusObservedGeneration{})).Should(Succeed())
			WaitForObjectToBeUpdatedInCache(byoMachine, func(object client.Object) bool {
				return object.(*infrav1.ByoMachine).Spec.InstallerRef != nil
			})

			result := k8sInstallerConfigReconciler.ByoMachineToK8sInstallerConfigMapFunc(byoMachine)
			Expect(len(result)).To(BeZero())
		})

		It("should return reconcile request if ByoMachine refer to K8sInstallerConfigTemplate installer", func() {
			ph, err := patch.NewHelper(byoMachine, k8sClientUncached)
			Expect(err).ShouldNot(HaveOccurred())
			byoMachine.Spec.InstallerRef = &corev1.ObjectReference{
				Kind:       "K8sInstallerConfigTemplate",
				Name:       k8sinstallerConfigTemplate.Name,
				Namespace:  k8sinstallerConfigTemplate.Namespace,
				APIVersion: infrav1.GroupVersion.String(),
			}
			Expect(ph.Patch(ctx, byoMachine, patch.WithStatusObservedGeneration{})).Should(Succeed())
			WaitForObjectToBeUpdatedInCache(byoMachine, func(object client.Object) bool {
				return object.(*infrav1.ByoMachine).Spec.InstallerRef != nil
			})

			result := k8sInstallerConfigReconciler.ByoMachineToK8sInstallerConfigMapFunc(byoMachine)
			Expect(len(result)).NotTo(BeZero())
		})
	})

})
