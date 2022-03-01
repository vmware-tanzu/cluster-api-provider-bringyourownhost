// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// nolint: testpackage
package e2e

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	infraproviderv1 "github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/apis/infrastructure/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/bootstrap"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/util"
)

// Test suite flags
var (
	// configPath is the path to the e2e config file.
	configPath string

	// useExistingCluster instructs the test to use the current cluster instead of creating a new one (default discovery rules apply).
	useExistingCluster bool

	existingClusterKubeConfig string

	// artifactFolder is the folder to store e2e test artifacts.
	artifactFolder string

	// skipCleanup prevents cleanup of test resources e.g. for debug purposes.
	skipCleanup bool

	// e2eConfig to be used for this test, read from configPath.
	e2eConfig *clusterctl.E2EConfig

	// clusterctlConfigPath to be used for this test, created by generating a clusterctl local repository
	// with the providers specified in the configPath.
	clusterctlConfigPath string

	// bootstrapClusterProvider manages provisioning of the bootstrap cluster to be used for the e2e tests.
	// Please note that provisioning will be skipped if e2e.use-existing-cluster is provided.
	bootstrapClusterProvider bootstrap.ClusterProvider

	// bootstrapClusterProxy allows interacting with the bootstrap cluster to be used for the e2e tests.
	bootstrapClusterProxy framework.ClusterProxy

	// TODO: Remove this later
	clusterConName string

	testScope string

	// added by huchen
	alltestsTime time.Time

	byoHostCapacityPool int

	ctx context.Context

	dockerClient           *client.Client
	allbyohostContainerIDs []string
	allAgentLogFiles       []string
	namespace              *corev1.Namespace
	allClusterNames        []string

	specGeneralName = "e2e"
	cancelWatches   context.CancelFunc
)

func init() {
	By("huchen: suit-init")
	flag.StringVar(&configPath, "e2e.config", "", "path to the e2e config file")
	flag.StringVar(&artifactFolder, "e2e.artifacts-folder", "", "folder where e2e test artifact should be stored")
	flag.BoolVar(&skipCleanup, "e2e.skip-resource-cleanup", false, "if true, the resource cleanup after tests will be skipped")
	flag.BoolVar(&useExistingCluster, "e2e.use-existing-cluster", false, "if true, the test uses the current cluster instead of creating a new one (default discovery rules apply)")
	flag.StringVar(&existingClusterKubeConfig, "e2e.existing-cluster-kubeconfig-path", "", "path to the existing cluster's kubeconfig")
	flag.StringVar(&testScope, "e2e.test-scope", "", "test scope")
}

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

// Using a SynchronizedBeforeSuite for controlling how to create resources shared across ParallelNodes (~ginkgo threads).
// The local clusterctl repository & the bootstrap cluster are created once and shared across all the tests.
var _ = SynchronizedBeforeSuite(func() []byte {
	// Before all ParallelNodes.
	By("huchen: SynchronizedBeforeSuite-1")
	Expect(configPath).To(BeAnExistingFile(), "Invalid test suite argument. e2e.config should be an existing file.")
	Expect(os.MkdirAll(artifactFolder, 0755)).To(Succeed(), "Invalid test suite argument. Can't create e2e.artifacts-folder %q", artifactFolder)

	By("Initializing a runtime.Scheme with all the GVK relevant for this test")
	scheme := initScheme()

	Byf("Loading the e2e test configuration from %q", configPath)
	e2eConfig = loadE2EConfig(configPath)

	Byf("Creating a clusterctl local repository into %q", artifactFolder)
	clusterctlConfigPath = createClusterctlLocalRepository(e2eConfig, filepath.Join(artifactFolder, "repository"))

	By("Setting up the bootstrap cluster")
	bootstrapClusterProvider, bootstrapClusterProxy = setupBootstrapCluster(e2eConfig, scheme, useExistingCluster, existingClusterKubeConfig)

	By("Initializing the bootstrap cluster")
	initBootstrapCluster(bootstrapClusterProxy, e2eConfig, clusterctlConfigPath, artifactFolder)

	clusterConName = e2eConfig.ManagementClusterName
	return []byte(
		strings.Join([]string{
			artifactFolder,
			configPath,
			clusterctlConfigPath,
			bootstrapClusterProxy.GetKubeconfigPath(),
		}, ","),
	)
}, func(data []byte) {
	// Before each ParallelNode.
	By("huchen: suit-2")
	parts := strings.Split(string(data), ",")
	Expect(parts).To(HaveLen(4))

	artifactFolder = parts[0]
	configPath = parts[1]
	clusterctlConfigPath = parts[2]
	kubeconfigPath := parts[3]

	e2eConfig = loadE2EConfig(configPath)
	bootstrapClusterProxy = framework.NewClusterProxy("bootstrap", kubeconfigPath, initScheme(), framework.WithMachineLogCollector(framework.DockerLogCollector{}))

	check()

	// set up a Namespace where to host objects for this spec and create a watcher for the namespace events.
	namespace, cancelWatches = setupSpecNamespace(ctx, specGeneralName, bootstrapClusterProxy, artifactFolder)

	createByohostCapacityPool()

})

// Using a SynchronizedAfterSuite for controlling how to delete resources shared across ParallelNodes (~ginkgo threads).
// The bootstrap cluster is shared across all the tests, so it should be deleted only after all ParallelNodes completes.
// The local clusterctl repository is preserved like everything else created into the artifact folder.
var _ = SynchronizedAfterSuite(func() {
	// After each ParallelNode.
}, func() {
	// After all ParallelNodes.
	By("huchen: SynchronizedAfterSuite-1")

	if CurrentGinkgoTestDescription().Failed {
		ShowInfo(allAgentLogFiles)
	}

	cleanUp()

	By("Tearing down the management cluster")
	if !skipCleanup {
		tearDown(bootstrapClusterProvider, bootstrapClusterProxy)
	}

	// added by huchen
	Showf("huchen: all tests: time elapse: %v", time.Since(alltestsTime))
})

func initScheme() *runtime.Scheme {
	sc := runtime.NewScheme()
	framework.TryAddDefaultSchemes(sc)
	Expect(infraproviderv1.AddToScheme(sc)).NotTo(HaveOccurred())
	return sc
}

func cleanUp() {
	// Dumps all the resources in the spec namespace, then cleanups the cluster object and the spec namespace itself.
	dumpSpecResourcesAndCleanup(ctx, specGeneralName, bootstrapClusterProxy, artifactFolder, namespace, cancelWatches, e2eConfig.GetIntervals, skipCleanup)

	if dockerClient != nil {
		for _, byohostContainerID := range allbyohostContainerIDs {
			err := dockerClient.ContainerStop(ctx, byohostContainerID, nil)
			Expect(err).NotTo(HaveOccurred())

			err = dockerClient.ContainerRemove(ctx, byohostContainerID, types.ContainerRemoveOptions{})
			Expect(err).NotTo(HaveOccurred())
		}

	}

	for _, agentLogFile := range allAgentLogFiles {
		err := os.Remove(agentLogFile)
		if err != nil {
			Showf("error removing file %s: %v", agentLogFile, err)
		}
	}
	err := os.Remove(ReadByohControllerManagerLogShellFile)
	if err != nil {
		Showf("error removing file %s: %v", ReadByohControllerManagerLogShellFile, err)
	}
	err = os.Remove(ReadAllPodsShellFile)
	if err != nil {
		Showf("error removing file %s: %v", ReadAllPodsShellFile, err)
	}
}

func check() {

	ctx = context.TODO()
	Expect(ctx).NotTo(BeNil(), "ctx is required for spec")

	Expect(e2eConfig).NotTo(BeNil(), "Invalid argument. e2eConfig can't be nil when calling spec")
	Expect(clusterctlConfigPath).To(BeAnExistingFile(), "Invalid argument. clusterctlConfigPath must be an existing file when calling spec")
	Expect(bootstrapClusterProxy).NotTo(BeNil(), "Invalid argument. bootstrapClusterProxy can't be nil when calling spec")
	Expect(os.MkdirAll(artifactFolder, 0755)).To(Succeed(), "Invalid argument. artifactFolder can't be created for spec")

	Expect(e2eConfig.Variables).To(HaveKey(KubernetesVersion))
}

func createByohostCapacityPool() {
	if testScope == "" {
		byoHostCapacityPool = 6
	} else {
		byoHostCapacityPool = 2
	}

	By("Creating byohost capacity pool containing serveral hosts")
	for i := 0; i < byoHostCapacityPool; i++ {
		byoHostName := fmt.Sprintf("byohost-%s", util.RandomString(6))
		output, byohostContainerID, err := setupByoDockerHost(ctx, clusterConName, byoHostName, namespace.Name, dockerClient, bootstrapClusterProxy)
		allbyohostContainerIDs = append(allbyohostContainerIDs, byohostContainerID)
		Expect(err).NotTo(HaveOccurred())

		// read the log of host agent container in backend, and write it
		agentLogFile := fmt.Sprintf("/tmp/host-agent-%d.log", i)
		func() {
			f := WriteDockerLog(output, agentLogFile)
			defer func() {
				deferredErr := f.Close()
				if deferredErr != nil {
					Showf("error closing file %s:, %v", agentLogFile, deferredErr)
				}
			}()
		}()
		allAgentLogFiles = append(allAgentLogFiles, agentLogFile)
	}
}

func loadE2EConfig(configPath string) *clusterctl.E2EConfig {
	config := clusterctl.LoadE2EConfig(context.TODO(), clusterctl.LoadE2EConfigInput{ConfigPath: configPath})
	Expect(config).NotTo(BeNil(), "Failed to load E2E config from %s", configPath)

	return config
}

func createClusterctlLocalRepository(config *clusterctl.E2EConfig, repositoryFolder string) string {
	createRepositoryInput := clusterctl.CreateRepositoryInput{
		E2EConfig:        config,
		RepositoryFolder: repositoryFolder,
	}

	// Ensuring a CNI file is defined in the config and register a FileTransformation to inject the referenced file in place of the CNI_RESOURCES envSubst variable.
	Expect(config.Variables).To(HaveKey(CNIPath), "Missing %s variable in the config", CNIPath)
	cniPath := config.GetVariable(CNIPath)
	Expect(cniPath).To(BeAnExistingFile(), "The %s variable should resolve to an existing file", CNIPath)

	createRepositoryInput.RegisterClusterResourceSetConfigMapTransformation(cniPath, CNIResources)

	clusterctlConfig := clusterctl.CreateRepository(context.TODO(), createRepositoryInput)
	Expect(clusterctlConfig).To(BeAnExistingFile(), "The clusterctl config file does not exists in the local repository %s", repositoryFolder)
	return clusterctlConfig
}

func setupBootstrapCluster(config *clusterctl.E2EConfig, scheme *runtime.Scheme, useExistingCluster bool, existingClusterKubeConfig string) (bootstrap.ClusterProvider, framework.ClusterProxy) {
	var clusterProvider bootstrap.ClusterProvider
	kubeconfigPath := existingClusterKubeConfig
	if !useExistingCluster {
		clusterProvider = bootstrap.CreateKindBootstrapClusterAndLoadImages(context.TODO(), bootstrap.CreateKindBootstrapClusterAndLoadImagesInput{
			Name:               config.ManagementClusterName,
			RequiresDockerSock: config.HasDockerProvider(),
			Images:             config.Images,
			IPFamily:           config.GetVariable(IPFamily),
		})
		Expect(clusterProvider).NotTo(BeNil(), "Failed to create a bootstrap cluster")

		kubeconfigPath = clusterProvider.GetKubeconfigPath()
		Expect(kubeconfigPath).To(BeAnExistingFile(), "Failed to get the kubeconfig file for the bootstrap cluster")
	}

	clusterProxy := framework.NewClusterProxy("bootstrap", kubeconfigPath, scheme)
	Expect(clusterProxy).NotTo(BeNil(), "Failed to get a bootstrap cluster proxy")

	return clusterProvider, clusterProxy
}

func initBootstrapCluster(bootstrapClusterProxy framework.ClusterProxy, config *clusterctl.E2EConfig, clusterctlConfig, artifactFolder string) {
	clusterctl.InitManagementClusterAndWatchControllerLogs(context.TODO(), clusterctl.InitManagementClusterAndWatchControllerLogsInput{
		ClusterProxy:            bootstrapClusterProxy,
		ClusterctlConfigPath:    clusterctlConfig,
		InfrastructureProviders: config.InfrastructureProviders(),
		LogFolder:               filepath.Join(artifactFolder, "clusters", bootstrapClusterProxy.GetName()),
	}, config.GetIntervals(bootstrapClusterProxy.GetName(), "wait-controllers")...)
}

func tearDown(bootstrapClusterProvider bootstrap.ClusterProvider, bootstrapClusterProxy framework.ClusterProxy) {
	if bootstrapClusterProxy != nil {
		bootstrapClusterProxy.Dispose(context.TODO())
	}
	if bootstrapClusterProvider != nil {
		bootstrapClusterProvider.Dispose(context.TODO())
	}
}

func setupSpecNamespace(ctx context.Context, specName string, clusterProxy framework.ClusterProxy, artifactFolder string) (*corev1.Namespace, context.CancelFunc) {
	Byf("Creating a namespace for hosting the %q test spec", specName)
	namespace, cancelWatches := framework.CreateNamespaceAndWatchEvents(ctx, framework.CreateNamespaceAndWatchEventsInput{
		Creator:   clusterProxy.GetClient(),
		ClientSet: clusterProxy.GetClientSet(),
		Name:      fmt.Sprintf("%s-%s", specName, util.RandomString(6)),
		LogFolder: filepath.Join(artifactFolder, "clusters", clusterProxy.GetName()),
	})

	return namespace, cancelWatches
}

func dumpSpecResourcesAndCleanup(ctx context.Context, specName string, clusterProxy framework.ClusterProxy, artifactFolder string, namespace *corev1.Namespace, cancelWatches context.CancelFunc, intervalsGetter func(spec, key string) []interface{}, skipCleanup bool) {

	clusters := framework.GetAllClustersByNamespace(ctx, framework.GetAllClustersByNamespaceInput{
		Lister:    clusterProxy.GetClient(),
		Namespace: namespace.Name,
	})

	for _, cluster := range clusters {

		Byf("Dumping logs from the %q workload cluster", cluster.Name)
		// Dump all the logs from the workload cluster before deleting them.
		clusterProxy.CollectWorkloadClusterLogs(ctx, cluster.Namespace, cluster.Name, filepath.Join(artifactFolder, "clusters", cluster.Name, "machines"))

	}

	Byf("Dumping all the Cluster API resources in the %q namespace", namespace.Name)
	// Dump all Cluster API related resources to artifacts before deleting them.
	framework.DumpAllResources(ctx, framework.DumpAllResourcesInput{
		Lister:    clusterProxy.GetClient(),
		Namespace: namespace.Name,
		LogPath:   filepath.Join(artifactFolder, "clusters", clusterProxy.GetName(), "resources"),
	})

	if !skipCleanup {
		Byf("Deleting all clusters")
		// While https://github.com/kubernetes-sigs/cluster-api/issues/2955 is addressed in future iterations, there is a chance
		// that cluster variable is not set even if the cluster exists, so we are calling DeleteAllClustersAndWait
		// instead of DeleteClusterAndWait
		framework.DeleteAllClustersAndWait(ctx, framework.DeleteAllClustersAndWaitInput{
			Client:    clusterProxy.GetClient(),
			Namespace: namespace.Name,
		}, intervalsGetter(specName, "wait-delete-cluster")...)

		Byf("Deleting namespace", specName)
		framework.DeleteNamespace(ctx, framework.DeleteNamespaceInput{
			Deleter: clusterProxy.GetClient(),
			Name:    namespace.Name,
		})
	}

	cancelWatches()
}

func Byf(format string, a ...interface{}) {
	By(fmt.Sprintf(format, a...))
}
