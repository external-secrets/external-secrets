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
package v1beta1

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PP struct{}

const shouldBeRegistered = "provider should be registered"

func (p *PP) Capabilities() SecretStoreCapabilities {
	return SecretStoreReadOnly
}

// New constructs a SecretsManager Provider.
func (p *PP) NewClient(_ context.Context, _ GenericStore, _ client.Client, _ string) (SecretsClient, error) {
	return p, nil
}

// PushSecret writes a single secret into a provider.
func (p *PP) PushSecret(ctx context.Context, values map[string][]byte, typed corev1.SecretType, metadata *apiextensionsv1.JSON, remoteRef PushRemoteRef) error {
	return nil
}

// DeleteSecret deletes a single secret from a provider.
func (p *PP) DeleteSecret(_ context.Context, _ PushRemoteRef) error {
	return nil
}

// GetSecret returns a single secret from the provider.
func (p *PP) GetSecret(_ context.Context, _ ExternalSecretDataRemoteRef) ([]byte, error) {
	return []byte("NOOP"), nil
}

// GetSecretMap returns multiple k/v pairs from the provider.
func (p *PP) GetSecretMap(_ context.Context, _ ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return map[string][]byte{}, nil
}

// Empty GetAllSecrets.
func (p *PP) GetAllSecrets(_ context.Context, _ ExternalSecretFind) (map[string][]byte, error) {
	// TO be implemented
	return map[string][]byte{}, nil
}

func (p *PP) Close(_ context.Context) error {
	return nil
}

func (p *PP) Validate() (ValidationResult, error) {
	return ValidationResultReady, nil
}

func (p *PP) ValidateStore(_ GenericStore) error {
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
		provider  *SecretStoreProvider
	}{
		{
			test:      "should panic when given an invalid provider",
			name:      "aws",
			expPanic:  true,
			expExists: false,
			provider:  &SecretStoreProvider{},
		},
		{
			test:      "should register an correct provider",
			name:      "aws",
			expExists: false,
			provider: &SecretStoreProvider{
				AWS: &AWSProvider{
					Service: AWSServiceSecretsManager,
				},
			},
		},
		{
			test:      "should panic if already exists",
			name:      "aws",
			expPanic:  true,
			expExists: true,
			provider: &SecretStoreProvider{
				AWS: &AWSProvider{
					Service: AWSServiceSecretsManager,
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

func runTest(t *testing.T, name string, provider *SecretStoreProvider, expPanic bool) {
	testProvider := &PP{}
	secretStore := &SecretStore{
		Spec: SecretStoreSpec{
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
	provider := &SecretStoreProvider{
		AWS: &AWSProvider{
			Service: AWSServiceParameterStore,
		},
	}
	secretStore := &SecretStore{
		Spec: SecretStoreSpec{
			Provider: provider,
		},
	}
	ForceRegister(testProvider, &SecretStoreProvider{
		AWS: &AWSProvider{
			Service: AWSServiceParameterStore,
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
	secretStore := &SecretStore{
		Spec: SecretStoreSpec{
			Provider: &SecretStoreProvider{
				GCPSM: &GCPSMProvider{},
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
