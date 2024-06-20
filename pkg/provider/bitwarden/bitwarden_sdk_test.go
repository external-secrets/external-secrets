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
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestSdkClient_CreateSecret(t *testing.T) {
	type fields struct {
		apiURL                func(c *httptest.Server) string
		identityURL           func(c *httptest.Server) string
		bitwardenSdkServerURL func(c *httptest.Server) string
		token                 string
		testServer            func(response any) *httptest.Server
		response              any
	}
	type args struct {
		ctx       context.Context
		createReq SecretCreateRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *SecretResponse
		wantErr bool
	}{
		{
			name: "create secret is successful",
			fields: fields{
				testServer: func(response any) *httptest.Server {
					testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						data, err := json.Marshal(response)
						if err != nil {
							http.Error(w, err.Error(), http.StatusInternalServerError)

							return
						}

						w.Write(data)
					}))

					return testServer
				},
				apiURL: func(c *httptest.Server) string {
					return c.URL
				},
				identityURL: func(c *httptest.Server) string {
					return c.URL
				},
				bitwardenSdkServerURL: func(c *httptest.Server) string {
					return c.URL
				},
				token: "token",
				response: &SecretResponse{
					ID:             "id",
					Key:            "key",
					Note:           "note",
					OrganizationID: "orgID",
					RevisionDate:   "2024-04-04",
					Value:          "value",
				},
			},
			args: args{
				ctx: context.Background(),
				createReq: SecretCreateRequest{
					Key:            "key",
					Note:           "note",
					OrganizationID: "orgID",
					ProjectIDS:     []string{"projectID"},
					Value:          "value",
				},
			},
			want: &SecretResponse{
				ID:             "id",
				Key:            "key",
				Note:           "note",
				OrganizationID: "orgID",
				RevisionDate:   "2024-04-04",
				Value:          "value",
			},
		},
		{
			name: "create secret fails",
			fields: fields{
				testServer: func(response any) *httptest.Server {
					testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						http.Error(w, "nope", http.StatusInternalServerError)
					}))

					return testServer
				},
				apiURL: func(c *httptest.Server) string {
					return c.URL
				},
				identityURL: func(c *httptest.Server) string {
					return c.URL
				},
				bitwardenSdkServerURL: func(c *httptest.Server) string {
					return c.URL
				},
				token: "token",
				response: &SecretResponse{
					ID:             "id",
					Key:            "key",
					Note:           "note",
					OrganizationID: "orgID",
					RevisionDate:   "2024-04-04",
					Value:          "value",
				},
			},
			args: args{
				ctx: context.Background(),
				createReq: SecretCreateRequest{
					Key:            "key",
					Note:           "note",
					OrganizationID: "orgID",
					ProjectIDS:     []string{"projectID"},
					Value:          "value",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.fields.testServer(tt.fields.response)
			defer server.Close()
			s := &SdkClient{
				apiURL:                tt.fields.apiURL(server),
				identityURL:           tt.fields.identityURL(server),
				bitwardenSdkServerURL: tt.fields.bitwardenSdkServerURL(server),
				token:                 tt.fields.token,
				client:                server.Client(),
			}
			got, err := s.CreateSecret(tt.args.ctx, tt.args.createReq)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateSecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateSecret() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSdkClient_DeleteSecret(t *testing.T) {
	type fields struct {
		apiURL                string
		identityURL           string
		token                 string
		bitwardenSdkServerURL string
		client                *http.Client
	}
	type args struct {
		ctx context.Context
		ids []string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *SecretsDeleteResponse
		wantErr bool
	}{
		{
			name: "delete secret is successful",
		},
		{
			name: "delete secret is unsuccessful",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SdkClient{
				apiURL:                tt.fields.apiURL,
				identityURL:           tt.fields.identityURL,
				token:                 tt.fields.token,
				bitwardenSdkServerURL: tt.fields.bitwardenSdkServerURL,
				client:                tt.fields.client,
			}
			got, err := s.DeleteSecret(tt.args.ctx, tt.args.ids)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteSecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DeleteSecret() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSdkClient_GetSecret(t *testing.T) {
	type fields struct {
		apiURL                string
		identityURL           string
		token                 string
		bitwardenSdkServerURL string
		client                *http.Client
	}
	type args struct {
		ctx context.Context
		id  string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *SecretResponse
		wantErr bool
	}{
		{
			name: "get secret is successful",
		},
		{
			name: "get secret is unsuccessful",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SdkClient{
				apiURL:                tt.fields.apiURL,
				identityURL:           tt.fields.identityURL,
				token:                 tt.fields.token,
				bitwardenSdkServerURL: tt.fields.bitwardenSdkServerURL,
				client:                tt.fields.client,
			}
			got, err := s.GetSecret(tt.args.ctx, tt.args.id)
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

func TestSdkClient_ListSecrets(t *testing.T) {
	type fields struct {
		apiURL                string
		identityURL           string
		token                 string
		bitwardenSdkServerURL string
		client                *http.Client
	}
	type args struct {
		ctx            context.Context
		organizationID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *SecretIdentifiersResponse
		wantErr bool
	}{
		{
			name: "list secrets is successful",
		},
		{
			name: "list secrets is unsuccessful",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SdkClient{
				apiURL:                tt.fields.apiURL,
				identityURL:           tt.fields.identityURL,
				token:                 tt.fields.token,
				bitwardenSdkServerURL: tt.fields.bitwardenSdkServerURL,
				client:                tt.fields.client,
			}
			got, err := s.ListSecrets(tt.args.ctx, tt.args.organizationID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListSecrets() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ListSecrets() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSdkClient_UpdateSecret(t *testing.T) {
	type fields struct {
		apiURL                string
		identityURL           string
		token                 string
		bitwardenSdkServerURL string
		client                *http.Client
	}
	type args struct {
		ctx    context.Context
		putReq SecretPutRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *SecretResponse
		wantErr bool
	}{
		{
			name: "update secret is successful",
		},
		{
			name: "update secret is unsuccessful",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SdkClient{
				apiURL:                tt.fields.apiURL,
				identityURL:           tt.fields.identityURL,
				token:                 tt.fields.token,
				bitwardenSdkServerURL: tt.fields.bitwardenSdkServerURL,
				client:                tt.fields.client,
			}
			got, err := s.UpdateSecret(tt.args.ctx, tt.args.putReq)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateSecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UpdateSecret() got = %v, want %v", got, tt.want)
			}
		})
	}
}
