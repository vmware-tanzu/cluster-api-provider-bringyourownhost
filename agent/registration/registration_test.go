package registration_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/agent/registration"
	certv1 "k8s.io/api/certificates/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Registration", func() {
	var (
		ctx      = context.TODO()
		byohcsr  = registration.ByohCSR{}
		ns       = "default"
		hostName = "test-host"
	)
	BeforeEach(func() {
		byohcsr = registration.ByohCSR{K8sClient: k8sClient}
	})
	Context("When CSR does not already exist", func() {
		It("should create CSR", func() {
			Expect(byohcsr.CreateCSR(hostName, ns)).NotTo(HaveOccurred())
			actualByohCSR := &certv1.CertificateSigningRequest{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: hostName, Namespace: ns}, actualByohCSR)).ToNot(HaveOccurred())
			pemData, _ := pem.Decode(actualByohCSR.Spec.Request)
			Expect(pemData).ToNot(Equal(nil))
			csr, err := x509.ParseCertificateRequest(pemData.Bytes)
			Expect(err).ToNot(HaveOccurred())
			Expect(csr.Subject.CommonName).To(Equal(hostName))
		})
	})

	Context("When CSR already exist", func() {
		It("Should not create csr", func() {
			privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
			Expect(err).ToNot(HaveOccurred())

			// Generate a new *x509.CertificateRequest template
			csrTemplate := x509.CertificateRequest{
				Subject: pkix.Name{
					Organization: []string{"test-org"},
					CommonName:   hostName,
				},
			}

			// Generate the CSR bytes
			csrData, err := x509.CreateCertificateRequest(rand.Reader, &csrTemplate, privateKey)
			Expect(err).ToNot(HaveOccurred())

			existingByohCSR := &certv1.CertificateSigningRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:        hostName,
					Namespace:   ns,
					Labels:      map[string]string{},
					Annotations: map[string]string{},
				},
				Spec: certv1.CertificateSigningRequestSpec{
					Request:    pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrData}),
					SignerName: certv1.KubeAPIServerClientSignerName,
					Usages:     []certv1.KeyUsage{certv1.UsageClientAuth},
				},
			}
			Expect(k8sClient.Create(ctx, existingByohCSR)).NotTo(HaveOccurred())
			Expect(byohcsr.CreateCSR(hostName, ns)).NotTo(HaveOccurred())

			actualByohCSRs := &certv1.CertificateSigningRequestList{}
			Expect(k8sClient.List(ctx, actualByohCSRs)).ToNot(HaveOccurred())
			Expect(len(actualByohCSRs.Items)).To(Equal(1))

			pemData, _ := pem.Decode(actualByohCSRs.Items[0].Spec.Request)
			Expect(pemData).ToNot(Equal(nil))
			csr, err := x509.ParseCertificateRequest(pemData.Bytes)
			Expect(err).ToNot(HaveOccurred())
			Expect(csr.Subject.Organization[0]).To(Equal("test-org"))
		})
	})

})
