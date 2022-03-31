// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// nolint: testpackage
package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/jackpal/gateway"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/version"
	infrastructurev1beta1 "github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/apis/infrastructure/v1beta1"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/test/builder"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/test/e2e"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
)

var _ = Describe("Agent", func() {

	Context("When the host is unable to register with the API server", func() {
		var (
			ns               *corev1.Namespace
			ctx              context.Context
			err              error
			hostName         string
			runner           *e2e.ByoHostRunner
			byoHostContainer *container.ContainerCreateCreatedBody
		)

		BeforeEach(func() {
			ns = builder.Namespace("testns").Build()
			ctx = context.TODO()
			Expect(k8sClient.Create(context.TODO(), ns)).NotTo(HaveOccurred(), "failed to create test namespace")

			hostName, err = os.Hostname()
			Expect(err).NotTo(HaveOccurred())
			runner = setupTestInfra(ctx, hostName, getKubeConfig().Name(), ns)

			byoHostContainer, err = runner.SetupByoDockerHost()
			Expect(err).NotTo(HaveOccurred())

		})

		AfterEach(func() {
			cleanup(runner.Context, byoHostContainer, ns, agentLogFile)
		})

		It("should not error out if the host already exists", func() {
			// not using the builder method here
			// because builder makes use of GenerateName that generates random names
			// For the below byoHost we need the name to be deterministic
			byoHost := &infrastructurev1beta1.ByoHost{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ByoHost",
					APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      hostName,
					Namespace: ns.Name,
				},
				Spec: infrastructurev1beta1.ByoHostSpec{},
			}
			Expect(k8sClient.Create(context.TODO(), byoHost)).NotTo(HaveOccurred())

			runner.CommandArgs["--downloadpath"] = fakeDownloadPath
			output, _, err := runner.ExecByoDockerHost(byoHostContainer)
			Expect(err).NotTo(HaveOccurred())

			defer output.Close()
			f := e2e.WriteDockerLog(output, agentLogFile)
			defer func() {
				deferredErr := f.Close()
				if deferredErr != nil {
					e2e.Showf("error closing file %s: %v", agentLogFile, deferredErr)
				}
			}()
			Consistently(func() (done bool) {
				_, err := os.Stat(agentLogFile)
				if err == nil {
					data, err := os.ReadFile(agentLogFile)
					if err == nil && strings.Contains(string(data), "\"msg\"=\"error\"") {
						return true
					}
				}
				return false
			}).Should(BeFalse())
		})

		It("should return an error when invalid kubeconfig is passed in", func() {

			runner.CommandArgs["--kubeconfig"] = fakeKubeConfig
			output, _, err := runner.ExecByoDockerHost(byoHostContainer)
			Expect(err).NotTo(HaveOccurred())
			defer output.Close()

			f := e2e.WriteDockerLog(output, agentLogFile)
			defer func() {
				deferredErr := f.Close()
				if deferredErr != nil {
					e2e.Showf("error closing file %s: %v", agentLogFile, deferredErr)
				}
			}()
			Eventually(func() (done bool) {
				_, err := os.Stat(agentLogFile)
				if err == nil {
					data, err := os.ReadFile(agentLogFile)
					if err == nil && strings.Contains(string(data), "\"msg\"=\"error getting kubeconfig\"") {
						return true
					}
				}
				return false
			}).Should(BeTrue())
		})
	})

	Context("When the host agent is able to connect to API Server", func() {

		var (
			ns               *corev1.Namespace
			ctx              context.Context
			hostName         string
			fakeDownloadPath = "fake-download-path"
			runner           *e2e.ByoHostRunner
			byoHostContainer *container.ContainerCreateCreatedBody
			output           dockertypes.HijackedResponse
		)

		BeforeEach(func() {
			ns = builder.Namespace("testns").Build()
			Expect(k8sClient.Create(context.TODO(), ns)).NotTo(HaveOccurred(), "failed to create test namespace")
			ctx = context.TODO()
			var err error
			hostName, err = os.Hostname()
			Expect(err).NotTo(HaveOccurred())

			runner = setupTestInfra(ctx, hostName, getKubeConfig().Name(), ns)
			runner.CommandArgs["--label"] = "site=apac"
			runner.CommandArgs["--downloadpath"] = fakeDownloadPath

			byoHostContainer, err = runner.SetupByoDockerHost()
			Expect(err).NotTo(HaveOccurred())

			output, _, err = runner.ExecByoDockerHost(byoHostContainer)
			Expect(err).NotTo(HaveOccurred())

			// wait until the agent process starts inside the byoh host container
			Eventually(func() bool {
				containerTop, _ := runner.DockerClient.ContainerTop(ctx, byoHostContainer.ID, []string{})
				for _, proc := range containerTop.Processes {
					if strings.Contains(proc[len(containerTop.Titles)-1], "agent") {
						return true
					}

				}
				return false
			}, 60).Should(BeTrue())
		})

		AfterEach(func() {
			cleanup(runner.Context, byoHostContainer, ns, agentLogFile)
		})

		It("should register the BYOHost with the management cluster", func() {
			byoHostLookupKey := types.NamespacedName{Name: hostName, Namespace: ns.Name}
			createdByoHost := &infrastructurev1beta1.ByoHost{}
			Eventually(func() *infrastructurev1beta1.ByoHost {
				err := k8sClient.Get(context.TODO(), byoHostLookupKey, createdByoHost)
				if err != nil {
					return nil
				}
				return createdByoHost
			}).ShouldNot(BeNil())
		})

		It("should register the BYOHost with the passed labels", func() {
			byoHostLookupKey := types.NamespacedName{Name: hostName, Namespace: ns.Name}
			createdByoHost := &infrastructurev1beta1.ByoHost{}
			Eventually(func() map[string]string {
				err := k8sClient.Get(context.TODO(), byoHostLookupKey, createdByoHost)
				if err != nil {
					return nil
				}
				return createdByoHost.ObjectMeta.Labels
			}).Should(Equal(map[string]string{"site": "apac"}))
		})

		It("should fetch networkstatus when register the BYOHost with the management cluster", func() {
			byoHostLookupKey := types.NamespacedName{Name: hostName, Namespace: ns.Name}
			defaultIP, err := gateway.DiscoverInterface()
			Expect(err).NotTo(HaveOccurred())
			Eventually(func() bool {
				createdByoHost := &infrastructurev1beta1.ByoHost{}
				err := k8sClient.Get(context.TODO(), byoHostLookupKey, createdByoHost)
				if err != nil {
					return false
				}
				// check if default ip and networkInterfaceName is right
				for _, item := range createdByoHost.Status.Network {
					if item.IsDefault {
						iface, err := net.InterfaceByName(item.NetworkInterfaceName)
						if err != nil {
							return false
						}

						addrs, err := iface.Addrs()
						if err != nil {
							return false
						}

						for _, addr := range addrs {
							var ip net.IP
							switch v := addr.(type) {
							case *net.IPNet:
								ip = v.IP
							case *net.IPAddr:
								ip = v.IP
							}
							if ip.String() == defaultIP.String() {
								return true
							}
						}
					}
				}
				return false
			}).Should(BeTrue())

		})

		It("should only reconcile ByoHost resource that the agent created", func() {
			byoHost := builder.ByoHost(ns.Name, "random-second-host").Build()
			Expect(k8sClient.Create(context.TODO(), byoHost)).NotTo(HaveOccurred(), "failed to create byohost")

			defer output.Close()

			f := e2e.WriteDockerLog(output, agentLogFile)
			defer func() {
				deferredErr := f.Close()
				if deferredErr != nil {
					e2e.Showf("error closing file %s: %v", agentLogFile, deferredErr)
				}
			}()
			Consistently(func() (done bool) {
				_, err := os.Stat(agentLogFile)
				if err == nil {
					data, err := os.ReadFile(agentLogFile)
					if err == nil && strings.Contains(string(data), byoHost.Name) {
						return true
					}
				}
				return false
			}, 10, 1).ShouldNot(BeTrue())
		})
		Context("when machineref & bootstrap secret is assigned", func() {
			var (
				byoMachine *infrastructurev1beta1.ByoMachine
				namespace  types.NamespacedName
			)
			BeforeEach(func() {
				byoMachine = builder.ByoMachine(ns.Name, defaultByoMachineName).Build()
				Expect(k8sClient.Create(ctx, byoMachine)).Should(Succeed())
				byoHost := &infrastructurev1beta1.ByoHost{}
				namespace = types.NamespacedName{Name: hostName, Namespace: ns.Name}
				Eventually(func() (err error) {
					err = k8sClient.Get(ctx, namespace, byoHost)
					return err
				}).Should(BeNil())

				patchHelper, _ := patch.NewHelper(byoHost, k8sClient)
				byoHost.Status.MachineRef = &corev1.ObjectReference{
					APIVersion: byoMachine.APIVersion,
					Kind:       byoMachine.Kind,
					Namespace:  byoMachine.Namespace,
					Name:       byoMachine.Name,
					UID:        byoMachine.UID,
				}
				byoHost.Annotations = map[string]string{}
				byoHost.Annotations[infrastructurev1beta1.K8sVersionAnnotation] = K8sVersion
				byoHost.Annotations[infrastructurev1beta1.BundleLookupBaseRegistryAnnotation] = bundleLookupBaseRegistry
				byoHost.Annotations[infrastructurev1beta1.BundleLookupTagAnnotation] = BundleLookupTag

				fakeBootstrapSecret := builder.Secret(ns.Name, fakeBootstrapSecret).Build()
				err := k8sClient.Create(ctx, fakeBootstrapSecret)
				Expect(err).ToNot(HaveOccurred())
				byoHost.Spec.BootstrapSecret = &corev1.ObjectReference{
					Kind:      "Secret",
					Namespace: byoMachine.Namespace,
					Name:      fakeBootstrapSecret.Name,
				}

				Expect(patchHelper.Patch(ctx, byoHost, patch.WithStatusObservedGeneration{})).NotTo(HaveOccurred())
			})

			It("should install k8s components", func() {

				defer output.Close()
				f := e2e.WriteDockerLog(output, agentLogFile)
				defer func() {
					deferredErr := f.Close()
					if deferredErr != nil {
						e2e.Showf("error closing file %s: %v", agentLogFile, deferredErr)
					}
				}()
				updatedByoHost := &infrastructurev1beta1.ByoHost{}
				Eventually(func() (condition corev1.ConditionStatus) {
					err := k8sClient.Get(ctx, namespace, updatedByoHost)
					if err == nil {
						kubeInstallStatus := conditions.Get(updatedByoHost, infrastructurev1beta1.K8sComponentsInstallationSucceeded)
						if kubeInstallStatus != nil {
							return kubeInstallStatus.Status
						}
					}
					return corev1.ConditionFalse
				}, 100).Should(Equal(corev1.ConditionTrue)) // installing K8s components is a lengthy operation, setting the timeout to 100s
			})
		})
	})

	Context("When host agent is executed with --version flag", func() {
		var (
			tmpHostAgentBinary string
		)
		BeforeEach(func() {
			date, err := exec.Command("date").Output()
			Expect(err).NotTo(HaveOccurred())

			version.GitMajor = "1"
			version.GitMinor = "2"
			version.GitVersion = "v1.2.3"
			version.GitCommit = "abc"
			version.GitTreeState = "clean"
			version.BuildDate = string(date)

			ldflags := fmt.Sprintf("-X 'github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/version.GitMajor=%s'"+
				"-X 'github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/version.GitMinor=%s'"+
				"-X 'github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/version.GitVersion=%s'"+
				"-X 'github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/version.GitCommit=%s'"+
				"-X 'github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/version.GitTreeState=%s'"+
				"-X 'github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/version.BuildDate=%s'",
				version.GitMajor, version.GitMinor, version.GitVersion, version.GitCommit, version.GitTreeState, version.BuildDate)

			tmpHostAgentBinary, err = gexec.Build("github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent", "-ldflags", ldflags)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			version.GitMajor = ""
			version.GitMinor = ""
			version.GitVersion = ""
			version.GitCommit = ""
			version.GitTreeState = ""
			version.BuildDate = ""
			tmpHostAgentBinary = ""
		})

		It("Shows the appropriate version of the agent", func() {
			expectedStruct := version.Info{
				Major:        "1",
				Minor:        "2",
				GitVersion:   "v1.2.3",
				GitCommit:    "abc",
				GitTreeState: "clean",
				BuildDate:    version.BuildDate,
				GoVersion:    runtime.Version(),
				Compiler:     runtime.Compiler,
				Platform:     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
			}
			expected := fmt.Sprintf("byoh-hostagent version: %#v\n", expectedStruct)
			out, err := exec.Command(tmpHostAgentBinary, "--version").Output()
			Expect(err).NotTo(HaveOccurred())
			output := string(out)
			Expect(output).Should(Equal(expected))
		})
	})

	Context("When --version flag is created using 'version.sh' script", func() {
		var (
			tmpHostAgentBinary string
			gitMajor           string
			gitMinor           string
			gitVersion         string
			err                error
		)
		BeforeEach(func() {
			command := exec.Command("/bin/sh", "-c", "git describe --tags --abbrev=14 --match 'v[0-9]*' 2>/dev/null")
			command.Stderr = os.Stderr
			cmdOut, _ := command.Output()
			gitVersion = strings.TrimSuffix(string(cmdOut), "\n")

			gitVersion = strings.Split(gitVersion, "-")[0]
			gitVars := strings.Split(gitVersion, ".")
			if len(gitVars) > 1 {
				gitMajor = gitVars[0][1:]
				gitMinor = gitVars[1]
			}

			root, _ := exec.Command("/bin/sh", "-c", "git rev-parse --show-toplevel").Output()
			cmd := exec.Command("/bin/sh", "-c", strings.TrimSuffix(string(root), "\n")+"/hack/version.sh")
			ldflags, _ := cmd.Output()
			tmpHostAgentBinary, err = gexec.Build("github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent", "-ldflags", string(ldflags))
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			tmpHostAgentBinary = ""
			gitMajor = ""
			gitMinor = ""
			gitVersion = ""
		})

		It("should match local generated git values", func() {
			out, err := exec.Command(tmpHostAgentBinary, "--version").Output()
			Expect(err).NotTo(HaveOccurred())

			majorExpected := "Major:\"" + gitMajor + "\""
			Expect(out).Should(ContainSubstring(majorExpected))

			minorExpected := "Minor:\"" + gitMinor + "\""
			Expect(out).Should(ContainSubstring(minorExpected))

			gitVersionExpected := "GitVersion:\"" + gitVersion
			Expect(out).Should(ContainSubstring(gitVersionExpected))

		})
	})

	Context("When the host agent is executed with --skip-installation flag", func() {
		var (
			ns               *corev1.Namespace
			ctx              context.Context
			err              error
			hostName         string
			fakeDownloadPath = "fake-download-path"
			runner           *e2e.ByoHostRunner
			byoHostContainer *container.ContainerCreateCreatedBody
		)

		BeforeEach(func() {
			ns = builder.Namespace("testns").Build()
			ctx = context.TODO()
			Expect(k8sClient.Create(context.TODO(), ns)).NotTo(HaveOccurred(), "failed to create test namespace")

			hostName, err = os.Hostname()
			Expect(err).NotTo(HaveOccurred())
			runner = setupTestInfra(ctx, hostName, getKubeConfig().Name(), ns)

			byoHostContainer, err = runner.SetupByoDockerHost()
			Expect(err).NotTo(HaveOccurred())

		})

		AfterEach(func() {
			cleanup(runner.Context, byoHostContainer, ns, agentLogFile)
		})

		It("should skip installation of k8s components", func() {
			runner.CommandArgs["--downloadpath"] = fakeDownloadPath
			runner.CommandArgs["--skip-installation"] = ""
			output, _, err := runner.ExecByoDockerHost(byoHostContainer)
			Expect(err).NotTo(HaveOccurred())

			defer output.Close()
			f := e2e.WriteDockerLog(output, agentLogFile)
			defer func() {
				deferredErr := f.Close()
				if deferredErr != nil {
					e2e.Showf("error closing file %s: %v", agentLogFile, deferredErr)
				}
			}()
			Eventually(func() (done bool) {
				_, err := os.Stat(agentLogFile)
				if err == nil {
					data, err := os.ReadFile(agentLogFile)
					if err == nil && strings.Contains(string(data), "\"msg\"=\"skip-installation flag set, skipping installer initialisation\"") {
						return true
					}
				}
				return false
			}, 30).Should(BeTrue())
		})
	})

	Context("When the host agent is executed with SecureAccess feature flag", func() {

		var (
			ns               *corev1.Namespace
			ctx              context.Context
			hostName         string
			runner           *e2e.ByoHostRunner
			byoHostContainer *container.ContainerCreateCreatedBody
			output           dockertypes.HijackedResponse
		)

		BeforeEach(func() {
			ns = builder.Namespace("testns").Build()
			Expect(k8sClient.Create(context.TODO(), ns)).NotTo(HaveOccurred(), "failed to create test namespace")
			ctx = context.TODO()
			var err error
			hostName, err = os.Hostname()
			Expect(err).NotTo(HaveOccurred())

			runner = setupTestInfra(ctx, hostName, getKubeConfig().Name(), ns)
			runner.CommandArgs["--feature-gates"] = "SecureAccess=true"

			byoHostContainer, err = runner.SetupByoDockerHost()
			Expect(err).NotTo(HaveOccurred())

			output, _, err = runner.ExecByoDockerHost(byoHostContainer)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			cleanup(runner.Context, byoHostContainer, ns, agentLogFile)
		})

		It("should enable the SecureAccess feature gate", func() {
			defer output.Close()
			f := e2e.WriteDockerLog(output, agentLogFile)
			defer func() {
				deferredErr := f.Close()
				if deferredErr != nil {
					e2e.Showf("error closing file %s: %v", agentLogFile, deferredErr)
				}
			}()
			Eventually(func() (done bool) {
				_, err := os.Stat(agentLogFile)
				if err == nil {
					data, err := os.ReadFile(agentLogFile)
					if err == nil && strings.Contains(string(data), "\"msg\"=\"secure access enabled, waiting for host to be registered by ByoAdmission Controller\"") {
						return true
					}
				}
				return false
			}).Should(BeTrue())
		})

		It("should not register the BYOHost with the management cluster", func() {
			byoHostLookupKey := types.NamespacedName{Name: hostName, Namespace: ns.Name}
			createdByoHost := &infrastructurev1beta1.ByoHost{}
			Consistently(func() *infrastructurev1beta1.ByoHost {
				err := k8sClient.Get(context.TODO(), byoHostLookupKey, createdByoHost)
				if err != nil {
					return nil
				}
				return createdByoHost
			}).Should(BeNil())
		})

	})
})
