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

package conjur

import (
	"context"
	"strings"
	"testing"

	"github.com/cyberark/conjur-api-go/conjurapi"
	"github.com/doodlesbykumbi/conjur-policy-go/pkg/conjurpolicy"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/external-secrets/external-secrets/providers/v1/conjur/fake"
	"github.com/external-secrets/external-secrets/runtime/esutils"
)

func TestDefaultPolicy(t *testing.T) {
	policy, err := conjurPolicy("secret1", []string{"foo", "bar", "baz"})
	assert.NoError(t, err)
	expected := `
- !policy
  id: secret1
  body:
  - !group
    id: delegation/consumers
    annotations:
      managed-by: "external-secrets"
      editable: "true"
  - !variable
    id: foo
    annotations:
      managed-by: "external-secrets"
  - !variable
    id: bar
    annotations:
      managed-by: "external-secrets"
  - !variable
    id: baz
    annotations:
      managed-by: "external-secrets"
  - !permit
    resource: !variable foo
    role: !group delegation/consumers
    privileges: [ read, execute ]
  - !permit
    resource: !variable bar
    role: !group delegation/consumers
    privileges: [ read, execute ]
  - !permit
    resource: !variable baz
    role: !group delegation/consumers
    privileges: [ read, execute ]`

	// roundtrip the expected output through a unmarshal/marshal to remove any formatting related issues
	p := conjurpolicy.PolicyStatements{}
	err = yaml.Unmarshal([]byte(expected), &p)
	assert.NoError(t, err)

	exp, err := yaml.Marshal(p)
	assert.NoError(t, err)

	assert.Equal(t, string(exp), policy)
}

type RemoteRef struct {
	RemoteKey string
	Property  string
	SecretKey string
}

func (r RemoteRef) GetRemoteKey() string {
	return r.RemoteKey
}

func (r RemoteRef) GetProperty() string {
	return r.Property
}

func (r RemoteRef) GetMetadata() *apiextensionsv1.JSON {
	return nil
}

func (r RemoteRef) GetSecretKey() string {
	return r.SecretKey
}

func TestPushSecret(t *testing.T) {
	tests := []struct {
		name           string
		secretValue    []byte
		remoteRef      RemoteRef
		expectedPolicy string // Partial match for the YAML policy
		expectedVar    string
		expectedVal    string
	}{
		{
			name:        "Push specified value to property",
			secretValue: []byte("password123"),
			remoteRef: RemoteRef{
				SecretKey: "password",
				RemoteKey: "data/vault/eso/db",
				Property:  "password",
			},
			expectedVar: "data/vault/eso/db/password",
			expectedVal: "password123",
		},
		{
			name:        "Push all values to a single property",
			secretValue: []byte("password123"),
			remoteRef: RemoteRef{
				RemoteKey: "data/vault/eso/db",
				Property:  "password",
			},
			expectedVar: "data/vault/eso/db/password",
			expectedVal: `{"password":"password123"}`,
		},
		{
			name:        "Push all values, unspecified property",
			secretValue: []byte("password123"),
			remoteRef: RemoteRef{
				RemoteKey: "data/vault/eso/db",
			},
			expectedVar: "data/vault/eso/db/password",
			expectedVal: "password123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &fake.ConjurMockClient{}
			provider := &Client{
				store: makeAPIKeySecretStore(svcURL, "conjur-hostid", "conjur-apikey", "myconjuraccount"),
				kube: clientfake.NewClientBuilder().
					WithObjects(makeFakeAPIKeySecrets()...).Build(),
				namespace: "default",
				client:    mockClient,
			}

			kubeSecret := &corev1.Secret{
				Data: map[string][]byte{
					"password": tt.secretValue,
				},
			}

			err := provider.PushSecret(context.Background(), kubeSecret, tt.remoteRef)

			if err != nil {
				t.Fatalf("PushSecret failed: %v", err)
			}

			if len(mockClient.AddSecretCalls) != 1 {
				t.Errorf("expected 1 AddSecret call, got %d", len(mockClient.AddSecretCalls))
			} else {
				call := mockClient.AddSecretCalls[0]
				if call.Variable != tt.expectedVar {
					t.Errorf("expected var %s, got %s", tt.expectedVar, call.Variable)
				}
				if call.Value != tt.expectedVal {
					t.Errorf("expected value %s, got %s", tt.expectedVal, call.Value)
				}
			}

			if tt.expectedPolicy != "" {
				if len(mockClient.LoadPolicyCalls) == 0 {
					t.Error("expected a LoadPolicy call but none occurred")
				} else {
					policy := mockClient.LoadPolicyCalls[0].Policy
					if !strings.Contains(policy, tt.expectedPolicy) {
						t.Errorf("policy missing expected string. Got: %s", policy)
					}
				}
			}
		})
	}
}

func TestCheckSecrets(t *testing.T) {
	tests := []struct {
		name             string
		conjurSecretName string
		conjurVars       []string
		secretData       map[string]string
		property         string
		key              string
		setupMock        func(m *fake.ConjurMockClient)
		expectedUpdates  []string
	}{
		{
			name:             "Unmanaged secret (should skip despite drift)",
			conjurSecretName: "db",
			conjurVars:       []string{"pass"},
			secretData:       map[string]string{"pass": "new-value"},
			setupMock: func(m *fake.ConjurMockClient) {
				m.SecretDetails = map[string]*conjurapi.StaticSecretResponse{
					"db/pass": {
						StaticSecret: conjurapi.StaticSecret{
							Annotations: map[string]string{"owner": "manual"},
						},
					},
				}
				m.SecretValues = map[string][]byte{"db/pass": []byte("old-value")}
			},
			expectedUpdates: []string{},
		},
		{
			name:             "Managed secret with drift (should update)",
			conjurSecretName: "db",
			conjurVars:       []string{"pass"},
			secretData:       map[string]string{"pass": "new-value"},
			setupMock: func(m *fake.ConjurMockClient) {
				m.SecretValues = map[string][]byte{"db/pass": []byte("old-value")}
			},
			expectedUpdates: []string{"pass"},
		},
		{
			name:             "Managed secret, no drift (should skip)",
			conjurSecretName: "db",
			conjurVars:       []string{"pass"},
			secretData:       map[string]string{"pass": "same-value"},
			setupMock: func(m *fake.ConjurMockClient) {
				m.SecretValues = map[string][]byte{"db/pass": []byte("same-value")}
			},
			expectedUpdates: []string{},
		},
		{
			name:             "Property + Key: Value differs (should update)",
			conjurSecretName: "api",
			conjurVars:       []string{"token"},
			property:         "credential",
			key:              "apikey",
			secretData:       map[string]string{"apikey": "new-val"},
			setupMock: func(m *fake.ConjurMockClient) {
				m.SecretValues = map[string][]byte{"api/token": []byte("old-val")}
			},
			expectedUpdates: []string{"token"},
		},
		{
			name:             "Property + Key: Value matches (should skip)",
			conjurSecretName: "api",
			conjurVars:       []string{"token"},
			property:         "credential",
			key:              "apikey",
			secretData:       map[string]string{"apikey": "same-val"},
			setupMock: func(m *fake.ConjurMockClient) {
				m.SecretValues = map[string][]byte{"api/token": []byte("same-val")}
			},
			expectedUpdates: []string{},
		},
		{
			name:             "Property only (JSON): Matches marshaled data (should skip)",
			conjurSecretName: "app",
			conjurVars:       []string{"config"},
			property:         "json-blob",
			key:              "",
			secretData:       map[string]string{"user": "admin", "port": "8080"},
			setupMock: func(m *fake.ConjurMockClient) {
				marshaled, _ := esutils.JSONMarshal(map[string]string{"user": "admin", "port": "8080"})
				m.SecretValues = map[string][]byte{"app/config": marshaled}
			},
			expectedUpdates: []string{},
		},
		{
			name:             "Property only (JSON): Data differs (should update)",
			conjurSecretName: "app",
			conjurVars:       []string{"config"},
			property:         "json-blob",
			key:              "",
			secretData:       map[string]string{"user": "admin"},
			setupMock: func(m *fake.ConjurMockClient) {
				m.SecretValues = map[string][]byte{"app/config": []byte(`{"user":"old"}`)}
			},
			expectedUpdates: []string{"config"},
		},
		{
			name:             "Key-only (Flat): Value matches (should skip)",
			conjurSecretName: "db",
			conjurVars:       []string{"username"},
			property:         "",
			secretData:       map[string]string{"username": "admin"},
			setupMock: func(m *fake.ConjurMockClient) {
				m.SecretValues = map[string][]byte{"db/username": []byte("admin")}
			},
			expectedUpdates: []string{},
		},
		{
			name:             "Key-only (Flat): Value differs (should update)",
			conjurSecretName: "db",
			conjurVars:       []string{"username"},
			property:         "",
			secretData:       map[string]string{"username": "new-admin"},
			setupMock: func(m *fake.ConjurMockClient) {
				m.SecretValues = map[string][]byte{"db/username": []byte("old-admin")}
			},
			expectedUpdates: []string{"username"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &fake.ConjurMockClient{
				SecretDetails: make(map[string]*conjurapi.StaticSecretResponse),
				SecretValues:  make(map[string][]byte),
			}
			tt.setupMock(mock)

			updates, err := checkSecrets(mock, tt.conjurSecretName, tt.conjurVars, tt.secretData, tt.property, tt.key)

			assert.NoError(t, err)

			assert.Equal(t, tt.expectedUpdates, updates)
		})
	}
}
