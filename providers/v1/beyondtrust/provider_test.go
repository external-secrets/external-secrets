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

package beyondtrust

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/utils/ptr"
	kubeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const (
	errTestCase  = "Test case Failed"
	fakeAPIURL   = "https://example.com:443/BeyondTrust/api/public/v3/"
	apiKey       = "fakeapikey00fakeapikeydd0000000000065b010f20fakeapikey0000000008700000a93fb5d74fddc0000000000000000000000000000000000000;runas=test_user"
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
		store    esv1.SecretStore
		kube     kubeclient.Client
		provider esv1.Provider
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
				store: esv1.SecretStore{
					Spec: esv1.SecretStoreSpec{
						Provider: &esv1.SecretStoreProvider{
							Beyondtrust: &esv1.BeyondtrustProvider{
								Server: &esv1.BeyondtrustServer{
									APIURL:        fakeAPIURL,
									RetrievalType: "SECRET",
								},

								Auth: &esv1.BeyondtrustAuth{
									ClientID: &esv1.BeyondTrustProviderSecretRef{
										Value: clientID,
									},
									ClientSecret: &esv1.BeyondTrustProviderSecretRef{
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
				store: esv1.SecretStore{
					Spec: esv1.SecretStoreSpec{
						Provider: &esv1.SecretStoreProvider{
							Beyondtrust: &esv1.BeyondtrustProvider{
								Server: &esv1.BeyondtrustServer{
									APIURL:        fakeAPIURL,
									RetrievalType: "SECRET",
								},

								Auth: &esv1.BeyondtrustAuth{
									ClientID: &esv1.BeyondTrustProviderSecretRef{
										Value: "6138d050",
									},
									ClientSecret: &esv1.BeyondTrustProviderSecretRef{
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
			expectedErrorText: "error in Inputs: Error in field ClientId : min / 36.",
		},
		{
			name:      "Bad Client Secret",
			nameSpace: "test",
			args: args{
				store: esv1.SecretStore{
					Spec: esv1.SecretStoreSpec{
						Provider: &esv1.SecretStoreProvider{
							Beyondtrust: &esv1.BeyondtrustProvider{
								Server: &esv1.BeyondtrustServer{
									APIURL:        fakeAPIURL,
									RetrievalType: "SECRET",
								},

								Auth: &esv1.BeyondtrustAuth{
									ClientSecret: &esv1.BeyondTrustProviderSecretRef{
										Value: "8i7U0Yulabon8mTc",
									},
									ClientID: &esv1.BeyondTrustProviderSecretRef{
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
			expectedErrorText: "error in Inputs: Error in field ClientSecret : min / 36.",
		},
		{
			name:      "Bad Separator",
			nameSpace: "test",
			args: args{
				store: esv1.SecretStore{
					Spec: esv1.SecretStoreSpec{
						Provider: &esv1.SecretStoreProvider{
							Beyondtrust: &esv1.BeyondtrustProvider{
								Server: &esv1.BeyondtrustServer{
									APIURL:        fakeAPIURL,
									Separator:     "//",
									RetrievalType: "SECRET",
								},
								Auth: &esv1.BeyondtrustAuth{
									ClientID: &esv1.BeyondTrustProviderSecretRef{
										Value: clientID,
									},
									ClientSecret: &esv1.BeyondTrustProviderSecretRef{
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
			expectedErrorText: "error in Inputs: Error in field ClientId : min / 36.",
		},
		{
			name:      "Time Out",
			nameSpace: "test",
			args: args{
				store: esv1.SecretStore{
					Spec: esv1.SecretStoreSpec{
						Provider: &esv1.SecretStoreProvider{
							Beyondtrust: &esv1.BeyondtrustProvider{
								Server: &esv1.BeyondtrustServer{
									APIURL:               fakeAPIURL,
									Separator:            "/",
									ClientTimeOutSeconds: 400,
									RetrievalType:        "SECRET",
								},
								Auth: &esv1.BeyondtrustAuth{
									ClientID: &esv1.BeyondTrustProviderSecretRef{
										Value: clientID,
									},
									ClientSecret: &esv1.BeyondTrustProviderSecretRef{
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
			expectedErrorText: "error in Inputs: Error in field ClientTimeOutinSeconds : lte / 300.",
		},
		{
			name:      "ApiKey ok",
			nameSpace: "test",
			args: args{
				store: esv1.SecretStore{
					Spec: esv1.SecretStoreSpec{
						Provider: &esv1.SecretStoreProvider{
							Beyondtrust: &esv1.BeyondtrustProvider{
								Server: &esv1.BeyondtrustServer{
									APIURL:        fakeAPIURL,
									RetrievalType: "SECRET",
								},

								Auth: &esv1.BeyondtrustAuth{
									APIKey: &esv1.BeyondTrustProviderSecretRef{
										Value: apiKey,
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
			name:      "Bad ApiKey",
			nameSpace: "test",
			args: args{
				store: esv1.SecretStore{
					Spec: esv1.SecretStoreSpec{
						Provider: &esv1.SecretStoreProvider{
							Beyondtrust: &esv1.BeyondtrustProvider{
								Server: &esv1.BeyondtrustServer{
									APIURL:        fakeAPIURL,
									RetrievalType: "SECRET",
								},

								Auth: &esv1.BeyondtrustAuth{
									APIKey: &esv1.BeyondTrustProviderSecretRef{
										Value: "bad_api_key",
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
			expectedErrorText: "error in Inputs: Error in field ApiKey : min / 128.",
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

func TestLoadConfigSecret_NamespacedStoreCannotCrossNamespace(t *testing.T) {
	kube := fake.NewClientBuilder().WithObjects(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "foo",
			Name:      "creds",
		},
		Data: map[string][]byte{
			"key": []byte("value"),
		},
	}).Build()
	ref := &esv1.BeyondTrustProviderSecretRef{
		SecretRef: &esmeta.SecretKeySelector{
			Namespace: ptr.To("foo"),
			Name:      "creds",
			Key:       "key",
		},
	}

	// For a namespaced SecretStore, attempting to read from another namespace must fail.
	_, err := loadConfigSecret(t.Context(), ref, kube, "ns2", esv1.SecretStoreKind)
	if err == nil {
		t.Fatalf("expected error when accessing secret across namespaces with SecretStore, got nil")
	}

	// For a namespaced SecretStore, attempting to read from the right namespace must not fail.
	val, err := loadConfigSecret(t.Context(), ref, kube, "foo", esv1.SecretStoreKind)
	if err != nil {
		t.Fatalf("expected error when accessing secret across namespaces with SecretStore, got nil")
	}
	if val != "value" {
		t.Fatalf("expected value, got %q", val)
	}
}

func TestLoadConfigSecret_ClusterStoreCanAccessOtherNamespace(t *testing.T) {
	kube := fake.NewClientBuilder().WithObjects(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "foo",
			Name:      "creds",
		},
		Data: map[string][]byte{
			"key": []byte("value"),
		},
	}).Build()

	ref := &esv1.BeyondTrustProviderSecretRef{
		SecretRef: &esmeta.SecretKeySelector{
			Namespace: ptr.To("foo"),
			Name:      "creds",
			Key:       "key",
		},
	}

	// ClusterSecretStore may access across namespaces when a namespace is provided in the selector.
	val, err := loadConfigSecret(t.Context(), ref, kube, "unrelated-namespace", esv1.ClusterSecretStoreKind)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "value" {
		t.Fatalf("expected valueA, got %q", val)
	}
}
