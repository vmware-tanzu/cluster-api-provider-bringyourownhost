package controllers

import (
	"context"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/apis/infrastructure/v1alpha4"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/common"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Controllers/ByomachineController", func() {
	var (
		byoMachineLookupKey types.NamespacedName
		byoHostLookupKey    types.NamespacedName
		ctx                 context.Context
		byoMachine          *infrastructurev1alpha4.ByoMachine
		machine             *clusterv1.Machine
		k8sClientUncached   client.Client
		byoHost             *infrastructurev1alpha4.ByoHost
	)

	BeforeEach(func() {
		ctx = context.Background()

		var clientErr error
		k8sClientUncached, clientErr = client.New(cfg, client.Options{Scheme: scheme.Scheme})
		Expect(clientErr).NotTo(HaveOccurred())

		machine = common.NewMachine(defaultMachineName, defaultNamespace, defaultClusterName)
		machine.Spec.Bootstrap = clusterv1.Bootstrap{
			DataSecretName: &fakeBootstrapSecret,
		}
		Expect(k8sClientUncached.Create(ctx, machine)).Should(Succeed())

		byoMachine = common.NewByoMachine(defaultByoMachineName, defaultNamespace, defaultClusterName, machine)
		Expect(k8sClientUncached.Create(ctx, byoMachine)).Should(Succeed())

		WaitForObjectsToBePopulatedInCache(machine, byoMachine)
		byoMachineLookupKey = types.NamespacedName{Name: byoMachine.Name, Namespace: byoMachine.Namespace}
	})

	It("should ignore byomachine if it is not found", func() {
		_, err := reconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      "non-existent-byomachine",
				Namespace: "non-existent-namespace"}})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should return error when cluster does not exist", func() {
		machineForByoMachineWithoutCluster := common.NewMachine("machine-for-a-byomachine-without-cluster", defaultNamespace, defaultClusterName)
		Expect(k8sClientUncached.Create(ctx, machineForByoMachineWithoutCluster)).Should(Succeed())

		byoMachineWithNonExistingCluster := common.NewByoMachine(defaultByoMachineName, defaultNamespace, "non-existent-cluster", machine)
		Expect(k8sClientUncached.Create(ctx, byoMachineWithNonExistingCluster)).Should(Succeed())

		WaitForObjectsToBePopulatedInCache(machineForByoMachineWithoutCluster, byoMachineWithNonExistingCluster)

		_, err := reconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      byoMachineWithNonExistingCluster.Name,
				Namespace: byoMachineWithNonExistingCluster.Namespace}})

		Expect(err).To(MatchError("Cluster.cluster.x-k8s.io \"non-existent-cluster\" not found"))
	})

	Context("When cluster infrastructure is ready", func() {
		BeforeEach(func() {
			ph, err := patch.NewHelper(capiCluster, k8sClientUncached)
			Expect(err).ShouldNot(HaveOccurred())
			capiCluster.Status.InfrastructureReady = true
			Expect(ph.Patch(ctx, capiCluster, patch.WithStatusObservedGeneration{})).Should(Succeed())

			WaitForObjectToBeUpdatedInCache(capiCluster, func(object client.Object) bool {
				return object.(*clusterv1.Cluster).Status.InfrastructureReady == true
			})
		})

		It("should return error when node is not available", func() {
			byoHost = common.NewByoHost("host-with-node-missing", defaultNamespace)
			Expect(k8sClientUncached.Create(ctx, byoHost)).Should(Succeed())

			WaitForObjectsToBePopulatedInCache(byoHost)

			_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: byoMachineLookupKey})
			Expect(err).To(MatchError("nodes \"" + byoHost.Name + "\" not found"))
		})

		Context("When BYO Hosts are not available", func() {
			It("should mark BYOHostReady as False", func() {
				_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: byoMachineLookupKey})
				Expect(err).To(MatchError("no hosts found"))

				createdByoMachine := &infrastructurev1alpha4.ByoMachine{}
				err = k8sClientUncached.Get(ctx, byoMachineLookupKey, createdByoMachine)
				Expect(err).ToNot(HaveOccurred())
				actualCondition := conditions.Get(createdByoMachine, infrastructurev1alpha4.BYOHostReady)

				Expect(*actualCondition).To(conditions.MatchCondition(clusterv1.Condition{
					Type:     infrastructurev1alpha4.BYOHostReady,
					Status:   corev1.ConditionFalse,
					Reason:   infrastructurev1alpha4.BYOHostsUnavailableReason,
					Severity: clusterv1.ConditionSeverityInfo,
				}))
			})

			It("should add MachineFinalizer on ByoMachine", func() {
				_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: byoMachineLookupKey})
				Expect(err).To(HaveOccurred())

				updatedByoMachine := &infrastructurev1alpha4.ByoMachine{}
				err = k8sClientUncached.Get(ctx, byoMachineLookupKey, updatedByoMachine)
				Expect(err).ToNot(HaveOccurred())
				Expect(controllerutil.ContainsFinalizer(updatedByoMachine, infrastructurev1alpha4.MachineFinalizer)).To(BeTrue())
			})
		})

		Context("When a single BYO Host is available", func() {
			BeforeEach(func() {
				byoHost = common.NewByoHost("single-available-default-host", defaultNamespace)
				Expect(k8sClientUncached.Create(ctx, byoHost)).Should(Succeed())

				Expect(clientFake.Create(ctx, common.NewNode(byoHost.Name, defaultNamespace))).Should(Succeed())
				WaitForObjectsToBePopulatedInCache(byoHost)

				byoHostLookupKey = types.NamespacedName{Name: byoHost.Name, Namespace: byoHost.Namespace}
			})

			AfterEach(func() {
				Expect(k8sClientUncached.Delete(ctx, byoHost)).ToNot(HaveOccurred())
			})

			It("claims the first available host", func() {
				_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: byoMachineLookupKey})
				Expect(err).ToNot(HaveOccurred())

				createdByoHost := &infrastructurev1alpha4.ByoHost{}
				err = k8sClientUncached.Get(ctx, byoHostLookupKey, createdByoHost)
				Expect(err).ToNot(HaveOccurred())
				Expect(createdByoHost.Status.MachineRef.Namespace).To(Equal(byoMachine.Namespace))
				Expect(createdByoHost.Status.MachineRef.Name).To(Equal(byoMachine.Name))

				createdByoMachine := &infrastructurev1alpha4.ByoMachine{}
				err = k8sClientUncached.Get(ctx, byoMachineLookupKey, createdByoMachine)
				Expect(err).ToNot(HaveOccurred())
				Expect(createdByoMachine.Spec.ProviderID).To(ContainSubstring(providerIDPrefix))
				Expect(createdByoMachine.Status.Ready).To(BeTrue())

				actualCondition := conditions.Get(createdByoMachine, infrastructurev1alpha4.BYOHostReady)
				Expect(*actualCondition).To(conditions.MatchCondition(clusterv1.Condition{
					Type:   infrastructurev1alpha4.BYOHostReady,
					Status: corev1.ConditionTrue,
				}))

				node := corev1.Node{}
				err = clientFake.Get(ctx, types.NamespacedName{Name: byoHost.Name, Namespace: defaultNamespace}, &node)
				Expect(err).NotTo(HaveOccurred())

				Expect(node.Spec.ProviderID).To(ContainSubstring(providerIDPrefix))
			})

			Context("When ByoMachine is attached to a host", func() {
				BeforeEach(func() {
					ph, err := patch.NewHelper(byoHost, k8sClientUncached)
					Expect(err).ShouldNot(HaveOccurred())
					byoHost.Status.MachineRef = &corev1.ObjectReference{
						Kind:       "ByoMachine",
						Namespace:  byoMachine.Namespace,
						Name:       byoMachine.Name,
						UID:        byoMachine.UID,
						APIVersion: byoHost.APIVersion,
					}
					Expect(ph.Patch(ctx, byoHost, patch.WithStatusObservedGeneration{})).Should(Succeed())

					WaitForObjectToBeUpdatedInCache(byoHost, func(object client.Object) bool {
						return object.(*infrastructurev1alpha4.ByoHost).Status.MachineRef != nil
					})
				})

				Context("When ByoMachine is deleted", func() {
					BeforeEach(func() {
						ph, err := patch.NewHelper(byoMachine, k8sClientUncached)
						Expect(err).ShouldNot(HaveOccurred())
						controllerutil.AddFinalizer(byoMachine, infrastructurev1alpha4.MachineFinalizer)
						Expect(ph.Patch(ctx, byoMachine, patch.WithStatusObservedGeneration{})).Should(Succeed())

						Expect(k8sClientUncached.Delete(ctx, byoMachine)).Should(Succeed())
						Eventually(func() bool {
							deletedByoMachine := &infrastructurev1alpha4.ByoMachine{}
							err := reconciler.Client.Get(ctx, byoMachineLookupKey, deletedByoMachine)
							if err != nil {
								return false
							}
							return !deletedByoMachine.ObjectMeta.DeletionTimestamp.IsZero()
						}).Should(BeTrue())
					})

					// TODO - To fix, the `reconcileDelete` should return an error if `K8sNodeBootstrapSucceeded` does not have a reason `K8sNodeAbsentReason`.
					// Not fixing now since the e2e test is failing. Will revisit this.
					XIt("should add cleanup annotation on byohost so that the host agent can cleanup", func() {
						_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: byoMachineLookupKey})
						Expect(err).NotTo(HaveOccurred())

						createdByoHost := &infrastructurev1alpha4.ByoHost{}
						Expect(k8sClientUncached.Get(ctx, byoHostLookupKey, createdByoHost)).NotTo(HaveOccurred())

						byoHostAnnotations := createdByoHost.GetAnnotations()
						_, ok := byoHostAnnotations[hostCleanupAnnotation]
						Expect(ok).To(BeTrue())
					})

					It("should remove host reservation when the host agent is done cleaning up", func() {
						ph, err := patch.NewHelper(byoHost, k8sClientUncached)
						Expect(err).ShouldNot(HaveOccurred())
						conditions.MarkFalse(byoHost, infrastructurev1alpha4.K8sNodeBootstrapSucceeded, infrastructurev1alpha4.K8sNodeAbsentReason, clusterv1.ConditionSeverityInfo, "")
						Expect(ph.Patch(ctx, byoHost, patch.WithStatusObservedGeneration{})).Should(Succeed())

						WaitForObjectToBeUpdatedInCache(byoHost, func(object client.Object) bool {
							return conditions.Get(object.(*infrastructurev1alpha4.ByoHost), infrastructurev1alpha4.K8sNodeBootstrapSucceeded).Reason == infrastructurev1alpha4.K8sNodeAbsentReason
						})

						_, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: byoMachineLookupKey})
						Expect(err).NotTo(HaveOccurred())

						createdByoHost := &infrastructurev1alpha4.ByoHost{}
						Expect(k8sClientUncached.Get(ctx, byoHostLookupKey, createdByoHost)).NotTo(HaveOccurred())
						Expect(createdByoHost.Status.MachineRef).To(BeNil())
						Expect(createdByoHost.Labels[clusterv1.ClusterLabelName]).To(BeEmpty())

						byoHostAnnotations := createdByoHost.GetAnnotations()
						_, ok := byoHostAnnotations[hostCleanupAnnotation]
						Expect(ok).To(BeFalse())
					})
				})
			})

			It("should mark BYOHostReady as False when byomachine is paused", func() {
				ph, err := patch.NewHelper(byoMachine, k8sClientUncached)
				Expect(err).ShouldNot(HaveOccurred())

				pauseAnnotations := map[string]string{
					clusterv1.PausedAnnotation: "paused",
				}
				annotations.AddAnnotations(byoMachine, pauseAnnotations)

				Expect(ph.Patch(ctx, byoMachine, patch.WithStatusObservedGeneration{})).Should(Succeed())

				WaitForObjectToBeUpdatedInCache(byoMachine, func(object client.Object) bool {
					return annotations.HasPausedAnnotation(object.(*infrastructurev1alpha4.ByoMachine))
				})

				_, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: byoMachineLookupKey})
				Expect(err).ToNot(HaveOccurred())

				createdByoMachine := &infrastructurev1alpha4.ByoMachine{}
				err = k8sClientUncached.Get(ctx, byoMachineLookupKey, createdByoMachine)
				Expect(err).ToNot(HaveOccurred())

				actualCondition := conditions.Get(createdByoMachine, infrastructurev1alpha4.BYOHostReady)
				Expect(*actualCondition).To(conditions.MatchCondition(clusterv1.Condition{
					Type:     infrastructurev1alpha4.BYOHostReady,
					Status:   corev1.ConditionFalse,
					Reason:   infrastructurev1alpha4.ClusterOrResourcePausedReason,
					Severity: clusterv1.ConditionSeverityInfo,
				}))
			})

			It("should mark BYOHostReady as False when cluster is paused", func() {
				pausedCluster := common.NewCluster("paused-cluster", defaultNamespace)
				pausedCluster.Spec.Paused = true
				Expect(k8sClientUncached.Create(ctx, pausedCluster)).Should(Succeed())

				pausedMachine := common.NewMachine("paused-machine", defaultNamespace, pausedCluster.Name)
				Expect(k8sClientUncached.Create(ctx, pausedMachine)).Should(Succeed())
				pausedByoMachine := common.NewByoMachine("paused-byo-machine", defaultNamespace, pausedCluster.Name, pausedMachine)
				Expect(k8sClientUncached.Create(ctx, pausedByoMachine)).Should(Succeed())

				WaitForObjectsToBePopulatedInCache(pausedCluster, pausedMachine, pausedByoMachine)

				pausedByoMachineLookupKey := types.NamespacedName{Name: pausedByoMachine.Name, Namespace: pausedByoMachine.Namespace}

				_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: pausedByoMachineLookupKey})
				Expect(err).ToNot(HaveOccurred())

				createdByoMachine := &infrastructurev1alpha4.ByoMachine{}
				err = k8sClientUncached.Get(ctx, pausedByoMachineLookupKey, createdByoMachine)
				Expect(err).ToNot(HaveOccurred())

				actualCondition := conditions.Get(createdByoMachine, infrastructurev1alpha4.BYOHostReady)
				Expect(*actualCondition).To(conditions.MatchCondition(clusterv1.Condition{
					Type:     infrastructurev1alpha4.BYOHostReady,
					Status:   corev1.ConditionFalse,
					Reason:   infrastructurev1alpha4.ClusterOrResourcePausedReason,
					Severity: clusterv1.ConditionSeverityInfo,
				}))

				Expect(k8sClientUncached.Delete(ctx, pausedCluster)).Should(Succeed())
				Expect(k8sClientUncached.Delete(ctx, pausedMachine)).Should(Succeed())
				Expect(k8sClientUncached.Delete(ctx, pausedByoMachine)).Should(Succeed())
			})

			It("should mark BYOHostReady as False when machine.Spec.Bootstrap.DataSecretName is not set", func() {
				ph, err := patch.NewHelper(machine, k8sClientUncached)
				Expect(err).ShouldNot(HaveOccurred())

				machine.Spec.Bootstrap = clusterv1.Bootstrap{DataSecretName: nil}
				Expect(ph.Patch(ctx, machine, patch.WithStatusObservedGeneration{})).Should(Succeed())

				WaitForObjectToBeUpdatedInCache(machine, func(object client.Object) bool {
					return object.(*clusterv1.Machine).Spec.Bootstrap.DataSecretName == nil
				})

				_, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: byoMachineLookupKey})
				Expect(err).To(MatchError("bootstrap data secret not available yet"))

				createdByoMachine := &infrastructurev1alpha4.ByoMachine{}
				err = k8sClientUncached.Get(ctx, byoMachineLookupKey, createdByoMachine)
				Expect(err).ShouldNot(HaveOccurred())

				actualCondition := conditions.Get(createdByoMachine, infrastructurev1alpha4.BYOHostReady)
				Expect(*actualCondition).To(conditions.MatchCondition(clusterv1.Condition{
					Type:     infrastructurev1alpha4.BYOHostReady,
					Status:   corev1.ConditionFalse,
					Reason:   infrastructurev1alpha4.WaitingForBootstrapDataSecretReason,
					Severity: clusterv1.ConditionSeverityInfo,
				}))
			})
		})

		Context("When no matching BYO Hosts are available", func() {
			BeforeEach(func() {
				byoHost = common.NewByoHost("byohost-with-different-label", defaultNamespace)
				byoHost.Labels = map[string]string{"CPUs": "2"}
				Expect(k8sClientUncached.Create(ctx, byoHost)).Should(Succeed())

				byoMachine = common.NewByoMachine("byomachine-with-label-selector", defaultNamespace, defaultClusterName, machine)
				byoMachine.Spec.Selector = &v1.LabelSelector{MatchLabels: map[string]string{"CPUs": "4"}}
				Expect(k8sClientUncached.Create(ctx, byoMachine)).Should(Succeed())

				WaitForObjectsToBePopulatedInCache(byoHost, byoMachine)
				byoMachineLookupKey = types.NamespacedName{Name: byoMachine.Name, Namespace: byoMachine.Namespace}
			})

			AfterEach(func() {
				Expect(k8sClientUncached.Delete(ctx, byoHost)).ToNot(HaveOccurred())
			})

			It("should mark BYOHostReady as False when BYOHosts is available but label mismatch", func() {
				_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: byoMachineLookupKey})
				Expect(err).To(MatchError("no hosts found"))

				createdByoMachine := &infrastructurev1alpha4.ByoMachine{}
				err = k8sClientUncached.Get(ctx, byoMachineLookupKey, createdByoMachine)
				Expect(err).ToNot(HaveOccurred())

				actualCondition := conditions.Get(createdByoMachine, infrastructurev1alpha4.BYOHostReady)
				Expect(*actualCondition).To(conditions.MatchCondition(clusterv1.Condition{
					Type:     infrastructurev1alpha4.BYOHostReady,
					Status:   corev1.ConditionFalse,
					Reason:   infrastructurev1alpha4.BYOHostsUnavailableReason,
					Severity: clusterv1.ConditionSeverityInfo,
				}))
			})
		})

		Context("When all ByoHost are attached", func() {
			BeforeEach(func() {
				byoHost = common.NewByoHost("byohost-attached-different-cluster", defaultNamespace)
				byoHost.Labels = map[string]string{clusterv1.ClusterLabelName: capiCluster.Name}
				Expect(k8sClientUncached.Create(ctx, byoHost)).Should(Succeed())

				WaitForObjectsToBePopulatedInCache(byoHost)
			})

			AfterEach(func() {
				Expect(k8sClientUncached.Delete(ctx, byoHost)).ToNot(HaveOccurred())
			})

			It("should mark BYOHostReady as False", func() {
				_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: byoMachineLookupKey})
				Expect(err).To(MatchError("no hosts found"))

				createdByoMachine := &infrastructurev1alpha4.ByoMachine{}
				err = k8sClientUncached.Get(ctx, byoMachineLookupKey, createdByoMachine)
				Expect(err).ToNot(HaveOccurred())

				actualCondition := conditions.Get(createdByoMachine, infrastructurev1alpha4.BYOHostReady)
				Expect(*actualCondition).To(conditions.MatchCondition(clusterv1.Condition{
					Type:     infrastructurev1alpha4.BYOHostReady,
					Status:   corev1.ConditionFalse,
					Reason:   infrastructurev1alpha4.BYOHostsUnavailableReason,
					Severity: clusterv1.ConditionSeverityInfo,
				}))
			})
		})

		Context("When multiple BYO Host are available", func() {
			var (
				byoHost1 *infrastructurev1alpha4.ByoHost
				byoHost2 *infrastructurev1alpha4.ByoHost
			)

			BeforeEach(func() {
				byoHost1 = common.NewByoHost(defaultByoHostName, defaultNamespace)
				Expect(k8sClientUncached.Create(ctx, byoHost1)).Should(Succeed())
				byoHost2 = common.NewByoHost(defaultByoHostName, defaultNamespace)
				Expect(k8sClientUncached.Create(ctx, byoHost2)).Should(Succeed())

				WaitForObjectsToBePopulatedInCache(byoHost1, byoHost2)

				Expect(clientFake.Create(ctx, common.NewNode(byoHost1.Name, defaultNamespace))).Should(Succeed())
				Expect(clientFake.Create(ctx, common.NewNode(byoHost2.Name, defaultNamespace))).Should(Succeed())
			})

			It("claims one of the available host", func() {
				_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: byoMachineLookupKey})
				Expect(err).ToNot(HaveOccurred())

				createdByoMachine := &infrastructurev1alpha4.ByoMachine{}
				err = k8sClientUncached.Get(ctx, byoMachineLookupKey, createdByoMachine)
				Expect(err).ToNot(HaveOccurred())

				Expect(createdByoMachine.Status.Ready).To(BeTrue())

				readyCondition := conditions.Get(createdByoMachine, infrastructurev1alpha4.BYOHostReady)
				Expect(*readyCondition).To(conditions.MatchCondition(clusterv1.Condition{
					Type:   infrastructurev1alpha4.BYOHostReady,
					Status: corev1.ConditionTrue,
				}))

				node1 := corev1.Node{}
				err = clientFake.Get(ctx, types.NamespacedName{Name: byoHost1.Name, Namespace: defaultNamespace}, &node1)
				Expect(err).NotTo(HaveOccurred())

				node2 := corev1.Node{}
				err = clientFake.Get(ctx, types.NamespacedName{Name: byoHost2.Name, Namespace: defaultNamespace}, &node2)
				Expect(err).NotTo(HaveOccurred())

				var nodeTagged bool
				if strings.Contains(node1.Spec.ProviderID, providerIDPrefix) || strings.Contains(node2.Spec.ProviderID, providerIDPrefix) {
					nodeTagged = true
				}
				Expect(nodeTagged).To(Equal(true))
			})

			It("does not claims the attached host", func() {
				ph, err := patch.NewHelper(byoHost2, k8sClientUncached)
				Expect(err).ShouldNot(HaveOccurred())
				byoHost2.Labels = map[string]string{clusterv1.ClusterLabelName: capiCluster.Name}
				Expect(ph.Patch(ctx, byoHost2, patch.WithStatusObservedGeneration{})).Should(Succeed())

				WaitForObjectToBeUpdatedInCache(byoHost2, func(object client.Object) bool {
					return object.(*infrastructurev1alpha4.ByoHost).Labels[clusterv1.ClusterLabelName] == capiCluster.Name
				})

				_, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: byoMachineLookupKey})
				Expect(err).ToNot(HaveOccurred())

				createdByoHost := &infrastructurev1alpha4.ByoHost{}
				err = k8sClientUncached.Get(ctx, types.NamespacedName{Name: byoHost1.Name, Namespace: defaultNamespace}, createdByoHost)
				Expect(err).ToNot(HaveOccurred())
				Expect(createdByoHost.Status.MachineRef.Namespace).To(Equal(defaultNamespace))
				Expect(createdByoHost.Status.MachineRef.Name).To(Equal(byoMachine.Name))

				createdByoMachine := &infrastructurev1alpha4.ByoMachine{}
				err = k8sClientUncached.Get(ctx, byoMachineLookupKey, createdByoMachine)
				Expect(err).ToNot(HaveOccurred())
				Expect(createdByoMachine.Status.Ready).To(BeTrue())

				readyCondition := conditions.Get(createdByoMachine, infrastructurev1alpha4.BYOHostReady)
				Expect(*readyCondition).To(conditions.MatchCondition(clusterv1.Condition{
					Type:   infrastructurev1alpha4.BYOHostReady,
					Status: corev1.ConditionTrue,
				}))

				node := corev1.Node{}
				err = clientFake.Get(ctx, types.NamespacedName{Name: byoHost1.Name, Namespace: defaultNamespace}, &node)
				Expect(err).NotTo(HaveOccurred())

				Expect(node.Spec.ProviderID).To(ContainSubstring(providerIDPrefix))
			})

			AfterEach(func() {
				Expect(k8sClientUncached.Delete(ctx, byoHost1)).Should(Succeed())
				Expect(k8sClientUncached.Delete(ctx, byoHost2)).Should(Succeed())
			})
		})
	})

	Context("When cluster infrastructure is not ready", func() {
		BeforeEach(func() {
			ph, err := patch.NewHelper(capiCluster, k8sClientUncached)
			Expect(err).ShouldNot(HaveOccurred())
			capiCluster.Status.InfrastructureReady = false
			err = ph.Patch(ctx, capiCluster, patch.WithStatusObservedGeneration{})
			Expect(err).ShouldNot(HaveOccurred())

			WaitForObjectToBeUpdatedInCache(capiCluster, func(object client.Object) bool {
				return object.(*clusterv1.Cluster).Status.InfrastructureReady == false
			})
		})

		It("should mark BYOHostReady as False", func() {
			_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: byoMachineLookupKey})
			Expect(err).To(MatchError("cluster infrastructure is not ready yet"))

			createdByoMachine := &infrastructurev1alpha4.ByoMachine{}
			err = k8sClientUncached.Get(ctx, byoMachineLookupKey, createdByoMachine)
			Expect(err).ShouldNot(HaveOccurred())

			actualCondition := conditions.Get(createdByoMachine, infrastructurev1alpha4.BYOHostReady)
			Expect(*actualCondition).To(conditions.MatchCondition(clusterv1.Condition{
				Type:     infrastructurev1alpha4.BYOHostReady,
				Status:   corev1.ConditionFalse,
				Reason:   infrastructurev1alpha4.WaitingForClusterInfrastructureReason,
				Severity: clusterv1.ConditionSeverityInfo,
			}))
		})

	})
})
