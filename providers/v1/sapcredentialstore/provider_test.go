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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
