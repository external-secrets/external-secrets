/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or impliec.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ngrok

import (
	"context"
	"errors"
	"testing"

	"github.com/ngrok/ngrok-api-go/v7"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/provider/ngrok/fake"
)

type pushSecretRemoteRef struct {
	remoteKey string
	property  string
}

func (p pushSecretRemoteRef) GetRemoteKey() string {
	return p.remoteKey
}
func (p pushSecretRemoteRef) GetProperty() string {
	return p.property
}

type testClientOpts struct {
	vaults         []*ngrok.Vault
	secrets        []*ngrok.Secret
	secretsListErr error
	vaultName      string
}

type testClientOpt func(opts *testClientOpts)

func WithVaults(vaults ...*ngrok.Vault) testClientOpt {
	return func(opts *testClientOpts) {
		opts.vaults = vaults
	}
}

func WithSecrets(secrets ...*ngrok.Secret) testClientOpt {
	return func(opts *testClientOpts) {
		opts.secrets = secrets
	}
}

func WithSecretsListError(err error) testClientOpt {
	return func(opts *testClientOpts) {
		opts.secretsListErr = err
	}
}

func WithVaultName(vaultName string) testClientOpt {
	return func(opts *testClientOpts) {
		opts.vaultName = vaultName
	}
}

func newTestClient(opts ...testClientOpt) *client {
	o := &testClientOpts{
		vaults:         []*ngrok.Vault{},
		secrets:        []*ngrok.Secret{},
		secretsListErr: nil,
		vaultName:      "vault_" + fake.GenerateRandomString(20),
	}

	for _, opt := range opts {
		opt(o)
	}

	return &client{
		vaultClient:   fake.NewVaultClient(o.vaults...),
		secretsClient: fake.NewSecretsClient(o.secrets...).WithListError(o.secretsListErr),
		vaultName:     o.vaultName,
	}
}

func TestClientImplementsClientInterface(t *testing.T) {
	client := newTestClient()
	assert.Implements(t, (*esv1.SecretsClient)(nil), client)
}

func TestClientGetSecret(t *testing.T) {
	client := newTestClient()

	secret, err := client.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{})
	assert.Error(t, err)
	assert.Nil(t, secret)
	assert.ErrorIs(t, err, errWriteOnlyOperations)
}

func TestClientGetSecretMap(t *testing.T) {
	client := newTestClient()

	secretMap, err := client.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{})
	assert.Error(t, err)
	assert.Nil(t, secretMap)
	assert.ErrorIs(t, err, errWriteOnlyOperations)
}

func TestClientGetAllSecrets(t *testing.T) {
	client := newTestClient()

	allSecrets, err := client.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{})
	assert.Error(t, err)
	assert.Nil(t, allSecrets)
	assert.ErrorIs(t, err, errWriteOnlyOperations)
}

func TestClientDeleteSecret(t *testing.T) {
	type testCase struct {
		name            string
		clientVaultName string
		vaults          []*ngrok.Vault
		secrets         []*ngrok.Secret
		ref             esv1.PushSecretRemoteRef
		err             error
	}
	tests := []testCase{
		{
			name: "when the vault does not exist",
			ref: pushSecretRemoteRef{
				remoteKey: "nonexistent-secret",
			},
			err: errVaultDoesNotExist,
		},
		{
			name: "when the vault exists but the secret does not",
			ref: pushSecretRemoteRef{
				remoteKey: "nonexistent-secret",
			},
			clientVaultName: "existing-vault",
			vaults: []*ngrok.Vault{
				{
					ID:   "vault_1",
					Name: "existing-vault",
				},
			},
			err: errVaultSecretDoesNotExist,
		},
		{
			name: "when the vault and secret both exist",
			ref: pushSecretRemoteRef{
				remoteKey: "i-exist",
			},
			clientVaultName: "existing-vault",
			vaults: []*ngrok.Vault{
				{
					ID:   "vault_1",
					Name: "existing-vault",
				},
			},
			secrets: []*ngrok.Secret{
				{
					ID:    "secret_1",
					Vault: ngrok.Ref{ID: "vault_1", URI: "vaults/vault_1"},
					Name:  "i-exist",
				},
			},
			err: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			opts := []testClientOpt{}
			if tc.clientVaultName != "" {
				opts = append(opts, WithVaultName(tc.clientVaultName))
			}
			if len(tc.vaults) > 0 {
				opts = append(opts, WithVaults(tc.vaults...))
			}
			if len(tc.secrets) > 0 {
				opts = append(opts, WithSecrets(tc.secrets...))
			}
			client := newTestClient(opts...)

			err := client.DeleteSecret(t.Context(), tc.ref)
			if tc.err == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tc.err)
			}
		})
	}
}

func TestClientSecretExists(t *testing.T) {
	type testCase struct {
		name            string
		clientVaultName string
		vaults          []*ngrok.Vault
		secrets         []*ngrok.Secret
		ref             esv1.PushSecretRemoteRef
		exists          bool
	}
	tests := []testCase{
		{
			name: "when the vault does not exist",
			ref: pushSecretRemoteRef{
				remoteKey: "nonexistent-secret",
			},
			exists: false,
		},
		{
			name: "when the vault exists but the secret does not",
			ref: pushSecretRemoteRef{
				remoteKey: "nonexistent-secret",
			},
			clientVaultName: "existing-vault",
			vaults: []*ngrok.Vault{
				{
					ID:   "vault_1",
					Name: "existing-vault",
				},
			},
			exists: false,
		},
		{
			name: "when the vault and secret both exist",
			ref: pushSecretRemoteRef{
				remoteKey: "i-exist",
			},
			clientVaultName: "existing-vault",
			vaults: []*ngrok.Vault{
				{
					ID:   "vault_1",
					Name: "existing-vault",
				},
			},
			secrets: []*ngrok.Secret{
				{
					ID:    "secret_1",
					Vault: ngrok.Ref{ID: "vault_1", URI: "vaults/vault_1"},
					Name:  "i-exist",
				},
			},
			exists: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			opts := []testClientOpt{}
			if tc.clientVaultName != "" {
				opts = append(opts, WithVaultName(tc.clientVaultName))
			}
			if len(tc.vaults) > 0 {
				opts = append(opts, WithVaults(tc.vaults...))
			}
			if len(tc.secrets) > 0 {
				opts = append(opts, WithSecrets(tc.secrets...))
			}
			client := newTestClient(opts...)

			exists, err := client.SecretExists(t.Context(), tc.ref)
			assert.NoError(t, err)
			assert.Equal(t, tc.exists, exists)
		})
	}
}

func TestClientPushSecret(t *testing.T) {
	type testCase struct {
		name      string
		vaultName string
		secret    *corev1.Secret
		data      esv1.PushSecretData
		err       error
		validate  func(t *testing.T, client *client)
	}
	testCases := []testCase{
		{
			name:   "when the secret is nil",
			secret: nil,
			data: v1alpha1.PushSecretData{
				Match:    v1alpha1.PushSecretMatch{},
				Metadata: nil,
			},
			err: errCannotPushNilSecret,
		},
		{
			name:      "it creates a vault if it does not exist",
			vaultName: "should-create-vault",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-secret",
				},
				Data: map[string][]byte{
					"key": []byte("value"),
				},
			},
			data: v1alpha1.PushSecretData{},
			err:  nil,
			validate: func(t *testing.T, client *client) {
				iter := client.vaultClient.List(nil)
				for iter.Next(t.Context()) {
					if iter.Err() != nil {
						t.Fatalf("failed to list vaults: %v", iter.Err())
					}

					// We should expect the vault to be created
					vault := iter.Item()
					assert.Equal(t, "should-create-vault", vault.Name)
				}
			},
		},
		{
			name:      "it pushes a secret to an existing vault",
			vaultName: "existing-vault",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-secret",
				},
				Data: map[string][]byte{
					"key": []byte("value"),
				},
			},
			data: v1alpha1.PushSecretData{
				Match: v1alpha1.PushSecretMatch{
					SecretKey: "key",
					RemoteRef: v1alpha1.PushSecretRemoteRef{
						RemoteKey: "my-ngrok-secret",
					},
				},
			},
			err: nil,
			validate: func(t *testing.T, client *client) {
				// Check if the secret was created in the vault
				secret, err := client.getSecretByVaultNameAndSecretName(t.Context(), "existing-vault", "my-ngrok-secret")
				assert.NoError(t, err)
				assert.NotNil(t, secret)
				assert.Equal(t, defaultDescription, secret.Description)
				assert.Equal(t, `{"_sha256":"cd42404d52ad55ccfa9aca4adc828aa5800ad9d385a0671fbcbf724118320619"}`, secret.Metadata)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := []testClientOpt{}
			if tc.vaultName != "" {
				opts = append(opts, WithVaultName(tc.vaultName))
			}
			client := newTestClient(opts...)

			err := client.PushSecret(t.Context(), tc.secret, tc.data)

			if tc.err == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tc.err)
			}

			if tc.validate != nil {
				tc.validate(t, client)
			}
		})
	}
}

func TestClientValidate(t *testing.T) {
	errListingSecrets := errors.New("failed to list secrets")

	type testCase struct {
		name    string
		secrets []*ngrok.Secret
		listErr error

		result esv1.ValidationResult
		err    error
	}
	testCases := []testCase{
		{
			name:    "valid client, no secrets",
			secrets: []*ngrok.Secret{},
			listErr: nil,
			result:  esv1.ValidationResultReady,
			err:     nil,
		},
		{
			name: "valid client, with secrets",
			secrets: []*ngrok.Secret{
				{
					ID: "secret_" + fake.GenerateRandomString(20),
					Vault: ngrok.Ref{
						ID:  "vault_" + fake.GenerateRandomString(20),
						URI: "vaults/vault_" + fake.GenerateRandomString(20),
					},
					Name: "my-secret",
				},
			},
			listErr: nil,
			result:  esv1.ValidationResultReady,
			err:     nil,
		},
		{
			name:    "error listing secrets",
			secrets: nil,
			listErr: errListingSecrets,
			result:  esv1.ValidationResultError,
			err:     errListingSecrets,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := []testClientOpt{
				WithSecretsListError(tc.listErr),
			}
			if len(tc.secrets) > 0 {
				opts = append(opts, WithSecrets(tc.secrets...))
			}
			client := newTestClient(opts...)

			result, err := client.Validate()
			assert.Equal(t, result, tc.result)
			if tc.err != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tc.err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
