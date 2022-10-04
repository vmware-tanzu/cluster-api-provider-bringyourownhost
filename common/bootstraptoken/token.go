// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package bootstraptoken

import (
	"fmt"
	"time"

	infrastructurev1beta1 "github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/apis/infrastructure/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	bootstrapapi "k8s.io/cluster-bootstrap/token/api"
	bootstraputil "k8s.io/cluster-bootstrap/token/util"
)

// GetTokenIDSecretFromBootstrapToken splits the token string and returns the tokenID and tokenSecret parts
func GetTokenIDSecretFromBootstrapToken(tokenStr string) (tokenID, tokenSecret string, err error) {
	substrs := bootstraputil.BootstrapTokenRegexp.FindStringSubmatch(tokenStr)
	if len(substrs) != 3 { //nolint: gomnd
		return "", "", fmt.Errorf("the bootstrap token %q was not of the form %q", tokenStr, bootstrapapi.BootstrapTokenPattern)
	}

	return substrs[1], substrs[2], nil
}

// GenerateSecretFromBootstrapTokenStr builds the secret object from the token string
// It also adds default description and auth groups that can be used by the bootstrap-kubeconfig
func GenerateSecretFromBootstrapToken(tokenStr string, ttl time.Duration) (*v1.Secret, error) {
	tokenID, tokenSecret, err := GetTokenIDSecretFromBootstrapToken(tokenStr)
	if err != nil {
		return nil, err
	}
	secretData := map[string][]byte{
		bootstrapapi.BootstrapTokenIDKey:               []byte(tokenID),
		bootstrapapi.BootstrapTokenSecretKey:           []byte(tokenSecret),
		bootstrapapi.BootstrapTokenExpirationKey:       []byte(time.Now().UTC().Add(ttl).Format(time.RFC3339)),
		bootstrapapi.BootstrapTokenUsageSigningKey:     []byte("true"),
		bootstrapapi.BootstrapTokenUsageAuthentication: []byte("true"),
		bootstrapapi.BootstrapTokenDescriptionKey:      []byte(infrastructurev1beta1.BootstrapTokenDescription),
		bootstrapapi.BootstrapTokenExtraGroupsKey:      []byte(infrastructurev1beta1.BootstrapTokenExtraGroups),
	}

	bootstrapSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      bootstraputil.BootstrapTokenSecretName(tokenID),
			Namespace: metav1.NamespaceSystem,
		},
		Type: bootstrapapi.SecretTypeBootstrapToken,
		Data: secretData,
	}
	return bootstrapSecret, nil
}

// GenerateBootstrapKubeconfigFromBootstrapToken creates a bootstrap kubeconfig object from the bootstrap token generated
// It also adds default cluster, context and auth info
func GenerateBootstrapKubeconfigFromBootstrapToken(tokenStr string, bootstrapKubeconfig *infrastructurev1beta1.BootstrapKubeconfig) (*clientcmdapi.Config, error) {
	tokenID, tokenSecret, err := GetTokenIDSecretFromBootstrapToken(tokenStr)
	if err != nil {
		return nil, err
	}

	// Build resulting kubeconfig.
	kubeconfigData := clientcmdapi.Config{
		// Define a cluster stanza based on the bootstrap kubeconfig.
		Clusters: map[string]*clientcmdapi.Cluster{infrastructurev1beta1.DefaultClusterName: {
			Server:                   bootstrapKubeconfig.Spec.APIServer,
			InsecureSkipTLSVerify:    bootstrapKubeconfig.Spec.InsecureSkipTLSVerify,
			CertificateAuthorityData: []byte(bootstrapKubeconfig.Spec.CertificateAuthorityData),
		}},
		// Define auth based on the obtained client cert.
		AuthInfos: map[string]*clientcmdapi.AuthInfo{infrastructurev1beta1.DefaultAuth: {
			Token: fmt.Sprintf(tokenID + "." + tokenSecret),
		}},
		// Define a context that connects the auth info and cluster, and set it as the default
		Contexts: map[string]*clientcmdapi.Context{infrastructurev1beta1.DefaultContext: {
			Cluster:   infrastructurev1beta1.DefaultClusterName,
			AuthInfo:  infrastructurev1beta1.DefaultAuth,
			Namespace: infrastructurev1beta1.DefaultNamespace,
		}},
		CurrentContext: infrastructurev1beta1.DefaultContext,
	}

	return &kubeconfigData, nil
}
