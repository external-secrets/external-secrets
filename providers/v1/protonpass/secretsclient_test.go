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

package protonpass

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/providers/v1/protonpass/internal/client"
)

type mockClient struct {
	item map[string][]byte
	err  error
}

func (m *mockClient) GetItem(_ context.Context, _ string) (map[string][]byte, error) {
	return m.item, m.err
}

func (m *mockClient) Validate(_ context.Context) error {
	return m.err
}

func TestGetSecret(t *testing.T) {
	sc := &Client{client: &mockClient{item: map[string][]byte{
		"title":    []byte("My Login"),
		"username": []byte("alice"),
		"password": []byte("p@ss"),
	}}}

	// Default property is password.
	val, err := sc.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "My Login"})
	require.NoError(t, err)
	assert.Equal(t, []byte("p@ss"), val)

	// Explicit property.
	val, err = sc.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "My Login", Property: "username"})
	require.NoError(t, err)
	assert.Equal(t, []byte("alice"), val)

	// Missing property errors.
	_, err = sc.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "My Login", Property: "nope"})
	assert.Error(t, err)
}

func TestGetSecretNoSecretErr(t *testing.T) {
	sc := &Client{client: &mockClient{err: esv1.NoSecretErr}}
	_, err := sc.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "missing"})
	assert.ErrorIs(t, err, esv1.NoSecretErr)
}

func TestGetSecretMap(t *testing.T) {
	sc := &Client{client: &mockClient{item: map[string][]byte{
		"username": []byte("alice"),
		"password": []byte("p@ss"),
	}}}
	got, err := sc.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "My Login"})
	require.NoError(t, err)
	assert.Equal(t, []byte("alice"), got["username"])

	sc = &Client{client: &mockClient{err: esv1.NoSecretErr}}
	_, err = sc.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "missing"})
	assert.ErrorIs(t, err, esv1.NoSecretErr)
}

func TestValidate(t *testing.T) {
	sc := &Client{client: &mockClient{}}
	result, err := sc.Validate()
	require.NoError(t, err)
	assert.Equal(t, esv1.ValidationResultReady, result)

	sc = &Client{client: &mockClient{err: errors.New("auth failed")}}
	result, err = sc.Validate()
	assert.Error(t, err)
	assert.Equal(t, esv1.ValidationResultError, result)
}

func TestNotSupportedMethods(t *testing.T) {
	sc := &Client{client: &mockClient{}}
	ctx := context.Background()

	err := sc.PushSecret(ctx, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not support")

	err = sc.DeleteSecret(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not support")

	_, err = sc.SecretExists(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not support")

	_, err = sc.GetAllSecrets(ctx, esv1.ExternalSecretFind{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not support")

	assert.NoError(t, sc.Close(ctx))
}

func TestGetSecretMissingProperty(t *testing.T) {
	sc := &Client{client: &mockClient{item: map[string][]byte{
		"title":    []byte("My Login"),
		"password": []byte("p@ss"),
	}}}
	_, err := sc.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "My Login", Property: "api_key"})
	assert.Error(t, err)
}

func TestGetSecretDefaultsToPassword(t *testing.T) {
	sc := &Client{client: &mockClient{item: map[string][]byte{
		"password": []byte("p@ss"),
	}}}
	val, err := sc.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "My Login"})
	require.NoError(t, err)
	assert.Equal(t, []byte("p@ss"), val)
}

var _ client.Interface = &mockClient{}
