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

package v1beta1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// ValidationProvider is a simple provider that we can use without cyclic import.
type ValidationProvider struct {
	ProviderInterface
}

func (v *ValidationProvider) ValidateStore(_ GenericStore) (admission.Warnings, error) {
	return nil, nil
}

func TestValidateSecretStore(t *testing.T) {
	tests := []struct {
		name      string
		obj       *SecretStore
		mock      func()
		assertErr func(t *testing.T, err error)
	}{
		{
			name: "valid regex",
			obj: &SecretStore{
				Spec: SecretStoreSpec{
					Conditions: []ClusterSecretStoreCondition{
						{
							NamespaceRegexes: []string{`.*`},
						},
					},
					Provider: &SecretStoreProvider{
						AWS: &AWSProvider{},
					},
				},
			},
			mock: func() {
				ForceRegister(&ValidationProvider{}, &SecretStoreProvider{
					AWS: &AWSProvider{},
				})
			},
			assertErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "invalid regex",
			obj: &SecretStore{
				Spec: SecretStoreSpec{
					Conditions: []ClusterSecretStoreCondition{
						{
							NamespaceRegexes: []string{`\1`},
						},
					},
					Provider: &SecretStoreProvider{
						AWS: &AWSProvider{},
					},
				},
			},
			mock: func() {
				ForceRegister(&ValidationProvider{}, &SecretStoreProvider{
					AWS: &AWSProvider{},
				})
			},
			assertErr: func(t *testing.T, err error) {
				assert.EqualError(t, err, "failed to compile 0th namespace regex in 0th condition: error parsing regexp: invalid escape sequence: `\\1`")
			},
		},
		{
			name: "multiple errors",
			obj: &SecretStore{
				Spec: SecretStoreSpec{
					Conditions: []ClusterSecretStoreCondition{
						{
							NamespaceRegexes: []string{`\1`, `\2`},
						},
					},
					Provider: &SecretStoreProvider{
						AWS: &AWSProvider{},
					},
				},
			},
			mock: func() {
				ForceRegister(&ValidationProvider{}, &SecretStoreProvider{
					AWS: &AWSProvider{},
				})
			},
			assertErr: func(t *testing.T, err error) {
				assert.EqualError(
					t,
					err,
					"failed to compile 0th namespace regex in 0th condition: error parsing regexp: invalid escape sequence: `\\1`\nfailed to compile 1th namespace regex in 0th condition: error parsing regexp: invalid escape sequence: `\\2`",
				)
			},
		},
		{
			name: "secret store must have only a single backend",
			obj: &SecretStore{
				Spec: SecretStoreSpec{
					Provider: &SecretStoreProvider{
						AWS:   &AWSProvider{},
						GCPSM: &GCPSMProvider{},
					},
				},
			},
			assertErr: func(t *testing.T, err error) {
				assert.EqualError(t, err, "store error for : secret stores must only have exactly one backend specified, found 2")
			},
		},
		{
			name: "requires provider or providerRef",
			obj: &SecretStore{
				Spec: SecretStoreSpec{
					Conditions: []ClusterSecretStoreCondition{
						{
							Namespaces: []string{"default"},
						},
					},
				},
			},
			assertErr: func(t *testing.T, err error) {
				assert.EqualError(t, err, "exactly one of spec.provider or spec.providerRef must be set")
			},
		},
		{
			name: "rejects provider and providerRef together",
			obj: &SecretStore{
				Spec: SecretStoreSpec{
					Provider: &SecretStoreProvider{
						AWS: &AWSProvider{},
					},
					ProviderRef: &StoreProviderRef{
						Name: "aws",
					},
				},
			},
			assertErr: func(t *testing.T, err error) {
				assert.EqualError(t, err, "exactly one of spec.provider or spec.providerRef must be set")
			},
		},
		{
			name: "rejects provider with runtimeRef",
			obj: &SecretStore{
				Spec: SecretStoreSpec{
					Provider: &SecretStoreProvider{
						AWS: &AWSProvider{},
					},
					RuntimeRef: &StoreRuntimeRef{
						Kind: "ProviderClass",
						Name: "aws",
					},
				},
			},
			assertErr: func(t *testing.T, err error) {
				assert.EqualError(t, err, "spec.runtimeRef must be empty when spec.provider is set")
			},
		},
		{
			name: "rejects providerRef without runtimeRef",
			obj: &SecretStore{
				Spec: SecretStoreSpec{
					ProviderRef: &StoreProviderRef{
						APIVersion: "external-secrets.io/v1beta1",
						Kind:       "Provider",
						Name:       "aws",
					},
				},
			},
			assertErr: func(t *testing.T, err error) {
				assert.EqualError(t, err, "spec.runtimeRef is required when spec.providerRef is set")
			},
		},
		{
			name: "rejects providerRef namespace mismatch",
			obj: &SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "store",
					Namespace: "default",
				},
				Spec: SecretStoreSpec{
					ProviderRef: &StoreProviderRef{
						APIVersion: "external-secrets.io/v1beta1",
						Kind:       "Provider",
						Name:       "aws",
						Namespace:  "other",
					},
					RuntimeRef: &StoreRuntimeRef{
						Kind: "ProviderClass",
						Name: "aws",
					},
				},
			},
			assertErr: func(t *testing.T, err error) {
				assert.EqualError(t, err, "spec.providerRef.namespace must be empty or match metadata.namespace")
			},
		},
		{
			name: "allows providerRef with runtimeRef",
			obj: &SecretStore{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "store",
					Namespace: "default",
				},
				Spec: SecretStoreSpec{
					ProviderRef: &StoreProviderRef{
						APIVersion: "external-secrets.io/v1beta1",
						Kind:       "Provider",
						Name:       "aws",
					},
					RuntimeRef: &StoreRuntimeRef{
						Kind: "ProviderClass",
						Name: "aws",
					},
				},
			},
			assertErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "rejects empty providerRef",
			obj: &SecretStore{
				Spec: SecretStoreSpec{
					ProviderRef: &StoreProviderRef{},
					RuntimeRef: &StoreRuntimeRef{
						Kind: "ProviderClass",
						Name: "aws",
					},
				},
			},
			assertErr: func(t *testing.T, err error) {
				assert.EqualError(t, err, "spec.providerRef.apiVersion is required")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mock != nil {
				tt.mock()
			}

			_, err := validateStore(tt.obj)
			tt.assertErr(t, err)
		})
	}
}

func TestValidateStoreRejectsProviderClassForClusterSecretStore(t *testing.T) {
	store := &ClusterSecretStore{
		Spec: SecretStoreSpec{
			ProviderRef: &StoreProviderRef{
				APIVersion: "external-secrets.io/v1beta1",
				Kind:       "Provider",
				Name:       "aws",
			},
			RuntimeRef: &StoreRuntimeRef{
				Kind: "ProviderClass",
				Name: "aws",
			},
		},
	}

	_, err := validateStore(store)
	require.Error(t, err)
	assert.EqualError(t, err, "ClusterSecretStore runtimeRef.kind must not be \"ProviderClass\"")
}

func TestValidateClusterSecretStoreAllowsProviderRefRuntimeRef(t *testing.T) {
	store := &ClusterSecretStore{
		Spec: SecretStoreSpec{
			ProviderRef: &StoreProviderRef{
				APIVersion: "external-secrets.io/v1beta1",
				Kind:       "Provider",
				Name:       "aws",
			},
			RuntimeRef: &StoreRuntimeRef{
				Kind: "ClusterProviderClass",
				Name: "aws",
			},
		},
	}

	_, err := validateStore(store)
	require.NoError(t, err)
}

func TestValidateStoreSkipsInlineProviderValidationForProviderRefMode(t *testing.T) {
	store := &SecretStore{
		ObjectMeta: metav1.ObjectMeta{Namespace: "team-a"},
		Spec: SecretStoreSpec{
			RuntimeRef: &StoreRuntimeRef{Name: "fake-runtime"},
			ProviderRef: &StoreProviderRef{
				APIVersion: "provider.external-secrets.io/v2alpha1",
				Kind:       "Fake",
				Name:       "fake-config",
			},
		},
	}

	_, err := validateStore(store)
	require.NoError(t, err)
}
