package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	infrastructurev1alpha4 "github.com/vmware-tanzu/cluster-api-provider-byoh/apis/infrastructure/v1alpha4"
	"github.com/vmware-tanzu/cluster-api-provider-byoh/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	"sigs.k8s.io/cluster-api/util/patch"
)

var _ = Describe("Agent", func() {

	Context("When the host is unable to register with the API server", func() {
		var (
			ns              *corev1.Namespace
			err             error
			hostName        string
			fakedKubeConfig string = "fake-kubeconfig-path"
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
			byoHost := common.NewByoHost(hostName, ns.Name, nil)
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

		It("should bootstrap the node when MachineRef is set", func() {

			bootstrapSecretName := "bootstrap-secret-1"
			machineName := "test-machine-1"
			byoMachineName := "test-byomachine-1"

			fileName1 := path.Join(workDir, "file-1.txt")
			fileOriginContent1 := "some-content-1"
			fileNewContent1 := " run cmd"

			fileName2 := path.Join(workDir, "file-2.txt")
			fileOriginContent2 := "some-content-2"
			fileAppendContent2 := "some-content-append-2"
			filePermission2 := 0777
			isAppend2 := true

			fileName3 := path.Join(workDir, "file-3.txt")
			fileContent3 := "some-content-3"
			fileBase64Content3 := base64.StdEncoding.EncodeToString([]byte(fileContent3))

			fileName4 := path.Join(workDir, "file-4.txt")
			fileContent4 := "some-content-4"
			fileGzipContent4, err := common.GzipData([]byte(fileContent4))
			Expect(err).NotTo(HaveOccurred())
			fileGzipBase64Content4 := base64.StdEncoding.EncodeToString(fileGzipContent4)

			//Init second file
			err = ioutil.WriteFile(fileName2, []byte(fileOriginContent2), 0644)
			Expect(err).NotTo(HaveOccurred())

			byoHost := &infrastructurev1alpha4.ByoHost{}
			byoHostLookupKey := types.NamespacedName{Name: hostName, Namespace: ns.Name}
			Eventually(func() bool {
				err = k8sClient.Get(context.TODO(), byoHostLookupKey, byoHost)
				return err == nil
			}).ShouldNot(BeFalse())

			bootstrapSecretUnencoded := fmt.Sprintf(`write_files:
- path: %s
  content: %s
- path: %s
  permissions: '%s'
  content: %s
  append: %v
- path: %s
  content: %s
  encoding: base64
- path: %s
  encoding: gzip+base64
  content: %s
runCmd:
- echo -n '%s' >> %s`, fileName1, fileOriginContent1, fileName2, strconv.FormatInt(int64(filePermission2), 8), fileAppendContent2, isAppend2, fileName3, fileBase64Content3, fileName4, fileGzipBase64Content4, fileNewContent1, fileName1)

			secret := common.NewSecret(bootstrapSecretName, bootstrapSecretUnencoded, ns.Name)
			Expect(k8sClient.Create(context.TODO(), secret)).NotTo(HaveOccurred())

			cluster := common.NewCluster(defaultClusterName, ns.Name)
			Expect(k8sClient.Create(context.TODO(), cluster)).NotTo(HaveOccurred())

			machine := common.NewMachine(machineName, ns.Name, cluster.Name)
			machine.Spec.Bootstrap = clusterv1.Bootstrap{
				DataSecretName: &secret.Name,
			}
			Expect(k8sClient.Create(context.TODO(), machine)).NotTo(HaveOccurred())

			byoMachine := common.NewByoMachine(byoMachineName, ns.Name, cluster.Name, machine)
			Expect(k8sClient.Create(context.TODO(), byoMachine)).NotTo(HaveOccurred())

			helper, err := patch.NewHelper(byoHost, k8sClient)
			Expect(err).NotTo(HaveOccurred())

			byoHost.Status.MachineRef = &corev1.ObjectReference{
				Kind:       "ByoMachine",
				Namespace:  ns.Name,
				Name:       byoMachine.Name,
				UID:        byoMachine.UID,
				APIVersion: byoHost.APIVersion,
			}
			byoHost.Spec.BootstrapSecret = &corev1.ObjectReference{
				Kind:      "Secret",
				Namespace: ns.Name,
				Name:      bootstrapSecretName,
			}
			err = helper.Patch(context.TODO(), byoHost)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() corev1.ConditionStatus {
				createdByoHost := &infrastructurev1alpha4.ByoHost{}
				err := k8sClient.Get(context.TODO(), byoHostLookupKey, createdByoHost)
				if err != nil {
					return corev1.ConditionFalse
				}
				for _, condition := range createdByoHost.Status.Conditions {
					if condition.Type == infrastructurev1alpha4.K8sComponentsInstallationSucceeded {
						return condition.Status
					}
				}
				return corev1.ConditionFalse
			}).Should(Equal(corev1.ConditionTrue))

			//check first file's content
			Eventually(func() string {
				buffer, err := ioutil.ReadFile(fileName1)
				if err != nil {
					return ""
				}
				return string(buffer)
			}).Should(Equal(fileOriginContent1 + fileNewContent1))

			//check second file's content
			Eventually(func() string {
				buffer, err := ioutil.ReadFile(fileName2)
				if err != nil {
					return ""
				}
				return string(buffer)
			}).Should(Equal(fileOriginContent2 + fileAppendContent2))

			//check second file permission
			Eventually(func() bool {
				stats, err := os.Stat(fileName2)
				if err == nil && stats.Mode() == fs.FileMode(filePermission2) {
					return true
				}
				return false
			}).Should(BeTrue())

			//check if third files's content decoded in base64 way successfully
			Eventually(func() string {
				buffer, err := ioutil.ReadFile(fileName3)
				if err != nil {
					return ""
				}
				return string(buffer)
			}).Should(Equal(fileContent3))

			//check if fourth files's content decoded in gzip+base64 way successfully
			Eventually(func() string {
				buffer, err := ioutil.ReadFile(fileName4)
				if err != nil {
					return ""
				}
				return string(buffer)
			}).Should(Equal(fileContent4))

		})
	})

})
