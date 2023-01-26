/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
limitations under the License.
*/
package azure

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/keyvault/keyvault"
	kvauth "github.com/Azure/go-autorest/autorest/azure/auth"

	// nolint
	. "github.com/onsi/ginkgo/v2"
	// nolint
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilpointer "k8s.io/utils/pointer"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

type azureProvider struct {
	clientID     string
	clientSecret string
	tenantID     string
	vaultURL     string
	client       *keyvault.BaseClient
	framework    *framework.Framework
}

func newazureProvider(f *framework.Framework, clientID, clientSecret, tenantID, vaultURL string) *azureProvider {
	clientCredentialsConfig := kvauth.NewClientCredentialsConfig(clientID, clientSecret, tenantID)
	clientCredentialsConfig.Resource = "https://vault.azure.net"
	basicClient := keyvault.New()
	prov := &azureProvider{
		framework:    f,
		client:       &basicClient,
		clientID:     clientID,
		clientSecret: clientSecret,
		tenantID:     tenantID,
		vaultURL:     vaultURL,
	}

	o := &sync.Once{}
	BeforeEach(func() {
		// run authorizor only if this spec is called
		o.Do(func() {
			authorizer, err := clientCredentialsConfig.Authorizer()
			if err != nil {
				Fail(err.Error())
			}
			prov.client.Authorizer = authorizer
		})
		prov.CreateSecretStoreWithWI()
		prov.CreateSecretStore()
		prov.CreateReferentSecretStore()
	})

	return prov
}

func newFromEnv(f *framework.Framework) *azureProvider {
	vaultURL := os.Getenv("VAULT_URL")
	tenantID := os.Getenv("TENANT_ID")
	clientID := os.Getenv("AZURE_CLIENT_ID")
	clientSecret := os.Getenv("AZURE_CLIENT_SECRET")
	return newazureProvider(f, clientID, clientSecret, tenantID, vaultURL)
}

func (s *azureProvider) CreateSecret(key string, val framework.SecretEntry) {
	_, err := s.client.SetSecret(
		context.Background(),
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
		context.Background(),
		s.vaultURL,
		key)
	Expect(err).ToNot(HaveOccurred())
}

func (s *azureProvider) CreateKey(key string) *keyvault.JSONWebKey {
	out, err := s.client.CreateKey(
		context.Background(),
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
	_, err := s.client.DeleteKey(context.Background(), s.vaultURL, key)
	Expect(err).ToNot(HaveOccurred())
}

func (s *azureProvider) CreateCertificate(key string) {
	_, err := s.client.CreateCertificate(
		context.Background(),
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
			context.Background(),
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
	_, err := s.client.DeleteCertificate(context.Background(), s.vaultURL, key)
	Expect(err).ToNot(HaveOccurred())
}

const (
	staticSecretName                  = "provider-secret"
	referentSecretName                = "referent-secret"
	workloadIdentityServiceAccountNme = "external-secrets-operator"
	credentialKeyClientID             = "client-id"
	credentialKeyClientSecret         = "client-secret"
)

func newProviderWithStaticCredentials(tenantID, vaultURL, secretName string) *esv1beta1.AzureKVProvider {
	return &esv1beta1.AzureKVProvider{
		TenantID: &tenantID,
		VaultURL: &vaultURL,
		AuthSecretRef: &esv1beta1.AzureKVAuth{
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

func newProviderWithServiceAccount(tenantID, vaultURL string, authType esv1beta1.AzureAuthType, serviceAccountName string, serviceAccountNamespace *string) *esv1beta1.AzureKVProvider {
	return &esv1beta1.AzureKVProvider{
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
	err := s.framework.CRClient.Create(context.Background(), azureCreds)
	Expect(err).ToNot(HaveOccurred())
	secretStore := &esv1beta1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.framework.Namespace.Name,
			Namespace: s.framework.Namespace.Name,
		},
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				AzureKV: newProviderWithStaticCredentials(s.tenantID, s.vaultURL, staticSecretName),
			},
		},
	}
	err = s.framework.CRClient.Create(context.Background(), secretStore)
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
	err := s.framework.CRClient.Create(context.Background(), azureCreds)
	Expect(err).ToNot(HaveOccurred())
	secretStore := &esv1beta1.ClusterSecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      referentAuthName(s.framework),
			Namespace: s.framework.Namespace.Name,
		},
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				AzureKV: newProviderWithStaticCredentials(s.tenantID, s.vaultURL, referentSecretName),
			},
		},
	}
	err = s.framework.CRClient.Create(context.Background(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}

func referentAuthName(f *framework.Framework) string {
	return "referent-auth-" + f.Namespace.Name
}

func (s *azureProvider) CreateSecretStoreWithWI() {
	authType := esv1beta1.AzureWorkloadIdentity
	namespace := "external-secrets-operator"
	ClusterSecretStore := &esv1beta1.ClusterSecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name: s.framework.Namespace.Name,
		},
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				AzureKV: newProviderWithServiceAccount(s.tenantID, s.vaultURL, authType, workloadIdentityServiceAccountNme, &namespace),
			},
		},
	}
	err := s.framework.CRClient.Create(context.Background(), ClusterSecretStore)
	Expect(err).ToNot(HaveOccurred())
}

func (s *azureProvider) CreateReferentSecretStoreWithWI() {
	authType := esv1beta1.AzureWorkloadIdentity
	ClusterSecretStore := &esv1beta1.ClusterSecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name: referentAuthName(s.framework),
		},
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				AzureKV: newProviderWithServiceAccount(s.tenantID, s.vaultURL, authType, workloadIdentityServiceAccountNme, nil),
			},
		},
	}
	err := s.framework.CRClient.Create(context.Background(), ClusterSecretStore)
	Expect(err).ToNot(HaveOccurred())
}
