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

package sapcredentialstore

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

func makeTestStore(serviceURL, namespace string, auth esv1.SAPCSAuth) *esv1.SecretStore {
	return &esv1.SecretStore{
		TypeMeta: metav1.TypeMeta{Kind: esv1.SecretStoreKind},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				SAPCredentialStore: &esv1.SAPCredentialStoreProvider{
					ServiceURL: serviceURL,
					Namespace:  namespace,
					Auth:       auth,
				},
			},
		},
	}
}

var validOAuth2Auth = esv1.SAPCSAuth{
	OAuth2: &esv1.SAPCSOAuth2Auth{
		TokenURL:     "https://auth.example.com/oauth/token",
		ClientID:     esmeta.SecretKeySelector{Name: "my-secret", Key: "client-id"},
		ClientSecret: esmeta.SecretKeySelector{Name: "my-secret", Key: "client-secret"},
	},
}

var validMTLSAuth = esv1.SAPCSAuth{
	MTLS: &esv1.SAPCSMTLSAuth{
		Certificate: esmeta.SecretKeySelector{Name: "my-cert", Key: "tls.crt"},
		PrivateKey:  esmeta.SecretKeySelector{Name: "my-cert", Key: "tls.key"},
	},
}

func TestValidateStore(t *testing.T) {
	p := Provider{}
	cases := []struct {
		name    string
		store   *esv1.SecretStore
		wantErr string
	}{
		{
			name:    "valid OAuth2",
			store:   makeTestStore("https://cred.example.com", "my-ns", validOAuth2Auth),
			wantErr: "",
		},
		{
			name:    "valid mTLS",
			store:   makeTestStore("https://cred.example.com", "my-ns", validMTLSAuth),
			wantErr: "",
		},
		{
			name:    "missing serviceURL",
			store:   makeTestStore("", "my-ns", validOAuth2Auth),
			wantErr: "serviceURL",
		},
		{
			name:    "missing namespace",
			store:   makeTestStore("https://cred.example.com", "", validOAuth2Auth),
			wantErr: "namespace",
		},
		{
			name:    "no auth mode set",
			store:   makeTestStore("https://cred.example.com", "my-ns", esv1.SAPCSAuth{}),
			wantErr: "auth",
		},
		{
			name: "both auth modes set",
			store: makeTestStore("https://cred.example.com", "my-ns", esv1.SAPCSAuth{
				OAuth2: validOAuth2Auth.OAuth2,
				MTLS:   validMTLSAuth.MTLS,
			}),
			wantErr: "auth",
		},
		{
			name: "OAuth2 missing tokenURL",
			store: makeTestStore("https://cred.example.com", "my-ns", esv1.SAPCSAuth{
				OAuth2: &esv1.SAPCSOAuth2Auth{
					ClientID:     esmeta.SecretKeySelector{Name: "s", Key: "id"},
					ClientSecret: esmeta.SecretKeySelector{Name: "s", Key: "secret"},
				},
			}),
			wantErr: "tokenURL",
		},
		{
			name: "OAuth2 missing clientID name",
			store: makeTestStore("https://cred.example.com", "my-ns", esv1.SAPCSAuth{
				OAuth2: &esv1.SAPCSOAuth2Auth{
					TokenURL:     "https://auth.example.com/oauth/token",
					ClientSecret: esmeta.SecretKeySelector{Name: "s", Key: "secret"},
				},
			}),
			wantErr: "clientId",
		},
		{
			name: "OAuth2 missing clientSecret name",
			store: makeTestStore("https://cred.example.com", "my-ns", esv1.SAPCSAuth{
				OAuth2: &esv1.SAPCSOAuth2Auth{
					TokenURL: "https://auth.example.com/oauth/token",
					ClientID: esmeta.SecretKeySelector{Name: "s", Key: "id"},
				},
			}),
			wantErr: "clientSecret",
		},
		{
			name: "mTLS missing certificate name",
			store: makeTestStore("https://cred.example.com", "my-ns", esv1.SAPCSAuth{
				MTLS: &esv1.SAPCSMTLSAuth{
					PrivateKey: esmeta.SecretKeySelector{Name: "s", Key: "tls.key"},
				},
			}),
			wantErr: "certificate",
		},
		{
			name: "mTLS missing privateKey name",
			store: makeTestStore("https://cred.example.com", "my-ns", esv1.SAPCSAuth{
				MTLS: &esv1.SAPCSMTLSAuth{
					Certificate: esmeta.SecretKeySelector{Name: "s", Key: "tls.crt"},
				},
			}),
			wantErr: "privateKey",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := p.ValidateStore(tc.store)
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			}
		})
	}
}

// --- US2: BTP Service Binding Secret tests (T012, T013) ---

func bindingSecret(ns, name, key, jsonData string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Data:       map[string][]byte{key: []byte(jsonData)},
	}
}

const validBindingJSON = `{
	"clientid":     "sb-my-client",
	"clientsecret": "super-secret",
	"url":          "https://credstore.example.com/api/v1",
	"tokenurl":     "https://auth.example.com/oauth/token"
}`

func makeBindingStore(bindingNS, bindingName, credKey string) *esv1.ClusterSecretStore {
	ref := &esv1.SAPCSServiceBindingRef{
		Name:           bindingName,
		Namespace:      bindingNS,
		CredentialsKey: credKey,
	}
	return &esv1.ClusterSecretStore{
		TypeMeta: metav1.TypeMeta{Kind: esv1.ClusterSecretStoreKind},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				SAPCredentialStore: &esv1.SAPCredentialStoreProvider{
					Namespace:               "default-ns",
					ServiceBindingSecretRef: ref,
				},
			},
		},
	}
}

// T012: ValidateStore with binding ref
func TestValidateStore_BindingRef(t *testing.T) {
	p := Provider{}
	cases := []struct {
		name     string
		store    esv1.GenericStore
		wantErr  string
		wantWarn bool
	}{
		{
			name:    "binding ref with valid secret → valid",
			store:   makeBindingStore("sap-bindings", "my-binding", "credentials"),
			wantErr: "",
		},
		{
			name: "binding ref with empty name → invalid",
			store: &esv1.ClusterSecretStore{
				TypeMeta: metav1.TypeMeta{Kind: esv1.ClusterSecretStoreKind},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						SAPCredentialStore: &esv1.SAPCredentialStoreProvider{
							Namespace: "default-ns",
							ServiceBindingSecretRef: &esv1.SAPCSServiceBindingRef{
								Name: "",
							},
						},
					},
				},
			},
			wantErr: "serviceBindingSecretRef.name is required",
		},
		{
			name: "both binding ref and inline auth → valid with warning",
			store: &esv1.ClusterSecretStore{
				TypeMeta: metav1.TypeMeta{Kind: esv1.ClusterSecretStoreKind},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						SAPCredentialStore: &esv1.SAPCredentialStoreProvider{
							ServiceURL: "https://credstore.example.com/api/v1",
							Namespace:  "default-ns",
							Auth:       validOAuth2Auth,
							ServiceBindingSecretRef: &esv1.SAPCSServiceBindingRef{
								Name:      "my-binding",
								Namespace: "sap-bindings",
							},
						},
					},
				},
			},
			wantErr:  "",
			wantWarn: true,
		},
		{
			name: "neither binding ref nor inline auth → invalid",
			store: &esv1.ClusterSecretStore{
				TypeMeta: metav1.TypeMeta{Kind: esv1.ClusterSecretStoreKind},
				Spec: esv1.SecretStoreSpec{
					Provider: &esv1.SecretStoreProvider{
						SAPCredentialStore: &esv1.SAPCredentialStoreProvider{
							ServiceURL: "https://credstore.example.com/api/v1",
							Namespace:  "default-ns",
							Auth:       esv1.SAPCSAuth{},
						},
					},
				},
			},
			wantErr: "auth",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			warnings, err := p.ValidateStore(tc.store)
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			}
			if tc.wantWarn {
				assert.NotEmpty(t, warnings)
			}
		})
	}
}

// T013: NewClient with binding ref
func TestNewClient_BindingRef(t *testing.T) {
	// Set up a token server that returns a minimal token response.
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
			"access_token": "test-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer tokenServer.Close()

	credServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer credServer.Close()

	bindingJSON := `{"clientid":"sb-my","clientsecret":"s3cr3t","url":"` + credServer.URL + `","tokenurl":"` + tokenServer.URL + `"}`

	t.Run("binding ref resolves credentials from Secret", func(t *testing.T) {
		secret := bindingSecret("sap-bindings", "my-binding", "credentials", bindingJSON)
		scheme := runtime.NewScheme()
		require.NoError(t, corev1.AddToScheme(scheme))
		kube := clientfake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()

		store := makeBindingStore("sap-bindings", "my-binding", "credentials")
		p := Provider{}
		c, err := p.NewClient(context.Background(), store, kube, "sap-bindings")
		require.NoError(t, err)
		assert.NotNil(t, c)
	})

	t.Run("binding secret not found returns error", func(t *testing.T) {
		scheme := runtime.NewScheme()
		require.NoError(t, corev1.AddToScheme(scheme))
		kube := clientfake.NewClientBuilder().WithScheme(scheme).Build()

		store := makeBindingStore("sap-bindings", "missing-binding", "credentials")
		p := Provider{}
		_, err := p.NewClient(context.Background(), store, kube, "sap-bindings")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}
