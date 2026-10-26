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

package beyondtrust

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/BeyondTrust/go-client-library-passwordsafe/api/authentication"
	"github.com/BeyondTrust/go-client-library-passwordsafe/api/logging"
	"github.com/BeyondTrust/go-client-library-passwordsafe/api/utils"
	"github.com/cenkalti/backoff/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	kubeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

const (
	errTestCase            = "Test case Failed"
	fakeAPIURL             = "https://example.com:443/BeyondTrust/api/public/v3/"
	apiKey                 = "fakeapikey00fakeapikeydd0000000000065b010f20fakeapikey0000000008700000a93fb5d74fddc0000000000000000000000000000000000000;runas=test_user"
	clientID               = "12345678-25fg-4b05-9ced-35e7dd5093ae"
	clientSecret           = "12345678-25fg-4b05-9ced-35e7dd5093ae"
	authConnectTokenPath   = "/Auth/connect/token"
	authSignAppInPath      = "/Auth/SignAppIn"
	secretsSafeFoldersPath = "/secrets-safe/folders/"
	secretsSafeSecretsPath = "/secrets-safe/secrets"
	folderSecretsPath      = "/secrets-safe/folders/cb871861-8b40-4556-820c-1ca6d522adfa/secrets"
	apiVersion32           = "3.2"
	fakeClientSecret       = "fake_client_secret"
	passwordKey            = "password"
	testCredentialKey      = "test-credential"
	testName               = "test"
)

func createMockPasswordSafeClient(t *testing.T) kubeclient.Client {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case authSignAppInPath:
			_, err := w.Write([]byte(`{"UserId":1, "EmailAddress":"fake@beyondtrust.com"}`))
			if err != nil {
				t.Error(errTestCase)
			}

		case "/Auth/Signout":
			_, err := w.Write([]byte(``))
			if err != nil {
				t.Error(errTestCase)
			}

		case secretsSafeSecretsPath:
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
			testName: {
				Server: server.URL,
			},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			testName: {
				Token: "token",
			},
		},
		Contexts: map[string]*clientcmdapi.Context{
			testName: {
				Cluster:  testName,
				AuthInfo: testName,
			},
		},
		CurrentContext: testName,
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
			nameSpace: testName,
			args: args{
				store: esv1.SecretStore{
					Spec: esv1.SecretStoreSpec{
						Provider: &esv1.SecretStoreProvider{
							Beyondtrust: &esv1.BeyondtrustProvider{
								Server: &esv1.BeyondtrustServer{
									APIURL:        fakeAPIURL,
									RetrievalType: secretTypeSecret,
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
			nameSpace: testName,
			args: args{
				store: esv1.SecretStore{
					Spec: esv1.SecretStoreSpec{
						Provider: &esv1.SecretStoreProvider{
							Beyondtrust: &esv1.BeyondtrustProvider{
								Server: &esv1.BeyondtrustServer{
									APIURL:        fakeAPIURL,
									RetrievalType: secretTypeSecret,
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
			nameSpace: testName,
			args: args{
				store: esv1.SecretStore{
					Spec: esv1.SecretStoreSpec{
						Provider: &esv1.SecretStoreProvider{
							Beyondtrust: &esv1.BeyondtrustProvider{
								Server: &esv1.BeyondtrustServer{
									APIURL:        fakeAPIURL,
									RetrievalType: secretTypeSecret,
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
			nameSpace: testName,
			args: args{
				store: esv1.SecretStore{
					Spec: esv1.SecretStoreSpec{
						Provider: &esv1.SecretStoreProvider{
							Beyondtrust: &esv1.BeyondtrustProvider{
								Server: &esv1.BeyondtrustServer{
									APIURL:        fakeAPIURL,
									Separator:     "//",
									RetrievalType: secretTypeSecret,
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
			nameSpace: testName,
			args: args{
				store: esv1.SecretStore{
					Spec: esv1.SecretStoreSpec{
						Provider: &esv1.SecretStoreProvider{
							Beyondtrust: &esv1.BeyondtrustProvider{
								Server: &esv1.BeyondtrustServer{
									APIURL:               fakeAPIURL,
									Separator:            "/",
									ClientTimeOutSeconds: 400,
									RetrievalType:        secretTypeSecret,
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
			nameSpace: testName,
			args: args{
				store: esv1.SecretStore{
					Spec: esv1.SecretStoreSpec{
						Provider: &esv1.SecretStoreProvider{
							Beyondtrust: &esv1.BeyondtrustProvider{
								Server: &esv1.BeyondtrustServer{
									APIURL:        fakeAPIURL,
									RetrievalType: secretTypeSecret,
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
			nameSpace: testName,
			args: args{
				store: esv1.SecretStore{
					Spec: esv1.SecretStoreSpec{
						Provider: &esv1.SecretStoreProvider{
							Beyondtrust: &esv1.BeyondtrustProvider{
								Server: &esv1.BeyondtrustServer{
									APIURL:        fakeAPIURL,
									RetrievalType: secretTypeSecret,
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
		{
			name:      "Missing RetrievalType",
			nameSpace: testName,
			args: args{
				store: esv1.SecretStore{
					Spec: esv1.SecretStoreSpec{
						Provider: &esv1.SecretStoreProvider{
							Beyondtrust: &esv1.BeyondtrustProvider{
								Server: &esv1.BeyondtrustServer{
									APIURL: fakeAPIURL,
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
			validateErrorNil:  false,
			validateErrorText: true,
			expectedErrorText: `invalid retrievalType "": must be one of MANAGED_ACCOUNT, SECRET, ALL`,
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
	kube := fake.NewClientBuilder().WithObjects(&v1.Secret{
		Namespace: "foo",
		Name:      "creds",
		Data: map[string][]byte{
			"key": []byte("value"),
		},
	}).Build()
	ref := &esv1.BeyondTrustProviderSecretRef{
		SecretRef: &esmeta.SecretKeySelector{
			Namespace: new("foo"),
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

func TestPushSecret(t *testing.T) {
	type testCase struct {
		name          string
		serverHandler http.HandlerFunc
		metadata      apiextensionsv1.JSON
		expectedError bool
		// apiVersion lets a case target a specific Password Safe API version.
		// Empty string keeps the historical default of 3.1.
		apiVersion string
	}

	tests := []testCase{
		{
			name: "successfully pushes credential secret",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case authConnectTokenPath:
					_, err := w.Write([]byte(`{"access_token": "fake_token", "expires_in": 600, "token_type": "Bearer"}`))
					if err != nil {
						t.Error(err)
					}
				case authSignAppInPath:
					_, err := w.Write([]byte(`{"UserId":1, "EmailAddress":"test@beyondtrust.com"}`))
					if err != nil {
						t.Error(err)
					}
				case secretsSafeFoldersPath:
					_, err := w.Write([]byte(`[{"Id": "cb871861-8b40-4556-820c-1ca6d522adfa","Name": "folder1"}]`))
					if err != nil {
						t.Error(err)
					}
				case folderSecretsPath:
					_, err := w.Write([]byte(`{"Id": "01ca9cf3-0751-4a90-4856-08dcf22d7472","Title": "Secret Title"}`))
					if err != nil {
						t.Error(err)
					}
				default:
					http.Error(w, "not found", http.StatusNotFound)
				}
			},
			expectedError: false,
			metadata: apiextensionsv1.JSON{
				Raw: []byte(`{
					"title": "Test Credential",
					"username": "admin",
					"description": "Test Credential Secret description",
					"secret_type": "CREDENTIAL",
					"folder_name": "folder1"
				}`),
			},
		},
		{
			name: "successfully pushes file secret",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case authConnectTokenPath:
					_, err := w.Write([]byte(`{"access_token": "fake_token", "expires_in": 600, "token_type": "Bearer"}`))
					if err != nil {
						t.Error(err)
					}
				case authSignAppInPath:
					_, err := w.Write([]byte(`{"UserId":1, "EmailAddress":"test@beyondtrust.com"}`))
					if err != nil {
						t.Error(err)
					}
				case secretsSafeFoldersPath:
					_, err := w.Write([]byte(`[{"Id": "cb871861-8b40-4556-820c-1ca6d522adfa","Name": "folder1"}]`))
					if err != nil {
						t.Error(err)
					}
				case "/secrets-safe/folders/cb871861-8b40-4556-820c-1ca6d522adfa/secrets/file":
					_, err := w.Write([]byte(`{"Id": "01ca9cf3-0751-4a90-4856-08dcf22d7472","Title": "Secret Title"}`))
					if err != nil {
						t.Error(err)
					}
				default:
					http.Error(w, "not found", http.StatusNotFound)
				}
			},
			expectedError: false,
			metadata: apiextensionsv1.JSON{
				Raw: []byte(`{
					"title": "Test File Secret",
					"username": "admin",
					"description": "Test File Secret description",
					"secret_type": "FILE",
					"folder_name": "folder1",
					"file_name": "credentials.txt"
				}`),
			},
		},
		{
			name: "successfully pushes text secret",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case authConnectTokenPath:
					_, err := w.Write([]byte(`{"access_token": "fake_token", "expires_in": 600, "token_type": "Bearer"}`))
					if err != nil {
						t.Error(err)
					}
				case authSignAppInPath:
					_, err := w.Write([]byte(`{"UserId":1, "EmailAddress":"test@beyondtrust.com"}`))
					if err != nil {
						t.Error(err)
					}
				case secretsSafeFoldersPath:
					_, err := w.Write([]byte(`[{"Id": "cb871861-8b40-4556-820c-1ca6d522adfa","Name": "folder1"}]`))
					if err != nil {
						t.Error(err)
					}
				case "/secrets-safe/folders/cb871861-8b40-4556-820c-1ca6d522adfa/secrets/text":
					_, err := w.Write([]byte(`{"Id": "01ca9cf3-0751-4a90-4856-08dcf22d7472","Title": "Secret Title"}`))
					if err != nil {
						t.Error(err)
					}
				default:
					http.Error(w, "not found", http.StatusNotFound)
				}
			},
			expectedError: false,
			metadata: apiextensionsv1.JSON{
				Raw: []byte(`{
					"title": "Test Text Secret",
					"username": "admin",
					"description": "Test File Secret description",
					"secret_type": "TEXT",
					"folder_name": "folder1"
				}`),
			},
		},
		{
			name: "successfully pushes credential secret v3.2",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case authConnectTokenPath:
					_, err := w.Write([]byte(`{"access_token": "fake_token", "expires_in": 600, "token_type": "Bearer"}`))
					if err != nil {
						t.Error(err)
					}
				case authSignAppInPath:
					_, err := w.Write([]byte(`{"UserId":1, "EmailAddress":"test@beyondtrust.com"}`))
					if err != nil {
						t.Error(err)
					}
				case secretsSafeFoldersPath:
					_, err := w.Write([]byte(`[{"Id": "cb871861-8b40-4556-820c-1ca6d522adfa","Name": "folder1"}]`))
					if err != nil {
						t.Error(err)
					}
				case folderSecretsPath:
					_, err := w.Write([]byte(`{"Id": "01ca9cf3-0751-4a90-4856-08dcf22d7472","Title": "Secret Title"}`))
					if err != nil {
						t.Error(err)
					}
				default:
					http.Error(w, "not found", http.StatusNotFound)
				}
			},
			expectedError: false,
			apiVersion:    apiVersion32,
			metadata: apiextensionsv1.JSON{
				Raw: []byte(`{
					"title": "Test Credential v3.2",
					"username": "admin",
					"description": "Test Credential Secret description",
					"secret_type": "CREDENTIAL",
					"folder_name": "folder1"
				}`),
			},
		},
		{
			name: "successfully pushes file secret v3.2",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case authConnectTokenPath:
					_, err := w.Write([]byte(`{"access_token": "fake_token", "expires_in": 600, "token_type": "Bearer"}`))
					if err != nil {
						t.Error(err)
					}
				case authSignAppInPath:
					_, err := w.Write([]byte(`{"UserId":1, "EmailAddress":"test@beyondtrust.com"}`))
					if err != nil {
						t.Error(err)
					}
				case secretsSafeFoldersPath:
					_, err := w.Write([]byte(`[{"Id": "cb871861-8b40-4556-820c-1ca6d522adfa","Name": "folder1"}]`))
					if err != nil {
						t.Error(err)
					}
				case "/secrets-safe/folders/cb871861-8b40-4556-820c-1ca6d522adfa/secrets/file":
					_, err := w.Write([]byte(`{"Id": "01ca9cf3-0751-4a90-4856-08dcf22d7472","Title": "Secret Title"}`))
					if err != nil {
						t.Error(err)
					}
				default:
					http.Error(w, "not found", http.StatusNotFound)
				}
			},
			expectedError: false,
			apiVersion:    apiVersion32,
			metadata: apiextensionsv1.JSON{
				Raw: []byte(`{
					"title": "Test File Secret v3.2",
					"username": "admin",
					"description": "Test File Secret description",
					"secret_type": "FILE",
					"folder_name": "folder1",
					"file_name": "credentials.txt"
				}`),
			},
		},
		{
			name: "successfully pushes text secret v3.2",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case authConnectTokenPath:
					_, err := w.Write([]byte(`{"access_token": "fake_token", "expires_in": 600, "token_type": "Bearer"}`))
					if err != nil {
						t.Error(err)
					}
				case authSignAppInPath:
					_, err := w.Write([]byte(`{"UserId":1, "EmailAddress":"test@beyondtrust.com"}`))
					if err != nil {
						t.Error(err)
					}
				case secretsSafeFoldersPath:
					_, err := w.Write([]byte(`[{"Id": "cb871861-8b40-4556-820c-1ca6d522adfa","Name": "folder1"}]`))
					if err != nil {
						t.Error(err)
					}
				case "/secrets-safe/folders/cb871861-8b40-4556-820c-1ca6d522adfa/secrets/text":
					_, err := w.Write([]byte(`{"Id": "01ca9cf3-0751-4a90-4856-08dcf22d7472","Title": "Secret Title"}`))
					if err != nil {
						t.Error(err)
					}
				default:
					http.Error(w, "not found", http.StatusNotFound)
				}
			},
			expectedError: false,
			apiVersion:    apiVersion32,
			metadata: apiextensionsv1.JSON{
				Raw: []byte(`{
					"title": "Test Text Secret v3.2",
					"username": "admin",
					"description": "Test Text Secret description",
					"secret_type": "TEXT",
					"folder_name": "folder1"
				}`),
			},
		},
		{
			name: "successfully pushes text secret - 404 error",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case authConnectTokenPath:
					_, err := w.Write([]byte(`{"access_token": "fake_token", "expires_in": 600, "token_type": "Bearer"}`))
					if err != nil {
						t.Error(err)
					}
				case authSignAppInPath:
					_, err := w.Write([]byte(`{"UserId":1, "EmailAddress":"test@beyondtrust.com"}`))
					if err != nil {
						t.Error(err)
					}
				case secretsSafeFoldersPath:
					_, err := w.Write([]byte(`[{"Id": "cb871861-8b40-4556-820c-1ca6d522adfa","Name": "folder1"}]`))
					if err != nil {
						t.Error(err)
					}
				default:
					http.Error(w, "not found", http.StatusNotFound)
				}
			},
			expectedError: true,
			metadata: apiextensionsv1.JSON{
				Raw: []byte(`{
					"title": "Test Text Secret",
					"username": "admin",
					"description": "Test File Secret description",
					"secret_type": "TEXT",
					"folder_name": "folder1"
				}`),
			},
		},
		{
			name: "fails authentication",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeServer := httptest.NewServer(tt.serverHandler)
			defer fakeServer.Close()

			logger, err := zap.NewDevelopment()

			if err != nil {
				t.Error(err)
			}

			zapLogger := logging.NewZapLogger(logger)

			clientTimeout := 30
			verifyCa := true
			retryMaxElapsedTimeMinutes := 2

			backoffDefinition := backoff.NewExponentialBackOff()
			backoffDefinition.InitialInterval = 1 * time.Second
			backoffDefinition.MaxElapsedTime = time.Duration(retryMaxElapsedTimeMinutes) * time.Second
			backoffDefinition.RandomizationFactor = 0.5

			httpClientObj, err := utils.GetHttpClient(clientTimeout, verifyCa, "", "", zapLogger)

			if err != nil {
				t.Error(err)
			}

			apiVersion := tt.apiVersion
			if apiVersion == "" {
				apiVersion = "3.1"
			}

			params := authentication.AuthenticationParametersObj{
				HTTPClient:                 *httpClientObj,
				BackoffDefinition:          backoffDefinition,
				EndpointURL:                fakeServer.URL,
				APIVersion:                 apiVersion,
				ClientID:                   "fake_clinet_id",
				ClientSecret:               fakeClientSecret,
				Logger:                     zapLogger,
				RetryMaxElapsedTimeSeconds: 30,
			}

			authObj, err := authentication.Authenticate(params)
			require.NoError(t, err)

			p := &Provider{authenticate: *authObj}

			secret := &v1.Secret{
				Data: map[string][]byte{passwordKey: []byte("supersecret")},
			}

			metadataJSON := &tt.metadata

			psd := v1alpha1.PushSecretData{
				Match: v1alpha1.PushSecretMatch{
					SecretKey: passwordKey,
					RemoteRef: v1alpha1.PushSecretRemoteRef{
						RemoteKey: testCredentialKey,
					},
				},
				Metadata: metadataJSON,
			}

			err = p.PushSecret(context.Background(), secret, psd)

			if tt.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSecretExists(t *testing.T) {
	type testCase struct {
		name             string
		serverHandler    http.HandlerFunc
		expectedExisting bool
	}

	tests := []testCase{
		{
			name: "Secret Exists",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case authConnectTokenPath:
					_, err := w.Write([]byte(`{"access_token": "fake_token", "expires_in": 600, "token_type": "Bearer"}`))
					if err != nil {
						t.Error(err)
					}
				case authSignAppInPath:
					_, err := w.Write([]byte(`{"UserId":1, "EmailAddress":"test@beyondtrust.com"}`))
					if err != nil {
						t.Error(err)
					}
				case secretsSafeSecretsPath:
					_, err := w.Write([]byte(`[{"Id": "01ca9cf3-0751-4a90-4856-08dcf22d7472","Title": "Secret Title"}]`))
					if err != nil {
						t.Error(err)
					}
				default:
					http.Error(w, "not found", http.StatusNotFound)
				}
			},
			expectedExisting: true,
		},
		{
			name: "Secret does not Exist",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case authConnectTokenPath:
					_, err := w.Write([]byte(`{"access_token": "fake_token", "expires_in": 600, "token_type": "Bearer"}`))
					if err != nil {
						t.Error(err)
					}
				case authSignAppInPath:
					_, err := w.Write([]byte(`{"UserId":1, "EmailAddress":"test@beyondtrust.com"}`))
					if err != nil {
						t.Error(err)
					}
				case secretsSafeSecretsPath:
					http.Error(w, "secret was not found", http.StatusNotFound)
				default:
					http.Error(w, "not found", http.StatusNotFound)
				}
			},
			expectedExisting: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeServer := httptest.NewServer(tt.serverHandler)
			defer fakeServer.Close()

			logger, err := zap.NewDevelopment()

			if err != nil {
				t.Error(err)
			}

			zapLogger := logging.NewZapLogger(logger)

			clientTimeout := 30
			verifyCa := true
			retryMaxElapsedTimeMinutes := 2

			backoffDefinition := backoff.NewExponentialBackOff()
			backoffDefinition.InitialInterval = 1 * time.Second
			backoffDefinition.MaxElapsedTime = time.Duration(retryMaxElapsedTimeMinutes) * time.Second
			backoffDefinition.RandomizationFactor = 0.5

			httpClientObj, err := utils.GetHttpClient(clientTimeout, verifyCa, "", "", zapLogger)

			if err != nil {
				t.Error(err)
			}

			params := authentication.AuthenticationParametersObj{
				HTTPClient:                 *httpClientObj,
				BackoffDefinition:          backoffDefinition,
				EndpointURL:                fakeServer.URL,
				APIVersion:                 "3.1",
				ClientID:                   "fake_clinet_id",
				ClientSecret:               fakeClientSecret,
				Logger:                     zapLogger,
				RetryMaxElapsedTimeSeconds: 30,
			}

			authObj, err := authentication.Authenticate(params)
			require.NoError(t, err)

			p := &Provider{authenticate: *authObj}

			remoteRef := v1alpha1.PushSecretRemoteRef{
				RemoteKey: testCredentialKey,
			}

			exists, err := p.SecretExists(context.Background(), remoteRef)

			if err != nil {
				t.Error(err)
			}

			if tt.expectedExisting {
				assert.True(t, exists)
			} else {
				assert.False(t, exists)
			}
		})
	}
}

// writeBody is a small helper for the mock servers below. It fails the test
// if the response cannot be written, instead of nesting an err-check inside
// every switch case in the handler.
func writeBody(t *testing.T, w http.ResponseWriter, body string) {
	t.Helper()
	if _, err := w.Write([]byte(body)); err != nil {
		t.Error(err)
	}
}

// newOwnerFieldsMockServer returns an httptest.Server that satisfies the
// auth + folder lookup + create-secret request flow. The body of the
// create-secret POST is captured into *captured for assertion.
func newOwnerFieldsMockServer(t *testing.T, secretsPath string, captured map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case authConnectTokenPath:
			writeBody(t, w, `{"access_token": "fake_token", "expires_in": 600, "token_type": "Bearer"}`)
		case authSignAppInPath:
			writeBody(t, w, `{"UserId":1, "EmailAddress":"test@beyondtrust.com"}`)
		case secretsSafeFoldersPath:
			writeBody(t, w, `[{"Id": "cb871861-8b40-4556-820c-1ca6d522adfa","Name": "folder1"}]`)
		case secretsPath:
			bodyBytes, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.NoError(t, json.Unmarshal(bodyBytes, &captured))
			writeBody(t, w, `{"Id": "01ca9cf3-0751-4a90-4856-08dcf22d7472","Title": "Secret Title"}`)
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
}

// newAuthenticatedProvider builds a Provider authenticated against serverURL
// with the requested API version. Used by tests that need a ready-to-call
// Provider without inlining the full auth/HTTP-client wiring.
func newAuthenticatedProvider(t *testing.T, serverURL, apiVersion string) *Provider {
	t.Helper()
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	zapLogger := logging.NewZapLogger(logger)

	backoffDefinition := backoff.NewExponentialBackOff()
	backoffDefinition.InitialInterval = 1 * time.Second
	backoffDefinition.MaxElapsedTime = 2 * time.Second
	backoffDefinition.RandomizationFactor = 0.5

	httpClientObj, err := utils.GetHttpClient(30, true, "", "", zapLogger)
	require.NoError(t, err)

	authObj, err := authentication.Authenticate(authentication.AuthenticationParametersObj{
		HTTPClient:                 *httpClientObj,
		BackoffDefinition:          backoffDefinition,
		EndpointURL:                serverURL,
		APIVersion:                 apiVersion,
		ClientID:                   "fake_client_id",
		ClientSecret:               fakeClientSecret,
		Logger:                     zapLogger,
		RetryMaxElapsedTimeSeconds: 30,
	})
	require.NoError(t, err)

	return &Provider{authenticate: *authObj}
}

func ownerFieldsMetadata(secretType string) apiextensionsv1.JSON {
	return apiextensionsv1.JSON{
		Raw: fmt.Appendf(nil, `{
			"title": "Owner Fields Test",
			"username": "admin",
			"secret_type": %q,
			"folder_name": "folder1",
			"owner_type": "User",
			"owner_id": 7,
			"group_id": 42
		}`, secretType),
	}
}

// TestPushSecret_OwnerFieldsArePropagated guards the dual-field migration in CreateSecret
// where every secret-type input carries both OwnersByOwnerId and OwnersByGroupId. The SDK
// then narrows down to one shape based on API version: v3.0 → OwnerDetailsOwnerId,
// v3.1/v3.2 → OwnerDetailsGroupId. We assert the request body the SDK actually emitted.
func TestPushSecret_OwnerFieldsArePropagated(t *testing.T) {
	type testCase struct {
		name        string
		apiVersion  string
		secretType  string
		secretsPath string
		assertBody  func(t *testing.T, body map[string]any)
	}

	tests := []testCase{
		{
			name:        "v3.0 propagates OwnersByOwnerId and top-level OwnerId/OwnerType for CREDENTIAL",
			apiVersion:  "3.0",
			secretType:  secretTypeCredential,
			secretsPath: folderSecretsPath,
			assertBody: func(t *testing.T, body map[string]any) {
				assert.Equal(t, float64(7), body["OwnerId"], "top-level OwnerId should come from metadata.owner_id")
				assert.Equal(t, "User", body["OwnerType"], "top-level OwnerType should come from metadata.owner_type")

				owners, ok := body["Owners"].([]any)
				require.True(t, ok, "Owners array missing or wrong type")
				require.NotEmpty(t, owners, "Owners array empty")
				first := owners[0].(map[string]any)
				assert.Equal(t, float64(1), first["OwnerId"], "main owner OwnerId should be UserId=1 from sign-in")
				assert.Equal(t, "test@beyondtrust.com", first["Email"])
			},
		},
		{
			name:        "v3.2 propagates OwnersByGroupId for CREDENTIAL",
			apiVersion:  apiVersion32,
			secretType:  secretTypeCredential,
			secretsPath: folderSecretsPath,
			assertBody: func(t *testing.T, body map[string]any) {
				_, hasOwnerId := body["OwnerId"]
				_, hasOwnerType := body["OwnerType"]
				assert.False(t, hasOwnerId, "v3.2 Config32 should not emit top-level OwnerId")
				assert.False(t, hasOwnerType, "v3.2 Config32 should not emit top-level OwnerType")

				owners, ok := body["Owners"].([]any)
				require.True(t, ok, "Owners array missing or wrong type")
				require.NotEmpty(t, owners, "Owners array empty")
				first := owners[0].(map[string]any)
				assert.Equal(t, float64(42), first["GroupId"], "main owner GroupId should come from metadata.group_id")
				assert.Equal(t, float64(1), first["UserId"], "main owner UserId should be UserId=1 from sign-in")
				assert.Equal(t, "test@beyondtrust.com", first["Email"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capturedBody := make(map[string]any)
			fakeServer := newOwnerFieldsMockServer(t, tt.secretsPath, capturedBody)
			defer fakeServer.Close()

			p := newAuthenticatedProvider(t, fakeServer.URL, tt.apiVersion)
			metadata := ownerFieldsMetadata(tt.secretType)

			psd := v1alpha1.PushSecretData{
				Match: v1alpha1.PushSecretMatch{
					SecretKey: passwordKey,
					RemoteRef: v1alpha1.PushSecretRemoteRef{RemoteKey: testCredentialKey},
				},
				Metadata: &metadata,
			}
			secret := &v1.Secret{Data: map[string][]byte{passwordKey: []byte("supersecret")}}

			require.NoError(t, p.PushSecret(context.Background(), secret, psd))
			require.NotNil(t, capturedBody, "create-secret endpoint was never hit")
			tt.assertBody(t, capturedBody)
		})
	}
}

func TestValidateStore(t *testing.T) {
	p := &Provider{}
	validRef := &esv1.BeyondTrustProviderSecretRef{Value: "x"}

	newStore := func(prov *esv1.BeyondtrustProvider) esv1.GenericStore {
		return &esv1.SecretStore{Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{Beyondtrust: prov},
		}}
	}

	tests := []struct {
		name     string
		store    esv1.GenericStore
		wantErr  bool
		wantWarn bool
	}{
		{
			name: "api key auth validates without panicking",
			store: newStore(&esv1.BeyondtrustProvider{
				Server: &esv1.BeyondtrustServer{APIURL: fakeAPIURL, RetrievalType: retrievalTypeManagedAccount},
				Auth:   &esv1.BeyondtrustAuth{APIKey: validRef},
			}),
			wantErr: false,
		},
		{
			name: "client id and secret auth validates",
			store: newStore(&esv1.BeyondtrustProvider{
				Server: &esv1.BeyondtrustServer{APIURL: fakeAPIURL, RetrievalType: retrievalTypeManagedAccount},
				Auth:   &esv1.BeyondtrustAuth{ClientID: validRef, ClientSecret: validRef},
			}),
			wantErr: false,
		},
		{
			name: "empty api url is rejected",
			store: newStore(&esv1.BeyondtrustProvider{
				Server: &esv1.BeyondtrustServer{APIURL: ""},
				Auth:   &esv1.BeyondtrustAuth{APIKey: validRef},
			}),
			wantErr: true,
		},
		{
			name: "missing server is rejected",
			store: newStore(&esv1.BeyondtrustProvider{
				Auth: &esv1.BeyondtrustAuth{APIKey: validRef},
			}),
			wantErr: true,
		},
		{
			name: "missing auth is rejected",
			store: newStore(&esv1.BeyondtrustProvider{
				Server: &esv1.BeyondtrustServer{APIURL: fakeAPIURL, RetrievalType: retrievalTypeManagedAccount},
			}),
			wantErr: true,
		},
		{
			name: "auth without any method is rejected",
			store: newStore(&esv1.BeyondtrustProvider{
				Server: &esv1.BeyondtrustServer{APIURL: fakeAPIURL, RetrievalType: retrievalTypeManagedAccount},
				Auth:   &esv1.BeyondtrustAuth{},
			}),
			wantErr: true,
		},
		{
			name: "client id without client secret is rejected",
			store: newStore(&esv1.BeyondtrustProvider{
				Server: &esv1.BeyondtrustServer{APIURL: fakeAPIURL, RetrievalType: retrievalTypeManagedAccount},
				Auth:   &esv1.BeyondtrustAuth{ClientID: validRef},
			}),
			wantErr: true,
		},
		{
			name: "auth ref without secretRef or value is rejected",
			store: newStore(&esv1.BeyondtrustProvider{
				Server: &esv1.BeyondtrustServer{APIURL: fakeAPIURL, RetrievalType: retrievalTypeManagedAccount},
				Auth:   &esv1.BeyondtrustAuth{APIKey: &esv1.BeyondTrustProviderSecretRef{}},
			}),
			wantErr: true,
		},
		{
			name: "cross-namespace secretRef is rejected for a namespaced store",
			store: newStore(&esv1.BeyondtrustProvider{
				Server: &esv1.BeyondtrustServer{APIURL: fakeAPIURL, RetrievalType: retrievalTypeManagedAccount},
				Auth: &esv1.BeyondtrustAuth{APIKey: &esv1.BeyondTrustProviderSecretRef{
					SecretRef: &esmeta.SecretKeySelector{
						Name:      "creds",
						Key:       "apikey",
						Namespace: new("other-namespace"),
					},
				}},
			}),
			wantErr: true,
		},
		{
			name: "certificate without key warns but validates",
			store: newStore(&esv1.BeyondtrustProvider{
				Server: &esv1.BeyondtrustServer{APIURL: fakeAPIURL, RetrievalType: retrievalTypeManagedAccount},
				Auth:   &esv1.BeyondtrustAuth{APIKey: validRef, Certificate: validRef},
			}),
			wantErr:  false,
			wantWarn: true,
		},
		{
			name: "complete certificate pair validates without warning",
			store: newStore(&esv1.BeyondtrustProvider{
				Server: &esv1.BeyondtrustServer{APIURL: fakeAPIURL, RetrievalType: retrievalTypeManagedAccount},
				Auth:   &esv1.BeyondtrustAuth{APIKey: validRef, Certificate: validRef, CertificateKey: validRef},
			}),
			wantErr: false,
		},
		{
			name: "retrievalType ALL validates without warning",
			store: newStore(&esv1.BeyondtrustProvider{
				Server: &esv1.BeyondtrustServer{APIURL: fakeAPIURL, RetrievalType: retrievalTypeAll},
				Auth:   &esv1.BeyondtrustAuth{APIKey: validRef},
			}),
			wantErr: false,
		},
		{
			name: "unknown retrievalType is rejected",
			store: newStore(&esv1.BeyondtrustProvider{
				Server: &esv1.BeyondtrustServer{APIURL: fakeAPIURL, RetrievalType: "BOTH"},
				Auth:   &esv1.BeyondtrustAuth{APIKey: validRef},
			}),
			wantErr: true,
		},
		{
			name: "empty retrievalType is rejected",
			store: newStore(&esv1.BeyondtrustProvider{
				Server: &esv1.BeyondtrustServer{APIURL: fakeAPIURL},
				Auth:   &esv1.BeyondtrustAuth{APIKey: validRef},
			}),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings, err := p.ValidateStore(tt.store)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			if tt.wantWarn {
				require.NotEmpty(t, warnings)
			} else {
				require.Empty(t, warnings)
			}
		})
	}
}

func TestResolveRetrievalKey(t *testing.T) {
	tests := []struct {
		name          string
		retrievalType string
		key           string
		wantManaged   bool
		wantPath      string
		wantErr       bool
	}{
		{name: "ALL with managed prefix", retrievalType: retrievalTypeAll, key: "managed:system01/account01", wantManaged: true, wantPath: "system01/account01"},
		{name: "ALL with secret prefix", retrievalType: retrievalTypeAll, key: "secret:app/folder/api-key", wantManaged: false, wantPath: "app/folder/api-key"},
		{name: "ALL is case-insensitive", retrievalType: "all", key: "secret:app/api-key", wantManaged: false, wantPath: "app/api-key"},
		{name: "ALL without prefix is rejected", retrievalType: retrievalTypeAll, key: "app/api-key", wantErr: true},
		{name: "ALL with unknown prefix is rejected", retrievalType: retrievalTypeAll, key: "file:app/api-key", wantErr: true},
		{name: "SECRET without prefix", retrievalType: secretTypeSecret, key: "app/api-key", wantManaged: false, wantPath: "app/api-key"},
		{name: "SECRET strips its own prefix", retrievalType: secretTypeSecret, key: "secret:app/api-key", wantManaged: false, wantPath: "app/api-key"},
		{name: "SECRET rejects the managed prefix", retrievalType: secretTypeSecret, key: "managed:system01/account01", wantErr: true},
		{name: "MANAGED_ACCOUNT without prefix", retrievalType: retrievalTypeManagedAccount, key: "system01/account01", wantManaged: true, wantPath: "system01/account01"},
		{name: "MANAGED_ACCOUNT strips its own prefix", retrievalType: retrievalTypeManagedAccount, key: "managed:system01/account01", wantManaged: true, wantPath: "system01/account01"},
		{name: "MANAGED_ACCOUNT rejects the secret prefix", retrievalType: retrievalTypeManagedAccount, key: "secret:app/api-key", wantErr: true},
		{name: "empty retrievalType is rejected", retrievalType: "", key: "system01/account01", wantErr: true},
		{name: "unknown retrievalType is rejected", retrievalType: "BOTH", key: "system01/account01", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			managed, path, err := resolveRetrievalKey(tt.retrievalType, tt.key)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantManaged, managed)
			assert.Equal(t, tt.wantPath, path)
		})
	}
}

// getSecretMock selects which request of the mocked retrieval flows fails.
type getSecretMock struct {
	missingManaged bool // the managed account lookup returns 404
	missingSecret  bool // the Secrets Safe search returns no secret
	failedCheckIn  bool // the managed account check-in returns 404
	fileSecret     bool // the secret is a FILE whose download returns 404
}

// newGetSecretMockServer serves both the managed account and the Secrets Safe
// retrieval flows, failing the requests that mock selects with the responses
// Password Safe sends.
func newGetSecretMockServer(t *testing.T, mock getSecretMock) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case authConnectTokenPath:
			writeBody(t, w, `{"access_token": "fake_token", "expires_in": 600, "token_type": "Bearer"}`)
		case authSignAppInPath:
			writeBody(t, w, `{"UserId":1, "EmailAddress":"test@beyondtrust.com"}`)
		case "/Auth/Signout":
			writeBody(t, w, ``)
		case "/ManagedAccounts":
			if mock.missingManaged {
				http.Error(w, "managed account not found", http.StatusNotFound)
				return
			}
			writeBody(t, w, `{"SystemId":1,"AccountId":10}`)
		case "/Requests":
			writeBody(t, w, `124`)
		case "/Credentials/124":
			writeBody(t, w, `"managed_password"`)
		case "/Requests/124/checkin":
			if mock.failedCheckIn {
				http.Error(w, "request not found", http.StatusNotFound)
				return
			}
			writeBody(t, w, ``)
		case secretsSafeSecretsPath:
			switch {
			case mock.missingSecret:
				writeBody(t, w, `[]`)
			case mock.fileSecret:
				// No handler serves the file download, so it returns 404.
				writeBody(t, w, `[{"SecretType": "FILE", "Id": "01ca9cf3-0751-4a90-4856-08dcf22d7472","Title": "api-key"}]`)
			default:
				writeBody(t, w, `[{"SecretType": "CREDENTIAL", "Password": "secret_password","Id": "01ca9cf3-0751-4a90-4856-08dcf22d7472","Title": "api-key"}]`)
			}
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
}

func TestGetSecret(t *testing.T) {
	tests := []struct {
		name          string
		retrievalType string
		key           string
		mock          getSecretMock
		want          string
		wantErr       error
		wantErrMsg    string
		wantFound     bool // the error must not be NoSecretErr
	}{
		{name: "ALL retrieves a managed account", retrievalType: retrievalTypeAll, key: "managed:system01/account01", want: "managed_password"},
		{name: "ALL retrieves a Secrets Safe secret", retrievalType: retrievalTypeAll, key: "secret:app/folder/api-key", want: "secret_password"},
		{name: "ALL without prefix is rejected", retrievalType: retrievalTypeAll, key: "app/folder/api-key", wantErrMsg: "retrievalType ALL requires"},
		{name: "ALL with invalid path after prefix", retrievalType: retrievalTypeAll, key: "managed:system01/folder/account01", wantErrMsg: errInvalidRetrievalPath},
		{name: "MANAGED_ACCOUNT without prefix is unchanged", retrievalType: retrievalTypeManagedAccount, key: "system01/account01", want: "managed_password"},
		{name: "SECRET without prefix is unchanged", retrievalType: secretTypeSecret, key: "app/folder/api-key", want: "secret_password"},
		{name: "MANAGED_ACCOUNT accepts the managed prefix", retrievalType: retrievalTypeManagedAccount, key: "managed:system01/account01", want: "managed_password"},
		{name: "SECRET accepts the secret prefix", retrievalType: secretTypeSecret, key: "secret:app/folder/api-key", want: "secret_password"},
		{name: "MANAGED_ACCOUNT rejects the secret prefix", retrievalType: retrievalTypeManagedAccount, key: "secret:app/folder/api-key", wantErrMsg: "does not match store retrievalType"},
		{name: "missing managed account returns NoSecretErr", retrievalType: retrievalTypeAll, key: "managed:system01/account01", mock: getSecretMock{missingManaged: true}, wantErr: esv1.NoSecretErr},
		{name: "missing secret returns NoSecretErr", retrievalType: retrievalTypeAll, key: "secret:app/folder/api-key", mock: getSecretMock{missingSecret: true}, wantErr: esv1.NoSecretErr},
		{name: "failed managed account check-in is not NoSecretErr", retrievalType: retrievalTypeAll, key: "managed:system01/account01", mock: getSecretMock{failedCheckIn: true}, wantFound: true},
		{name: "failed file download is not NoSecretErr", retrievalType: retrievalTypeAll, key: "secret:app/folder/api-key", mock: getSecretMock{fileSecret: true}, wantFound: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeServer := newGetSecretMockServer(t, tt.mock)
			defer fakeServer.Close()

			p := newAuthenticatedProvider(t, fakeServer.URL, "3.1")
			p.retrievaltype = tt.retrievalType
			p.separator = "/"
			p.decrypt = true
			p.log = *logging.NewLogrLogger(&ESOLogger)

			got, err := p.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: tt.key})
			switch {
			case tt.wantErr != nil:
				require.ErrorIs(t, err, tt.wantErr)
			case tt.wantErrMsg != "":
				require.ErrorContains(t, err, tt.wantErrMsg)
			case tt.wantFound:
				require.Error(t, err)
				require.NotErrorIs(t, err, esv1.NoSecretErr)
			default:
				require.NoError(t, err)
				assert.Equal(t, tt.want, string(got))
			}
		})
	}
}
