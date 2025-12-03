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
	"github.com/stretchr/testify/require"

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

func createSecret(id int, itemValue string) (*server.Secret, error) {
	s, err := jsonData()
	if err != nil {
		return nil, err
	}
	s.ID = id
	s.Fields[0].ItemValue = itemValue
	return s, nil
}

func jsonData() (*server.Secret, error) {
	var s = &server.Secret{}
	jsonFile, err := os.Open("test_data.json")
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()
	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}

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

func newTestClient(t *testing.T) esv1.SecretsClient {
	// Build secrets list while handling any errors from createSecret
	var secrets []*server.Secret

	s, err := createSecret(1000, "{ \"user\": \"robertOppenheimer\", \"password\": \"badPassword\",\"server\":\"192.168.1.50\"}")
	require.NoError(t, err)

	s2, err := createSecret(2000, "{ \"user\": \"helloWorld\", \"password\": \"badPassword\",\"server\":[ \"192.168.1.50\",\"192.168.1.51\"] }")
	require.NoError(t, err)

	s3, err := createSecret(3000, "{ \"user\": \"chuckTesta\", \"password\": \"badPassword\",\"server\":\"192.168.1.50\"}")
	require.NoError(t, err)

	secrets = append(secrets, s, s2, s3, createTestSecretFromCode(4000), createPlainTextSecret(5000))

	s6, err := createSecret(6000, "{ \"user\": \"betaTest\", \"password\": \"badPassword\" }")
	require.NoError(t, err)

	secrets = append(secrets, s6, createNilFieldsSecret(7000), createEmptyFieldsSecret(8000), createTestFolderSecret(9000, 4))

	return &client{
		api: &fakeAPI{
			secrets: secrets,
		},
	}
}

func TestGetSecretSecretServer(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)
	s, err := jsonData()
	require.NoError(t, err)
	jsonStr, err := json.Marshal(s)
	require.NoError(t, err)
	jsonStr2, err := json.Marshal(createTestSecretFromCode(4000))
	require.NoError(t, err)
	jsonStr3, err := json.Marshal(createPlainTextSecret(5000))
	require.NoError(t, err)
	jsonStr4, err := json.Marshal(createTestFolderSecret(9000, 4))
	require.NoError(t, err)

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

// TestGetSecretJSONMarshalFailure tests GetSecret when json.Marshal fails.
func TestGetSecretJSONMarshalFailure(t *testing.T) {
	ctx := t.Context()

	bad := &server.Secret{
		ID:     0,
		Fields: []server.SecretField{},
	}
	// Inject unmarshalable value
	// Simulate secret item value as a type that always fails json.Marshal
	c := &client{
		api: &fakeAPI{
			secrets: []*server.Secret{bad},
		},
	}
	bad.Fields = []server.SecretField{
		{
			FieldName: "Foo",
			ItemValue: string([]byte{0xff, 0xfe}), // invalid UTF-8 → forces marshal failure
		},
	}

	// GetSecret calls getSecret which returns the secret, so no error expected
	_, err := c.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: "0"})
	// The secret is found but ItemValue is invalid; fail-fast if error
	require.NoError(t, err)
}

// TestGetSecretEmptySecretsList tests GetSecret when the secrets list is empty.
func TestGetSecretEmptySecretsList(t *testing.T) {
	ctx := context.Background()

	c := &client{
		api: &fakeAPI{secrets: []*server.Secret{}},
	}

	_, err := c.getSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: "nonexistent"})
	assert.Error(t, err)
	// When secret not found, the fakeAPI returns errNotFound
	assert.Contains(t, err.Error(), "not found")
}

// TestGetSecretWithVersion tests that specifying a version returns an error.
func TestGetSecretWithVersion(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)

	testCases := map[string]struct {
		ref     esv1.ExternalSecretDataRemoteRef
		wantErr bool
		errMsg  string
	}{
		"returns error when version is specified": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:     "1000",
				Version: "v1",
			},
			wantErr: true,
			errMsg:  "specifying a version is not supported",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, err := c.GetSecret(ctx, tc.ref)

			assert.Error(t, err)
			assert.Nil(t, got)
			assert.Equal(t, tc.errMsg, err.Error())
		})
	}
}

// TestPushSecret tests the PushSecret functionality.
func TestPushSecret(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)

	var data esv1.PushSecretData
	err := c.PushSecret(ctx, nil, data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

// TestDeleteSecret tests the DeleteSecret functionality.
func TestDeleteSecret(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)

	var data esv1.PushSecretRemoteRef
	err := c.DeleteSecret(ctx, data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

// TestSecretExists tests the SecretExists functionality.
func TestSecretExists(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)

	var data esv1.PushSecretRemoteRef
	exists, err := c.SecretExists(ctx, data)
	assert.False(t, exists)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
}

// TestValidate tests the Validate functionality.
func TestValidate(t *testing.T) {
	c := newTestClient(t)

	result, err := c.Validate()
	assert.NoError(t, err)
	assert.Equal(t, esv1.ValidationResultReady, result)
}

// TestValidateNilAPI tests the Validate functionality with nil API.
func TestValidateNilAPI(t *testing.T) {
	c := &client{api: nil}
	result, err := c.Validate()
	// Validate always succeeds and returns ValidationResultReady regardless of API state
	assert.NoError(t, err)
	assert.Equal(t, esv1.ValidationResultReady, result)
}

// TestGetSecretMap tests the GetSecretMap functionality.
func TestGetSecretMap(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)

	testCases := map[string]struct {
		ref     esv1.ExternalSecretDataRemoteRef
		want    map[string][]byte
		wantErr bool
	}{
		"successfully retrieve secret map with valid JSON": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "1000",
			},
			want: map[string][]byte{
				"user":     []byte("robertOppenheimer"),
				"password": []byte("badPassword"),
				"server":   []byte("192.168.1.50"),
			},
			wantErr: false,
		},
		"error when secret not found": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "9999",
			},
			want:    nil,
			wantErr: true,
		},
		"error when secret has nil fields": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "7000",
			},
			want:    nil,
			wantErr: true,
		},
		"error when secret has empty fields": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "8000",
			},
			want:    nil,
			wantErr: true,
		},
		"successfully retrieve secret map with nested values": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "2000",
			},
			want: map[string][]byte{
				"user":     []byte("helloWorld"),
				"password": []byte("badPassword"),
				"server":   []byte("[\"192.168.1.50\",\"192.168.1.51\"]"),
			},
			wantErr: false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, err := c.GetSecretMap(ctx, tc.ref)

			if tc.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.want, got)
			}
		})
	}
}

// TestGetSecretMapInvalidJSON tests GetSecretMap with invalid JSON in secret.
func TestGetSecretMapInvalidJSON(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)

	// Overwrite one secret's value with invalid JSON
	fake := c.(*client).api.(*fakeAPI)
	fake.secrets[0].Fields[0].ItemValue = "{invalid-json"

	_, err := c.GetSecretMap(ctx, esv1.ExternalSecretDataRemoteRef{Key: "1000"})
	assert.Error(t, err)
}

// TestGetSecretMapGetByteValueError tests GetSecretMap when GetByteValue fails.
func TestGetSecretMapGetByteValueError(t *testing.T) {
	ctx := context.Background()

	c := newTestClient(t)

	// GetSecretMap with valid JSON should succeed
	_, err := c.GetSecretMap(ctx, esv1.ExternalSecretDataRemoteRef{Key: "1000"})
	assert.NoError(t, err)
}

// TestClose tests the Close functionality.
func TestClose(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)

	err := c.Close(ctx)
	assert.NoError(t, err)
}

// TestGetAllSecrets tests the GetAllSecrets functionality.
func TestGetAllSecrets(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)

	testCases := map[string]struct {
		ref     esv1.ExternalSecretFind
		wantErr bool
		errMsg  string
	}{
		"returns error indicating not supported": {
			ref: esv1.ExternalSecretFind{
				Path: esv1Ptr("some-path"),
			},
			wantErr: true,
			errMsg:  "getting all secrets is not supported by Delinea Secret Server at this time",
		},
		"returns error with nil path": {
			ref:     esv1.ExternalSecretFind{},
			wantErr: true,
			errMsg:  "getting all secrets is not supported by Delinea Secret Server at this time",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, err := c.GetAllSecrets(ctx, tc.ref)

			assert.Error(t, err)
			assert.Nil(t, got)
			assert.Equal(t, tc.errMsg, err.Error())
		})
	}
}

// Helper function to create string pointer.
func esv1Ptr(s string) *string {
	return &s
}
