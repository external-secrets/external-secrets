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
	"regexp"
	"sync"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

var _ esv1beta1.Provider = &Client{}

type PushSecretCallArgs struct {
	Value     []byte
	RemoteRef esv1beta1.PushSecretRemoteRef
}

// Client is a fake client for testing.
type Client struct {
	// WARNING: only interact with this map using the provided methods so that this is thread-safe.
	pushedSecrets     map[string]PushSecretCallArgs
	pushedSecretsLock sync.RWMutex

	GetSecretFn     func(context.Context, esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error)
	GetSecretMapFn  func(context.Context, esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error)
	GetAllSecretsFn func(context.Context, esv1beta1.ExternalSecretFind) (map[string][]byte, error)
	SecretExistsFn  func(context.Context, esv1beta1.PushSecretRemoteRef) (bool, error)
	SetSecretFn     func(ctx context.Context, secret *corev1.Secret, data esv1beta1.PushSecretData) error
	DeleteSecretFn  func(ctx context.Context, ref esv1beta1.PushSecretRemoteRef) error

	// NewFn returns the fake client as a SecretsClient interface.
	NewFn func(context.Context, esv1beta1.GenericStore, client.Client, string) (esv1beta1.SecretsClient, error)
}

// New returns a fake provider/client.
func New() *Client {
	v := &Client{}
	v.Reset()
	return v
}

// GetAllSecrets is used to retrieve data for `spec.dataFrom[].find` fields.
func (v *Client) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	return v.GetAllSecretsFn(ctx, ref)
}

// WithGetAllSecrets wraps the secret data map returned by this fake provider.
func (v *Client) WithGetAllSecrets(secData map[string][]byte, err error) *Client {
	v.GetAllSecretsFn = func(context.Context, esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
		return secData, err
	}
	return v
}

func (v *Client) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1beta1.PushSecretData) error {
	v.StorePushedSecret(data.GetRemoteKey(), PushSecretCallArgs{
		Value:     secret.Data[data.GetSecretKey()],
		RemoteRef: data,
	})
	return v.SetSecretFn(ctx, secret, data)
}

// WithSetSecret wraps the secret response to the fake provider.
func (v *Client) WithSetSecret(err error) *Client {
	v.SetSecretFn = func(ctx context.Context, secret *corev1.Secret, data esv1beta1.PushSecretData) error {
		return err
	}
	return v
}

// ClearPushedSecrets clears the pushed secrets map.
func (v *Client) ClearPushedSecrets() {
	v.pushedSecretsLock.Lock()
	defer v.pushedSecretsLock.Unlock()
	v.pushedSecrets = make(map[string]PushSecretCallArgs)
}

// LoadPushedSecret returns the pushed secret for the given remote key.
func (v *Client) LoadPushedSecret(remoteKey string) (PushSecretCallArgs, bool) {
	v.pushedSecretsLock.RLock()
	defer v.pushedSecretsLock.RUnlock()
	val, ok := v.pushedSecrets[remoteKey]
	return val, ok
}

// StorePushedSecret stores the pushed secret for the given remote key.
func (v *Client) StorePushedSecret(remoteKey string, val PushSecretCallArgs) {
	v.pushedSecretsLock.Lock()
	defer v.pushedSecretsLock.Unlock()
	v.pushedSecrets[remoteKey] = val
}

func (v *Client) DeleteSecret(ctx context.Context, ref esv1beta1.PushSecretRemoteRef) error {
	return v.DeleteSecretFn(ctx, ref)
}

func (v *Client) SecretExists(ctx context.Context, ref esv1beta1.PushSecretRemoteRef) (bool, error) {
	return v.SecretExistsFn(ctx, ref)
}

// GetSecret is used to retrieve data for `spec.data[]` fields.
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

// GetSecretMap is used to retrieve data for `spec.dataFrom[].extract` fields.
func (v *Client) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return v.GetSecretMapFn(ctx, ref)
}

// WithGetSecretMap wraps the secret data map returned by this fake provider.
func (v *Client) WithGetSecretMap(secData map[string][]byte, err error) *Client {
	v.GetSecretMapFn = func(context.Context, esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
		return secData, err
	}
	return v
}

func (v *Client) Close(_ context.Context) error {
	return nil
}

func (v *Client) Validate() (esv1beta1.ValidationResult, error) {
	return esv1beta1.ValidationResultReady, nil
}

func (v *Client) ValidateStore(_ esv1beta1.GenericStore) (admission.Warnings, error) {
	return nil, nil
}

// WithNew wraps the fake provider factory function.
func (v *Client) WithNew(f func(context.Context, esv1beta1.GenericStore, client.Client, string) (esv1beta1.SecretsClient, error)) *Client {
	v.NewFn = f
	return v
}

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (v *Client) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadOnly
}

// NewClient returns a new fake provider.
func (v *Client) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string) (esv1beta1.SecretsClient, error) {
	c, err := v.NewFn(ctx, store, kube, namespace)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// Reset the fake provider.
func (v *Client) Reset() {
	// Reset the internal state.
	v.ClearPushedSecrets()

	// Reset all functions to their default values.
	v.GetSecretFn = defaultGetSecretFn
	v.GetSecretMapFn = defaultGetSecretMapFn
	v.GetAllSecretsFn = defaultGetAllSecretsFn
	v.SecretExistsFn = defaultSecretExistsFn
	v.SetSecretFn = defaultSetSecretFn
	v.DeleteSecretFn = defaultDeleteSecretFn
	v.NewFn = v.defaultNewFn
}

// ResetWithStructuredData resets the provider to return values like a structured provider (e.g. one which stores JSON data in each remoteKey).
// providerData is structured: {remoteKey: {secretProperty: secretValue}}.
func (v *Client) ResetWithStructuredData(providerData map[string]map[string][]byte) {
	// Reset the internal state.
	v.Reset()

	// NOTE: this is used by `spec.data[]` fields
	v.GetSecretFn = func(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
		if data, ok := providerData[ref.Key]; ok {
			if val, ok := data[ref.Property]; ok {
				return val, nil
			}
		}
		return nil, esv1beta1.NoSecretErr
	}

	// NOTE: this is used by `spec.dataFrom[].extract` fields
	v.GetSecretMapFn = func(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
		if data, ok := providerData[ref.Key]; ok {
			secretMap := make(map[string][]byte)
			if ref.Property != "" {
				// if a property is specified, only return that property
				if val, ok := data[ref.Property]; ok {
					secretMap[ref.Property] = val
					return secretMap, nil
				}
			} else {
				// if no property is specified, return all properties
				for property, value := range data {
					secretMap[property] = value
				}
				return secretMap, nil
			}
		}
		return nil, esv1beta1.NoSecretErr
	}

	// NOTE: this is used by `spec.dataFrom[].find` fields
	v.GetAllSecretsFn = func(ctx context.Context, find esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
		secretMap := make(map[string][]byte)
		for remoteKey, dataMap := range providerData {
			m, err := regexp.MatchString(find.Name.RegExp, remoteKey)
			if err != nil {
				return nil, err
			}
			if m {
				// dataFrom[].find returns a JSON serialized version of the data in the remote key
				// NOTE: because go base64 encodes []bytes, we first convert the inner map to a map[string]string
				out := make(map[string]string)
				for k, v := range dataMap {
					out[k] = string(v)
				}
				jsonData, err := utils.JSONMarshal(out)
				if err != nil {
					return nil, err
				}
				secretMap[remoteKey] = jsonData
			}
		}
		if len(secretMap) == 0 {
			return nil, esv1beta1.NoSecretErr
		}
		return secretMap, nil
	}
}

func (v *Client) defaultNewFn(context.Context, esv1beta1.GenericStore, client.Client, string) (esv1beta1.SecretsClient, error) {
	return v, nil
}

func defaultGetSecretFn(context.Context, esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	return nil, nil
}

func defaultGetSecretMapFn(context.Context, esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return nil, nil
}

func defaultGetAllSecretsFn(context.Context, esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, nil
}

func defaultSecretExistsFn(context.Context, esv1beta1.PushSecretRemoteRef) (bool, error) {
	return false, nil
}

func defaultSetSecretFn(context.Context, *corev1.Secret, esv1beta1.PushSecretData) error {
	return nil
}

func defaultDeleteSecretFn(context.Context, esv1beta1.PushSecretRemoteRef) error {
	return nil
}
