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

// Package v1_test exercises the esv1 provider lookup stubs via the runtime/provider registry.
package v1_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/provider"
)

// PP is a minimal test provider that satisfies esv1.Provider.
type PP struct{}

func (p *PP) Capabilities() esv1.SecretStoreCapabilities { return esv1.SecretStoreReadOnly }
func (p *PP) NewClient(_ context.Context, _ esv1.GenericStore, _ client.Client, _ string) (esv1.SecretsClient, error) {
	return p, nil
}
func (p *PP) PushSecret(_ context.Context, _ *corev1.Secret, _ esv1.PushSecretData) error { return nil }
func (p *PP) DeleteSecret(_ context.Context, _ esv1.PushSecretRemoteRef) error            { return nil }
func (p *PP) SecretExists(_ context.Context, _ esv1.PushSecretRemoteRef) (bool, error)    { return false, nil }
func (p *PP) GetSecret(_ context.Context, _ esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	return []byte("NOOP"), nil
}
func (p *PP) GetSecretMap(_ context.Context, _ esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return map[string][]byte{}, nil
}
func (p *PP) GetAllSecrets(_ context.Context, _ esv1.ExternalSecretFind) (map[string][]byte, error) {
	return map[string][]byte{}, nil
}
func (p *PP) Close(_ context.Context) error                      { return nil }
func (p *PP) Validate() (esv1.ValidationResult, error)           { return esv1.ValidationResultReady, nil }
func (p *PP) ValidateStore(_ esv1.GenericStore) (admission.Warnings, error) { return nil, nil }

const shouldBeRegistered = "provider should be registered"

func TestGetProviderByName(t *testing.T) {
	testProvider := &PP{}
	provider.ForceRegisterProvider(testProvider, &esv1.SecretStoreProvider{
		AWS: &esv1.AWSProvider{
			Service: esv1.AWSServiceSecretsManager,
		},
	})

	p1, ok := esv1.GetProviderByName("aws")
	assert.True(t, ok, shouldBeRegistered)
	assert.Equal(t, testProvider, p1)
}

func TestGetProvider(t *testing.T) {
	testProvider := &PP{}
	spec := &esv1.SecretStoreProvider{
		GCPSM: &esv1.GCPSMProvider{},
	}
	provider.ForceRegisterProvider(testProvider, spec)

	secretStore := &esv1.SecretStore{
		Spec: esv1.SecretStoreSpec{
			Provider: spec,
		},
	}
	p2, err := esv1.GetProvider(secretStore)
	assert.Nil(t, err)
	assert.Equal(t, testProvider, p2)
}

func TestListProviders(t *testing.T) {
	testProvider := &PP{}
	provider.ForceRegisterProvider(testProvider, &esv1.SecretStoreProvider{
		AWS: &esv1.AWSProvider{
			Service: esv1.AWSServiceParameterStore,
		},
	})

	providers := esv1.List()
	assert.NotNil(t, providers)
	_, ok := providers["aws"]
	assert.True(t, ok, "aws provider should be in list")
}
