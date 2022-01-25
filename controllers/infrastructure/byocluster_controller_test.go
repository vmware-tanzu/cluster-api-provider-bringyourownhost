// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controllers_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	infrastructurev1beta1 "github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/apis/infrastructure/v1beta1"
	controllers "github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/controllers/infrastructure"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/test/builder"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Controllers/ByoclusterController", func() {

	var (
		ctx               context.Context
		k8sClientUncached client.Client
		byoCluster        *infrastructurev1beta1.ByoCluster
		cluster           *clusterv1.Cluster
	)

	BeforeEach(func() {
		ctx = context.Background()
		var clientErr error

		k8sClientUncached, clientErr = client.New(cfg, client.Options{Scheme: scheme.Scheme})
		Expect(clientErr).NotTo(HaveOccurred())
	})

	It("should not throw error when byocluster does not exist", func() {
		_, err := byoClusterReconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      "non-existent-byocluster",
				Namespace: "non-existent-namespace"}})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should not throw error when OwnerRef is not set", func() {
		byoCluster = builder.ByoCluster(defaultNamespace, "byocluster-not-link-cluster").Build()
		Expect(k8sClientUncached.Create(ctx, byoCluster)).Should(Succeed())
		WaitForObjectsToBePopulatedInCache(byoCluster)

		_, err := byoClusterReconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      byoCluster.Name,
				Namespace: byoCluster.Namespace}})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should not throw error when byocluster is paused", func() {
		cluster = builder.Cluster(defaultNamespace, "cluster-paused").
			WithPausedField(true).
			Build()
		Expect(k8sClientUncached.Create(ctx, cluster)).Should(Succeed())
		WaitForObjectsToBePopulatedInCache(cluster)

		byoCluster = builder.ByoCluster(defaultNamespace, "byocluster-paused").
			WithOwnerCluster(cluster).
			Build()
		Expect(k8sClientUncached.Create(ctx, byoCluster)).Should(Succeed())
		WaitForObjectsToBePopulatedInCache(byoCluster)

		_, err := byoClusterReconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      byoCluster.Name,
				Namespace: byoCluster.Namespace}})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should be able to delete ByoCluster", func() {
		cluster = builder.Cluster(defaultNamespace, "byocluster-deleted").
			Build()
		Expect(k8sClientUncached.Create(ctx, cluster)).Should(Succeed())
		WaitForObjectsToBePopulatedInCache(cluster)

		byoCluster = builder.ByoCluster(defaultNamespace, "byocluster-deleted").
			WithOwnerCluster(cluster).
			Build()
		Expect(k8sClientUncached.Create(ctx, byoCluster)).Should(Succeed())
		WaitForObjectsToBePopulatedInCache(byoCluster)

		byoClusterLookupKey := types.NamespacedName{Name: byoCluster.Name, Namespace: byoCluster.Namespace}
		_, err := byoClusterReconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: byoClusterLookupKey})
		Expect(err).NotTo(HaveOccurred())

		Expect(k8sClientUncached.Delete(ctx, byoCluster)).Should(Succeed())
		WaitForObjectToBeUpdatedInCache(byoCluster, func(object client.Object) bool {
			return !object.(*infrastructurev1beta1.ByoCluster).ObjectMeta.DeletionTimestamp.IsZero()
		})

		_, err = byoClusterReconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: byoClusterLookupKey})
		Expect(err).NotTo(HaveOccurred())

		// assert ByoCluster does not exists
		deletedByoCluster := &infrastructurev1beta1.ByoCluster{}
		err = k8sClientUncached.Get(ctx, byoClusterLookupKey, deletedByoCluster)
		Expect(err).To(MatchError(fmt.Sprintf("byoclusters.infrastructure.cluster.x-k8s.io %q not found", byoClusterLookupKey.Name)))

	})

	It("should get valid value of fields when ByoClusterController gets a create request", func() {
		cluster = builder.Cluster(defaultNamespace, "byocluster-finalizer").
			Build()
		Expect(k8sClientUncached.Create(ctx, cluster)).Should(Succeed())
		WaitForObjectsToBePopulatedInCache(cluster)

		byoCluster = builder.ByoCluster(defaultNamespace, "byocluster-finalizer").
			WithOwnerCluster(cluster).
			Build()
		Expect(k8sClientUncached.Create(ctx, byoCluster)).Should(Succeed())
		WaitForObjectsToBePopulatedInCache(byoCluster)

		byoClusterLookupKey := types.NamespacedName{Name: byoCluster.Name, Namespace: byoCluster.Namespace}
		_, err := byoClusterReconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: byoClusterLookupKey})
		Expect(err).NotTo(HaveOccurred())

		createdByoCluster := &infrastructurev1beta1.ByoCluster{}
		err = k8sClientUncached.Get(ctx, byoClusterLookupKey, createdByoCluster)
		Expect(err).ToNot(HaveOccurred())
		Expect(controllerutil.ContainsFinalizer(createdByoCluster, infrastructurev1beta1.ClusterFinalizer)).To(BeTrue())
		Expect(createdByoCluster.Status.Ready).To(BeTrue())
		Expect(createdByoCluster.Spec.ControlPlaneEndpoint.Port).To(Equal(int32(controllers.DefaultAPIEndpointPort)))
	})

})
