// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1beta1_test

import (
	b64 "encoding/base64"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	byohv1beta1 "github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/apis/infrastructure/v1beta1"
	"github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/test/builder"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/cluster-api/util/patch"
)

var _ = Describe("BootstrapKubeconfig Webhook", func() {

	var (
		bootstrapKubeconfig         *byohv1beta1.BootstrapKubeconfig
		err                         error
		defaultNamespace            = "default"
		testBootstrapKubeconfigName = "test-bootstrap-kubeconfig"
		testServerEmpty             = ""
		testServerInvalidURL        = "htt p://test.com"
		testServerWithoutScheme     = "abc.com"
		testServerWithoutHostname   = "https://test-server"
		testServerWithoutPort       = "https://test.com"
		testServerValid             = "https://abc.com:1234"
		testCADataEmpty             = ""
		testCADataInvalid           = "test-ca-data"
		testPEMDataInvalid          = b64.StdEncoding.EncodeToString([]byte(testCADataInvalid))
	)
	Context("When BootstrapKubeconfig gets a create request", func() {

		It("should reject the request if APIServer is not a valid URL", func() {
			bootstrapKubeconfig = builder.BootstrapKubeconfig(defaultNamespace, testBootstrapKubeconfigName).
				WithServer(testServerInvalidURL).
				Build()
			err = k8sClient.Create(ctx, bootstrapKubeconfig)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fmt.Sprintf("admission webhook \"vbootstrapkubeconfig.kb.io\" denied the request: spec.apiserver: Invalid value: %q: APIServer URL is not valid", testServerInvalidURL)))
		})

		It("should reject the request if APIServer field is empty", func() {
			bootstrapKubeconfig = builder.BootstrapKubeconfig(defaultNamespace, testBootstrapKubeconfigName).
				WithServer(testServerEmpty).
				Build()
			err = k8sClient.Create(ctx, bootstrapKubeconfig)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("admission webhook \"vbootstrapkubeconfig.kb.io\" denied the request: spec.apiserver: Invalid value: \"\": APIServer field cannot be empty"))

		})

		It("should reject the request if APIServer address does not have https scheme specified", func() {
			bootstrapKubeconfig = builder.BootstrapKubeconfig(defaultNamespace, testBootstrapKubeconfigName).
				WithServer(testServerWithoutScheme).
				Build()
			err = k8sClient.Create(ctx, bootstrapKubeconfig)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fmt.Sprintf("admission webhook \"vbootstrapkubeconfig.kb.io\" denied the request: spec.apiserver: Invalid value: %q: APIServer is not of the format https://hostname:port", testServerWithoutScheme)))
		})

		It("should reject the request if APIServer address hostname is not specified", func() {
			bootstrapKubeconfig = builder.BootstrapKubeconfig(defaultNamespace, testBootstrapKubeconfigName).
				WithServer(testServerWithoutHostname).
				Build()
			err = k8sClient.Create(ctx, bootstrapKubeconfig)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fmt.Sprintf("admission webhook \"vbootstrapkubeconfig.kb.io\" denied the request: spec.apiserver: Invalid value: %q: APIServer is not of the format https://hostname:port", testServerWithoutHostname)))
		})

		It("should reject the request if APIServer address does not have the port info", func() {
			bootstrapKubeconfig = builder.BootstrapKubeconfig(defaultNamespace, testBootstrapKubeconfigName).
				WithServer(testServerWithoutPort).
				Build()
			err = k8sClient.Create(ctx, bootstrapKubeconfig)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fmt.Sprintf("admission webhook \"vbootstrapkubeconfig.kb.io\" denied the request: spec.apiserver: Invalid value: %q: APIServer is not of the format https://hostname:port", testServerWithoutPort)))
		})

		It("should reject the request if CertificateAuthorityData field is empty", func() {
			bootstrapKubeconfig = builder.BootstrapKubeconfig(defaultNamespace, testBootstrapKubeconfigName).
				WithServer(testServerValid).
				WithCAData(testCADataEmpty).
				Build()
			err = k8sClient.Create(ctx, bootstrapKubeconfig)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("admission webhook \"vbootstrapkubeconfig.kb.io\" denied the request: spec.caData: Invalid value: \"\": CertificateAuthorityData field cannot be empty"))

		})

		It("should reject request if CertificateAuthorityData cannot be base64 decoded", func() {
			bootstrapKubeconfig = builder.BootstrapKubeconfig(defaultNamespace, testBootstrapKubeconfigName).
				WithServer(testServerValid).
				WithCAData(testCADataInvalid).
				Build()
			err = k8sClient.Create(ctx, bootstrapKubeconfig)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fmt.Sprintf("admission webhook \"vbootstrapkubeconfig.kb.io\" denied the request: spec.caData: Invalid value: %q: cannot base64 decode CertificateAuthorityData", testCADataInvalid)))

		})

		It("should reject request if CertificateAuthorityData is not PEM encoded", func() {
			bootstrapKubeconfig = builder.BootstrapKubeconfig(defaultNamespace, testBootstrapKubeconfigName).
				WithServer(testServerValid).
				WithCAData(testPEMDataInvalid).
				Build()
			err = k8sClient.Create(ctx, bootstrapKubeconfig)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fmt.Sprintf("admission webhook \"vbootstrapkubeconfig.kb.io\" denied the request: spec.caData: Invalid value: %q: CertificateAuthorityData is not PEM encoded", testPEMDataInvalid)))

		})

		It("should accept the request if all fields are valid", func() {
			// use from config of envtest
			testCADataValid := b64.StdEncoding.EncodeToString(cfg.CAData)

			bootstrapKubeconfig = builder.BootstrapKubeconfig(defaultNamespace, testBootstrapKubeconfigName).
				WithServer(testServerValid).
				WithCAData(testCADataValid).
				Build()
			err = k8sClient.Create(ctx, bootstrapKubeconfig)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When BootstrapKubeconfig gets an update request", func() {
		var (
			ph                         *patch.Helper
			createdBootstrapKubeconfig *byohv1beta1.BootstrapKubeconfig
		)
		BeforeEach(func() {
			// use from config of envtest
			testCADataValid := b64.StdEncoding.EncodeToString(cfg.CAData)

			bootstrapKubeconfig = builder.BootstrapKubeconfig(defaultNamespace, testBootstrapKubeconfigName).
				WithServer(testServerValid).
				WithCAData(testCADataValid).
				Build()
			err = k8sClient.Create(ctx, bootstrapKubeconfig)
			Expect(err).NotTo(HaveOccurred())

			createdBootstrapKubeconfig = &byohv1beta1.BootstrapKubeconfig{}
			namespacedName := types.NamespacedName{Name: bootstrapKubeconfig.Name, Namespace: defaultNamespace}
			Eventually(func() error {
				err = k8sClient.Get(ctx, namespacedName, createdBootstrapKubeconfig)
				if err != nil {
					return err
				}
				return nil
			}).Should(BeNil())

			// create a patch helper
			ph, err = patch.NewHelper(bootstrapKubeconfig, k8sClient)
			Expect(err).ShouldNot(HaveOccurred())
		})

		AfterEach(func() {
			err = k8sClient.Delete(ctx, bootstrapKubeconfig)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should reject the request if APIServer field is empty", func() {
			createdBootstrapKubeconfig.Spec.APIServer = testServerEmpty
			err = ph.Patch(ctx, createdBootstrapKubeconfig)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("admission webhook \"vbootstrapkubeconfig.kb.io\" denied the request: spec.apiserver: Invalid value: \"\": APIServer field cannot be empty"))

		})

		It("should reject the request if APIServer is not of the correct format", func() {
			createdBootstrapKubeconfig.Spec.APIServer = testServerWithoutHostname
			err = ph.Patch(ctx, createdBootstrapKubeconfig)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fmt.Sprintf("admission webhook \"vbootstrapkubeconfig.kb.io\" denied the request: spec.apiserver: Invalid value: %q: APIServer is not of the format https://hostname:port", testServerWithoutHostname)))
		})

		It("should reject the request if CertificateAuthorityData field is empty", func() {
			createdBootstrapKubeconfig.Spec.CertificateAuthorityData = testCADataEmpty
			err = ph.Patch(ctx, createdBootstrapKubeconfig)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("admission webhook \"vbootstrapkubeconfig.kb.io\" denied the request: spec.caData: Invalid value: \"\": CertificateAuthorityData field cannot be empty"))

		})

		It("should reject request if CertificateAuthorityData cannot be base64 decoded", func() {
			createdBootstrapKubeconfig.Spec.CertificateAuthorityData = testCADataInvalid
			err = ph.Patch(ctx, createdBootstrapKubeconfig)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fmt.Sprintf("admission webhook \"vbootstrapkubeconfig.kb.io\" denied the request: spec.caData: Invalid value: %q: cannot base64 decode CertificateAuthorityData", testCADataInvalid)))

		})

		It("should reject request if CertificateAuthorityData is not PEM encoded", func() {
			createdBootstrapKubeconfig.Spec.CertificateAuthorityData = testPEMDataInvalid
			err = ph.Patch(ctx, createdBootstrapKubeconfig)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fmt.Sprintf("admission webhook \"vbootstrapkubeconfig.kb.io\" denied the request: spec.caData: Invalid value: %q: CertificateAuthorityData is not PEM encoded", testPEMDataInvalid)))

		})

		It("should accept the request if all fields are valid", func() {
			// patch a valid APIServer value
			createdBootstrapKubeconfig.Spec.APIServer = "https://1.2.3.4:5678"
			err = ph.Patch(ctx, createdBootstrapKubeconfig)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When BootstrapKubeconfig gets a delete request", func() {
		var (
			createdBootstrapKubeconfig *byohv1beta1.BootstrapKubeconfig
			namespacedName             types.NamespacedName
		)
		BeforeEach(func() {
			// use from config of envtest
			testCADataValid := b64.StdEncoding.EncodeToString(cfg.CAData)

			bootstrapKubeconfig = builder.BootstrapKubeconfig(defaultNamespace, testBootstrapKubeconfigName).
				WithServer(testServerValid).
				WithCAData(testCADataValid).
				Build()
			err = k8sClient.Create(ctx, bootstrapKubeconfig)
			Expect(err).NotTo(HaveOccurred())

			createdBootstrapKubeconfig = &byohv1beta1.BootstrapKubeconfig{}
			namespacedName = types.NamespacedName{Name: bootstrapKubeconfig.Name, Namespace: defaultNamespace}
			Eventually(func() error {
				err = k8sClient.Get(ctx, namespacedName, createdBootstrapKubeconfig)
				if err != nil {
					return err
				}
				return nil
			}).Should(BeNil())

		})

		It("should delete the BootstrapKubeconfig instance", func() {
			err = k8sClient.Delete(ctx, createdBootstrapKubeconfig)
			Expect(err).NotTo(HaveOccurred())

			deletedBootstrapKubeconfig := &byohv1beta1.BootstrapKubeconfig{}
			Eventually(func() bool {
				err = k8sClient.Get(ctx, namespacedName, deletedBootstrapKubeconfig)
				if err != nil {
					if apierrors.IsNotFound(err) {
						return true
					}
				}
				return false
			}).Should(BeTrue())
		})
	})
})
