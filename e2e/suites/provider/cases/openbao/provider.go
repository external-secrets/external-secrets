/*
Copyright © The ESO Authors

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

package openbao

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/framework/addon"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type openBaoProvider struct {
	addon     *addon.OpenBao
	framework *framework.Framework
}

const (
	secretStorePath = "secret"
)

func newOpenBaoProvider(f *framework.Framework, addon *addon.OpenBao) *openBaoProvider {
	return &openBaoProvider{
		addon:     addon,
		framework: f,
	}
}

func (s *openBaoProvider) CreateSecret(key string, val framework.SecretEntry) {
	s.updateSecret(http.MethodPost, fmt.Sprintf("v1/secret/data/%s", key), strings.NewReader(fmt.Sprintf(`{"data": %s}`, val.Value)), http.StatusOK)
}

func (s *openBaoProvider) DeleteSecret(key string) {
	s.updateSecret(http.MethodDelete, fmt.Sprintf("v1/secret/metadata/%s", key), nil, http.StatusNoContent)
}

func (s *openBaoProvider) updateSecret(method string, path string, body io.Reader, expectedStatus int) {
	req, err := http.NewRequest(method, fmt.Sprintf("%s/%s", s.addon.URLs.Local, path), body)
	Expect(err).ToNot(HaveOccurred())
	req.Header.Add("X-Vault-Token", s.addon.RootToken)

	res, err := http.DefaultClient.Do(req)
	Expect(err).ToNot(HaveOccurred())
	defer res.Body.Close()

	ExpectWithOffset(1, res.StatusCode).To(Equal(expectedStatus), func() string {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return fmt.Sprintf("http request failed: could not read response body: %v", err)
		}
		return fmt.Sprintf("http request failed: %s", string(body))
	})
}

func makeStore(name, ns string, v *addon.OpenBao) *esv1.SecretStore {
	return &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				OpenBao: &esv1.OpenBaoProvider{
					Version:  esv1.OpenBaoKVStoreV2,
					Path:     new(secretStorePath),
					Server:   v.URLs.InClusterTLS,
					CABundle: v.ServerCA,
				},
			},
		},
	}
}

func (s openBaoProvider) CreateTokenStore() {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "token-provider",
			Namespace: s.framework.Namespace.Name,
		},
		Data: map[string][]byte{
			"token": []byte(s.addon.RootToken),
		},
	}
	secretStore := makeStore(s.framework.Namespace.Name, s.framework.Namespace.Name, s.addon)
	secretStore.Spec.Provider.OpenBao.Auth = &esv1.OpenBaoAuth{
		TokenSecretRef: &esmeta.SecretKeySelector{
			Name: secret.Name,
			Key:  "token",
		},
	}

	err := s.framework.CRClient.Create(GinkgoT().Context(), secret)
	Expect(err).ToNot(HaveOccurred())
	err = s.framework.CRClient.Create(GinkgoT().Context(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}
