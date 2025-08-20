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
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeClient "sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
)

func TestProviderCapabilities(t *testing.T) {
	provider := &Provider{}
	capabilities := provider.Capabilities()

	assert.NotNil(t, capabilities)
	assert.Equal(t, esv1.SecretStoreWriteOnly, capabilities)
}

func TestProviderValidateStore(t *testing.T) {
	type expected struct {
		err      error
		warnings admission.Warnings
	}
	type testCase struct {
		name     string
		store    esv1.GenericStore
		expected expected
	}
	testCases := []testCase{
		{
			name:  "nil store",
			store: nil,
			expected: expected{
				err:      errInvalidStore,
				warnings: nil,
			},
		},
		{
			name: "nil provider",
			store: &esv1.SecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: "SecretStore",
				},
				Spec: esv1.SecretStoreSpec{
					Provider: nil,
				},
			},
			expected: expected{
				err:      errInvalidStoreProv,
				warnings: nil,
			},
		},
		{
			name: "invalid ngrok provider",
			store: &esv1.SecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: "SecretStore",
				},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Ngrok: nil,
					},
				},
			},
			expected: expected{
				err:      errInvalidNgrokProv,
				warnings: nil,
			},
		},
		{
			name: "invalid API URL",
			store: &esv1.SecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: "SecretStore",
				},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Ngrok: &esv1.NgrokProvider{
							APIURL: "http://example.com/path\n",
						},
					},
				},
			},
			expected: expected{
				err:      errInvalidAPIURL,
				warnings: nil,
			},
		},
		{
			name: "valid store",
			store: &esv1.SecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: "SecretStore",
				},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Ngrok: &esv1.NgrokProvider{
							Vault: esv1.NgrokVault{
								Name: "test-vault",
							},
							Auth: esv1.NgrokAuth{
								APIKey: &esv1.NgrokProviderSecretRef{
									SecretRef: &v1.SecretKeySelector{
										Key:  "apiKey",
										Name: "ngrok-credentials",
									},
								},
							},
						},
					},
				},
			},
			expected: expected{
				err:      nil,
				warnings: nil,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			provider := &Provider{}

			warnings, err := provider.ValidateStore(tc.store)
			assert.Equal(t, tc.expected.warnings, warnings)

			if tc.expected.err != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tc.expected.err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProviderNewClient(t *testing.T) {
	ngrokCredentials := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ngrok-credentials",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"API_KEY": []byte("secret-api-key"),
		},
	}

	type testCase struct {
		name       string
		store      esv1.GenericStore
		kubeClient kubeClient.Client
		namespace  string

		err error
	}

	testCases := []testCase{
		{
			name:       "valid store",
			namespace:  "default",
			kubeClient: clientfake.NewClientBuilder().WithObjects(ngrokCredentials).Build(),
			store: &esv1.SecretStore{
				TypeMeta: metav1.TypeMeta{
					Kind: "SecretStore",
				},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						Ngrok: &esv1.NgrokProvider{
							Auth: esv1.NgrokAuth{
								APIKey: &esv1.NgrokProviderSecretRef{
									SecretRef: &v1.SecretKeySelector{
										Key:  "API_KEY",
										Name: "ngrok-credentials",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			provider := &Provider{}
			client, err := provider.NewClient(t.Context(), tc.store, tc.kubeClient, tc.namespace)

			if tc.err != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tc.err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, client)
		})
	}
}
