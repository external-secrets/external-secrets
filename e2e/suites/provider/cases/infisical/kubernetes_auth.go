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
	"time"

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
	withKubernetesAuth        = "with kubernetes auth"
	withKubernetesAuthCluster = "with kubernetes auth and cluster store"

	kubernetesIdentitySecretName = "infisical-kubernetes-identity"
	identityIDKey                = "identityId"
)

// createKubernetesAuthResources creates the ServiceAccount that the store
// logs in as and the Secret that holds the Kubernetes auth identity ID.
func (s *infisicalProvider) createKubernetesAuthResources(ns string) {
	sa := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      addon.InfisicalKubernetesAuthServiceAccount,
			Namespace: ns,
		},
	}
	err := s.framework.CRClient.Create(GinkgoT().Context(), sa)
	Expect(err).ToNot(HaveOccurred())

	identity := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kubernetesIdentitySecretName,
			Namespace: ns,
		},
		Data: map[string][]byte{
			identityIDKey: []byte(s.addon.KubernetesIdentityID),
		},
	}
	err = s.framework.CRClient.Create(GinkgoT().Context(), identity)
	Expect(err).ToNot(HaveOccurred())
}

// kubernetesAuthProviderSpec authenticates through serviceAccountRef. A
// ClusterSecretStore must name the ServiceAccount namespace, so ns is set for
// it and nil for a SecretStore.
func (s *infisicalProvider) kubernetesAuthProviderSpec(ns *string) *esv1.InfisicalProvider {
	spec := s.infisicalProviderSpec(ns)
	spec.Auth = esv1.InfisicalAuth{
		KubernetesAuthCredentials: &esv1.KubernetesAuthCredentials{
			IdentityID: esmeta.SecretKeySelector{
				Name:      kubernetesIdentitySecretName,
				Key:       identityIDKey,
				Namespace: ns,
			},
			ServiceAccountRef: &esmeta.ServiceAccountSelector{
				Name:      addon.InfisicalKubernetesAuthServiceAccount,
				Namespace: ns,
				Audiences: []string{addon.InfisicalKubernetesAuthAudience},
			},
		},
	}
	return spec
}

// CreateKubernetesAuthStore creates a namespaced SecretStore that logs in as
// the test namespace's ServiceAccount.
func (s *infisicalProvider) CreateKubernetesAuthStore() {
	ns := s.framework.Namespace.Name
	s.createKubernetesAuthResources(ns)

	store := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ns,
			Namespace: ns,
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				Infisical: s.kubernetesAuthProviderSpec(nil),
			},
		},
	}
	err := s.framework.CRClient.Create(GinkgoT().Context(), store)
	Expect(err).ToNot(HaveOccurred())
}

// CreateKubernetesAuthClusterStore creates a ClusterSecretStore that logs in
// as the ServiceAccount in the test namespace.
func (s *infisicalProvider) CreateKubernetesAuthClusterStore() {
	ns := s.framework.Namespace.Name
	s.createKubernetesAuthResources(ns)

	store := &esv1.ClusterSecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterStoreName(s.framework),
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				Infisical: s.kubernetesAuthProviderSpec(&ns),
			},
		},
	}
	err := s.framework.CRClient.Create(GinkgoT().Context(), store)
	Expect(err).ToNot(HaveOccurred())

	DeferCleanup(func() {
		// Cannot use the ginkgo context inside DeferCleanup, it would register
		// another cleanup. Use a plain context with a short timeout instead.
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		_ = s.framework.CRClient.Delete(ctx, store)
	})
}

func useKubernetesAuth(prov *infisicalProvider) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		prov.CreateKubernetesAuthStore()
		tc.ExternalSecret.Spec.SecretStoreRef.Name = tc.Framework.Namespace.Name
	}
}

func useKubernetesAuthClusterStore(prov *infisicalProvider) func(*framework.TestCase) {
	return func(tc *framework.TestCase) {
		prov.CreateKubernetesAuthClusterStore()
		tc.ExternalSecret.Spec.SecretStoreRef.Name = clusterStoreName(tc.Framework)
		tc.ExternalSecret.Spec.SecretStoreRef.Kind = esv1.ClusterSecretStoreKind
	}
}
