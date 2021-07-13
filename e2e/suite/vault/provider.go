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
	. "github.com/onsi/ginkgo"

	//nolint
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/e2e/framework"
)

type vaultProvider struct {
	url       string
	token     string
	client    *vault.Client
	framework *framework.Framework
}

func newVaultProvider(f *framework.Framework, url, token string) *vaultProvider {
	vc, err := vault.NewClient(&vault.Config{
		Address: url,
	})
	Expect(err).ToNot(HaveOccurred())
	vc.SetToken(token)

	prov := &vaultProvider{
		framework: f,
		url:       url,
		token:     token,
		client:    vc,
	}
	BeforeEach(prov.BeforeEach)
	return prov
}

func (s *vaultProvider) CreateSecret(key, val string) {
	req := s.client.NewRequest(http.MethodPost, fmt.Sprintf("/v1/secret/data/%s", key))
	req.BodyBytes = []byte(fmt.Sprintf(`{"data": %s}`, val))
	_, err := s.client.RawRequestWithContext(context.Background(), req)
	Expect(err).ToNot(HaveOccurred())
}

func (s *vaultProvider) DeleteSecret(key string) {
	req := s.client.NewRequest(http.MethodDelete, fmt.Sprintf("/v1/secret/data/%s", key))
	_, err := s.client.RawRequestWithContext(context.Background(), req)
	Expect(err).ToNot(HaveOccurred())
}

func (s *vaultProvider) BeforeEach() {
	By("creating a vault secret")
	vaultCreds := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "provider-secret",
			Namespace: s.framework.Namespace.Name,
		},
		StringData: map[string]string{
			"token": s.token, // vault dev-mode default token
		},
	}
	err := s.framework.CRClient.Create(context.Background(), vaultCreds)
	Expect(err).ToNot(HaveOccurred())

	By("creating an secret store for vault")
	secretStore := &esv1alpha1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.framework.Namespace.Name,
			Namespace: s.framework.Namespace.Name,
		},
		Spec: esv1alpha1.SecretStoreSpec{
			Provider: &esv1alpha1.SecretStoreProvider{
				Vault: &esv1alpha1.VaultProvider{
					Version: esv1alpha1.VaultKVStoreV2,
					Path:    "secret",
					Server:  s.url,
					Auth: esv1alpha1.VaultAuth{
						TokenSecretRef: &esmeta.SecretKeySelector{
							Name: "provider-secret",
							Key:  "token",
						},
					},
				},
			},
		},
	}
	err = s.framework.CRClient.Create(context.Background(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}
