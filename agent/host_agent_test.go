// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// nolint: testpackage
package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"runtime"
	

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jackpal/gateway"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/version"
	infrastructurev1beta1 "github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/apis/infrastructure/v1beta1"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/test/builder"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Agent", func() {

	Context("When the host is unable to register with the API server", func() {
		var (
			ns               *corev1.Namespace
			err              error
			hostName         string
			fakedKubeConfig  = "fake-kubeconfig-path"
			fakeDownloadPath = "fake-download-path"
			session          *gexec.Session
		)

		BeforeEach(func() {
			ns = builder.Namespace("testns").Build()
			Expect(k8sClient.Create(context.TODO(), ns)).NotTo(HaveOccurred(), "failed to create test namespace")

			hostName, err = os.Hostname()
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			err = k8sClient.Delete(context.TODO(), ns)
			Expect(err).NotTo(HaveOccurred(), "failed to delete test namespace")
			session.Terminate().Wait()
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

			command := exec.Command(pathToHostAgentBinary, "--kubeconfig", kubeconfigFile.Name(), "--namespace", ns.Name, "--downloadpath", fakeDownloadPath)
			session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Consistently(session).ShouldNot(gexec.Exit(0))
		})

		It("should return an error when invalid kubeconfig is passed in", func() {
			command := exec.Command(pathToHostAgentBinary, "--kubeconfig", fakedKubeConfig)
			session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
		})
	})

	Context("When the host agent is able to connect to API Server", func() {

		var (
			ns               *corev1.Namespace
			session          *gexec.Session
			err              error
			workDir          string
			hostName         string
			fakeDownloadPath = "fake-download-path"
		)

		BeforeEach(func() {
			ns = builder.Namespace("testns").Build()
			Expect(k8sClient.Create(context.TODO(), ns)).NotTo(HaveOccurred(), "failed to create test namespace")

			hostName, err = os.Hostname()
			Expect(err).NotTo(HaveOccurred())

			command := exec.Command(pathToHostAgentBinary, "--kubeconfig", kubeconfigFile.Name(), "--namespace", ns.Name, "--label", "site=apac", "--downloadpath", fakeDownloadPath)

			session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			workDir, err = ioutil.TempDir("", "host-agent-ut")
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			err = k8sClient.Delete(context.TODO(), ns)
			Expect(err).NotTo(HaveOccurred())
			err = os.RemoveAll(workDir)
			Expect(err).NotTo(HaveOccurred())
			session.Terminate().Wait()
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

			Consistently(session.Err, "10s").ShouldNot(gbytes.Say(byoHost.Name))
		})
	})

	Context("When host agent is executed with --version flag", func() {
		var (
			tmpHostAgentBinary string
		)
		BeforeEach(func() {
			date, err := exec.Command("date").Output()
			Expect(err).NotTo(HaveOccurred())
			version.BuildDate = string(date)
			version.Version = "v1.2.3"
			ldflags := fmt.Sprintf("-X 'github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/version.Version=%s'"+
				" -X 'github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/version.BuildDate=%s'", version.Version, version.BuildDate)
			tmpHostAgentBinary, err = gexec.Build("github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent", "-ldflags", ldflags)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			version.BuildDate = ""
			version.Version = ""
			tmpHostAgentBinary = ""
		})

		It("Shows the appropriate version of the agent", func() {
			expectedStruct := version.Info{
				Major:     "1",
				Minor:     "2",
				Patch:     "3",
				BuildDate: version.BuildDate,
				GoVersion: runtime.Version(),
				Compiler:  runtime.Compiler,
				Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
			}
			expected := fmt.Sprintf("byoh-hostagent version: %#v\n", expectedStruct)
			out, err := exec.Command(tmpHostAgentBinary, "--version").Output()
			Expect(err).NotTo(HaveOccurred())
			output := string(out)
			Expect(output).Should(Equal(expected))
		})
	})
})
