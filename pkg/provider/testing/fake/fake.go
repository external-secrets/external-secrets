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

package fake

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

var _ esv1beta1.Provider = &Client{}

// Client is a fake client for testing.
type Client struct {
	NewFn           func(context.Context, esv1beta1.GenericStore, client.Client, string) (esv1beta1.SecretsClient, error)
	GetSecretFn     func(context.Context, esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error)
	GetSecretMapFn  func(context.Context, esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error)
	GetAllSecretsFn func(context.Context, esv1beta1.ExternalSecretFind) (map[string][]byte, error)
}

// New returns a fake provider/client.
func New() *Client {
	v := &Client{
		GetSecretFn: func(context.Context, esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
			return nil, nil
		},
		GetSecretMapFn: func(context.Context, esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
			return nil, nil
		},
		GetAllSecretsFn: func(context.Context, esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
			return nil, nil
		},
	}

	v.NewFn = func(context.Context, esv1beta1.GenericStore, client.Client, string) (esv1beta1.SecretsClient, error) {
		return v, nil
	}

	return v
}

// RegisterAs registers the fake client in the schema.
func (v *Client) RegisterAs(provider *esv1beta1.SecretStoreProvider) {
	esv1beta1.ForceRegister(v, provider)
}

// GetAllSecrets implements the provider.Provider interface.
func (v *Client) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	return v.GetAllSecretsFn(ctx, ref)
}

// GetSecret implements the provider.Provider interface.
func (v *Client) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	return v.GetSecretFn(ctx, ref)
}

// WithGetSecret wraps secret data returned by this provider.
func (v *Client) WithGetSecret(secData []byte, err error) *Client {
	v.GetSecretFn = func(context.Context, esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
		return secData, err
	}
	return v
}

// GetSecretMap implements the provider.Provider interface.
func (v *Client) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return v.GetSecretMapFn(ctx, ref)
}

func (v *Client) Close(ctx context.Context) error {
	return nil
}

func (v *Client) Validate() (esv1beta1.ValidationResult, error) {
	return esv1beta1.ValidationResultReady, nil
}

func (v *Client) ValidateStore(store esv1beta1.GenericStore) error {
	return nil
}

// WithGetSecretMap wraps the secret data map returned by this fake provider.
func (v *Client) WithGetSecretMap(secData map[string][]byte, err error) *Client {
	v.GetSecretMapFn = func(context.Context, esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
		return secData, err
	}
	return v
}

// WithGetAllSecrets wraps the secret data map returned by this fake provider.
func (v *Client) WithGetAllSecrets(secData map[string][]byte, err error) *Client {
	v.GetAllSecretsFn = func(context.Context, esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
		return secData, err
	}
	return v
}

// WithNew wraps the fake provider factory function.
func (v *Client) WithNew(f func(context.Context, esv1beta1.GenericStore, client.Client,
	string) (esv1beta1.SecretsClient, error)) *Client {
	v.NewFn = f
	return v
}

// NewClient returns a new fake provider.
func (v *Client) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string) (esv1beta1.SecretsClient, error) {
	c, err := v.NewFn(ctx, store, kube, namespace)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (v *Client) Reset() {
	v.WithNew(func(context.Context, esv1beta1.GenericStore, client.Client,
		string) (esv1beta1.SecretsClient, error) {
		return v, nil
	})
}
