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
package passworddepot

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"testing"
	"time"

	fakepassworddepot "github.com/external-secrets/external-secrets/pkg/provider/passworddepot/fake"
)

const fingerprint1 = "53fe39bd-0d5c-4b46-83b3-122fef14364e"
const mySecret = "my-secret"
const someDB = "some-db"

var (
	mockDatabaseList = Databases{
		Databases: []struct {
			Name         string    `json:"name"`
			Fingerprint  string    `json:"fingerprint"`
			Date         time.Time `json:"date"`
			Rights       string    `json:"rights"`
			Reasondelete string    `json:"reasondelete"`
		}{
			{
				Name:        someDB,
				Fingerprint: "434da246-c165-499b-8996-6ee2a9673429",
			},
		},
	}

	mockDatabaseEntries = DatabaseEntries{
		Entries: []Entry{
			{
				Name:        mySecret,
				Fingerprint: fingerprint1,
			},
		},
	}
)

func TestPasswortDepotApiListDatabases(t *testing.T) {
	type fields struct {
		funcStack []func(req *http.Request) (*http.Response, error)
	}
	tests := []struct {
		name    string
		fields  fields
		want    Databases
		wantErr bool
	}{
		{
			name: "list databases",
			fields: fields{
				funcStack: []func(req *http.Request) (*http.Response, error){
					createResponder(mockDatabaseList, true), //nolint:bodyclose // linters bug
				},
			},
			want: Databases{
				Databases: []struct {
					Name         string    `json:"name"`
					Fingerprint  string    `json:"fingerprint"`
					Date         time.Time `json:"date"`
					Rights       string    `json:"rights"`
					Reasondelete string    `json:"reasondelete"`
				}{
					{
						Name:        someDB,
						Fingerprint: "434da246-c165-499b-8996-6ee2a9673429",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "bad response body",
			fields: fields{
				funcStack: []func(req *http.Request) (*http.Response, error){
					createResponder([]byte("bad response"), false), //nolint:bodyclose // linters bug
				},
			},
			want:    Databases{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &API{
				client: &fakepassworddepot.MockClient{
					FuncStack: tt.fields.funcStack,
				},
				baseURL:  "localhost",
				hostPort: "8714",
				password: "test",
				username: "test",
				secret: &AccessData{
					ClientID:    "12345",
					AccessToken: "f02a1f7c-7629-422e-96f5-a98b29da1287",
				},
			}
			got, err := api.ListDatabases()
			if (err != nil) != tt.wantErr {
				t.Errorf("PasswortDepotApi.ListDatabases() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PasswortDepotApi.ListDatabases() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPasswortDepotApiGetSecret(t *testing.T) {
	type fields struct {
		funcStack []func(req *http.Request) (*http.Response, error)
	}
	type args struct {
		database   string
		secretName string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    SecretEntry
		wantErr bool
	}{
		{
			name: "get secret",
			fields: fields{
				funcStack: []func(req *http.Request) (*http.Response, error){
					createResponder(mockDatabaseList, true),    //nolint:bodyclose // linters bug
					createResponder(mockDatabaseEntries, true), //nolint:bodyclose // linters bug
					createResponder(SecretEntry{ //nolint:bodyclose // linters bug
						Name:        mySecret,
						Fingerprint: fingerprint1,
						Pass:        "yery53cr3t",
					}, true),
				},
			},
			args: args{
				database:   someDB,
				secretName: mySecret,
			},
			want: SecretEntry{
				Name:        mySecret,
				Fingerprint: fingerprint1,
				Pass:        "yery53cr3t",
			},
			wantErr: false,
		},
		{
			name: "get nested secret",
			fields: fields{
				funcStack: []func(req *http.Request) (*http.Response, error){
					createResponder(mockDatabaseList, true), //nolint:bodyclose // linters bug
					createResponder(DatabaseEntries{ //nolint:bodyclose // linters bug
						Entries: []Entry{
							{
								Name:        "Production",
								Fingerprint: "33f918ae-ec60-41cb-8131-67d8f21eef4c",
							},
						},
					}, true),
					createResponder(mockDatabaseEntries, true), //nolint:bodyclose // linters bug
					createResponder(SecretEntry{ //nolint:bodyclose // linters bug
						Name:        mySecret,
						Fingerprint: fingerprint1,
						Pass:        "yery53cr3t",
					}, true),
				},
			},
			args: args{
				database:   someDB,
				secretName: "Production.my-secret",
			},
			want: SecretEntry{
				Name:        mySecret,
				Fingerprint: fingerprint1,
				Pass:        "yery53cr3t",
			},
			wantErr: false,
		},
		{
			name: "bad response body on database entries",
			fields: fields{
				funcStack: []func(req *http.Request) (*http.Response, error){
					createResponder(mockDatabaseList, true),    //nolint:bodyclose // linters bug
					createResponder([]byte("bad body"), false), //nolint:bodyclose // linters bug
				},
			},
			args: args{
				database:   someDB,
				secretName: mySecret,
			},
			want:    SecretEntry{},
			wantErr: true,
		},
		{
			name: "bad response on secret entry",
			fields: fields{
				funcStack: []func(req *http.Request) (*http.Response, error){
					createResponder(mockDatabaseList, true),             //nolint:bodyclose // linters bug
					createResponder(mockDatabaseEntries, true),          //nolint:bodyclose // linters bug
					createResponder([]byte("bad response body"), false), //nolint:bodyclose // linters bug
				},
			},
			args: args{
				database:   someDB,
				secretName: mySecret,
			},
			want:    SecretEntry{},
			wantErr: true,
		},
		{
			name: "no secret with name",
			fields: fields{
				funcStack: []func(req *http.Request) (*http.Response, error){
					createResponder(mockDatabaseList, true),                    //nolint:bodyclose // linters bug
					createResponder(DatabaseEntries{Entries: []Entry{}}, true), //nolint:bodyclose // linters bug
				},
			},
			args: args{
				database:   someDB,
				secretName: mySecret,
			},
			want:    SecretEntry{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &API{
				client: &fakepassworddepot.MockClient{
					FuncStack: tt.fields.funcStack,
				},
				baseURL:  "localhost",
				hostPort: "8714",
				password: "test",
				username: "test",
				secret: &AccessData{
					ClientID:    "12345",
					AccessToken: "f02a1f7c-7629-422e-96f5-a98b29da1287",
				},
			}
			got, err := api.GetSecret(tt.args.database, tt.args.secretName)
			if (err != nil) != tt.wantErr {
				t.Errorf("PasswortDepotApi.GetSecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PasswortDepotApi.GetSecret() = %v, want %v", got, tt.want)
			}
		})
	}
}

func createResponder(payload any, withMarshal bool) func(*http.Request) (*http.Response, error) {
	return func(req *http.Request) (*http.Response, error) {
		var payloadBytes []byte
		if withMarshal {
			payloadBytes, _ = json.Marshal(payload)
		} else {
			payloadBytes = payload.([]byte)
		}
		res := http.Response{
			Status:     "OK",
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(payloadBytes)),
		}
		return &res, nil
	}
}
