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
package device42

import (
	"bytes"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	fakedevice42 "github.com/external-secrets/external-secrets/pkg/provider/device42/fake"
)

const device42PasswordID = "12345"

func d42PasswordResponse() D42PasswordResponse {
	return D42PasswordResponse{Passwords: []D42Password{d42Password()}}
}

func d42Password() D42Password {
	return D42Password{
		Password: "test_Password",
		ID:       12345,
	}
}

func TestDevice42ApiGetSecret(t *testing.T) {
	type fields struct {
		funcStack []func(req *http.Request) (*http.Response, error)
	}
	type args struct {
		secretID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    D42Password
		wantErr bool
	}{
		{
			name: "get secret",
			fields: fields{
				funcStack: []func(req *http.Request) (*http.Response, error){
					createResponder(d42PasswordResponse(), true), //nolint:bodyclose
				},
			},
			args: args{
				secretID: device42PasswordID,
			},
			want:    d42Password(),
			wantErr: false,
		},
		{
			name: "bad response on secret entry",
			fields: fields{
				funcStack: []func(req *http.Request) (*http.Response, error){
					createResponder([]byte("bad response body"), false), //nolint:bodyclose // linters bug
				},
			},
			args: args{
				secretID: device42PasswordID,
			},
			want:    D42Password{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &API{
				client: &fakedevice42.MockClient{
					FuncStack: tt.fields.funcStack,
				},
				baseURL:  "localhost",
				hostPort: "8714",
				password: "test",
				username: "test",
			}
			got, err := api.GetSecret(tt.args.secretID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Device42.GetSecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Device42.GetSecret() = %v, want %v", got, tt.want)
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
			Body:       &closeableBuffer{bytes.NewReader(payloadBytes)},
		}
		return &res, nil
	}
}

type closeableBuffer struct {
	*bytes.Reader
}

func (cb *closeableBuffer) Close() error {
	// Here you can add any cleanup code if needed
	return nil
}
