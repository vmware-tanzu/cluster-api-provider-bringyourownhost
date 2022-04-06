// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/system"
	. "github.com/onsi/gomega" // nolint: stylecheck
	"github.com/pkg/errors"
	"sigs.k8s.io/cluster-api/test/framework"
)

const (
	kindImage          = "byoh/node:e2e"
	tempKubeconfigPath = "/tmp/mgmt.conf"
)

type cpConfig struct {
	followLink bool
	copyUIDGID bool
	sourcePath string
	destPath   string
	container  string
}

// ByoHostRunner runs bring-you-own-host cluster in docker
type ByoHostRunner struct {
	Context               context.Context
	clusterConName        string
	ByoHostName           string
	PathToHostAgentBinary string
	Namespace             string
	DockerClient          *client.Client
	NetworkInterface      string
	bootstrapClusterProxy framework.ClusterProxy
	CommandArgs           map[string]string
	Port                  string
	KubeconfigFile        string
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
	defer func() {
		deferredErr := srcArchive.Close()
		if deferredErr != nil {
			Showf("error in closing the src archive %v", deferredErr)
		}
	}()

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
	defer func() {
		deferredErr := preparedArchive.Close()
		if deferredErr != nil {
			Showf("error in closing the prepared archive %v", deferredErr)
		}
	}()

	resolvedDstPath = dstDir
	content = preparedArchive

	options := types.CopyToContainerOptions{
		AllowOverwriteDirWithFile: false,
		CopyUIDGID:                copyConfig.copyUIDGID,
	}

	return cli.CopyToContainer(ctx, copyConfig.container, resolvedDstPath, content, options)
}

func (r *ByoHostRunner) createDockerContainer() (container.ContainerCreateCreatedBody, error) {
	tmpfs := map[string]string{"/run": "", "/tmp": ""}

	return r.DockerClient.ContainerCreate(r.Context,
		&container.Config{Hostname: r.ByoHostName,
			Image: kindImage,
		},
		&container.HostConfig{Privileged: true,
			SecurityOpt: []string{"seccomp=unconfined"},
			Tmpfs:       tmpfs,
			NetworkMode: container.NetworkMode(r.NetworkInterface),
			Binds:       []string{"/var", "/lib/modules:/lib/modules:ro"},
		},
		&network.NetworkingConfig{EndpointsConfig: map[string]*network.EndpointSettings{r.NetworkInterface: {}}},
		nil, r.ByoHostName)
}

func (r *ByoHostRunner) copyKubeconfig(config cpConfig, listopt types.ContainerListOptions) error {
	var kubeconfig []byte
	if r.NetworkInterface == "host" {
		listopt.Filters.Add("name", r.ByoHostName)
		containers, err := r.DockerClient.ContainerList(r.Context, listopt)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(containers)).To(Equal(1))

		kubeconfig, err = os.ReadFile(r.KubeconfigFile)
		Expect(err).NotTo(HaveOccurred())

		re := regexp.MustCompile("server:.*")
		kubeconfig = re.ReplaceAll(kubeconfig, []byte("server: https://127.0.0.1:"+r.Port))
	} else {
		listopt.Filters.Add("name", r.clusterConName+"-control-plane")
		containers, err := r.DockerClient.ContainerList(r.Context, listopt)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(containers)).To(Equal(1))

		profile, err := r.DockerClient.ContainerInspect(r.Context, containers[0].ID)
		Expect(err).NotTo(HaveOccurred())

		kubeconfig, err = os.ReadFile(r.bootstrapClusterProxy.GetKubeconfigPath())
		Expect(err).NotTo(HaveOccurred())

		re := regexp.MustCompile("server:.*")
		kubeconfig = re.ReplaceAll(kubeconfig, []byte("server: https://"+profile.NetworkSettings.Networks[r.NetworkInterface].IPAddress+":6443"))
	}
	Expect(os.WriteFile(tempKubeconfigPath, kubeconfig, 0644)).NotTo(HaveOccurred()) // nolint: gosec,gomnd

	config.sourcePath = tempKubeconfigPath
	config.destPath = r.CommandArgs["--kubeconfig"]
	err := copyToContainer(r.Context, r.DockerClient, config)
	return err
}

// SetupByoDockerHost sets up the byohost docker container
func (r *ByoHostRunner) SetupByoDockerHost() (*container.ContainerCreateCreatedBody, error) {
	var byohost container.ContainerCreateCreatedBody
	var err error

	byohost, err = r.createDockerContainer()

	Expect(err).NotTo(HaveOccurred())
	Expect(r.DockerClient.ContainerStart(r.Context, byohost.ID, types.ContainerStartOptions{})).NotTo(HaveOccurred())

	config := cpConfig{
		sourcePath: r.PathToHostAgentBinary,
		destPath:   "/agent",
		container:  byohost.ID,
	}
	Expect(copyToContainer(r.Context, r.DockerClient, config)).NotTo(HaveOccurred())

	listopt := types.ContainerListOptions{}
	listopt.Filters = filters.NewArgs()

	err = r.copyKubeconfig(config, listopt)
	return &byohost, err
}

// ExecByoDockerHost runs the exec command in the byohost docker container
func (r *ByoHostRunner) ExecByoDockerHost(byohost *container.ContainerCreateCreatedBody) (types.HijackedResponse, string, error) {
	var cmdArgs []string
	cmdArgs = append(cmdArgs, "./agent")
	for flag, arg := range r.CommandArgs {
		cmdArgs = append(cmdArgs, flag, arg)
	}
	rconfig := types.ExecConfig{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          cmdArgs,
	}

	resp, err := r.DockerClient.ContainerExecCreate(r.Context, byohost.ID, rconfig)
	Expect(err).NotTo(HaveOccurred())

	output, err := r.DockerClient.ContainerExecAttach(r.Context, resp.ID, types.ExecStartCheck{})
	return output, byohost.ID, err
}

func setControlPlaneIP(ctx context.Context, dockerClient *client.Client) {
	_, ok := os.LookupEnv("CONTROL_PLANE_ENDPOINT_IP")
	if ok {
		return
	}
	inspect, _ := dockerClient.NetworkInspect(ctx, "kind", types.NetworkInspectOptions{})
	ipOctets := strings.Split(inspect.IPAM.Config[0].Subnet, ".")

	// The ControlPlaneEndpoint is a static IP that is in the hosts'
	// subnet but outside of its DHCP range. We believe 151 is a pretty
	// high number and we have < 10 containers being spun up, so we
	// can safely use this IP for the ControlPlaneEndpoint
	ipOctets[3] = "151"
	ip := strings.Join(ipOctets, ".")
	err := os.Setenv("CONTROL_PLANE_ENDPOINT_IP", ip)
	if err != nil {
		Expect(err).NotTo(HaveOccurred())
	}
}
