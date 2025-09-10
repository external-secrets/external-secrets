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
package secretserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/DelineaXPM/tss-sdk-go/v3/server"
	"github.com/stretchr/testify/assert"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

var (
	errNotFound = errors.New("not found")
)

type fakeAPI struct {
	secrets []*server.Secret
}

const (
	usernameSlug = "username"
	passwordSlug = "password"
)

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

func (f *fakeAPI) SecretByPath(path string) (*server.Secret, error) {
	for _, s := range f.secrets {
		if "/"+s.Name == path {
			return s, nil
		}
	}
	return nil, errNotFound
}

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
	s.Fields[0].Slug = usernameSlug
	s.Fields[1].FieldName = "Password"
	s.Fields[1].Slug = passwordSlug
	s.Fields[1].ItemValue = "passwordvalue"
	return s
}

func createTestFolderSecret(id, folderId int) *server.Secret {
	s := new(server.Secret)
	s.FolderID = folderId
	s.ID = id
	s.Name = "FolderSecretname"
	s.Fields = make([]server.SecretField, 2)
	s.Fields[0].ItemValue = "usernamevalue"
	s.Fields[0].FieldName = "Username"
	s.Fields[0].Slug = usernameSlug
	s.Fields[1].FieldName = "Password"
	s.Fields[1].Slug = passwordSlug
	s.Fields[1].ItemValue = "passwordvalue"
	return s
}

func createPlainTextSecret(id int) *server.Secret {
	s := new(server.Secret)
	s.ID = id
	s.Name = "PlainTextSecret"
	s.Fields = make([]server.SecretField, 1)
	s.Fields[0].FieldName = "Content"
	s.Fields[0].Slug = "content"
	s.Fields[0].ItemValue = `non-json-secret-value`
	return s
}

func createNilFieldsSecret(id int) *server.Secret {
	s := new(server.Secret)
	s.ID = id
	s.Name = "NilFieldsSecret"
	s.Fields = nil
	return s
}

func createEmptyFieldsSecret(id int) *server.Secret {
	s := new(server.Secret)
	s.ID = id
	s.Name = "EmptyFieldsSecret"
	s.Fields = []server.SecretField{}
	return s
}

func newTestClient() esv1.SecretsClient {
	return &client{
		api: &fakeAPI{
			secrets: []*server.Secret{
				createSecret(1000, "{ \"user\": \"robertOppenheimer\", \"password\": \"badPassword\",\"server\":\"192.168.1.50\"}"),
				createSecret(2000, "{ \"user\": \"helloWorld\", \"password\": \"badPassword\",\"server\":[ \"192.168.1.50\",\"192.168.1.51\"] }"),
				createSecret(3000, "{ \"user\": \"chuckTesta\", \"password\": \"badPassword\",\"server\":\"192.168.1.50\"}"),
				createTestSecretFromCode(4000),
				createPlainTextSecret(5000),
				createSecret(6000, "{ \"user\": \"betaTest\", \"password\": \"badPassword\" }"),
				createNilFieldsSecret(7000),
				createEmptyFieldsSecret(8000),
				createTestFolderSecret(9000, 4),
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
	jsonStr3, _ := json.Marshal(createPlainTextSecret(5000))
	jsonStr4, _ := json.Marshal(createTestFolderSecret(9000, 4))

	testCases := map[string]struct {
		ref  esv1.ExternalSecretDataRemoteRef
		want []byte
		err  error
	}{
		"incorrect key returns nil and error": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "0",
			},
			want: []byte(nil),
			err:  errNotFound,
		},
		"key = 'secret name' and user property returns a single value": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      "ESO-test-secret",
				Property: "user",
			},
			want: []byte(`robertOppenheimer`),
		},
		"Secret from JSON: key and password property returns a single value": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      "1000",
				Property: "password",
			},
			want: []byte(`badPassword`),
		},
		"Secret from JSON: key and nested property returns a single value": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      "2000",
				Property: "server.1",
			},
			want: []byte(`192.168.1.51`),
		},
		"Secret from JSON: existent key with non-existing property": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      "3000",
				Property: "foo.bar",
			},
			err: esv1.NoSecretError{},
		},
		"Secret from JSON: existent 'name' key with no property": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "1000",
			},
			want: jsonStr,
		},
		"Secret from code: existent key with no property": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "4000",
			},
			want: jsonStr2,
		},
		"Secret from code: key and username fieldnamereturns a single value": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      "4000",
				Property: "Username",
			},
			want: []byte(`usernamevalue`),
		},
		"Plain text secret: existent key with no property": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "5000",
			},
			want: jsonStr3,
		},
		"Plain text secret: key with property returns expected value": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      "5000",
				Property: "Content",
			},
			want: []byte(`non-json-secret-value`),
		},
		"Secret from code: valid ItemValue but incorrect property returns noSecretError": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      "6000",
				Property: "missing",
			},
			want: []byte(nil),
			err:  esv1.NoSecretError{},
		},
		"Secret from code: valid ItemValue but nil Fields returns nil": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "7000",
			},
			want: []byte(nil),
		},
		"Secret from code: empty Fields returns noSecretError": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      "8000",
				Property: "missing",
			},
			want: []byte(nil),
			err:  esv1.NoSecretError{},
		},
		"Secret from code: 'name' and password slug returns a single value": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      "Secretname",
				Property: "password",
			},
			want: []byte(`passwordvalue`),
		},
		"Secret from code: 'name' not found and password slug returns error": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      "Secretnameerror",
				Property: "password",
			},
			want: []byte(nil),
			err:  errNotFound,
		},
		"Secret from code: 'name' found and non-existent attribute slug returns noSecretError": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      "Secretname",
				Property: "passwordkey",
			},
			want: []byte(nil),
			err:  esv1.NoSecretError{},
		},
		"Secret by path: valid path returns secret": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "/FolderSecretname",
			},
			want: jsonStr4,
		},
		"Secret by path: invalid path returns error": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "/invalid/secret/path",
			},
			want: []byte(nil),
			err:  errNotFound,
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
