// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// nolint: testpackage
package e2e

import (
	"context"
	"fmt"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/util"
)

const (
	KubernetesVersion = "KUBERNETES_VERSION"
	CNIPath           = "CNI"
	CNIResources      = "CNI_RESOURCES"
	IPFamily          = "IP_FAMILY"
)

// creating a workload cluster
// This test is meant to provide a first, fast signal to detect regression; it is recommended to use it as a PR blocker test.
var _ = Describe("When BYOH joins existing cluster [PR-Blocking]", func() {

	var (
		specName = "quick-start"
	)

	It("Should create a workload cluster with single BYOH host", func() {
		clusterResources := new(clusterctl.ApplyClusterTemplateAndWaitResult)
		clusterName := fmt.Sprintf("%s-%s", specName, util.RandomString(6))

		setControlPlaneIP(context.Background(), dockerClient)
		clusterctl.ApplyClusterTemplateAndWait(ctx, clusterctl.ApplyClusterTemplateAndWaitInput{
			ClusterProxy: bootstrapClusterProxy,
			ConfigCluster: clusterctl.ConfigClusterInput{
				LogFolder:                filepath.Join(artifactFolder, "clusters", bootstrapClusterProxy.GetName()),
				ClusterctlConfigPath:     clusterctlConfigPath,
				KubeconfigPath:           bootstrapClusterProxy.GetKubeconfigPath(),
				InfrastructureProvider:   clusterctl.DefaultInfrastructureProvider,
				Flavor:                   clusterctl.DefaultFlavor,
				Namespace:                namespace.Name,
				ClusterName:              clusterName,
				KubernetesVersion:        e2eConfig.GetVariable(KubernetesVersion),
				ControlPlaneMachineCount: pointer.Int64Ptr(1),
				WorkerMachineCount:       pointer.Int64Ptr(1),
			},
			WaitForClusterIntervals:      e2eConfig.GetIntervals(specName, "wait-cluster"),
			WaitForControlPlaneIntervals: e2eConfig.GetIntervals(specName, "wait-control-plane"),
			WaitForMachineDeployments:    e2eConfig.GetIntervals(specName, "wait-worker-nodes"),
		}, clusterResources)

	})
})
