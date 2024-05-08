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
package conjur

import (
	"context"
	"encoding/base64"
	"strings"

	//nolint
	. "github.com/onsi/ginkgo/v2"

	//nolint
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/cyberark/conjur-api-go/conjurapi"
	"github.com/external-secrets/external-secrets-e2e/framework"
	"github.com/external-secrets/external-secrets-e2e/framework/addon"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

type conjurProvider struct {
	url       string
	client    *conjurapi.Client
	framework *framework.Framework
}

const (
	jwtK8sProviderName       = "jwt-k8s-provider"
	jwtK8sHostIDProviderName = "jwt-k8s-hostid-provider"
)

func newConjurProvider(f *framework.Framework) *conjurProvider {
	prov := &conjurProvider{
		framework: f,
	}
	BeforeEach(prov.BeforeEach)
	return prov
}

func (s *conjurProvider) CreateSecret(key string, val framework.SecretEntry) {
	// Generate a policy file for the secret key
	policy := createVariablePolicy(key, s.framework.Namespace.Name, val.Tags)

	_, err := s.client.LoadPolicy(conjurapi.PolicyModePost, "root", strings.NewReader(policy))
	Expect(err).ToNot(HaveOccurred())

	// Add the secret value
	err = s.client.AddSecret(key, val.Value)
	Expect(err).ToNot(HaveOccurred())
}

func (s *conjurProvider) DeleteSecret(key string) {
	policy := deleteVariablePolicy(key)
	_, err := s.client.LoadPolicy(conjurapi.PolicyModePatch, "root", strings.NewReader(policy))

	Expect(err).ToNot(HaveOccurred())
}

func (s *conjurProvider) BeforeEach() {
	ns := s.framework.Namespace.Name
	c := addon.NewConjur(ns)
	s.framework.Install(c)
	s.client = c.ConjurClient
	s.url = c.ConjurURL

	s.CreateApiKeyStore(c, ns)
	s.CreateJWTK8sStore(c, ns)
	s.CreateJWTK8sHostIDStore(c, ns)
}

func makeStore(name, ns string, c *addon.Conjur) *esv1beta1.SecretStore {
	return &esv1beta1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Conjur: &esv1beta1.ConjurProvider{
					URL:      c.ConjurURL,
					CABundle: base64.StdEncoding.EncodeToString(c.ConjurServerCA),
				},
			},
		},
	}
}

func (s *conjurProvider) CreateApiKeyStore(c *addon.Conjur, ns string) {
	By("creating a conjur secret")
	conjurCreds := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ns,
			Namespace: ns,
		},
		Data: map[string][]byte{
			"apikey":   []byte(c.AdminApiKey),
			"username": []byte("admin"),
		},
	}
	err := s.framework.CRClient.Create(context.Background(), conjurCreds)
	Expect(err).ToNot(HaveOccurred())

	By("creating an secret store for conjur")
	secretStore := makeStore(ns, ns, c)
	secretStore.Spec.Provider.Conjur.Auth = esv1beta1.ConjurAuth{
		APIKey: &esv1beta1.ConjurAPIKey{
			Account: "default",
			UserRef: &esmeta.SecretKeySelector{
				Name: ns,
				Key:  "username",
			},
			APIKeyRef: &esmeta.SecretKeySelector{
				Name: ns,
				Key:  "apikey",
			},
		},
	}
	err = s.framework.CRClient.Create(context.Background(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}

func (s conjurProvider) CreateJWTK8sStore(c *addon.Conjur, ns string) {
	// Create a service account
	sa := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app-sa",
			Namespace: ns,
		},
	}
	err := s.framework.CRClient.Create(context.Background(), sa)
	Expect(err).ToNot(HaveOccurred())

	// Add the service account to the Conjur policy with permissions to
	// authenticate with authn-jwt
	saName := "system:serviceaccount:" + ns + ":test-app-sa"
	policy := createJwtHostPolicy(saName, "eso-tests")

	_, err = s.client.LoadPolicy(conjurapi.PolicyModePost, "root", strings.NewReader(policy))
	Expect(err).ToNot(HaveOccurred())

	// Now create a secret store that uses the service account to authenticate
	secretStore := makeStore(jwtK8sProviderName, ns, c)
	secretStore.Spec.Provider.Conjur.Auth = esv1beta1.ConjurAuth{
		Jwt: &esv1beta1.ConjurJWT{
			Account:   "default",
			ServiceID: "eso-tests",
			ServiceAccountRef: &esmeta.ServiceAccountSelector{
				Name: "test-app-sa",
				Audiences: []string{
					c.ConjurURL,
				},
			},
		},
	}
	err = s.framework.CRClient.Create(context.Background(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}

func (s conjurProvider) CreateJWTK8sHostIDStore(c *addon.Conjur, ns string) {
	// Create a service account
	sa := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app-hostid-sa",
			Namespace: ns,
		},
	}
	err := s.framework.CRClient.Create(context.Background(), sa)
	Expect(err).ToNot(HaveOccurred())

	// Add the service account to the Conjur policy with permissions to
	// authenticate with authn-jwt
	saName := "system:serviceaccount:" + ns + ":test-app-hostid-sa"
	policy := createJwtHostPolicy(saName, "eso-tests-hostid")

	_, err = s.client.LoadPolicy(conjurapi.PolicyModePost, "root", strings.NewReader(policy))
	Expect(err).ToNot(HaveOccurred())

	// Now create a secret store that uses the service account to authenticate
	secretStore := makeStore(jwtK8sHostIDProviderName, ns, c)
	secretStore.Spec.Provider.Conjur.Auth = esv1beta1.ConjurAuth{
		Jwt: &esv1beta1.ConjurJWT{
			Account:   "default",
			HostID:    "host/" + saName,
			ServiceID: "eso-tests-hostid",
			ServiceAccountRef: &esmeta.ServiceAccountSelector{
				Name: "test-app-hostid-sa",
				Audiences: []string{
					c.ConjurURL,
				},
			},
		},
	}
	err = s.framework.CRClient.Create(context.Background(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}
