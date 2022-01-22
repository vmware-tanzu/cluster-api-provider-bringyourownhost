// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// nolint: testpackage
package e2e

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	infrastructurev1beta1 "github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/apis/infrastructure/v1beta1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/util"
)

var _ = Describe("When BYO Host rejoins the capacity pool", func() {

	var (
		caseContextData *CaseContext        = nil
		collectInfoData *CollectInfoContext = nil
		byoHostPoolData *ByoHostPoolContext = nil
	)

	BeforeEach(func() {

		caseContextData = new(CaseContext)
		Expect(caseContextData).NotTo(BeNil())
		caseContextData.CaseName = "reuse"
		caseContextData.ClusterConName = clusterConName
		caseContextData.clusterProxy = bootstrapClusterProxy
		caseContextData.ClusterResources = new(clusterctl.ApplyClusterTemplateAndWaitResult)
		Expect(caseContextData.ClusterResources).NotTo(BeNil())

		specName := caseContextData.CaseName
		caseContextData.ctx = context.TODO()
		Expect(caseContextData.ctx).NotTo(BeNil(), "ctx is required for %s spec", specName)

		Expect(e2eConfig).NotTo(BeNil(), "Invalid argument. e2eConfig can't be nil when calling %s spec", specName)
		Expect(clusterctlConfigPath).To(BeAnExistingFile(), "Invalid argument. clusterctlConfigPath must be an existing file when calling %s spec", specName)
		Expect(bootstrapClusterProxy).NotTo(BeNil(), "Invalid argument. bootstrapClusterProxy can't be nil when calling %s spec", specName)
		Expect(os.MkdirAll(artifactFolder, 0755)).To(Succeed(), "Invalid argument. artifactFolder can't be created for %s spec", specName)
		Expect(e2eConfig.Variables).To(HaveKey(KubernetesVersion))

		// Setup a Namespace where to host objects for this spec and create a watcher for the namespace events.
		setupSpecNamespace(caseContextData, artifactFolder)

		byoHostPoolData = new(ByoHostPoolContext)
		Expect(byoHostPoolData).NotTo(BeNil())

		collectInfoData = new(CollectInfoContext)
		Expect(collectInfoData).NotTo(BeNil())
		collectInfoData.DeploymentLogDir = fmt.Sprintf("/tmp/%s-deplymentlogs", caseContextData.CaseName)
	})

	It("Should reuse the same BYO Host after it is reset", func() {
		ctx := caseContextData.ctx
		clusterProxy := caseContextData.clusterProxy
		namespace := caseContextData.Namespace
		specName := caseContextData.CaseName
		clusterResources := caseContextData.ClusterResources
		caseContextData.ClusterName = fmt.Sprintf("%s-%s", specName, util.RandomString(6))

		byoHostPoolData.Capacity = 2
		Byf("Creating byohost capacity pool containing %d hosts", byoHostPoolData.Capacity)
		fl := setupByohostPool(caseContextData, collectInfoData, byoHostPoolData)
		for _, f := range fl {
			defer f.Close()
		}

		By("Creating a cluster")
		clusterctl.ApplyClusterTemplateAndWait(ctx, clusterctl.ApplyClusterTemplateAndWaitInput{
			ClusterProxy: clusterProxy,
			ConfigCluster: clusterctl.ConfigClusterInput{
				LogFolder:                filepath.Join(artifactFolder, "clusters", bootstrapClusterProxy.GetName()),
				ClusterctlConfigPath:     clusterctlConfigPath,
				KubeconfigPath:           clusterProxy.GetKubeconfigPath(),
				InfrastructureProvider:   clusterctl.DefaultInfrastructureProvider,
				Flavor:                   clusterctl.DefaultFlavor,
				Namespace:                namespace.Name,
				ClusterName:              caseContextData.ClusterName,
				KubernetesVersion:        e2eConfig.GetVariable(KubernetesVersion),
				ControlPlaneMachineCount: pointer.Int64Ptr(1),
				WorkerMachineCount:       pointer.Int64Ptr(1),
			},
			WaitForClusterIntervals:      e2eConfig.GetIntervals(specName, "wait-cluster"),
			WaitForControlPlaneIntervals: e2eConfig.GetIntervals(specName, "wait-control-plane"),
			WaitForMachineDeployments:    e2eConfig.GetIntervals(specName, "wait-worker-nodes"),
		}, clusterResources)

		// Assert on byohost cluster label to match clusterName
		byoHostLookupKey := k8stypes.NamespacedName{Name: byoHostPoolData.ByoHostNames[1], Namespace: namespace.Name}
		byoHostToBeReused := &infrastructurev1beta1.ByoHost{}
		Expect(clusterProxy.GetClient().Get(ctx, byoHostLookupKey, byoHostToBeReused)).Should(Succeed())
		cluster, ok := byoHostToBeReused.Labels[clusterv1.ClusterLabelName]
		Expect(ok).To(BeTrue())
		Expect(cluster).To(Equal(caseContextData.ClusterName))

		By("Delete the cluster and freeing the ByoHosts")
		framework.DeleteAllClustersAndWait(ctx, framework.DeleteAllClustersAndWaitInput{
			Client:    clusterProxy.GetClient(),
			Namespace: namespace.Name,
		}, e2eConfig.GetIntervals(specName, "wait-delete-cluster")...)

		// Assert if cluster label is removed
		// This verifies that the byohost has rejoined the capacity pool
		byoHostToBeReused = &infrastructurev1beta1.ByoHost{}
		Expect(clusterProxy.GetClient().Get(ctx, byoHostLookupKey, byoHostToBeReused)).Should(Succeed())
		_, ok = byoHostToBeReused.Labels[clusterv1.ClusterLabelName]
		Expect(ok).To(BeFalse())

		By("Creating a new cluster")
		caseContextData.ClusterName = fmt.Sprintf("%s-%s", specName, util.RandomString(6))
		clusterctl.ApplyClusterTemplateAndWait(ctx, clusterctl.ApplyClusterTemplateAndWaitInput{
			ClusterProxy: clusterProxy,
			ConfigCluster: clusterctl.ConfigClusterInput{
				LogFolder:                filepath.Join(artifactFolder, "clusters", clusterProxy.GetName()),
				ClusterctlConfigPath:     clusterctlConfigPath,
				KubeconfigPath:           clusterProxy.GetKubeconfigPath(),
				InfrastructureProvider:   clusterctl.DefaultInfrastructureProvider,
				Flavor:                   clusterctl.DefaultFlavor,
				Namespace:                namespace.Name,
				ClusterName:              caseContextData.ClusterName,
				KubernetesVersion:        e2eConfig.GetVariable(KubernetesVersion),
				ControlPlaneMachineCount: pointer.Int64Ptr(1),
				WorkerMachineCount:       pointer.Int64Ptr(1),
			},
			WaitForClusterIntervals:      e2eConfig.GetIntervals(specName, "wait-cluster"),
			WaitForControlPlaneIntervals: e2eConfig.GetIntervals(specName, "wait-control-plane"),
			WaitForMachineDeployments:    e2eConfig.GetIntervals(specName, "wait-worker-nodes"),
		}, clusterResources)

		// Assert on byohost cluster label to match clusterName
		byoHostToBeReused = &infrastructurev1beta1.ByoHost{}
		Expect(clusterProxy.GetClient().Get(ctx, byoHostLookupKey, byoHostToBeReused)).Should(Succeed())
		cluster, ok = byoHostToBeReused.Labels[clusterv1.ClusterLabelName]
		Expect(ok).To(BeTrue())
		Expect(cluster).To(Equal(caseContextData.ClusterName))
	})

	JustAfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			CollectInfo(caseContextData, collectInfoData)
			ShowInfoBeforeCaseQuit()
		}
	})

	AfterEach(func() {
		dumpSpecResourcesAndCleanup(caseContextData, artifactFolder, e2eConfig.GetIntervals, skipCleanup)
		cleanByohostPool(caseContextData, byoHostPoolData)
		if CurrentGinkgoTestDescription().Failed {
			ShowInfoAfterCaseQuit(collectInfoData)
		}
		RemoveLogs(collectInfoData)
	})
})
