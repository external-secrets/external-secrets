/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

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

	vault "github.com/hashicorp/vault/api"

	//nolint
	. "github.com/onsi/ginkgo/v2"

	//nolint
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/e2e/framework"
	"github.com/external-secrets/external-secrets/e2e/framework/addon"
)

type vaultProvider struct {
	url       string
	client    *vault.Client
	framework *framework.Framework
}

const (
	certAuthProviderName    = "cert-auth-provider"
	appRoleAuthProviderName = "app-role-provider"
	kvv1ProviderName        = "kv-v1-provider"
	jwtProviderName         = "jwt-provider"
	jwtK8sProviderName      = "jwt-k8s-provider"
	kubernetesProviderName  = "kubernetes-provider"
)

var (
	secretStorePath = "secret"
)

func newVaultProvider(f *framework.Framework) *vaultProvider {
	prov := &vaultProvider{
		framework: f,
	}
	BeforeEach(prov.BeforeEach)
	return prov
}

// CreateSecret creates a secret in both kv v1 and v2 provider.
func (s *vaultProvider) CreateSecret(key string, val framework.SecretEntry) {
	req := s.client.NewRequest(http.MethodPost, fmt.Sprintf("/v1/secret/data/%s", key))
	req.BodyBytes = []byte(fmt.Sprintf(`{"data": %s}`, val.Value))
	_, err := s.client.RawRequestWithContext(context.Background(), req) //nolint:staticcheck
	Expect(err).ToNot(HaveOccurred())

	req = s.client.NewRequest(http.MethodPost, fmt.Sprintf("/v1/secret_v1/%s", key))
	req.BodyBytes = []byte(val.Value)
	_, err = s.client.RawRequestWithContext(context.Background(), req) //nolint:staticcheck
	Expect(err).ToNot(HaveOccurred())
}

func (s *vaultProvider) DeleteSecret(key string) {
	req := s.client.NewRequest(http.MethodDelete, fmt.Sprintf("/v1/secret/data/%s", key))
	_, err := s.client.RawRequestWithContext(context.Background(), req) //nolint:staticcheck
	Expect(err).ToNot(HaveOccurred())

	req = s.client.NewRequest(http.MethodDelete, fmt.Sprintf("/v1/secret_v1/%s", key))
	_, err = s.client.RawRequestWithContext(context.Background(), req) //nolint:staticcheck
	Expect(err).ToNot(HaveOccurred())
}

func (s *vaultProvider) BeforeEach() {
	ns := s.framework.Namespace.Name
	v := addon.NewVault(ns)
	s.framework.Install(v)
	s.client = v.VaultClient
	s.url = v.VaultURL

	s.CreateCertStore(v, ns)
	s.CreateTokenStore(v, ns)
	s.CreateAppRoleStore(v, ns)
	s.CreateV1Store(v, ns)
	s.CreateJWTStore(v, ns)
	s.CreateJWTK8sStore(v, ns)
	s.CreateKubernetesAuthStore(v, ns)
}

func makeStore(name, ns string, v *addon.Vault) *esv1beta1.SecretStore {
	return &esv1beta1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Vault: &esv1beta1.VaultProvider{
					Version:  esv1beta1.VaultKVStoreV2,
					Path:     &secretStorePath,
					Server:   v.VaultURL,
					CABundle: v.VaultServerCA,
				},
			},
		},
	}
}

func (s *vaultProvider) CreateCertStore(v *addon.Vault, ns string) {
	By("creating a vault secret")
	clientCert := v.ClientCert
	clientKey := v.ClientKey
	vaultCreds := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      certAuthProviderName,
			Namespace: ns,
		},
		Data: map[string][]byte{
			"token":       []byte(v.RootToken),
			"client_cert": clientCert,
			"client_key":  clientKey,
		},
	}
	err := s.framework.CRClient.Create(context.Background(), vaultCreds)
	Expect(err).ToNot(HaveOccurred())

	By("creating an secret store for vault")
	secretStore := makeStore(certAuthProviderName, ns, v)
	secretStore.Spec.Provider.Vault.Auth = esv1beta1.VaultAuth{
		Cert: &esv1beta1.VaultCertAuth{
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
	err = s.framework.CRClient.Create(context.Background(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}

func (s vaultProvider) CreateTokenStore(v *addon.Vault, ns string) {
	vaultCreds := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "token-provider",
			Namespace: ns,
		},
		Data: map[string][]byte{
			"token": []byte(v.RootToken),
		},
	}
	err := s.framework.CRClient.Create(context.Background(), vaultCreds)
	Expect(err).ToNot(HaveOccurred())
	secretStore := makeStore(s.framework.Namespace.Name, ns, v)
	secretStore.Spec.Provider.Vault.Auth = esv1beta1.VaultAuth{
		TokenSecretRef: &esmeta.SecretKeySelector{
			Name: "token-provider",
			Key:  "token",
		},
	}
	err = s.framework.CRClient.Create(context.Background(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}

func (s vaultProvider) CreateAppRoleStore(v *addon.Vault, ns string) {
	By("creating a vault secret")
	vaultCreds := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appRoleAuthProviderName,
			Namespace: ns,
		},
		Data: map[string][]byte{
			"approle_secret": []byte(v.AppRoleSecret),
		},
	}
	err := s.framework.CRClient.Create(context.Background(), vaultCreds)
	Expect(err).ToNot(HaveOccurred())

	By("creating an secret store for vault")
	secretStore := makeStore(appRoleAuthProviderName, ns, v)
	secretStore.Spec.Provider.Vault.Auth = esv1beta1.VaultAuth{
		AppRole: &esv1beta1.VaultAppRole{
			Path:   v.AppRolePath,
			RoleID: v.AppRoleID,
			SecretRef: esmeta.SecretKeySelector{
				Name: appRoleAuthProviderName,
				Key:  "approle_secret",
			},
		},
	}
	err = s.framework.CRClient.Create(context.Background(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}

func (s vaultProvider) CreateV1Store(v *addon.Vault, ns string) {
	vaultCreds := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "v1-provider",
			Namespace: ns,
		},
		Data: map[string][]byte{
			"token": []byte(v.RootToken),
		},
	}
	err := s.framework.CRClient.Create(context.Background(), vaultCreds)
	Expect(err).ToNot(HaveOccurred())
	secretStore := makeStore(kvv1ProviderName, ns, v)
	secretV1StorePath := "secret_v1"
	secretStore.Spec.Provider.Vault.Version = esv1beta1.VaultKVStoreV1
	secretStore.Spec.Provider.Vault.Path = &secretV1StorePath
	secretStore.Spec.Provider.Vault.Auth = esv1beta1.VaultAuth{
		TokenSecretRef: &esmeta.SecretKeySelector{
			Name: "v1-provider",
			Key:  "token",
		},
	}
	err = s.framework.CRClient.Create(context.Background(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}

func (s vaultProvider) CreateJWTStore(v *addon.Vault, ns string) {
	vaultCreds := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jwt-provider",
			Namespace: ns,
		},
		Data: map[string][]byte{
			"jwt": []byte(v.JWTToken),
		},
	}
	err := s.framework.CRClient.Create(context.Background(), vaultCreds)
	Expect(err).ToNot(HaveOccurred())
	secretStore := makeStore(jwtProviderName, ns, v)
	secretStore.Spec.Provider.Vault.Auth = esv1beta1.VaultAuth{
		Jwt: &esv1beta1.VaultJwtAuth{
			Path: v.JWTPath,
			Role: v.JWTRole,
			SecretRef: &esmeta.SecretKeySelector{
				Name: "jwt-provider",
				Key:  "jwt",
			},
		},
	}
	err = s.framework.CRClient.Create(context.Background(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}

func (s vaultProvider) CreateJWTK8sStore(v *addon.Vault, ns string) {
	secretStore := makeStore(jwtK8sProviderName, ns, v)
	secretStore.Spec.Provider.Vault.Auth = esv1beta1.VaultAuth{
		Jwt: &esv1beta1.VaultJwtAuth{
			Path: v.JWTK8sPath,
			Role: v.JWTRole,
			KubernetesServiceAccountToken: &esv1beta1.VaultKubernetesServiceAccountTokenAuth{
				ServiceAccountRef: esmeta.ServiceAccountSelector{
					Name: "default",
				},
				Audiences: &[]string{
					"vault.client",
				},
			},
		},
	}
	err := s.framework.CRClient.Create(context.Background(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}

func (s vaultProvider) CreateKubernetesAuthStore(v *addon.Vault, ns string) {
	secretStore := makeStore(kubernetesProviderName, ns, v)
	secretStore.Spec.Provider.Vault.Auth = esv1beta1.VaultAuth{
		Kubernetes: &esv1beta1.VaultKubernetesAuth{
			Path: v.KubernetesAuthPath,
			Role: v.KubernetesAuthRole,
			ServiceAccountRef: &esmeta.ServiceAccountSelector{
				Name: "default",
			},
		},
	}
	err := s.framework.CRClient.Create(context.Background(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}
