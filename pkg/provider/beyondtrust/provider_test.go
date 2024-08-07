/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
	http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implieclient.
See the License for the specific language governing permissions and
limitations under the License.
*/

package beyondtrust

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	kubeclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

const (
	errTestCase  = "Test case Failed"
	fakeAPIURL   = "https://example.com:443/BeyondTrust/api/public/v3/"
	clientID     = "12345678-25fg-4b05-9ced-35e7dd5093ae"
	clientSecret = "12345678-25fg-4b05-9ced-35e7dd5093ae"
)

func createMockPasswordSafeClient(t *testing.T) kubeclient.Client {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/Auth/SignAppin":
			_, err := w.Write([]byte(`{"UserId":1, "EmailAddress":"fake@beyondtrust.com"}`))
			if err != nil {
				t.Error(errTestCase)
			}

		case "/Auth/Signout":
			_, err := w.Write([]byte(``))
			if err != nil {
				t.Error(errTestCase)
			}

		case "/secrets-safe/secrets":
			_, err := w.Write([]byte(`[{"SecretType": "FILE", "Password": "credential_in_sub_3_password","Id": "12345678-07d6-4955-175a-08db047219ce","Title": "credential_in_sub_3"}]`))
			if err != nil {
				t.Error(errTestCase)
			}

		case "/secrets-safe/secrets/12345678-07d6-4955-175a-08db047219ce/file/download":
			_, err := w.Write([]byte(`fake_password`))
			if err != nil {
				t.Error(errTestCase)
			}

		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	clientConfig := clientcmd.NewDefaultClientConfig(clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			"test": {
				Server: server.URL,
			},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			"test": {
				Token: "token",
			},
		},
		Contexts: map[string]*clientcmdapi.Context{
			"test": {
				Cluster:  "test",
				AuthInfo: "test",
			},
		},
		CurrentContext: "test",
	}, &clientcmd.ConfigOverrides{})

	restConfig, err := clientConfig.ClientConfig()
	assert.Nil(t, err)
	c, err := kubeclient.New(restConfig, kubeclient.Options{})
	assert.Nil(t, err)

	return c
}

func TestNewClient(t *testing.T) {
	type args struct {
		store    esv1beta1.SecretStore
		kube     kubeclient.Client
		provider esv1beta1.Provider
	}
	tests := []struct {
		name              string
		nameSpace         string
		args              args
		validateErrorNil  bool
		validateErrorText bool
		expectedErrorText string
	}{
		{
			name:      "Client ok",
			nameSpace: "test",
			args: args{
				store: esv1beta1.SecretStore{
					Spec: esv1beta1.SecretStoreSpec{
						Provider: &esv1beta1.SecretStoreProvider{
							Beyondtrust: &esv1beta1.BeyondtrustProvider{
								Server: &esv1beta1.BeyondtrustServer{
									APIURL:        fakeAPIURL,
									RetrievalType: "SECRET",
								},

								Auth: &esv1beta1.BeyondtrustAuth{
									ClientID: &esv1beta1.BeyondTrustProviderSecretRef{
										Value: clientID,
									},
									ClientSecret: &esv1beta1.BeyondTrustProviderSecretRef{
										Value: clientSecret,
									},
								},
							},
						},
					},
				},
				kube:     createMockPasswordSafeClient(t),
				provider: &Provider{},
			},
			validateErrorNil:  true,
			validateErrorText: false,
		},
		{
			name:      "Bad Client Id",
			nameSpace: "test",
			args: args{
				store: esv1beta1.SecretStore{
					Spec: esv1beta1.SecretStoreSpec{
						Provider: &esv1beta1.SecretStoreProvider{
							Beyondtrust: &esv1beta1.BeyondtrustProvider{
								Server: &esv1beta1.BeyondtrustServer{
									APIURL:        fakeAPIURL,
									RetrievalType: "SECRET",
								},

								Auth: &esv1beta1.BeyondtrustAuth{
									ClientID: &esv1beta1.BeyondTrustProviderSecretRef{
										Value: "6138d050",
									},
									ClientSecret: &esv1beta1.BeyondTrustProviderSecretRef{
										Value: clientSecret,
									},
								},
							},
						},
					},
				},
				kube:     createMockPasswordSafeClient(t),
				provider: &Provider{},
			},
			validateErrorNil:  false,
			validateErrorText: true,
			expectedErrorText: "error in Inputs: Key: 'UserInputValidaton.ClientId' Error:Field validation for 'ClientId' failed on the 'min' tag",
		},
		{
			name:      "Bad Client Secret",
			nameSpace: "test",
			args: args{
				store: esv1beta1.SecretStore{
					Spec: esv1beta1.SecretStoreSpec{
						Provider: &esv1beta1.SecretStoreProvider{
							Beyondtrust: &esv1beta1.BeyondtrustProvider{
								Server: &esv1beta1.BeyondtrustServer{
									APIURL:        fakeAPIURL,
									RetrievalType: "SECRET",
								},

								Auth: &esv1beta1.BeyondtrustAuth{
									ClientSecret: &esv1beta1.BeyondTrustProviderSecretRef{
										Value: "8i7U0Yulabon8mTc",
									},
									ClientID: &esv1beta1.BeyondTrustProviderSecretRef{
										Value: clientID,
									},
								},
							},
						},
					},
				},
				kube:     createMockPasswordSafeClient(t),
				provider: &Provider{},
			},
			validateErrorNil:  false,
			validateErrorText: true,
			expectedErrorText: "error in Inputs: Key: 'UserInputValidaton.ClientSecret' Error:Field validation for 'ClientSecret' failed on the 'min' tag",
		},
		{
			name:      "Bad Separator",
			nameSpace: "test",
			args: args{
				store: esv1beta1.SecretStore{
					Spec: esv1beta1.SecretStoreSpec{
						Provider: &esv1beta1.SecretStoreProvider{
							Beyondtrust: &esv1beta1.BeyondtrustProvider{
								Server: &esv1beta1.BeyondtrustServer{
									APIURL:        fakeAPIURL,
									Separator:     "//",
									RetrievalType: "SECRET",
								},
								Auth: &esv1beta1.BeyondtrustAuth{
									ClientID: &esv1beta1.BeyondTrustProviderSecretRef{
										Value: clientID,
									},
									ClientSecret: &esv1beta1.BeyondTrustProviderSecretRef{
										Value: clientSecret,
									},
								},
							},
						},
					},
				},
				kube:     createMockPasswordSafeClient(t),
				provider: &Provider{},
			},
			validateErrorNil:  false,
			validateErrorText: true,
			expectedErrorText: "error in Inputs: Key: 'UserInputValidaton.Separator' Error:Field validation for 'Separator' failed on the 'max' tag",
		},
		{
			name:      "Time Out",
			nameSpace: "test",
			args: args{
				store: esv1beta1.SecretStore{
					Spec: esv1beta1.SecretStoreSpec{
						Provider: &esv1beta1.SecretStoreProvider{
							Beyondtrust: &esv1beta1.BeyondtrustProvider{
								Server: &esv1beta1.BeyondtrustServer{
									APIURL:               fakeAPIURL,
									Separator:            "/",
									ClientTimeOutSeconds: 400,
									RetrievalType:        "SECRET",
								},
								Auth: &esv1beta1.BeyondtrustAuth{
									ClientID: &esv1beta1.BeyondTrustProviderSecretRef{
										Value: clientID,
									},
									ClientSecret: &esv1beta1.BeyondTrustProviderSecretRef{
										Value: clientSecret,
									},
								},
							},
						},
					},
				},
				kube:     createMockPasswordSafeClient(t),
				provider: &Provider{},
			},
			validateErrorNil:  false,
			validateErrorText: true,
			expectedErrorText: "error in Inputs: Key: 'UserInputValidaton.ClientTimeOutinSeconds' Error:Field validation for 'ClientTimeOutinSeconds' failed on the 'lte' tag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.args.provider.NewClient(context.Background(), &tt.args.store, tt.args.kube, tt.nameSpace)
			if err != nil && tt.validateErrorNil {
				t.Errorf("ProviderBeyondtrust.NewClient() error = %v", err)
			}

			if err != nil && tt.validateErrorText {
				assert.Equal(t, err.Error(), tt.expectedErrorText)
			}
		})
	}
}
