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
	"k8s.io/utils/pointer"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/util"
)

var _ = Describe("When testing MachineDeployment scale out/in", func() {

	var (
		caseContextData *CaseContext        = nil
		collectInfoData *CollectInfoContext = nil
		byoHostPoolData *ByoHostPoolContext = nil
	)

	BeforeEach(func() {

		caseContextData = new(CaseContext)
		Expect(caseContextData).NotTo(BeNil())
		caseContextData.CaseName = "md-scale"
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

	It("Should successfully scale a MachineDeployment up and down upon changes to the MachineDeployment replica count", func() {

		ctx := caseContextData.ctx
		clusterProxy := caseContextData.clusterProxy
		namespace := caseContextData.Namespace
		specName := caseContextData.CaseName
		clusterResources := caseContextData.ClusterResources

		caseContextData.ClusterName = fmt.Sprintf("%s-%s", specName, util.RandomString(6))

		byoHostPoolData.Capacity = 6
		Byf("Creating byohost capacity pool containing %d hosts", byoHostPoolData.Capacity)
		fl := setupByohostPool(caseContextData, collectInfoData, byoHostPoolData)
		for _, f := range fl {
			defer f.Close()
		}

		By("creating a workload cluster with one control plane node and one worker node")
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
				ControlPlaneMachineCount: pointer.Int64Ptr(3),
				WorkerMachineCount:       pointer.Int64Ptr(1),
			},
			WaitForClusterIntervals:      e2eConfig.GetIntervals(specName, "wait-cluster"),
			WaitForControlPlaneIntervals: e2eConfig.GetIntervals(specName, "wait-control-plane"),
			WaitForMachineDeployments:    e2eConfig.GetIntervals(specName, "wait-worker-nodes"),
		}, clusterResources)

		Expect(clusterResources.MachineDeployments[0].Spec.Replicas).To(Equal(pointer.Int32Ptr(1)))

		By("Scaling the MachineDeployment out to 3")
		framework.ScaleAndWaitMachineDeployment(ctx, framework.ScaleAndWaitMachineDeploymentInput{
			ClusterProxy:              clusterProxy,
			Cluster:                   clusterResources.Cluster,
			MachineDeployment:         clusterResources.MachineDeployments[0],
			Replicas:                  3,
			WaitForMachineDeployments: e2eConfig.GetIntervals(specName, "wait-worker-nodes"),
		})

		Expect(clusterResources.MachineDeployments[0].Spec.Replicas).To(Equal(pointer.Int32Ptr(3)))

		By("Scaling the MachineDeployment down to 2")
		framework.ScaleAndWaitMachineDeployment(ctx, framework.ScaleAndWaitMachineDeploymentInput{
			ClusterProxy:              clusterProxy,
			Cluster:                   clusterResources.Cluster,
			MachineDeployment:         clusterResources.MachineDeployments[0],
			Replicas:                  2,
			WaitForMachineDeployments: e2eConfig.GetIntervals(specName, "wait-worker-nodes"),
		})
		Expect(clusterResources.MachineDeployments[0].Spec.Replicas).To(Equal(pointer.Int32Ptr(2)))
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
