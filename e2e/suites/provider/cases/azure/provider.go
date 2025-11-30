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
package azure

import (
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/keyvault/keyvault"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	kvauth "github.com/Azure/go-autorest/autorest/azure/auth"

	// nolint
	. "github.com/onsi/ginkgo/v2"
	// nolint
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilpointer "k8s.io/utils/pointer"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	esoazkv "github.com/external-secrets/external-secrets/providers/v1/azure/keyvault"
)

type azureProvider struct {
	clientID     string
	clientSecret string
	tenantID     string
	vaultURL     string
	client       *keyvault.BaseClient
	framework    *framework.Framework
}

// newFromEnv creates a new Azure KeyVault e2e test provider
// which uses client credentials flow to authenticate with azure.
func newFromEnv(f *framework.Framework) *azureProvider {
	vaultURL := os.Getenv("TFC_VAULT_URL")
	tenantID := os.Getenv("TFC_AZURE_TENANT_ID")
	clientID := os.Getenv("TFC_AZURE_CLIENT_ID")
	clientSecret := os.Getenv("TFC_AZURE_CLIENT_SECRET")

	basicClient := keyvault.New()
	prov := &azureProvider{
		framework:    f,
		clientID:     clientID,
		tenantID:     tenantID,
		vaultURL:     vaultURL,
		client:       &basicClient,
		clientSecret: clientSecret,
	}

	o := &sync.Once{}
	BeforeEach(func() {
		// run authorizor only if this spec is called
		// this allows us to run OTHER providers using GINKGO_LABELS without bailing out
		o.Do(func() {
			defer GinkgoRecover()
			clientCredentialsConfig := kvauth.NewClientCredentialsConfig(clientID, clientSecret, tenantID)
			clientCredentialsConfig.Resource = "https://vault.azure.net"
			authorizer, err := clientCredentialsConfig.Authorizer()
			if err != nil {
				Fail(err.Error())
			}
			prov.client.Authorizer = authorizer
		})
		prov.CreateSecretStore()
		prov.CreateReferentSecretStore()
		prov.CreateSecretStoreNewSDK()
		prov.CreateReferentSecretStoreNewSDK()
	})

	return prov
}

// create a new provider from workload identity
// the azwi webhook injects `AZURE_*` env vars into the container.
// we use these credentials to authenticate with azure using the federated token flow.
// please see here for details: https://azure.github.io/azure-workload-identity/docs/quick-start.html
func newFromWorkloadIdentity(f *framework.Framework) *azureProvider {
	// from azwi webhook
	tenantID := os.Getenv("AZURE_TENANT_ID")
	clientID := os.Getenv("AZURE_CLIENT_ID")
	tokenFilePath := os.Getenv("AZURE_FEDERATED_TOKEN_FILE")

	// from run.sh
	vaultURL := "https://eso-testing.vault.azure.net/"

	basicClient := keyvault.New()
	prov := &azureProvider{
		framework: f,
		client:    &basicClient,
		clientID:  clientID,
		tenantID:  tenantID,
		vaultURL:  vaultURL,
	}

	o := &sync.Once{}
	BeforeEach(func() {
		prov.CreateSecretStoreWithWI()
		// run authorizor only if this spec is called
		o.Do(func() {
			defer GinkgoRecover()
			token, err := os.ReadFile(tokenFilePath)
			if err != nil {
				Fail(err.Error())
			}

			// exchange the federated token for an access token
			aadEndpoint := esoazkv.AadEndpointForType(esv1.AzureEnvironmentPublicCloud)
			kvResource := strings.TrimSuffix(azure.PublicCloud.KeyVaultEndpoint, "/")
			tokenProvider, err := esoazkv.NewTokenProvider(GinkgoT().Context(), string(token), clientID, tenantID, aadEndpoint, kvResource)
			if err != nil {
				Fail(err.Error())
			}
			basicClient.Authorizer = autorest.NewBearerAuthorizer(tokenProvider)
		})
	})

	return prov
}

func (s *azureProvider) CreateSecret(key string, val framework.SecretEntry) {
	_, err := s.client.SetSecret(
		GinkgoT().Context(),
		s.vaultURL,
		key,
		keyvault.SecretSetParameters{
			Value: &val.Value,
			SecretAttributes: &keyvault.SecretAttributes{
				RecoveryLevel: keyvault.Purgeable,
				Enabled:       utilpointer.Bool(true),
			},
		})
	Expect(err).ToNot(HaveOccurred())
}

func (s *azureProvider) DeleteSecret(key string) {
	_, err := s.client.DeleteSecret(
		GinkgoT().Context(),
		s.vaultURL,
		key)
	Expect(err).ToNot(HaveOccurred())
}

func (s *azureProvider) CreateKey(key string) *keyvault.JSONWebKey {
	out, err := s.client.CreateKey(
		GinkgoT().Context(),
		s.vaultURL,
		key,
		keyvault.KeyCreateParameters{
			Kty: keyvault.RSA,
			KeyAttributes: &keyvault.KeyAttributes{
				RecoveryLevel: keyvault.Purgeable,
				Enabled:       utilpointer.Bool(true),
			},
		},
	)
	Expect(err).ToNot(HaveOccurred())
	return out.Key
}

func (s *azureProvider) DeleteKey(key string) {
	_, err := s.client.DeleteKey(GinkgoT().Context(), s.vaultURL, key)
	Expect(err).ToNot(HaveOccurred())
}

func (s *azureProvider) CreateCertificate(key string) {
	_, err := s.client.CreateCertificate(
		GinkgoT().Context(),
		s.vaultURL,
		key,
		keyvault.CertificateCreateParameters{
			CertificatePolicy: &keyvault.CertificatePolicy{
				X509CertificateProperties: &keyvault.X509CertificateProperties{
					Subject:          utilpointer.String("CN=e2e.test"),
					ValidityInMonths: utilpointer.Int32(42),
				},
				IssuerParameters: &keyvault.IssuerParameters{
					Name: utilpointer.String("Self"),
				},
				Attributes: &keyvault.CertificateAttributes{
					RecoveryLevel: keyvault.Purgeable,
					Enabled:       utilpointer.Bool(true),
				},
			},
			CertificateAttributes: &keyvault.CertificateAttributes{
				RecoveryLevel: keyvault.Purgeable,
				Enabled:       utilpointer.Bool(true),
			},
		},
	)
	Expect(err).ToNot(HaveOccurred())
}

func (s *azureProvider) GetCertificate(key string) []byte {
	attempts := 60
	for {
		out, err := s.client.GetCertificate(
			GinkgoT().Context(),
			s.vaultURL,
			key,
			"",
		)
		Expect(err).ToNot(HaveOccurred())
		if out.Cer != nil {
			return *out.Cer
		}

		attempts--
		if attempts <= 0 {
			Fail("failed fetching azkv certificate")
		}
		<-time.After(time.Second * 5)
	}
}

func (s *azureProvider) DeleteCertificate(key string) {
	_, err := s.client.DeleteCertificate(GinkgoT().Context(), s.vaultURL, key)
	Expect(err).ToNot(HaveOccurred())
}

const (
	staticSecretName                  = "provider-secret"
	referentSecretName                = "referent-secret"
	workloadIdentityServiceAccountNme = "external-secrets-operator"
	credentialKeyClientID             = "client-id"
	credentialKeyClientSecret         = "client-secret"
)

func newProviderWithStaticCredentials(tenantID, vaultURL, secretName string) *esv1.AzureKVProvider {
	return &esv1.AzureKVProvider{
		TenantID: &tenantID,
		VaultURL: &vaultURL,
		AuthSecretRef: &esv1.AzureKVAuth{
			ClientID: &esmeta.SecretKeySelector{
				Name: staticSecretName,
				Key:  credentialKeyClientID,
			},
			ClientSecret: &esmeta.SecretKeySelector{
				Name: staticSecretName,
				Key:  credentialKeyClientSecret,
			},
		},
	}
}

func newProviderWithStaticCredentialsNewSDK(tenantID, vaultURL, secretName string) *esv1.AzureKVProvider {
	useNewSDK := true
	return &esv1.AzureKVProvider{
		TenantID:    &tenantID,
		VaultURL:    &vaultURL,
		UseAzureSDK: &useNewSDK,
		AuthSecretRef: &esv1.AzureKVAuth{
			ClientID: &esmeta.SecretKeySelector{
				Name: staticSecretName,
				Key:  credentialKeyClientID,
			},
			ClientSecret: &esmeta.SecretKeySelector{
				Name: staticSecretName,
				Key:  credentialKeyClientSecret,
			},
		},
	}
}

func newProviderWithServiceAccount(tenantID, vaultURL string, authType esv1.AzureAuthType, serviceAccountName string, serviceAccountNamespace *string) *esv1.AzureKVProvider {
	return &esv1.AzureKVProvider{
		TenantID: &tenantID,
		VaultURL: &vaultURL,
		AuthType: &authType,
		ServiceAccountRef: &esmeta.ServiceAccountSelector{
			Name:      serviceAccountName,
			Namespace: serviceAccountNamespace,
		},
	}
}

func (s *azureProvider) CreateSecretStore() {
	azureCreds := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      staticSecretName,
			Namespace: s.framework.Namespace.Name,
		},
		StringData: map[string]string{
			credentialKeyClientID:     s.clientID,
			credentialKeyClientSecret: s.clientSecret,
		},
	}
	err := s.framework.CRClient.Create(GinkgoT().Context(), azureCreds)
	Expect(err).ToNot(HaveOccurred())
	secretStore := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.framework.Namespace.Name,
			Namespace: s.framework.Namespace.Name,
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				AzureKV: newProviderWithStaticCredentials(s.tenantID, s.vaultURL, staticSecretName),
			},
		},
	}
	err = s.framework.CRClient.Create(GinkgoT().Context(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}

func (s *azureProvider) CreateSecretStoreNewSDK() {
	azureCreds := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      staticSecretName,
			Namespace: s.framework.Namespace.Name,
		},
		StringData: map[string]string{
			credentialKeyClientID:     s.clientID,
			credentialKeyClientSecret: s.clientSecret,
		},
	}
	err := s.framework.CRClient.Create(GinkgoT().Context(), azureCreds)
	// Ignore AlreadyExists error since CreateSecretStore() might have already created this secret
	if err != nil && !apierrors.IsAlreadyExists(err) {
		Expect(err).ToNot(HaveOccurred())
	}
	secretStore := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.framework.Namespace.Name + "-new-sdk",
			Namespace: s.framework.Namespace.Name,
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				AzureKV: newProviderWithStaticCredentialsNewSDK(s.tenantID, s.vaultURL, staticSecretName),
			},
		},
	}
	err = s.framework.CRClient.Create(GinkgoT().Context(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}

func (s *azureProvider) CreateReferentSecretStore() {
	azureCreds := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      referentSecretName,
			Namespace: s.framework.Namespace.Name,
		},
		StringData: map[string]string{
			credentialKeyClientID:     s.clientID,
			credentialKeyClientSecret: s.clientSecret,
		},
	}
	err := s.framework.CRClient.Create(GinkgoT().Context(), azureCreds)
	Expect(err).ToNot(HaveOccurred())
	secretStore := &esv1.ClusterSecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      referentAuthName(s.framework),
			Namespace: s.framework.Namespace.Name,
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				AzureKV: newProviderWithStaticCredentials(s.tenantID, s.vaultURL, referentSecretName),
			},
		},
	}
	err = s.framework.CRClient.Create(GinkgoT().Context(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}

func (s *azureProvider) CreateReferentSecretStoreNewSDK() {
	azureCreds := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      referentSecretName + "-new-sdk",
			Namespace: s.framework.Namespace.Name,
		},
		StringData: map[string]string{
			credentialKeyClientID:     s.clientID,
			credentialKeyClientSecret: s.clientSecret,
		},
	}
	err := s.framework.CRClient.Create(GinkgoT().Context(), azureCreds)
	Expect(err).ToNot(HaveOccurred())
	secretStore := &esv1.ClusterSecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      referentAuthName(s.framework) + "-new-sdk",
			Namespace: s.framework.Namespace.Name,
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				AzureKV: newProviderWithStaticCredentialsNewSDK(s.tenantID, s.vaultURL, referentSecretName+"-new-sdk"),
			},
		},
	}
	err = s.framework.CRClient.Create(GinkgoT().Context(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}

func referentAuthName(f *framework.Framework) string {
	return "referent-auth-" + f.Namespace.Name
}

func (s *azureProvider) CreateSecretStoreWithWI() {
	authType := esv1.AzureWorkloadIdentity
	namespace := "external-secrets-operator"
	ClusterSecretStore := &esv1.ClusterSecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name: s.framework.Namespace.Name,
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				AzureKV: newProviderWithServiceAccount(s.tenantID, s.vaultURL, authType, workloadIdentityServiceAccountNme, &namespace),
			},
		},
	}
	err := s.framework.CRClient.Create(GinkgoT().Context(), ClusterSecretStore)
	Expect(err).ToNot(HaveOccurred())
}

func (s *azureProvider) CreateReferentSecretStoreWithWI() {
	authType := esv1.AzureWorkloadIdentity
	ClusterSecretStore := &esv1.ClusterSecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name: referentAuthName(s.framework),
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				AzureKV: newProviderWithServiceAccount(s.tenantID, s.vaultURL, authType, workloadIdentityServiceAccountNme, nil),
			},
		},
	}
	err := s.framework.CRClient.Create(GinkgoT().Context(), ClusterSecretStore)
	Expect(err).ToNot(HaveOccurred())
}
