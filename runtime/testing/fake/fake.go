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

package fake

import (
	"context"
	"sync"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

var _ esv1.Provider = &Client{}

type SetSecretCallArgs struct {
	Value     []byte
	RemoteRef esv1.PushSecretRemoteRef
}

// Client is a fake client for testing.
type Client struct {
	mu              *sync.RWMutex
	pushSecretData  map[string]SetSecretCallArgs
	NewFn           func(context.Context, esv1.GenericStore, client.Client, string) (esv1.SecretsClient, error)
	GetSecretFn     func(context.Context, esv1.ExternalSecretDataRemoteRef) ([]byte, error)
	GetSecretMapFn  func(context.Context, esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error)
	GetAllSecretsFn func(context.Context, esv1.ExternalSecretFind) (map[string][]byte, error)
	SecretExistsFn  func(context.Context, esv1.PushSecretRemoteRef) (bool, error)
	SetSecretFn     func() error
	DeleteSecretFn  func() error
}

// New returns a fake provider/client.
func New() *Client {
	v := &Client{
		mu: &sync.RWMutex{},
		GetSecretFn: func(context.Context, esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
			return nil, nil
		},
		GetSecretMapFn: func(context.Context, esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
			return nil, nil
		},
		GetAllSecretsFn: func(context.Context, esv1.ExternalSecretFind) (map[string][]byte, error) {
			return nil, nil
		},
		SecretExistsFn: func(context.Context, esv1.PushSecretRemoteRef) (bool, error) {
			return false, nil
		},
		SetSecretFn: func() error {
			return nil
		},
		DeleteSecretFn: func() error {
			return nil
		},
		pushSecretData: map[string]SetSecretCallArgs{},
	}

	v.NewFn = func(context.Context, esv1.GenericStore, client.Client, string) (esv1.SecretsClient, error) {
		return v, nil
	}

	return v
}

// RegisterAs registers the fake client in the schema.
func (v *Client) RegisterAs(provider *esv1.SecretStoreProvider) {
	esv1.ForceRegister(v, provider, esv1.MaintenanceStatusMaintained)
}

// GetAllSecrets implements the provider.Provider interface.
func (v *Client) GetAllSecrets(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	return v.GetAllSecretsFn(ctx, ref)
}

func (v *Client) PushSecret(_ context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.pushSecretData[data.GetRemoteKey()] = SetSecretCallArgs{
		Value:     secret.Data[data.GetSecretKey()],
		RemoteRef: data,
	}
	return v.SetSecretFn()
}

// GetPushSecretData safely retrieves the push secret data map for reading.
func (v *Client) GetPushSecretData() map[string]SetSecretCallArgs {
	v.mu.RLock()
	defer v.mu.RUnlock()
	// Create a copy to avoid race conditions
	result := make(map[string]SetSecretCallArgs, len(v.pushSecretData))
	for k, v := range v.pushSecretData {
		result[k] = v
	}
	return result
}

func (v *Client) DeleteSecret(_ context.Context, _ esv1.PushSecretRemoteRef) error {
	return v.DeleteSecretFn()
}

func (v *Client) SecretExists(ctx context.Context, ref esv1.PushSecretRemoteRef) (bool, error) {
	return v.SecretExistsFn(ctx, ref)
}

// GetSecret implements the provider.Provider interface.
func (v *Client) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	return v.GetSecretFn(ctx, ref)
}

// WithGetSecret wraps secret data returned by this provider.
func (v *Client) WithGetSecret(secData []byte, err error) *Client {
	v.GetSecretFn = func(context.Context, esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
		return secData, err
	}
	return v
}

// GetSecretMap implements the provider.Provider interface.
func (v *Client) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return v.GetSecretMapFn(ctx, ref)
}

func (v *Client) Close(_ context.Context) error {
	return nil
}

func (v *Client) Validate() (esv1.ValidationResult, error) {
	return esv1.ValidationResultReady, nil
}

func (v *Client) ValidateStore(_ esv1.GenericStore) (admission.Warnings, error) {
	return nil, nil
}

// WithGetSecretMap wraps the secret data map returned by this fake provider.
func (v *Client) WithGetSecretMap(secData map[string][]byte, err error) *Client {
	v.GetSecretMapFn = func(context.Context, esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
		return secData, err
	}
	return v
}

// WithGetAllSecrets wraps the secret data map returned by this fake provider.
func (v *Client) WithGetAllSecrets(secData map[string][]byte, err error) *Client {
	v.GetAllSecretsFn = func(context.Context, esv1.ExternalSecretFind) (map[string][]byte, error) {
		return secData, err
	}
	return v
}

// WithSetSecret wraps the secret response to the fake provider.
func (v *Client) WithSetSecret(err error) *Client {
	v.SetSecretFn = func() error {
		return err
	}
	return v
}

// WithNew wraps the fake provider factory function.
func (v *Client) WithNew(f func(context.Context, esv1.GenericStore, client.Client,
	string) (esv1.SecretsClient, error)) *Client {
	v.NewFn = f
	return v
}

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (v *Client) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadOnly
}

// NewClient returns a new fake provider.
func (v *Client) NewClient(ctx context.Context, store esv1.GenericStore, kube client.Client, namespace string) (esv1.SecretsClient, error) {
	c, err := v.NewFn(ctx, store, kube, namespace)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (v *Client) Reset() {
	v.WithNew(func(context.Context, esv1.GenericStore, client.Client,
		string) (esv1.SecretsClient, error) {
		return v, nil
	})
	v.mu.Lock()
	defer v.mu.Unlock()
	v.pushSecretData = map[string]SetSecretCallArgs{}
}
