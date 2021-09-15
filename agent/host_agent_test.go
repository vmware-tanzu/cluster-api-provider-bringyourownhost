package main

import (
	"context"
	"os"
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/agent/registration"
	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/apis/infrastructure/v1alpha4"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Agent", func() {

	Context("When the host is unable to register with the API server", func() {
		var (
			ns                      *corev1.Namespace
			err                     error
			fakedKubeConfig         = "fake-kubeconfig-path"
			session                 *gexec.Session
			ByoHostRegsiterFileName string
			hostName                string
		)

		BeforeEach(func() {
			ns = common.NewNamespace("testns")
			Expect(k8sClient.Create(context.TODO(), ns)).NotTo(HaveOccurred(), "failed to create test namespace")

			hostName, err = os.Hostname()
			Expect(err).NotTo(HaveOccurred())

			registerClient := &registration.HostRegistrar{}
			err = registerClient.GetByoHostRegsiterFileName()
			Expect(err).NotTo(HaveOccurred(), "failed to get Regsiter File")
			ByoHostRegsiterFileName = registerClient.ByoHostRegsiterFileName
		})

		AfterEach(func() {
			err = k8sClient.Delete(context.TODO(), ns)
			Expect(err).NotTo(HaveOccurred(), "failed to delete test namespace")
		})

		It("should not error out when the host restart", func() {
			command := exec.Command(pathToHostAgentBinary, "--kubeconfig", kubeconfigFile.Name(), "--namespace", ns.Name)
			session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			// Make sure byohost is created
			Eventually(func() bool {
				isExisted, _ := common.IsFileExists(ByoHostRegsiterFileName)
				return isExisted
			}).Should(BeTrue())

			session.Terminate().Wait()

			command = exec.Command(pathToHostAgentBinary, "--kubeconfig", kubeconfigFile.Name(), "--namespace", ns.Name)
			session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Consistently(session).ShouldNot(gexec.Exit(0))
			session.Terminate().Wait()

			Expect(os.Remove(ByoHostRegsiterFileName)).NotTo(HaveOccurred())

		})

		It("should error out when a byohost with same hostname existed", func() {
			byoHost := common.NewByoHost(hostName, ns.Name)
			Expect(k8sClient.Create(context.TODO(), byoHost)).NotTo(HaveOccurred())

			// Make sure byohost is created
			Eventually(func() bool {
				byoHostLookupKey := types.NamespacedName{Name: hostName, Namespace: ns.Name}
				createdByoHost := &infrastructurev1alpha4.ByoHost{}
				err = k8sClient.Get(context.TODO(), byoHostLookupKey, createdByoHost)
				return (err == nil)
			}).Should(BeTrue())

			command := exec.Command(pathToHostAgentBinary, "--kubeconfig", kubeconfigFile.Name(), "--namespace", ns.Name)
			session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
			session.Terminate().Wait()
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
			ns                      *corev1.Namespace
			session                 *gexec.Session
			err                     error
			hostName                string
			ByoHostRegsiterFileName string
		)

		BeforeEach(func() {
			registerClient := &registration.HostRegistrar{}
			err = registerClient.GetByoHostRegsiterFileName()
			Expect(err).NotTo(HaveOccurred(), "failed to get Regsiter File")
			ByoHostRegsiterFileName = registerClient.ByoHostRegsiterFileName

			ns = common.NewNamespace("testns")
			Expect(k8sClient.Create(context.TODO(), ns)).NotTo(HaveOccurred(), "failed to create test namespace")

			hostName, err = os.Hostname()
			Expect(err).NotTo(HaveOccurred())

			command := exec.Command(pathToHostAgentBinary, "--kubeconfig", kubeconfigFile.Name(), "--namespace", ns.Name)

			session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			err = k8sClient.Delete(context.TODO(), ns)
			Expect(err).NotTo(HaveOccurred())

			session.Terminate().Wait()
			Expect(os.Remove(ByoHostRegsiterFileName)).NotTo(HaveOccurred())
		})

		It("should register the BYOHost with the management cluster", func() {
			byoHostLookupKey := types.NamespacedName{Name: hostName, Namespace: ns.Name}
			Eventually(func() *infrastructurev1alpha4.ByoHost {
				createdByoHost := &infrastructurev1alpha4.ByoHost{}
				err := k8sClient.Get(context.TODO(), byoHostLookupKey, createdByoHost)
				if err != nil {
					return nil
				}
				return createdByoHost
			}).ShouldNot(BeNil())
		})

		It("should fetch networkstatus when register the BYOHost with the management cluster", func() {
			byoHostLookupKey := types.NamespacedName{Name: hostName, Namespace: ns.Name}
			Eventually(func() bool {
				createdByoHost := &infrastructurev1alpha4.ByoHost{}
				err := k8sClient.Get(context.TODO(), byoHostLookupKey, createdByoHost)
				if err != nil {
					return false
				}
				if len(createdByoHost.Status.Network) != 0 {
					return true
				}
				return false
			}).Should(BeTrue())

		})
	})
})
