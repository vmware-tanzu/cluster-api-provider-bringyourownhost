// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controllers_test

import (
	"context"

	b64 "encoding/base64"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	infrav1 "github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/apis/infrastructure/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Controllers/BoottrapKubeconfigController", func() {
	var (
		k8sClientUncached               client.Client
		bootstrapKubeconfigLookupKey    types.NamespacedName
		bootstrapKubeConfig             *infrav1.BootstrapKubeconfig
		ctx                             = context.Background()
		testServer                      = "123.123.123.123:1234"
		testCAData                      = "test-ca-data"
		existingBootstrapKubeconfigData = "i am already present"
	)

	It("should ignore bootstrapkubeconfig if it is not found", func() {
		_, err := bootstrapKubeconfigReconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      "non-existent-bootstrap-kubeconfig",
				Namespace: "non-existent-namespace"}})
		Expect(err).NotTo(HaveOccurred())
	})

	Context("When BootstrapKubeconfig CRD is created", func() {
		BeforeEach(func() {
			var clientErr error
			k8sClientUncached, clientErr = client.New(cfg, client.Options{Scheme: scheme.Scheme})
			Expect(clientErr).NotTo(HaveOccurred())
			bootstrapKubeConfig = &infrav1.BootstrapKubeconfig{
				TypeMeta:   metav1.TypeMeta{Kind: "BootstrapKubeconfig", APIVersion: infrav1.GroupVersion.String()},
				ObjectMeta: metav1.ObjectMeta{GenerateName: "bootstrap-kubeconfig", Namespace: "default"},
				Spec: infrav1.BootstrapKubeconfigSpec{
					APIServer:                testServer,
					InsecureSkipTLSVerify:    false,
					CertificateAuthorityData: b64.StdEncoding.EncodeToString([]byte(testCAData)),
				},
				Status: infrav1.BootstrapKubeconfigStatus{},
			}
			Expect(k8sClientUncached.Create(ctx, bootstrapKubeConfig)).Should(Succeed())
			WaitForObjectsToBePopulatedInCache(bootstrapKubeConfig)

			bootstrapKubeconfigLookupKey = types.NamespacedName{Name: bootstrapKubeConfig.Name, Namespace: bootstrapKubeConfig.Namespace}
		})

		It("should return empty result if BootstrapKubeconfigData is already present", func() {
			helper, err := patch.NewHelper(bootstrapKubeConfig, k8sClientUncached)
			Expect(err).NotTo(HaveOccurred())
			bootstrapKubeConfig.Status.BootstrapKubeconfigData = &existingBootstrapKubeconfigData
			Expect(helper.Patch(ctx, bootstrapKubeConfig)).NotTo(HaveOccurred())

			res, err := bootstrapKubeconfigReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: bootstrapKubeconfigLookupKey})
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(ctrl.Result{}))
		})

		It("should generate the bootstrap kubeconfig data", func() {
			_, err := bootstrapKubeconfigReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: bootstrapKubeconfigLookupKey})
			Expect(err).NotTo(HaveOccurred())

			createdBootstrapKubeconfig := &infrav1.BootstrapKubeconfig{}
			err = k8sClientUncached.Get(ctx, bootstrapKubeconfigLookupKey, createdBootstrapKubeconfig)
			Expect(err).ToNot(HaveOccurred())

			kubeconfigData := createdBootstrapKubeconfig.Status.BootstrapKubeconfigData
			Expect(kubeconfigData).ShouldNot(BeNil())

			bootstrapKubeconfigFileData, err := clientcmd.Load([]byte(*kubeconfigData))
			Expect(err).NotTo(HaveOccurred())

			// assert Server and CertificateAuthorityData are the same as that we have passed
			Expect(bootstrapKubeconfigFileData.Clusters[infrav1.DefaultClusterName].Server).To(Equal(testServer))

			caDataFromStatus := bootstrapKubeconfigFileData.Clusters[infrav1.DefaultClusterName].CertificateAuthorityData
			Expect(string(caDataFromStatus)).To(Equal(testCAData))

		})

		AfterEach(func() {
			Expect(k8sClientUncached.Delete(ctx, bootstrapKubeConfig)).ToNot(HaveOccurred())
		})
	})
})
