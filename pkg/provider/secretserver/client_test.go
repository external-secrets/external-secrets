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
package secretserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/DelineaXPM/tss-sdk-go/v2/server"
	"github.com/stretchr/testify/assert"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

var (
	errNotFound = errors.New("not found")
)

type fakeAPI struct {
	secrets []*server.Secret
}

func (f *fakeAPI) Secret(id int) (*server.Secret, error) {
	for _, s := range f.secrets {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, errNotFound
}

func (f *fakeAPI) Secrets(searchText, _ string) ([]server.Secret, error) {
	secret := make([]server.Secret, 1)
	for _, s := range f.secrets {
		if s.Name == searchText {
			secret[0] = *s
			return secret, nil
		}
	}
	return nil, errNotFound
}

// createSecret assembles a server.Secret from file test_data.json.
func createSecret(id int, itemValue string) *server.Secret {
	s, _ := getJSONData()
	s.ID = id
	s.Fields[0].ItemValue = itemValue
	return s
}

func getJSONData() (*server.Secret, error) {
	var s = &server.Secret{}
	jsonFile, err := os.Open("test_data.json")
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)
	err = json.Unmarshal(byteValue, &s)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func createTestSecretFromCode(id int) *server.Secret {
	s := new(server.Secret)
	s.ID = id
	s.Name = "Secretname"
	s.Fields = make([]server.SecretField, 2)
	s.Fields[0].ItemValue = "usernamevalue"
	s.Fields[0].FieldName = "Username"
	s.Fields[0].Slug = "username"
	s.Fields[1].FieldName = "Password"
	s.Fields[1].Slug = "password"
	s.Fields[1].ItemValue = "passwordvalue"
	return s
}

func newTestClient() esv1beta1.SecretsClient {
	return &client{
		api: &fakeAPI{
			secrets: []*server.Secret{
				createSecret(1000, "{ \"user\": \"robertOppenheimer\", \"password\": \"badPassword\",\"server\":\"192.168.1.50\"}"),
				createSecret(2000, "{ \"user\": \"helloWorld\", \"password\": \"badPassword\",\"server\":[ \"192.168.1.50\",\"192.168.1.51\"] }"),
				createSecret(3000, "{ \"user\": \"chuckTesta\", \"password\": \"badPassword\",\"server\":\"192.168.1.50\"}"),
				createTestSecretFromCode(4000),
			},
		},
	}
}

func TestGetSecretSecretServer(t *testing.T) {
	ctx := context.Background()
	c := newTestClient()
	s, _ := getJSONData()
	jsonStr, _ := json.Marshal(s)
	jsonStr2, _ := json.Marshal(createTestSecretFromCode(4000))

	testCases := map[string]struct {
		ref  esv1beta1.ExternalSecretDataRemoteRef
		want []byte
		err  error
	}{
		"incorrect key returns nil and error": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key: "0",
			},
			want: []byte(nil),
			err:  errNotFound,
		},
		"key = 'secret name' and user property returns a single value": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:      "ESO-test-secret",
				Property: "user",
			},
			want: []byte(`robertOppenheimer`),
		},
		"Secret from JSON: key and password property returns a single value": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:      "1000",
				Property: "password",
			},
			want: []byte(`badPassword`),
		},
		"Secret from JSON: key and nested property returns a single value": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:      "2000",
				Property: "server.1",
			},
			want: []byte(`192.168.1.51`),
		},
		"Secret from JSON: existent key with non-existing propery": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:      "3000",
				Property: "foo.bar",
			},
			err: esv1beta1.NoSecretError{},
		},
		"Secret from JSON: existent 'name' key with no propery": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key: "1000",
			},
			want: jsonStr,
		},
		"Secret from code: existent key with no property": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key: "4000",
			},
			want: jsonStr2,
		},
		"Secret from code: key and username fieldnamereturns a single value": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:      "4000",
				Property: "Username",
			},
			want: []byte(`usernamevalue`),
		},
		"Secret from code: 'name' and password slug returns a single value": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:      "Secretname",
				Property: "password",
			},
			want: []byte(`passwordvalue`),
		},
		"Secret from code: 'name' not found and password slug returns error": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:      "Secretnameerror",
				Property: "password",
			},
			want: []byte(nil),
			err:  errNotFound,
		},
		"Secret from code: 'name' found and non-existent attribute slug returns noSecretError": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key:      "Secretname",
				Property: "passwordkey",
			},
			want: []byte(nil),
			err:  esv1beta1.NoSecretError{},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, err := c.GetSecret(ctx, tc.ref)

			if tc.err == nil {
				assert.NoError(t, err)
				assert.Equal(t, tc.want, got)
			} else {
				assert.Nil(t, got)
				assert.ErrorIs(t, err, tc.err)
				assert.Equal(t, tc.err, err)
			}
		})
	}
}
