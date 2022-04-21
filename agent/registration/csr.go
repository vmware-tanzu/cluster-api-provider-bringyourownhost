package registration

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"

	certv1 "k8s.io/api/certificates/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const keySize = 2048

type ByohCSR struct {
	K8sClient client.Client
}

func (bcsr *ByohCSR) CreateCSR(hostname, namespace string) error {
	ctx := context.TODO()
	byoCSR := &certv1.CertificateSigningRequest{}
	err := bcsr.K8sClient.Get(ctx, types.NamespacedName{Name: hostname, Namespace: namespace}, byoCSR)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			klog.Errorf("error getting csr %s in namespace %s, err=%v", hostname, namespace, err)
			return err
		}
		byoCSR, err := bcsr.generateCSR(hostname, namespace)
		if err != nil {
			klog.Errorf("error generating csr %s in namespace %s, err=%v", hostname, namespace, err)
			return err
		}
		err = bcsr.K8sClient.Create(ctx, byoCSR)
		if err != nil {
			klog.Errorf("error creating host %s in namespace %s, err=%v", hostname, namespace, err)
			return err
		}
	}
	return nil
}

func (r *ByohCSR) generateCSR(hostname, namespace string) (*certv1.CertificateSigningRequest, error) {
	// Generate Private Key
	privateKey, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return nil, err
	}

	// Generate a new *x509.CertificateRequest template
	// TODO validate template
	csrTemplate := x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName: hostname,
		},
	}

	// Generate the CSR bytes
	csrData, err := x509.CreateCertificateRequest(rand.Reader, &csrTemplate, privateKey)
	if err != nil {
		return nil, err
	}
	// Create the CSR object
	csr := &certv1.CertificateSigningRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:        hostname,
			Namespace:   namespace,
			Labels:      map[string]string{},
			Annotations: map[string]string{},
		},
		Spec: certv1.CertificateSigningRequestSpec{
			Request:           pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrData}),
			SignerName:        certv1.KubeAPIServerClientSignerName,
			Usages:            []certv1.KeyUsage{certv1.UsageClientAuth},
			ExpirationSeconds: pointer.Int32(86400),
		},
	}
	return csr, nil
}
