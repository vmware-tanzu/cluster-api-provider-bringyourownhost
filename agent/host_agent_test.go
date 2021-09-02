package main

import (
	"context"
	"io/ioutil"
	"os"
	"os/exec"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/apis/infrastructure/v1alpha4"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Agent", func() {

	Context("When the host is unable to register with the API server", func() {
		var (
			ns              *corev1.Namespace
			err             error
			hostName        string
			fakedKubeConfig = "fake-kubeconfig-path"
			session         *gexec.Session
		)

		BeforeEach(func() {
			ns = common.NewNamespace(common.RandStr("testns-", 5))
			Expect(k8sClient.Create(context.TODO(), ns)).NotTo(HaveOccurred(), "failed to create test namespace")

			hostName, err = os.Hostname()
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			err = k8sClient.Delete(context.TODO(), ns)
			Expect(err).NotTo(HaveOccurred(), "failed to delete test namespace")
			session.Terminate().Wait()
		})

		It("should error out if the host already exists", func() {
			byoHost := &infrastructurev1alpha4.ByoHost{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ByoHost",
					APIVersion: "infrastructure.cluster.x-k8s.io/v1alpha4",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      hostName,
					Namespace: ns.Name,
				},
				Spec: infrastructurev1alpha4.ByoHostSpec{},
			}
			Expect(k8sClient.Create(context.TODO(), byoHost)).NotTo(HaveOccurred())

			command := exec.Command(pathToHostAgentBinary, "--kubeconfig", kubeconfigFile.Name(), "--namespace", ns.Name)
			session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
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
			ns       *corev1.Namespace
			session  *gexec.Session
			err      error
			workDir  string
			hostName string
		)

		BeforeEach(func() {
			ns = common.NewNamespace(common.RandStr("testns-", 5))
			Expect(k8sClient.Create(context.TODO(), ns)).NotTo(HaveOccurred(), "failed to create test namespace")

			hostName, err = os.Hostname()
			Expect(err).NotTo(HaveOccurred())

			command := exec.Command(pathToHostAgentBinary, "--kubeconfig", kubeconfigFile.Name(), "--namespace", ns.Name)

			session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			workDir, err = ioutil.TempDir("", "host-agent-ut")
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			err = k8sClient.Delete(context.TODO(), ns)
			Expect(err).NotTo(HaveOccurred())
			os.RemoveAll(workDir)
			session.Terminate().Wait()
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
