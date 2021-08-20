/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/system"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/pkg/errors"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/util"
)

const (
	KubernetesVersion  = "KUBERNETES_VERSION"
	CNIPath            = "CNI"
	CNIResources       = "CNI_RESOURCES"
	IPFamily           = "IP_FAMILY"
	KindImage          = "byoh/node:v1.19.11"
	TempKubeconfigPath = "/tmp/mgmt.conf"
)

var (
	AgentLogFile string
)

type cpConfig struct {
	followLink bool
	copyUIDGID bool
	sourcePath string
	destPath   string
	container  string
}

func resolveLocalPath(localPath string) (absPath string, err error) {
	if absPath, err = filepath.Abs(localPath); err != nil {
		return
	}
	return archive.PreserveTrailingDotOrSeparator(absPath, localPath, filepath.Separator), nil
}

func copyToContainer(ctx context.Context, cli *client.Client, copyConfig cpConfig) (err error) {
	srcPath := copyConfig.sourcePath
	dstPath := copyConfig.destPath

	srcPath, err = resolveLocalPath(srcPath)
	if err != nil {
		return err
	}

	// Prepare destination copy info by stat-ing the container path.
	dstInfo := archive.CopyInfo{Path: dstPath}
	dstStat, err := cli.ContainerStatPath(ctx, copyConfig.container, dstPath)

	// If the destination is a symbolic link, we should evaluate it.
	if err == nil && dstStat.Mode&os.ModeSymlink != 0 {
		linkTarget := dstStat.LinkTarget
		if !system.IsAbs(linkTarget) {
			// Join with the parent directory.
			dstParent, _ := archive.SplitPathDirEntry(dstPath)
			linkTarget = filepath.Join(dstParent, linkTarget)
		}

		dstInfo.Path = linkTarget
		dstStat, err = cli.ContainerStatPath(ctx, copyConfig.container, linkTarget)
	}

	// Validate the destination path
	if err = command.ValidateOutputPathFileMode(dstStat.Mode); err != nil {
		return errors.Wrapf(err, `destination "%s:%s" must be a directory or a regular file`, copyConfig.container, dstPath)
	}

	// Ignore any error and assume that the parent directory of the destination
	// path exists, in which case the copy may still succeed. If there is any
	// type of conflict (e.g., non-directory overwriting an existing directory
	// or vice versa) the extraction will fail. If the destination simply did
	// not exist, but the parent directory does, the extraction will still
	// succeed.
	if err == nil {
		dstInfo.Exists, dstInfo.IsDir = true, dstStat.Mode.IsDir()
	}

	var (
		content         io.ReadCloser
		resolvedDstPath string
	)

	// Prepare source copy info.
	srcInfo, err := archive.CopyInfoSourcePath(srcPath, copyConfig.followLink)
	if err != nil {
		return err
	}

	srcArchive, err := archive.TarResource(srcInfo)
	if err != nil {
		return err
	}
	defer srcArchive.Close()

	// With the stat info about the local source as well as the
	// destination, we have enough information to know whether we need to
	// alter the archive that we upload so that when the server extracts
	// it to the specified directory in the container we get the desired
	// copy behavior.

	// See comments in the implementation of `archive.PrepareArchiveCopy`
	// for exactly what goes into deciding how and whether the source
	// archive needs to be altered for the correct copy behavior when it is
	// extracted. This function also infers from the source and destination
	// info which directory to extract to, which may be the parent of the
	// destination that the user specified.
	dstDir, preparedArchive, err := archive.PrepareArchiveCopy(srcArchive, srcInfo, dstInfo)
	if err != nil {
		return err
	}
	defer preparedArchive.Close()

	resolvedDstPath = dstDir
	content = preparedArchive

	options := types.CopyToContainerOptions{
		AllowOverwriteDirWithFile: false,
		CopyUIDGID:                copyConfig.copyUIDGID,
	}

	return cli.CopyToContainer(ctx, copyConfig.container, resolvedDstPath, content, options)
}

// creating a workload cluster
// This test is meant to provide a first, fast signal to detect regression; it is recommended to use it as a PR blocker test.
var _ = Describe("When BYOH joins existing cluster", func() {

	var (
		ctx              context.Context
		specName         = "quick-start"
		namespace        *corev1.Namespace
		cancelWatches    context.CancelFunc
		clusterResources *clusterctl.ApplyClusterTemplateAndWaitResult
		dockerClient     *client.Client
		byohost          container.ContainerCreateCreatedBody
		err              error
	)

	BeforeEach(func() {

		ctx = context.TODO()
		Expect(ctx).NotTo(BeNil(), "ctx is required for %s spec", specName)

		Expect(e2eConfig).NotTo(BeNil(), "Invalid argument. e2eConfig can't be nil when calling %s spec", specName)
		Expect(clusterctlConfigPath).To(BeAnExistingFile(), "Invalid argument. clusterctlConfigPath must be an existing file when calling %s spec", specName)
		Expect(bootstrapClusterProxy).NotTo(BeNil(), "Invalid argument. bootstrapClusterProxy can't be nil when calling %s spec", specName)
		Expect(os.MkdirAll(artifactFolder, 0755)).To(Succeed(), "Invalid argument. artifactFolder can't be created for %s spec", specName)

		Expect(e2eConfig.Variables).To(HaveKey(KubernetesVersion))

		// Setup a Namespace where to host objects for this spec and create a watcher for the namespace events.
		namespace, cancelWatches = setupSpecNamespace(ctx, specName, bootstrapClusterProxy, artifactFolder)
		clusterResources = new(clusterctl.ApplyClusterTemplateAndWaitResult)
		AgentLogFile = common.RandStr("/tmp/agent", 5) + ".log"
	})

	It("Should create a workload cluster with single BYOH host", func() {

		clusterName := fmt.Sprintf("%s-%s", specName, util.RandomString(6))
		dockerClient, err = client.NewClientWithOpts(client.FromEnv)
		Expect(err).NotTo(HaveOccurred())

		tmpfs := map[string]string{"/run": "", "/tmp": ""}

		byohost, err = dockerClient.ContainerCreate(ctx,
			&container.Config{Hostname: "byohost",
				Image: KindImage,
			},
			&container.HostConfig{Privileged: true,
				SecurityOpt: []string{"seccomp=unconfined"},
				Tmpfs:       tmpfs,
				NetworkMode: "kind",
				Binds:       []string{"/var", "/lib/modules:/lib/modules:ro"},
			},
			&network.NetworkingConfig{EndpointsConfig: map[string]*network.EndpointSettings{"kind": {}}},
			nil, "byohost")

		Expect(err).NotTo(HaveOccurred())

		Expect(dockerClient.ContainerStart(ctx, byohost.ID, types.ContainerStartOptions{})).NotTo(HaveOccurred())

		pathToHostAgentBinary, err := gexec.Build("github.com/vmware-tanzu/cluster-api-provider-byoh/agent")
		Expect(err).NotTo(HaveOccurred())

		config := cpConfig{
			sourcePath: pathToHostAgentBinary,
			destPath:   "/agent",
			container:  byohost.ID,
		}

		Expect(copyToContainer(ctx, dockerClient, config)).NotTo(HaveOccurred())

		listopt := types.ContainerListOptions{}
		listopt.Filters = filters.NewArgs()
		listopt.Filters.Add("name", clusterConName+"-control-plane")

		containers, err := dockerClient.ContainerList(ctx, listopt)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(containers)).To(Equal(1))

		profile, err := dockerClient.ContainerInspect(ctx, containers[0].ID)
		Expect(err).NotTo(HaveOccurred())

		kubeconfig, err := os.ReadFile(bootstrapClusterProxy.GetKubeconfigPath())
		Expect(err).NotTo(HaveOccurred())

		re := regexp.MustCompile("server:.*")
		kubeconfig = re.ReplaceAll(kubeconfig, []byte("server: https://"+profile.NetworkSettings.Networks["kind"].IPAddress+":6443"))

		Expect(os.WriteFile(TempKubeconfigPath, kubeconfig, 0644)).NotTo(HaveOccurred())

		config.sourcePath = TempKubeconfigPath
		config.destPath = "/mgmt.conf"
		Expect(copyToContainer(ctx, dockerClient, config)).NotTo(HaveOccurred())

		rconfig := types.ExecConfig{
			AttachStdout: true,
			AttachStderr: true,
			Cmd:          []string{"./agent", "--kubeconfig", "/mgmt.conf"},
		}

		resp, err := dockerClient.ContainerExecCreate(ctx, byohost.ID, rconfig)
		Expect(err).NotTo(HaveOccurred())

		output, err := dockerClient.ContainerExecAttach(ctx, resp.ID, types.ExecStartCheck{})
		Expect(err).NotTo(HaveOccurred())
		defer output.Close()

		// write agent log for debug
		if AgentLogFile != "" {
			s := make(chan string)
			e := make(chan error)
			buf := bufio.NewReader(output.Reader)
			f, err := os.OpenFile(AgentLogFile, os.O_CREATE|os.O_WRONLY, 0666)
			Expect(err).NotTo(HaveOccurred())

			defer func() {
				f.Close()
			}()

			go func() {
				for {
					line, _, err := buf.ReadLine()
					if err != nil {
						// will be quit by this err: read unix @->/run/docker.sock: use of closed network connection
						e <- err
						break
					} else {
						s <- string(line)
					}
				}
			}()

			go func() {
				defer GinkgoRecover()
				for {
					select {
					case line := <-s:
						_, err2 := f.WriteString(line + "\n")
						if err2 != nil {
							Byf("Write String to file failed, err2=%v", err2)
						}
						_ = f.Sync()
					case err := <-e:
						// Please ignore this error if you see it in output
						Byf("Get err %v", err)
						return
					}
				}
			}()
		}

		clusterctl.ApplyClusterTemplateAndWait(ctx, clusterctl.ApplyClusterTemplateAndWaitInput{
			ClusterProxy: bootstrapClusterProxy,
			ConfigCluster: clusterctl.ConfigClusterInput{
				LogFolder:                filepath.Join(artifactFolder, "clusters", bootstrapClusterProxy.GetName()),
				ClusterctlConfigPath:     clusterctlConfigPath,
				KubeconfigPath:           bootstrapClusterProxy.GetKubeconfigPath(),
				InfrastructureProvider:   "byoh:v0.4.0",
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

	AfterEach(func() {
		if dockerClient != nil && byohost.ID != "" {
			err := dockerClient.ContainerStop(ctx, byohost.ID, nil)
			Expect(err).NotTo(HaveOccurred())

			err = dockerClient.ContainerRemove(ctx, byohost.ID, types.ContainerRemoveOptions{})
			Expect(err).NotTo(HaveOccurred())
		}

		if AgentLogFile != "" {
			os.Remove(AgentLogFile)
		}

		// Dumps all the resources in the spec namespace, then cleanups the cluster object and the spec namespace itself.
		dumpSpecResourcesAndCleanup(ctx, specName, bootstrapClusterProxy, artifactFolder, namespace, cancelWatches, clusterResources.Cluster, e2eConfig.GetIntervals, skipCleanup)
	})
})
