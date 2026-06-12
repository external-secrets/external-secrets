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

package infisical

import (
	"context"
	"path"
	"strings"
	"sync"
	"time"

	infisicalSdk "github.com/infisical/go-sdk"

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

const (
	credentialsSecretName = "infisical-credentials"
	clientIDKey           = "clientId"
	clientSecretKey       = "clientSecret"
	// scopePath is the secret path the store is scoped to. All e2e keys live
	// directly under it; the provider resolves bare keys against this path.
	scopePath = "/"
)

type infisicalProvider struct {
	addon     *addon.Infisical
	framework *framework.Framework

	// created tracks the keys seeded for the current spec so DeleteSecret is
	// idempotent: the framework's deferred cleanup may delete a key that a
	// test (e.g. DeletionPolicyDelete) already removed, and Infisical errors
	// on deleting a missing secret.
	mu      sync.Mutex
	created map[string]bool
}

func newInfisicalProvider(f *framework.Framework, a *addon.Infisical) *infisicalProvider {
	prov := &infisicalProvider{
		addon:     a,
		framework: f,
	}
	BeforeEach(func() {
		prov.mu.Lock()
		prov.created = map[string]bool{}
		prov.mu.Unlock()
	})
	return prov
}

// CreateSecret seeds a secret in Infisical. The provider is read-only, so the
// suite writes through the SDK rather than via PushSecret.
func (s *infisicalProvider) CreateSecret(key string, val framework.SecretEntry) {
	secretPath, name := secretAddress(scopePath, key)
	_, err := s.addon.SDKClient.Secrets().Create(infisicalSdk.CreateSecretOptions{
		ProjectID:             s.addon.ProjectID,
		Environment:           s.addon.EnvironmentSlug,
		SecretPath:            secretPath,
		SecretKey:             name,
		SecretValue:           val.Value,
		SkipMultiLineEncoding: true,
	})
	Expect(err).ToNot(HaveOccurred())

	s.mu.Lock()
	s.created[key] = true
	s.mu.Unlock()
}

func (s *infisicalProvider) DeleteSecret(key string) {
	s.mu.Lock()
	seeded := s.created[key]
	delete(s.created, key)
	s.mu.Unlock()
	if !seeded {
		return
	}

	secretPath, name := secretAddress(scopePath, key)
	_, err := s.addon.SDKClient.Secrets().Delete(infisicalSdk.DeleteSecretOptions{
		ProjectID:   s.addon.ProjectID,
		Environment: s.addon.EnvironmentSlug,
		SecretPath:  secretPath,
		SecretKey:   name,
	})
	Expect(err).ToNot(HaveOccurred())
}

// secretAddress mirrors the provider's key resolution so the seeded path
// matches where the provider looks the secret up.
//   - no slash:        ("foo", "/scope")      -> ("/scope", "foo")
//   - leading slash:   ("/a/b/foo", "/scope") -> ("/a/b", "foo")
//   - relative path:   ("sub/foo", "/scope")  -> ("/scope/sub", "foo")
func secretAddress(defaultPath, key string) (string, string) {
	if !strings.Contains(key, "/") {
		return defaultPath, key
	}
	lastIndex := strings.LastIndex(key, "/")
	folder, name := key[:lastIndex], key[lastIndex+1:]
	if strings.HasPrefix(key, "/") {
		return folder, name
	}
	return path.Join(defaultPath, folder), name
}

func (s *infisicalProvider) credentialsSecret(ns string) *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      credentialsSecretName,
			Namespace: ns,
		},
		Data: map[string][]byte{
			clientIDKey:     []byte(s.addon.ClientID),
			clientSecretKey: []byte(s.addon.ClientSecret),
		},
	}
}

func (s *infisicalProvider) infisicalProviderSpec(credsNamespace *string) *esv1.InfisicalProvider {
	return &esv1.InfisicalProvider{
		HostAPI: s.addon.HostAPI,
		Auth: esv1.InfisicalAuth{
			UniversalAuthCredentials: &esv1.UniversalAuthCredentials{
				ClientID: esmeta.SecretKeySelector{
					Name:      credentialsSecretName,
					Key:       clientIDKey,
					Namespace: credsNamespace,
				},
				ClientSecret: esmeta.SecretKeySelector{
					Name:      credentialsSecretName,
					Key:       clientSecretKey,
					Namespace: credsNamespace,
				},
			},
		},
		SecretsScope: esv1.MachineIdentityScopeInWorkspace{
			ProjectSlug:            s.addon.ProjectSlug,
			EnvironmentSlug:        s.addon.EnvironmentSlug,
			SecretsPath:            scopePath,
			ExpandSecretReferences: true,
		},
	}
}

// CreateUniversalAuthStore creates a namespaced SecretStore authenticated with
// the Universal Auth machine identity.
func (s *infisicalProvider) CreateUniversalAuthStore() {
	ns := s.framework.Namespace.Name
	err := s.framework.CRClient.Create(GinkgoT().Context(), s.credentialsSecret(ns))
	Expect(err).ToNot(HaveOccurred())

	store := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ns,
			Namespace: ns,
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				Infisical: s.infisicalProviderSpec(nil),
			},
		},
	}
	err = s.framework.CRClient.Create(GinkgoT().Context(), store)
	Expect(err).ToNot(HaveOccurred())
}

// CreateUniversalAuthClusterStore creates a ClusterSecretStore that references
// the credentials Secret in the test namespace.
func (s *infisicalProvider) CreateUniversalAuthClusterStore() {
	ns := s.framework.Namespace.Name
	err := s.framework.CRClient.Create(GinkgoT().Context(), s.credentialsSecret(ns))
	Expect(err).ToNot(HaveOccurred())

	store := &esv1.ClusterSecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterStoreName(s.framework),
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				Infisical: s.infisicalProviderSpec(&ns),
			},
		},
	}
	err = s.framework.CRClient.Create(GinkgoT().Context(), store)
	Expect(err).ToNot(HaveOccurred())

	DeferCleanup(func() {
		// Cannot use the ginkgo context inside DeferCleanup, it would register
		// another cleanup. Use a plain context with a short timeout instead.
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		_ = s.framework.CRClient.Delete(ctx, store)
	})
}

func clusterStoreName(f *framework.Framework) string {
	return "infisical-" + f.Namespace.Name
}
