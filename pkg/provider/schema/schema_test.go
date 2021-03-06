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
package schema

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/provider"
)

type PP struct{}

// New constructs a SecretsManager Provider.
func (p *PP) NewClient(ctx context.Context, store esv1alpha1.GenericStore, kube client.Client, namespace string) (provider.SecretsClient, error) {
	return p, nil
}

// GetSecret returns a single secret from the provider.
func (p *PP) GetSecret(ctx context.Context, ref esv1alpha1.ExternalSecretDataRemoteRef) ([]byte, error) {
	return []byte("NOOP"), nil
}

// GetSecretMap returns multiple k/v pairs from the provider.
func (p *PP) GetSecretMap(ctx context.Context, ref esv1alpha1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return map[string][]byte{}, nil
}

// TestRegister tests if the Register function
// (1) panics if it tries to register something invalid
// (2) stores the correct provider
func TestRegister(t *testing.T) {

	for _, row := range []struct {
		name     string
		expPanic bool
		provider *esv1alpha1.SecretStoreProvider
	}{
		{ // should panic
			name:     "aws/SecretsManager",
			expPanic: true,
			provider: &esv1alpha1.SecretStoreProvider{},
		},
		{
			// should register
			name: "aws/SecretsManager",
			provider: &esv1alpha1.SecretStoreProvider{
				AWS: &esv1alpha1.AWSProvider{
					Service: esv1alpha1.AWSServiceSecretsManager,
				},
			},
		},
		{
			// should panic: already exists
			name:     "aws/SecretsManager",
			expPanic: true,
			provider: &esv1alpha1.SecretStoreProvider{
				AWS: &esv1alpha1.AWSProvider{
					Service: esv1alpha1.AWSServiceSecretsManager,
				},
			},
		},
		{
			// should register pm service
			name:     "aws/ParameterStore",
			expPanic: true,
			provider: &esv1alpha1.SecretStoreProvider{
				AWS: &esv1alpha1.AWSProvider{
					Service: esv1alpha1.AWSServiceParameterStore,
				},
			},
		},
	} {
		p, ok := GetProviderByName(row.name)
		assert.Nil(t, p)
		assert.False(t, ok, "provider should not be registered")

		testProvider := &PP{}
		secretStore := &esv1alpha1.SecretStore{
			Spec: esv1alpha1.SecretStoreSpec{
				Provider: row.provider,
			},
		}

		if row.expPanic {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("Register should panic")
				}
			}()
		}
		Register(testProvider, secretStore.Spec.Provider)
		p1, ok := GetProviderByName(row.name)
		assert.True(t, ok, "provider should be registered")
		assert.Equal(t, testProvider, p1)

		p2, err := GetProvider(secretStore)
		assert.Nil(t, err)
		assert.Equal(t, testProvider, p2)
	}
}
