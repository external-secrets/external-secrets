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

package bitwarden

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

func XTestProvider_DeleteSecret(t *testing.T) {
	type fields struct {
		kube               client.Client
		namespace          string
		store              v1beta1.GenericStore
		bitwardenSdkClient *SdkClient
	}
	type args struct {
		ctx context.Context
		ref v1beta1.PushSecretRemoteRef
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "delete secret is successfully",
			wantErr: true,
			args: args{
				ctx: context.TODO(),
				ref: v1alpha1.PushSecretRemoteRef{
					RemoteKey: "d8f29773-3019-4973-9bbc-66327d077fe2",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Provider{
				kube:               tt.fields.kube,
				namespace:          tt.fields.namespace,
				store:              tt.fields.store,
				bitwardenSdkClient: tt.fields.bitwardenSdkClient,
			}
			if err := p.DeleteSecret(tt.args.ctx, tt.args.ref); (err != nil) != tt.wantErr {
				t.Errorf("DeleteSecret() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func XTestProvider_GetAllSecrets(t *testing.T) {
	type fields struct {
		kube               client.Client
		namespace          string
		store              v1beta1.GenericStore
		bitwardenSdkClient *SdkClient
	}
	type args struct {
		ctx context.Context
		ref v1beta1.ExternalSecretFind
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    map[string][]byte
		wantErr bool
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Provider{
				kube:               tt.fields.kube,
				namespace:          tt.fields.namespace,
				store:              tt.fields.store,
				bitwardenSdkClient: tt.fields.bitwardenSdkClient,
			}
			got, err := p.GetAllSecrets(tt.args.ctx, tt.args.ref)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAllSecrets() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAllSecrets() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProvider_GetSecret(t *testing.T) {
	type fields struct {
		kube               func() client.Client
		namespace          string
		store              v1beta1.GenericStore
		bitwardenSdkClient func(t *testing.T) (*SdkClient, func())
	}
	type args struct {
		ctx context.Context
		ref v1beta1.ExternalSecretDataRemoteRef
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "get secret with UUID",
			fields: fields{
				kube: func() client.Client {
					return fake.NewFakeClient()
				},
				namespace: "default",
				store:     &v1beta1.SecretStore{},
				bitwardenSdkClient: func(t *testing.T) (*SdkClient, func()) {
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						content, _ := io.ReadAll(r.Body)
						assert.Equal(t, string(content), `{"id":"d8f29773-3019-4973-9bbc-66327d077fe2"}`)
						resp := &SecretResponse{
							ID:             "id",
							Key:            "key",
							Note:           "note",
							OrganizationID: "org",
							Value:          "value",
						}
						data, _ := json.Marshal(resp)
						w.Write(data)
					}))

					return &SdkClient{
						apiURL:                server.URL,
						identityURL:           server.URL,
						token:                 "token",
						bitwardenSdkServerURL: server.URL,
						client:                server.Client(),
					}, server.Close
				},
			},
			args: args{
				ctx: context.Background(),
				ref: v1beta1.ExternalSecretDataRemoteRef{
					Key: "d8f29773-3019-4973-9bbc-66327d077fe2",
				},
			},
			want: []byte("value"),
		},
		{
			name: "get secret by name",
			fields: fields{
				kube: func() client.Client {
					return fake.NewFakeClient()
				},
				namespace: "default",
				store: &v1beta1.SecretStore{
					Spec: v1beta1.SecretStoreSpec{
						Provider: &v1beta1.SecretStoreProvider{
							BitwardenSecretsManager: &v1beta1.BitwardenSecretsManagerProvider{
								OrganizationID: "orgid",
							},
						},
					},
				},
				bitwardenSdkClient: func(t *testing.T) (*SdkClient, func()) {
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if strings.Contains(r.URL.String(), "rest/api/1/secrets") {
							resp := &SecretIdentifiersResponse{
								Data: []SecretIdentifierResponse{
									{
										ID:             "d8f29773-3019-4973-9bbc-66327d077fe2",
										Key:            "this-is-a-name",
										OrganizationID: "orgid",
									},
								},
							}
							data, _ := json.Marshal(resp)
							w.Write(data)
						} else {
							projectID := "projectID"
							resp := &SecretResponse{
								ID:             "d8f29773-3019-4973-9bbc-66327d077fe2",
								Key:            "key",
								Note:           "note",
								OrganizationID: "org",
								Value:          "value",
								ProjectID:      &projectID,
							}
							data, _ := json.Marshal(resp)
							w.Write(data)
						}
					}))

					return &SdkClient{
						apiURL:                server.URL,
						identityURL:           server.URL,
						token:                 "token",
						bitwardenSdkServerURL: server.URL,
						client:                server.Client(),
					}, server.Close
				},
			},
			args: args{
				ctx: context.Background(),
				ref: v1beta1.ExternalSecretDataRemoteRef{
					Key:      "this-is-a-name",
					Property: "projectID",
				},
			},
			want: []byte("value"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sdkClient, serverClose := tt.fields.bitwardenSdkClient(t)
			defer serverClose()

			p := &Provider{
				kube:               tt.fields.kube(),
				namespace:          tt.fields.namespace,
				store:              tt.fields.store,
				bitwardenSdkClient: sdkClient,
			}
			got, err := p.GetSecret(tt.args.ctx, tt.args.ref)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetSecret() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProvider_PushSecret(t *testing.T) {
	type fields struct {
		kube               func() client.Client
		namespace          string
		store              v1beta1.GenericStore
		bitwardenSdkClient func(t *testing.T) (*SdkClient, func())
	}
	type args struct {
		ctx    context.Context
		secret *corev1.Secret
		data   v1beta1.PushSecretData
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "push secret is successful for a none existent remote secret",
			args: args{
				ctx: context.Background(),
				secret: &corev1.Secret{
					Data: map[string][]byte{
						"key": []byte("value"),
					},
				},
				data: v1alpha1.PushSecretData{
					Match: v1alpha1.PushSecretMatch{
						SecretKey: "key",
						RemoteRef: v1alpha1.PushSecretRemoteRef{
							RemoteKey: "this-is-a-name",
							Property:  "projectID",
						},
					},
				},
			},
			fields: fields{
				kube: func() client.Client {
					return fake.NewFakeClient()
				},
				namespace: "default",
				store: &v1beta1.SecretStore{
					Spec: v1beta1.SecretStoreSpec{
						Provider: &v1beta1.SecretStoreProvider{
							BitwardenSecretsManager: &v1beta1.BitwardenSecretsManagerProvider{
								OrganizationID: "orgid",
							},
						},
					},
				},
				bitwardenSdkClient: func(t *testing.T) (*SdkClient, func()) {
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						switch {
						case strings.Contains(r.URL.String(), "rest/api/1/secrets"):
							resp := &SecretIdentifiersResponse{
								Data: []SecretIdentifierResponse{
									{
										ID:             "d8f29773-3019-4973-9bbc-66327d077fe2",
										Key:            "this-is-a-name",
										OrganizationID: "orgid",
									},
								},
							}
							data, _ := json.Marshal(resp)
							w.Write(data)
						case strings.Contains(r.URL.String(), "rest/api/1/secret") && r.Method == http.MethodGet:
							projectID := "projectID"
							resp := &SecretResponse{
								ID:             "d8f29773-3019-4973-9bbc-66327d077fe2",
								Key:            "no-match", // if this is this-is-a-name it would match
								Note:           "",
								OrganizationID: "orgid",
								Value:          "value",
								ProjectID:      &projectID,
							}
							data, _ := json.Marshal(resp)
							w.Write(data)
						case strings.Contains(r.URL.String(), "rest/api/1/secret") && r.Method == http.MethodPost:
							body := &SecretCreateRequest{}
							decoder := json.NewDecoder(r.Body)
							require.NoError(t, decoder.Decode(body))

							assert.Equal(t, &SecretCreateRequest{
								Key:            "this-is-a-name",
								Note:           "",
								OrganizationID: "orgid",
								ProjectIDS:     []string{"projectID"},
								Value:          "value",
							}, body)

							// write something back so we don't fail on Create.
							w.Write([]byte(`{}`))
						}
					}))

					return &SdkClient{
						apiURL:                server.URL,
						identityURL:           server.URL,
						token:                 "token",
						bitwardenSdkServerURL: server.URL,
						client:                server.Client(),
					}, server.Close
				},
			},
		},
		{
			name: "push secret is successful for a existing remote secret but only the value differs will call update",
			args: args{
				ctx: context.Background(),
				secret: &corev1.Secret{
					Data: map[string][]byte{
						"key": []byte("new-value"),
					},
				},
				data: v1alpha1.PushSecretData{
					Match: v1alpha1.PushSecretMatch{
						SecretKey: "key",
						RemoteRef: v1alpha1.PushSecretRemoteRef{
							RemoteKey: "this-is-a-name",
							Property:  "projectID",
						},
					},
				},
			},
			fields: fields{
				kube: func() client.Client {
					return fake.NewFakeClient()
				},
				namespace: "default",
				store: &v1beta1.SecretStore{
					Spec: v1beta1.SecretStoreSpec{
						Provider: &v1beta1.SecretStoreProvider{
							BitwardenSecretsManager: &v1beta1.BitwardenSecretsManagerProvider{
								OrganizationID: "orgid",
							},
						},
					},
				},
				bitwardenSdkClient: func(t *testing.T) (*SdkClient, func()) {
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						switch {
						case strings.Contains(r.URL.String(), "rest/api/1/secrets"):
							resp := &SecretIdentifiersResponse{
								Data: []SecretIdentifierResponse{
									{
										ID:             "d8f29773-3019-4973-9bbc-66327d077fe2",
										Key:            "this-is-a-name",
										OrganizationID: "orgid",
									},
								},
							}
							data, _ := json.Marshal(resp)
							w.Write(data)
						case strings.Contains(r.URL.String(), "rest/api/1/secret") && r.Method == http.MethodGet:
							projectID := "projectID"
							resp := &SecretResponse{
								ID:             "d8f29773-3019-4973-9bbc-66327d077fe2",
								Key:            "this-is-a-name",
								Note:           "",
								OrganizationID: "orgid",
								Value:          "value",
								ProjectID:      &projectID,
							}
							data, _ := json.Marshal(resp)
							w.Write(data)
						case strings.Contains(r.URL.String(), "rest/api/1/secret") && r.Method == http.MethodPut:
							body := &SecretCreateRequest{}
							decoder := json.NewDecoder(r.Body)
							require.NoError(t, decoder.Decode(body))

							assert.Equal(t, &SecretCreateRequest{
								Key:            "this-is-a-name",
								Note:           "",
								OrganizationID: "orgid",
								ProjectIDS:     []string{"projectID"},
								Value:          "new-value",
							}, body)

							// write something back so we don't fail on Create.
							w.Write([]byte(`{}`))
						}
					}))

					return &SdkClient{
						apiURL:                server.URL,
						identityURL:           server.URL,
						token:                 "token",
						bitwardenSdkServerURL: server.URL,
						client:                server.Client(),
					}, server.Close
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sdkClient, cancel := tt.fields.bitwardenSdkClient(t)
			defer cancel()
			p := &Provider{
				kube:               tt.fields.kube(),
				namespace:          tt.fields.namespace,
				store:              tt.fields.store,
				bitwardenSdkClient: sdkClient,
			}

			if err := p.PushSecret(tt.args.ctx, tt.args.secret, tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("PushSecret() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func XTestProvider_SecretExists(t *testing.T) {
	type fields struct {
		kube               client.Client
		namespace          string
		store              v1beta1.GenericStore
		bitwardenSdkClient *SdkClient
	}
	type args struct {
		ctx context.Context
		ref v1beta1.PushSecretRemoteRef
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Provider{
				kube:               tt.fields.kube,
				namespace:          tt.fields.namespace,
				store:              tt.fields.store,
				bitwardenSdkClient: tt.fields.bitwardenSdkClient,
			}
			got, err := p.SecretExists(tt.args.ctx, tt.args.ref)
			if (err != nil) != tt.wantErr {
				t.Errorf("SecretExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SecretExists() got = %v, want %v", got, tt.want)
			}
		})
	}
}
