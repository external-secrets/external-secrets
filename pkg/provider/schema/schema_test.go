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
func (p *PP) New(ctx context.Context, store esv1alpha1.SecretStoreProvider, kube client.Client, namespace string) (provider.Provider, error) {
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

func TestRegister(t *testing.T) {
	p, ok := GetProviderByName("awssm")
	assert.Nil(t, p)
	assert.False(t, ok, "provider should not be registered")

	testProvider := &PP{}
	secretStore := &esv1alpha1.SecretStore{
		Spec: esv1alpha1.SecretStoreSpec{
			Provider: &esv1alpha1.SecretStoreProvider{
				AWSSM: &esv1alpha1.AWSSMProvider{},
			},
		},
	}

	ForceRegister(testProvider, secretStore.Spec.Provider)
	p1, ok := GetProviderByName("awssm")
	assert.True(t, ok, "provider should be registered")
	assert.Equal(t, testProvider, p1)

	p2, err := GetProvider(secretStore)
	assert.Nil(t, err)
	assert.Equal(t, testProvider, p2)
}
