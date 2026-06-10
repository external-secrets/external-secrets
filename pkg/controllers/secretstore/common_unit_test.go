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

package secretstore

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

type fakeCapabilityClient struct {
	caps esapi.SecretStoreCapabilities
}

func (f *fakeCapabilityClient) GetSecret(context.Context, esapi.ExternalSecretDataRemoteRef) ([]byte, error) {
	return nil, nil
}

func (f *fakeCapabilityClient) PushSecret(context.Context, *corev1.Secret, esapi.PushSecretData) error {
	return nil
}

func (f *fakeCapabilityClient) DeleteSecret(context.Context, esapi.PushSecretRemoteRef) error {
	return nil
}

func (f *fakeCapabilityClient) SecretExists(context.Context, esapi.PushSecretRemoteRef) (bool, error) {
	return false, nil
}

func (f *fakeCapabilityClient) Validate() (esapi.ValidationResult, error) {
	return esapi.ValidationResultReady, nil
}

func (f *fakeCapabilityClient) GetSecretMap(context.Context, esapi.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return nil, nil
}

func (f *fakeCapabilityClient) GetAllSecrets(context.Context, esapi.ExternalSecretFind) (map[string][]byte, error) {
	return nil, nil
}

func (f *fakeCapabilityClient) Close(context.Context) error {
	return nil
}

func (f *fakeCapabilityClient) Capabilities(context.Context) (esapi.SecretStoreCapabilities, error) {
	return f.caps, nil
}

func TestResolveStoreCapabilitiesUsesRemoteClientWhenProviderRefIsSet(t *testing.T) {
	store := &esapi.SecretStore{
		Spec: esapi.SecretStoreSpec{
			RuntimeRef: &esapi.StoreRuntimeRef{Name: "fake-runtime"},
			ProviderRef: &esapi.StoreProviderRef{
				APIVersion: "provider.external-secrets.io/v2alpha1",
				Kind:       "Fake",
				Name:       "fake-config",
			},
		},
	}

	client := &fakeCapabilityClient{caps: esapi.SecretStoreReadWrite}
	caps, err := resolveStoreCapabilities(context.Background(), store, client)
	if err != nil {
		t.Fatalf("resolveStoreCapabilities() error = %v", err)
	}
	if caps != esapi.SecretStoreReadWrite {
		t.Fatalf("expected %v, got %v", esapi.SecretStoreReadWrite, caps)
	}
}

func TestShouldDeferClusterStoreValidation(t *testing.T) {
	t.Run("defer caller dependent cluster store", func(t *testing.T) {
		store := &esapi.ClusterSecretStore{
			TypeMeta: metav1.TypeMeta{Kind: esapi.ClusterSecretStoreKind},
			Spec: esapi.SecretStoreSpec{
				RuntimeRef: &esapi.StoreRuntimeRef{Name: "fake-runtime"},
				ProviderRef: &esapi.StoreProviderRef{
					APIVersion: "provider.external-secrets.io/v2alpha1",
					Kind:       "Fake",
					Name:       "fake-config",
				},
			},
		}

		if !shouldDeferClusterStoreValidation(store, "") {
			t.Fatalf("expected cluster store validation to defer without caller namespace")
		}
	})

	t.Run("do not defer namespaced provider ref", func(t *testing.T) {
		store := &esapi.ClusterSecretStore{
			TypeMeta: metav1.TypeMeta{Kind: esapi.ClusterSecretStoreKind},
			Spec: esapi.SecretStoreSpec{
				RuntimeRef: &esapi.StoreRuntimeRef{Name: "fake-runtime"},
				ProviderRef: &esapi.StoreProviderRef{
					APIVersion: "provider.external-secrets.io/v2alpha1",
					Kind:       "Fake",
					Name:       "fake-config",
					Namespace:  "provider-ns",
				},
			},
		}

		if shouldDeferClusterStoreValidation(store, "") {
			t.Fatalf("did not expect namespaced provider ref to defer validation")
		}
	})

	t.Run("do not defer when caller namespace is available", func(t *testing.T) {
		store := &esapi.ClusterSecretStore{
			TypeMeta: metav1.TypeMeta{Kind: esapi.ClusterSecretStoreKind},
			Spec: esapi.SecretStoreSpec{
				RuntimeRef: &esapi.StoreRuntimeRef{Name: "fake-runtime"},
				ProviderRef: &esapi.StoreProviderRef{
					APIVersion: "provider.external-secrets.io/v2alpha1",
					Kind:       "Fake",
					Name:       "fake-config",
				},
			},
		}

		if shouldDeferClusterStoreValidation(store, "workload-ns") {
			t.Fatalf("did not expect validation to defer when caller namespace is known")
		}
	})
}
