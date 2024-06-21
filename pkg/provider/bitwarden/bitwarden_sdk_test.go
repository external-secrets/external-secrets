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

// The rest of the tests much look the same, it would be nice if I could find a way
// to nicely unify the tests for all of them.

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
					ProjectIDS:     []string{projectID},
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
					ProjectIDS:     []string{projectID},
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
