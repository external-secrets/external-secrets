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

	esv1alpha2 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha2"
	"github.com/external-secrets/external-secrets/pkg/provider"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

type PP struct{}

const shouldBeRegistered = "provider should be registered"

// New constructs a SecretsManager Provider.
func (p *PP) NewClient(ctx context.Context, store esv1alpha2.GenericStore, kube client.Client, namespace string) (provider.SecretsClient, error) {
	return p, nil
}

// GetSecret returns a single secret from the provider.
func (p *PP) GetSecret(ctx context.Context, ref esv1alpha2.ExternalSecretDataRemoteRef) ([]byte, error) {
	return []byte("NOOP"), nil
}

// GetSecretMap returns multiple k/v pairs from the provider.
func (p *PP) GetSecretMap(ctx context.Context, ref esv1alpha2.ExternalSecretDataFromRemoteRef) (map[string][]byte, error) {
	return map[string][]byte{}, nil
}

// Implements store.Client.GetAllSecrets Interface.
// New version of GetAllSecrets.
func (p *PP) GetAllSecrets(ctx context.Context, ref esv1alpha2.ExternalSecretDataFromRemoteRef) (map[string][]byte, error) {
	// TO be implemented
	return nil, utils.ThrowNotImplemented()
}

func (p *PP) Close(ctx context.Context) error {
	return nil
}

// TestRegister tests if the Register function
// (1) panics if it tries to register something invalid
// (2) stores the correct provider.
func TestRegister(t *testing.T) {
	tbl := []struct {
		test      string
		name      string
		expPanic  bool
		expExists bool
		provider  *esv1alpha2.SecretStoreProvider
	}{
		{
			test:      "should panic when given an invalid provider",
			name:      "aws",
			expPanic:  true,
			expExists: false,
			provider:  &esv1alpha2.SecretStoreProvider{},
		},
		{
			test:      "should register an correct provider",
			name:      "aws",
			expExists: false,
			provider: &esv1alpha2.SecretStoreProvider{
				AWS: &esv1alpha2.AWSProvider{
					Service: esv1alpha2.AWSServiceSecretsManager,
				},
			},
		},
		{
			test:      "should panic if already exists",
			name:      "aws",
			expPanic:  true,
			expExists: true,
			provider: &esv1alpha2.SecretStoreProvider{
				AWS: &esv1alpha2.AWSProvider{
					Service: esv1alpha2.AWSServiceSecretsManager,
				},
			},
		},
	}
	for i := range tbl {
		row := tbl[i]
		t.Run(row.test, func(t *testing.T) {
			runTest(t,
				row.name,
				row.provider,
				row.expPanic,
			)
		})
	}
}

func runTest(t *testing.T, name string, provider *esv1alpha2.SecretStoreProvider, expPanic bool) {
	testProvider := &PP{}
	secretStore := &esv1alpha2.SecretStore{
		Spec: esv1alpha2.SecretStoreSpec{
			Provider: provider,
		},
	}
	if expPanic {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Register should panic")
			}
		}()
	}
	Register(testProvider, secretStore.Spec.Provider)
	p1, ok := GetProviderByName(name)
	assert.True(t, ok, shouldBeRegistered)
	assert.Equal(t, testProvider, p1)
	p2, err := GetProvider(secretStore)
	assert.Nil(t, err)
	assert.Equal(t, testProvider, p2)
}

// ForceRegister is used by other tests, we should ensure it works as expected.
func TestForceRegister(t *testing.T) {
	testProvider := &PP{}
	provider := &esv1alpha2.SecretStoreProvider{
		AWS: &esv1alpha2.AWSProvider{
			Service: esv1alpha2.AWSServiceParameterStore,
		},
	}
	secretStore := &esv1alpha2.SecretStore{
		Spec: esv1alpha2.SecretStoreSpec{
			Provider: provider,
		},
	}
	ForceRegister(testProvider, &esv1alpha2.SecretStoreProvider{
		AWS: &esv1alpha2.AWSProvider{
			Service: esv1alpha2.AWSServiceParameterStore,
		},
	})
	p1, ok := GetProviderByName("aws")
	assert.True(t, ok, shouldBeRegistered)
	assert.Equal(t, testProvider, p1)
	p2, err := GetProvider(secretStore)
	assert.Nil(t, err)
	assert.Equal(t, testProvider, p2)
}

func TestRegisterGCP(t *testing.T) {
	p, ok := GetProviderByName("gcpsm")
	assert.Nil(t, p)
	assert.False(t, ok, "provider should not be registered")

	testProvider := &PP{}
	secretStore := &esv1alpha2.SecretStore{
		Spec: esv1alpha2.SecretStoreSpec{
			Provider: &esv1alpha2.SecretStoreProvider{
				GCPSM: &esv1alpha2.GCPSMProvider{},
			},
		},
	}

	ForceRegister(testProvider, secretStore.Spec.Provider)
	p1, ok := GetProviderByName("gcpsm")
	assert.True(t, ok, shouldBeRegistered)
	assert.Equal(t, testProvider, p1)

	p2, err := GetProvider(secretStore)
	assert.Nil(t, err)
	assert.Equal(t, testProvider, p2)
}
