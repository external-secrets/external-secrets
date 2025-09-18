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
package conjur

import (
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
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

type conjurProvider struct {
	addon     *addon.Conjur
	framework *framework.Framework
}

const (
	defaultStoreName         = "conjur"
	secretName               = "conjur-creds"
	jwtK8sProviderName       = "jwt-k8s-provider"
	jwtK8sHostIDProviderName = "jwt-k8s-hostid-provider"
	hostidServiceAccountName = "test-app-hostid-sa"
	appServiceAccountName    = "test-app-sa"
)

func newConjurProvider(f *framework.Framework, conjur *addon.Conjur) *conjurProvider {
	prov := &conjurProvider{
		framework: f,
		addon:     conjur,
	}

	BeforeEach(prov.BeforeEach)

	return prov
}

func (s *conjurProvider) CreateSecret(key string, val framework.SecretEntry) {
	// Generate a policy file for the secret key
	policy := createVariablePolicy(key, s.framework.Namespace.Name, val.Tags)

	_, err := s.addon.ConjurClient.LoadPolicy(conjurapi.PolicyModePost, "root", strings.NewReader(policy))
	Expect(err).ToNot(HaveOccurred())

	// Add the secret value
	err = s.addon.ConjurClient.AddSecret(key, val.Value)
	Expect(err).ToNot(HaveOccurred())
}

func (s *conjurProvider) DeleteSecret(key string) {
	policy := deleteVariablePolicy(key)
	_, err := s.addon.ConjurClient.LoadPolicy(conjurapi.PolicyModePatch, "root", strings.NewReader(policy))

	Expect(err).ToNot(HaveOccurred())
}

func (s *conjurProvider) BeforeEach() {
	// setup policy
	saName := "system:serviceaccount:" + s.framework.Namespace.Name + ":" + appServiceAccountName
	policy := createJwtHostPolicy(saName, "eso-tests")
	_, err := s.addon.ConjurClient.LoadPolicy(conjurapi.PolicyModePost, "root", strings.NewReader(policy))
	Expect(err).ToNot(HaveOccurred())

	// setup policy
	saName = "system:serviceaccount:" + s.framework.Namespace.Name + ":" + hostidServiceAccountName
	policy = createJwtHostPolicy(saName, "eso-tests-hostid")

	_, err = s.addon.ConjurClient.LoadPolicy(conjurapi.PolicyModePost, "root", strings.NewReader(policy))
	Expect(err).ToNot(HaveOccurred())
}

func makeStore(name, ns string, c *addon.Conjur) *esv1.SecretStore {
	return &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				Conjur: &esv1.ConjurProvider{
					URL:      c.ConjurURL,
					CABundle: base64.StdEncoding.EncodeToString(c.ConjurServerCA),
				},
			},
		},
	}
}

func (s *conjurProvider) CreateApiKeyStore() {
	By("creating a conjur secret")
	conjurCreds := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: s.framework.Namespace.Name,
		},
		Data: map[string][]byte{
			"apikey":   []byte(s.addon.AdminApiKey),
			"username": []byte("admin"),
		},
	}
	err := s.framework.CRClient.Create(GinkgoT().Context(), conjurCreds)
	Expect(err).ToNot(HaveOccurred())

	By("creating an secret store for conjur")
	secretStore := makeStore(defaultStoreName, s.framework.Namespace.Name, s.addon)
	secretStore.Spec.Provider.Conjur.Auth = esv1.ConjurAuth{
		APIKey: &esv1.ConjurAPIKey{
			Account: "default",
			UserRef: &esmeta.SecretKeySelector{
				Name: secretName,
				Key:  "username",
			},
			APIKeyRef: &esmeta.SecretKeySelector{
				Name: secretName,
				Key:  "apikey",
			},
		},
	}
	err = s.framework.CRClient.Create(GinkgoT().Context(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}

func (s conjurProvider) CreateJWTK8sStore() {
	// Create a service account
	sa := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appServiceAccountName,
			Namespace: s.framework.Namespace.Name,
		},
	}
	err := s.framework.CRClient.Create(GinkgoT().Context(), sa)
	Expect(err).ToNot(HaveOccurred())

	// Now create a secret store that uses the service account to authenticate
	secretStore := makeStore(jwtK8sProviderName, s.framework.Namespace.Name, s.addon)
	secretStore.Spec.Provider.Conjur.Auth = esv1.ConjurAuth{
		Jwt: &esv1.ConjurJWT{
			Account:   "default",
			ServiceID: "eso-tests",
			ServiceAccountRef: &esmeta.ServiceAccountSelector{
				Name: appServiceAccountName,
				Audiences: []string{
					s.addon.ConjurURL,
				},
			},
		},
	}
	err = s.framework.CRClient.Create(GinkgoT().Context(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}

func (s conjurProvider) CreateJWTK8sHostIDStore() {
	// Create a service account
	sa := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hostidServiceAccountName,
			Namespace: s.framework.Namespace.Name,
		},
	}
	err := s.framework.CRClient.Create(GinkgoT().Context(), sa)
	Expect(err).ToNot(HaveOccurred())

	saName := "system:serviceaccount:" + s.framework.Namespace.Name + ":" + hostidServiceAccountName

	// Now create a secret store that uses the service account to authenticate
	secretStore := makeStore(jwtK8sHostIDProviderName, s.framework.Namespace.Name, s.addon)
	secretStore.Spec.Provider.Conjur.Auth = esv1.ConjurAuth{
		Jwt: &esv1.ConjurJWT{
			Account:   "default",
			HostID:    "host/" + saName,
			ServiceID: "eso-tests-hostid",
			ServiceAccountRef: &esmeta.ServiceAccountSelector{
				Name: hostidServiceAccountName,
				Audiences: []string{
					s.addon.ConjurURL,
				},
			},
		},
	}
	err = s.framework.CRClient.Create(GinkgoT().Context(), secretStore)
	Expect(err).ToNot(HaveOccurred())
}
