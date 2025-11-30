/*
Copyright © 2025 ESO Maintainer Team

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package vault

import (
	"context"
	"fmt"
	"net/http"
	"time"

	vault "github.com/hashicorp/vault/api"

	//nolint
	. "github.com/onsi/ginkgo/v2"

	//nolint
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/framework/addon"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

type vaultProvider struct {
	addon     *addon.Vault
	url       string
	mtlsUrl   string
	client    *vault.Client
	framework *framework.Framework
}

type StoreCustomizer = func(provider *vaultProvider, secret *v1.Secret, secretStore *metav1.ObjectMeta, secretStoreSpec *esv1.SecretStoreSpec, isClusterStore bool)

const (
	clientTlsCertName       = "vault-client-tls"
	certAuthProviderName    = "cert-auth-provider"
	appRoleAuthProviderName = "app-role-provider"
	kvv1ProviderName        = "kv-v1-provider"
	jwtProviderName         = "jwt-provider"
	jwtProviderSecretName   = "jwt-provider-credentials"
	jwtK8sProviderName      = "jwt-k8s-provider"
	kubernetesProviderName  = "kubernetes-provider"
	referentSecretName      = "referent-secret"
	referentKey             = "referent-secret-key"
)

var (
	secretStorePath  = "secret"
	mtlsSuffix       = "-mtls"
	invalidMtlSuffix = "-invalid-mtls"
)

func newVaultProvider(f *framework.Framework, addon *addon.Vault) *vaultProvider {
	prov := &vaultProvider{
		addon:     addon,
		framework: f,
	}

	BeforeEach(prov.BeforeEach)
	AfterEach(prov.AfterEach)
	return prov
}

// CreateSecret creates a secret in both kv v1 and v2 provider.
func (s *vaultProvider) CreateSecret(key string, val framework.SecretEntry) {
	req := s.client.NewRequest(http.MethodPost, fmt.Sprintf("/v1/secret/data/%s", key))
	req.BodyBytes = []byte(fmt.Sprintf(`{"data": %s}`, val.Value))
	_, err := s.client.RawRequestWithContext(GinkgoT().Context(), req) //nolint:staticcheck
	Expect(err).ToNot(HaveOccurred())

	req = s.client.NewRequest(http.MethodPost, fmt.Sprintf("/v1/secret_v1/%s", key))
	req.BodyBytes = []byte(val.Value)
	_, err = s.client.RawRequestWithContext(GinkgoT().Context(), req) //nolint:staticcheck
	Expect(err).ToNot(HaveOccurred())
}

func (s *vaultProvider) DeleteSecret(key string) {
	req := s.client.NewRequest(http.MethodDelete, fmt.Sprintf("/v1/secret/data/%s", key))
	_, err := s.client.RawRequestWithContext(GinkgoT().Context(), req) //nolint:staticcheck
	Expect(err).ToNot(HaveOccurred())

	req = s.client.NewRequest(http.MethodDelete, fmt.Sprintf("/v1/secret_v1/%s", key))
	_, err = s.client.RawRequestWithContext(GinkgoT().Context(), req) //nolint:staticcheck
	Expect(err).ToNot(HaveOccurred())
}

func WithMTLS(provider *vaultProvider, secret *v1.Secret, secretStore *metav1.ObjectMeta, secretStoreSpec *esv1.SecretStoreSpec, isClusterStore bool) {
	provider.CreateClientTlsCert()
	secret.Name = secret.Name + mtlsSuffix
	secretStore.Name = secretStore.Name + mtlsSuffix
	secretStoreSpec.Provider.Vault.Server = provider.mtlsUrl
	secretStoreSpec.Provider.Vault.ClientTLS = esv1.VaultClientTLS{
		CertSecretRef: &esmeta.SecretKeySelector{
			Name: clientTlsCertName,
		},
		KeySecretRef: &esmeta.SecretKeySelector{
			Name: clientTlsCertName,
		},
	}
	if isClusterStore {
		secretStoreSpec.Provider.Vault.ClientTLS.CertSecretRef.Namespace = &provider.framework.Namespace.Name
		secretStoreSpec.Provider.Vault.ClientTLS.KeySecretRef.Namespace = &provider.framework.Namespace.Name
	}
}

func WithInvalidMTLS(provider *vaultProvider, secret *v1.Secret, secretStore *metav1.ObjectMeta, secretStoreSpec *esv1.SecretStoreSpec, isClusterStore bool) {
	secret.Name = secret.Name + invalidMtlSuffix
	secretStore.Name = secretStore.Name + invalidMtlSuffix
	secretStoreSpec.Provider.Vault.Server = provider.mtlsUrl
}

func (s *vaultProvider) BeforeEach() {
	s.client = s.addon.VaultClient
	s.url = s.addon.VaultURL
	s.mtlsUrl = s.addon.VaultMtlsURL
}

func (s *vaultProvider) AfterEach() {
}

func makeStore(name, ns string, v *addon.Vault) *esv1.SecretStore {
	return &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				Vault: &esv1.VaultProvider{
					Version:  esv1.VaultKVStoreV2,
					Path:     &secretStorePath,
					Server:   v.VaultURL,
					CABundle: v.VaultServerCA,
				},
			},
		},
	}
}

func makeClusterStore(name, ns string, v *addon.Vault) *esv1.ClusterSecretStore {
	store := makeStore(name, ns, v)
	return &esv1.ClusterSecretStore{
		ObjectMeta: store.ObjectMeta,
		Spec:       store.Spec,
	}
}

func (s *vaultProvider) CreateClientTlsCert() {
	By("creating a secret containing the Vault TLS client certificate")
	clientCert := s.addon.ClientCert
	clientKey := s.addon.ClientKey
	vaultClientCert := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clientTlsCertName,
			Namespace: s.framework.Namespace.Name,
		},
		Data: map[string][]byte{
			"tls.crt": clientCert,
			"tls.key": clientKey,
		},
	}
	err := s.framework.CRClient.Create(GinkgoT().Context(), vaultClientCert)
	Expect(err).ToNot(HaveOccurred())
}

func (s *vaultProvider) CreateCertStore() {
	By("creating a vault secret")
	clientCert := s.addon.ClientCert
	clientKey := s.addon.ClientKey
	vaultCreds := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      certAuthProviderName,
			Namespace: s.framework.Namespace.Name,
		},
		Data: map[string][]byte{
			"token":       []byte(s.addon.RootToken),
			"client_cert": clientCert,
			"client_key":  clientKey,
		},
	}
	err := s.framework.CRClient.Create(GinkgoT().Context(), vaultCreds)
	Expect(err).ToNot(HaveOccurred())

	By("creating an secret store for vault")
	secretStore := makeStore(certAuthProviderName, s.framework.Namespace.Name, s.addon)
	secretStore.Spec.Provider.Vault.Auth = &esv1.VaultAuth{
		Cert: &esv1.VaultCertAuth{
			ClientCert: esmeta.SecretKeySelector{
				Name: certAuthProviderName,
				Key:  "client_cert",
			},
			SecretRef: esmeta.SecretKeySelector{
				Name: certAuthProviderName,
				Key:  "client_key",
			},
		},
	}
	err = s.framework.CRClient.Create(GinkgoT().Context(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}

func (s vaultProvider) CreateTokenStore(customizers ...StoreCustomizer) {
	vaultCreds := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "token-provider",
			Namespace: s.framework.Namespace.Name,
		},
		Data: map[string][]byte{
			"token": []byte(s.addon.RootToken),
		},
	}
	secretStore := makeStore(s.framework.Namespace.Name, s.framework.Namespace.Name, s.addon)
	secretStore.Spec.Provider.Vault.Auth = &esv1.VaultAuth{
		TokenSecretRef: &esmeta.SecretKeySelector{
			Name: vaultCreds.Name,
			Key:  "token",
		},
	}
	for _, customizer := range customizers {
		customizer(&s, vaultCreds, &secretStore.ObjectMeta, &secretStore.Spec, false)
	}

	secretStore.Spec.Provider.Vault.Auth.TokenSecretRef.Name = vaultCreds.Name
	err := s.framework.CRClient.Create(GinkgoT().Context(), vaultCreds)
	Expect(err).ToNot(HaveOccurred())
	err = s.framework.CRClient.Create(GinkgoT().Context(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}

// CreateReferentTokenStore creates a secret in the ExternalSecrets
// namespace and creates a ClusterSecretStore with an empty namespace
// that can be used to test the referent namespace feature.
func (s vaultProvider) CreateReferentTokenStore(customizers ...StoreCustomizer) {
	referentSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      referentSecretName,
			Namespace: s.framework.Namespace.Name,
		},
		Data: map[string][]byte{
			referentKey: []byte(s.addon.RootToken),
		},
	}
	secretStore := makeClusterStore(referentSecretStoreName(s.framework), s.framework.Namespace.Name, s.addon)
	secretStore.Spec.Provider.Vault.Auth = &esv1.VaultAuth{
		TokenSecretRef: &esmeta.SecretKeySelector{
			Name: referentSecret.Name,
			Key:  referentKey,
		},
	}
	for _, customizer := range customizers {
		customizer(&s, referentSecret, &secretStore.ObjectMeta, &secretStore.Spec, true)
	}

	DeferCleanup(func() {
		// cannot use ginkgo context nested in DeferCleanup
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		s.framework.CRClient.Delete(ctx, secretStore)
	})

	secretStore.Spec.Provider.Vault.Auth.TokenSecretRef.Name = referentSecret.Name
	_, err := s.framework.KubeClientSet.CoreV1().Secrets(s.framework.Namespace.Name).Create(GinkgoT().Context(), referentSecret, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	err = s.framework.CRClient.Create(GinkgoT().Context(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}

func (s vaultProvider) CreateAppRoleStore() {
	By("creating a vault secret")
	vaultCreds := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appRoleAuthProviderName,
			Namespace: s.framework.Namespace.Name,
		},
		Data: map[string][]byte{
			"approle_secret": []byte(s.addon.AppRoleSecret),
		},
	}
	err := s.framework.CRClient.Create(GinkgoT().Context(), vaultCreds)
	Expect(err).ToNot(HaveOccurred())

	By("creating an secret store for vault")
	secretStore := makeStore(appRoleAuthProviderName, s.framework.Namespace.Name, s.addon)
	secretStore.Spec.Provider.Vault.Auth = &esv1.VaultAuth{
		AppRole: &esv1.VaultAppRole{
			Path:   s.addon.AppRolePath,
			RoleID: s.addon.AppRoleID,
			SecretRef: esmeta.SecretKeySelector{
				Name: appRoleAuthProviderName,
				Key:  "approle_secret",
			},
		},
	}
	err = s.framework.CRClient.Create(GinkgoT().Context(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}

func (s vaultProvider) CreateV1Store() {
	vaultCreds := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "v1-provider",
			Namespace: s.framework.Namespace.Name,
		},
		Data: map[string][]byte{
			"token": []byte(s.addon.RootToken),
		},
	}
	err := s.framework.CRClient.Create(GinkgoT().Context(), vaultCreds)
	Expect(err).ToNot(HaveOccurred())
	secretStore := makeStore(kvv1ProviderName, s.framework.Namespace.Name, s.addon)
	secretV1StorePath := "secret_v1"
	secretStore.Spec.Provider.Vault.Version = esv1.VaultKVStoreV1
	secretStore.Spec.Provider.Vault.Path = &secretV1StorePath
	secretStore.Spec.Provider.Vault.Auth = &esv1.VaultAuth{
		TokenSecretRef: &esmeta.SecretKeySelector{
			Name: "v1-provider",
			Key:  "token",
		},
	}
	err = s.framework.CRClient.Create(GinkgoT().Context(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}

func (s vaultProvider) CreateJWTStore() {
	vaultCreds := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jwtProviderSecretName,
			Namespace: s.framework.Namespace.Name,
		},
		Data: map[string][]byte{
			"jwt": []byte(s.addon.JWTToken),
		},
	}
	err := s.framework.CRClient.Create(GinkgoT().Context(), vaultCreds)
	Expect(err).ToNot(HaveOccurred())
	secretStore := makeStore(jwtProviderName, s.framework.Namespace.Name, s.addon)
	secretStore.Spec.Provider.Vault.Auth = &esv1.VaultAuth{
		Jwt: &esv1.VaultJwtAuth{
			Path: s.addon.JWTPath,
			Role: s.addon.JWTRole,
			SecretRef: &esmeta.SecretKeySelector{
				Name: jwtProviderSecretName,
				Key:  "jwt",
			},
		},
	}
	err = s.framework.CRClient.Create(GinkgoT().Context(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}

func (s vaultProvider) CreateJWTK8sStore() {
	secretStore := makeStore(jwtK8sProviderName, s.framework.Namespace.Name, s.addon)
	secretStore.Spec.Provider.Vault.Auth = &esv1.VaultAuth{
		Jwt: &esv1.VaultJwtAuth{
			Path: s.addon.JWTK8sPath,
			Role: s.addon.JWTRole,
			KubernetesServiceAccountToken: &esv1.VaultKubernetesServiceAccountTokenAuth{
				ServiceAccountRef: esmeta.ServiceAccountSelector{
					Name: "default",
				},
				Audiences: &[]string{
					"vault.client",
				},
			},
		},
	}
	err := s.framework.CRClient.Create(GinkgoT().Context(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}

func (s vaultProvider) CreateKubernetesAuthStore() {
	secretStore := makeStore(kubernetesProviderName, s.framework.Namespace.Name, s.addon)
	secretStore.Spec.Provider.Vault.Auth = &esv1.VaultAuth{
		Kubernetes: &esv1.VaultKubernetesAuth{
			Path: s.addon.KubernetesAuthPath,
			Role: s.addon.KubernetesAuthRole,
			ServiceAccountRef: &esmeta.ServiceAccountSelector{
				Name: "default",
			},
		},
	}
	err := s.framework.CRClient.Create(GinkgoT().Context(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}

func referentSecretStoreName(f *framework.Framework) string {
	return "referent-provider-" + f.Namespace.Name
}
