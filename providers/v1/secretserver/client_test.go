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
package secretserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/DelineaXPM/tss-sdk-go/v3/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

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
	// Match real SDK behavior: return ([]Secret{}, nil) for zero matches,
	// NOT (nil, errNotFound). The real SDK's searchResources returns an empty
	// SearchResult.Records slice and make([]Secret, 0).
	var secrets []server.Secret
	for _, s := range f.secrets {
		if s.Name == searchText {
			secrets = append(secrets, *s)
		}
	}
	if secrets == nil {
		secrets = []server.Secret{}
	}
	return secrets, nil
}

func (f *fakeAPI) SecretByPath(path string) (*server.Secret, error) {
	for _, s := range f.secrets {
		if "/"+s.Name == path || s.Name == path {
			return s, nil
		}
	}
	return nil, errNotFound
}

// CreateSecret is a mock implementation of the Secret Server API CreateSecret method.
// It returns a predefined secret based on the SecretTemplateID provided.
func (f *fakeAPI) CreateSecret(secret server.Secret) (*server.Secret, error) {
	if secret.Name == "simulate-create-error" {
		return nil, errors.New("simulated create error")
	}
	secret.ID = len(f.secrets) + 10000

	// Simulate populating FieldName and Slug based on FieldID
	template, _ := f.SecretTemplate(secret.SecretTemplateID)
	if template != nil {
		for i, field := range secret.Fields {
			for _, tField := range template.Fields {
				if tField.SecretTemplateFieldID == field.FieldID {
					secret.Fields[i].Slug = tField.FieldSlugName
					secret.Fields[i].FieldName = tField.Name
				}
			}
		}
	}

	f.secrets = append(f.secrets, &secret)
	return &secret, nil
}

// UpdateSecret is a mock implementation of the Secret Server API UpdateSecret method.
// It returns an error if a predefined test condition is met, otherwise it simulates success.
func (f *fakeAPI) UpdateSecret(secret server.Secret) (*server.Secret, error) {
	for i, s := range f.secrets {
		if s.ID == secret.ID {
			f.secrets[i] = &secret
			return &secret, nil
		}
	}
	return nil, errNotFound
}

// DeleteSecret is a mock implementation of the Secret Server API DeleteSecret method.
// It returns an error if the id corresponds to a simulated failure case.
func (f *fakeAPI) DeleteSecret(id int) error {
	if id == 9999 {
		return errors.New("simulated backend deletion error")
	}
	for i, s := range f.secrets {
		if s.ID == id {
			f.secrets = append(f.secrets[:i], f.secrets[i+1:]...)
			return nil
		}
	}
	return errNotFound
}

// SecretTemplate is a mock implementation of the Secret Server API SecretTemplate method.
// It returns a predefined template or an error based on the requested id.
func (f *fakeAPI) SecretTemplate(id int) (*server.SecretTemplate, error) {
	if id == 999 {
		return nil, errors.New("template not found")
	}
	return &server.SecretTemplate{
		ID:   id,
		Name: "Test Template",
		Fields: []server.SecretTemplateField{
			{
				SecretTemplateFieldID: 1,
				FieldSlugName:         "username",
				Name:                  "Username",
			},
			{
				SecretTemplateFieldID: 2,
				FieldSlugName:         "password",
				Name:                  "Password",
			},
			{
				SecretTemplateFieldID: 3,
				FieldSlugName:         "notes",
				Name:                  "Notes",
			},
		},
	}, nil
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
	var secrets []*server.Secret //nolint:prealloc // populated incrementally

	s, err := createSecret(1000, "{ \"user\": \"robertOppenheimer\", \"password\": \"badPassword\",\"server\":\"192.168.1.50\"}")
	require.NoError(t, err)

	s2, err := createSecret(2000, "{ \"user\": \"helloWorld\", \"password\": \"badPassword\",\"server\":[ \"192.168.1.50\",\"192.168.1.51\"] }")
	require.NoError(t, err)

	s3, err := createSecret(3000, "{ \"user\": \"chuckTesta\", \"password\": \"badPassword\",\"server\":\"192.168.1.50\"}")
	require.NoError(t, err)

	secrets = append(secrets, s, s2, s3, createTestSecretFromCode(4000), createPlainTextSecret(5000))

	s6, err := createSecret(6000, "{ \"user\": \"betaTest\", \"password\": \"badPassword\" }")
	require.NoError(t, err)

	secrets = append(secrets, s6, createNilFieldsSecret(7000), createEmptyFieldsSecret(8000), createTestFolderSecret(9000, 4), createTestFolderSecret(9001, 5))

	// Create a secret for path-based test
	pathSecret := &server.Secret{
		ID:       9002,
		Name:     "/some/path/secret",
		FolderID: 6,
		Fields: []server.SecretField{
			{FieldName: "Password", Slug: "password", ItemValue: "old_path_value"},
		},
	}
	secrets = append(secrets, pathSecret)

	s9999, err := createSecret(9999, "simulated error")
	require.NoError(t, err)
	secrets = append(secrets, s9999)

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
		ref    esv1.ExternalSecretDataRemoteRef
		want   []byte
		err    error
		errMsg string // when set, asserts Contains(err.Error(), errMsg) instead of exact error match
	}{
		"incorrect key returns nil and error": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "0",
			},
			want:   []byte(nil),
			errMsg: errMsgNoMatchingSecrets,
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
		"Secret from code: nil Fields returns error": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "7000",
			},
			want:   []byte(nil),
			errMsg: "secret contains no fields",
		},
		"Secret from code: empty Fields returns error": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "8000",
			},
			want:   []byte(nil),
			errMsg: "secret contains no fields",
		},
		"Secret from code: 'name' and password slug returns a single value": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      "Secretname",
				Property: "password",
			},
			want: []byte(`passwordvalue`),
		},
		"Secret from code: 'name' not found returns no matching secrets error": {
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      "Secretnameerror",
				Property: "password",
			},
			want:   []byte(nil),
			errMsg: errMsgNoMatchingSecrets,
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
			want:   []byte(nil),
			errMsg: "not found",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, err := c.GetSecret(ctx, tc.ref)

			if tc.err == nil && tc.errMsg == "" {
				assert.NoError(t, err)
				assert.Equal(t, tc.want, got)
			} else {
				assert.Nil(t, got)
				if tc.errMsg != "" {
					assert.ErrorContains(t, err, tc.errMsg)
				} else {
					assert.ErrorIs(t, err, tc.err)
				}
			}
		})
	}
}

// TestGetSecretWithInvalidUTF8ItemValue tests GetSecret with invalid UTF-8 in ItemValue.
// json.Marshal in Go handles invalid UTF-8 strings without error, so this verifies
// that GetSecret succeeds in this edge case.
func TestGetSecretWithInvalidUTF8ItemValue(t *testing.T) {
	ctx := t.Context()

	bad := &server.Secret{
		ID:     0,
		Fields: []server.SecretField{},
	}
	c := &client{
		api: &fakeAPI{
			secrets: []*server.Secret{bad},
		},
	}
	bad.Fields = []server.SecretField{
		{
			FieldName: "Foo",
			ItemValue: string([]byte{0xff, 0xfe}), // invalid UTF-8
		},
	}

	// GetSecret with no property returns the full JSON; json.Marshal handles invalid UTF-8.
	_, err := c.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: "0"})
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
	// fakeAPI.Secrets now returns ([]Secret{}, nil) for zero matches (matching real SDK),
	// so getSecretByName returns errMsgNoMatchingSecrets.
	assert.Contains(t, err.Error(), errMsgNoMatchingSecrets)
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

// fakePushSecretData implements esv1.PushSecretData for testing.
type fakePushSecretData struct {
	remoteKey string
	property  string
	secretKey string
	metadata  *apiextensionsv1.JSON
}

// GetRemoteKey returns the remote key for the fake push secret data.
func (f fakePushSecretData) GetRemoteKey() string { return f.remoteKey }

// GetProperty returns the property for the fake push secret data.
func (f fakePushSecretData) GetProperty() string { return f.property }

// GetSecretKey returns the secret key for the fake push secret data.
func (f fakePushSecretData) GetSecretKey() string { return f.secretKey }

// GetMetadata returns the metadata for the fake push secret data.
func (f fakePushSecretData) GetMetadata() *apiextensionsv1.JSON { return f.metadata }

// fakePushSecretRemoteRef implements esv1.PushSecretRemoteRef for testing.
type fakePushSecretRemoteRef struct {
	remoteKey string
	property  string
}

// GetRemoteKey returns the remote key for the fake remote ref.
func (f fakePushSecretRemoteRef) GetRemoteKey() string { return f.remoteKey }

// GetProperty returns the property for the fake remote ref.
func (f fakePushSecretRemoteRef) GetProperty() string { return f.property }

// TestPushSecret tests the PushSecret functionality.
func TestPushSecret(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)

	secret := &corev1.Secret{
		Data: map[string][]byte{
			"my-key": []byte("my-value"),
		},
	}

	metadataJSON := apiextensionsv1.JSON{
		Raw: []byte(`{"apiVersion":"kubernetes.external-secrets.io/v1alpha1","kind":"PushSecretMetadata","spec":{"folderId": 1, "secretTemplateId": 1}}`),
	}

	// Create a new secret
	data := fakePushSecretData{
		remoteKey: "new-secret",
		property:  "username",
		secretKey: "my-key",
		metadata:  &metadataJSON,
	}
	err := c.PushSecret(ctx, secret, data)
	assert.NoError(t, err)

	// Verify the secret was created
	createdSecret, _ := c.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: "new-secret", Property: "username"})
	assert.Equal(t, []byte("my-value"), createdSecret)

	// Create a new secret with path-like key and folderId
	dataPathCreate := fakePushSecretData{
		remoteKey: "/some/new/path/secretname",
		property:  "username",
		secretKey: "my-key",
		metadata:  &metadataJSON,
	}
	err = c.PushSecret(ctx, secret, dataPathCreate)
	assert.NoError(t, err)

	// verify that the created secret has just the basename "secretname"
	// and since it's the 10th secret created by fakeAPI, its ID would be 10000 + len(secrets)
	foundSecrets, _ := c.(*client).api.Secrets("secretname", "Name")
	assert.Len(t, foundSecrets, 1)
	assert.Equal(t, "secretname", foundSecrets[0].Name)
	assert.Equal(t, 1, foundSecrets[0].FolderID)

	// Update an existing secret
	dataUpdate := fakePushSecretData{
		remoteKey: "4000",
		property:  "password",
		secretKey: "my-key", // "my-value" will replace the badPassword
	}
	err = c.PushSecret(ctx, secret, dataUpdate)
	assert.NoError(t, err)

	// Verify update
	updatedSecret, _ := c.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: "4000", Property: "password"})
	assert.Equal(t, []byte("my-value"), updatedSecret)

	// Missing metadata for new secret
	dataMissingMeta := fakePushSecretData{
		remoteKey: "new-secret-no-meta",
		property:  "username",
		secretKey: "my-key",
		metadata:  nil,
	}
	err = c.PushSecret(ctx, secret, dataMissingMeta)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "folderId and secretTemplateId must be provided in metadata to create a new secret")

	// Invalid secretTemplateId in metadata
	invalidMetadataJSON := apiextensionsv1.JSON{
		Raw: []byte(`{"apiVersion":"kubernetes.external-secrets.io/v1alpha1","kind":"PushSecretMetadata","spec":{"folderId": 1, "secretTemplateId": 999}}`), // non-existent template
	}
	dataInvalidMeta := fakePushSecretData{
		remoteKey: "new-secret-invalid-meta",
		property:  "username",
		secretKey: "my-key",
		metadata:  &invalidMetadataJSON,
	}
	err = c.PushSecret(ctx, secret, dataInvalidMeta)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get secret template")

	// Simulate create error
	// Requires modifying fakeAPI to return an error when Name == "simulate-create-error"
	dataCreateError := fakePushSecretData{
		remoteKey: "simulate-create-error",
		property:  "username",
		secretKey: "my-key",
		metadata:  &metadataJSON,
	}
	err = c.PushSecret(ctx, secret, dataCreateError)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create secret")

	// Update with non-existent property
	dataUpdateInvalidProp := fakePushSecretData{
		remoteKey: "4000",
		property:  "non-existent-property",
		secretKey: "my-key",
	}
	err = c.PushSecret(ctx, secret, dataUpdateInvalidProp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "field non-existent-property not found in secret")

	// Update duplicate-named secret in specific folder (ID 9001 in FolderID 5)
	metadataFolder5 := apiextensionsv1.JSON{
		Raw: []byte(`{"apiVersion":"kubernetes.external-secrets.io/v1alpha1","kind":"PushSecretMetadata","spec":{"folderId": 5, "secretTemplateId": 1}}`),
	}
	dataFolderUpdate := fakePushSecretData{
		remoteKey: "FolderSecretname",
		property:  "password",
		secretKey: "my-key",
		metadata:  &metadataFolder5,
	}
	err = c.PushSecret(ctx, secret, dataFolderUpdate)
	assert.NoError(t, err)

	// Verify only the secret in folder 5 was updated
	s9001, _ := c.(*client).api.Secret(9001)
	s9000, _ := c.(*client).api.Secret(9000)
	// Check the password field
	var s9001PW, s9000PW string
	for _, f := range s9001.Fields {
		if f.Slug == passwordSlug {
			s9001PW = f.ItemValue
		}
	}
	for _, f := range s9000.Fields {
		if f.Slug == passwordSlug {
			s9000PW = f.ItemValue
		}
	}
	assert.Equal(t, "my-value", s9001PW)
	assert.Equal(t, "passwordvalue", s9000PW) // Unchanged

	// Update path-based key secret
	dataPathUpdate := fakePushSecretData{
		remoteKey: "/some/path/secret",
		property:  "password",
		secretKey: "my-key",
	}
	err = c.PushSecret(ctx, secret, dataPathUpdate)
	assert.NoError(t, err)

	sPath, _ := c.(*client).api.Secret(9002)
	var sPathPW string
	for _, f := range sPath.Fields {
		if f.Slug == passwordSlug {
			sPathPW = f.ItemValue
		}
	}
	assert.Equal(t, "my-value", sPathPW)

	// Push invalid UTF-8 secret
	invalidUtf8Secret := &corev1.Secret{
		Data: map[string][]byte{
			"invalid-utf8": {0xff, 0xfe, 0xfd},
		},
	}
	dataInvalidUtf8 := fakePushSecretData{
		remoteKey: "new-secret-utf8",
		property:  "username",
		secretKey: "invalid-utf8",
		metadata:  &metadataJSON,
	}
	err = c.PushSecret(ctx, invalidUtf8Secret, dataInvalidUtf8)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "secret value is not valid UTF-8")
}

// TestDeleteSecret tests the DeleteSecret functionality.
func TestDeleteSecret(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)

	ref := fakePushSecretRemoteRef{
		remoteKey: "1000",
	}

	// Should exist initially
	exists, err := c.SecretExists(ctx, ref)
	assert.NoError(t, err)
	assert.True(t, exists)

	// Delete it
	err = c.DeleteSecret(ctx, ref)
	assert.NoError(t, err)

	// Should not exist now
	exists, err = c.SecretExists(ctx, ref)
	assert.NoError(t, err)
	assert.False(t, exists)

	// Test idempotency: delete again should not error
	err = c.DeleteSecret(ctx, ref)
	assert.NoError(t, err)

	// Test path-based key deletion
	pathRef := fakePushSecretRemoteRef{
		remoteKey: "/some/path/secret",
	}

	exists, err = c.SecretExists(ctx, pathRef)
	assert.NoError(t, err)
	assert.True(t, exists)

	err = c.DeleteSecret(ctx, pathRef)
	assert.NoError(t, err)

	exists, err = c.SecretExists(ctx, pathRef)
	assert.NoError(t, err)
	assert.False(t, exists)
}

// TestDeleteSecret_Error tests that an error from the backend during DeleteSecret is propagated.
func TestDeleteSecret_Error(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)

	ref := fakePushSecretRemoteRef{
		remoteKey: "9999",
	}

	// Should exist initially
	exists, err := c.SecretExists(ctx, ref)
	assert.NoError(t, err)
	assert.True(t, exists)

	// Attempt to delete it, expecting an error
	err = c.DeleteSecret(ctx, ref)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete secret")

	// Verify it still exists
	exists, err = c.SecretExists(ctx, ref)
	assert.NoError(t, err)
	assert.True(t, exists)
}

// TestSecretExists tests the SecretExists functionality.
func TestSecretExists(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)

	testCases := map[string]struct {
		ref     esv1.PushSecretRemoteRef
		want    bool
		wantErr bool
	}{
		"existing secret": {
			ref:     fakePushSecretRemoteRef{remoteKey: "1000"},
			want:    true,
			wantErr: false,
		},
		"non-existing secret": {
			ref:     fakePushSecretRemoteRef{remoteKey: "does-not-exist"},
			want:    false,
			wantErr: false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, err := c.SecretExists(ctx, tc.ref)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.want, got)
			}
		})
	}
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
		// The following test case expects an error because the secret with Key "9999"
		// contains invalid JSON ("simulated error") which causes unmarshalling to fail
		// in GetSecretMap, rather than because the secret is missing.
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

// TestGetSecretMapValidJSON tests GetSecretMap with valid JSON data succeeds.
func TestGetSecretMapValidJSON(t *testing.T) {
	ctx := context.Background()

	c := newTestClient(t)

	// GetSecretMap with valid JSON should succeed
	result, err := c.GetSecretMap(ctx, esv1.ExternalSecretDataRemoteRef{Key: "1000"})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, []byte("robertOppenheimer"), result["user"])
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
				Path: new("some-path"),
			},
			wantErr: true,
			errMsg:  "getting all secrets is not supported by Delinea Secret Server",
		},
		"returns error with nil path": {
			ref:     esv1.ExternalSecretFind{},
			wantErr: true,
			errMsg:  "getting all secrets is not supported by Delinea Secret Server",
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

// TestIsNotFoundError tests the isNotFoundError function with various error formats.
func TestIsNotFoundError(t *testing.T) {
	testCases := map[string]struct {
		err  error
		want bool
	}{
		"nil error": {
			err:  nil,
			want: false,
		},
		"exact lowercase not found": {
			err:  errors.New("not found"),
			want: true,
		},
		"SDK HTTP 404 format": {
			err:  errors.New("404 Not Found: no secret was found"),
			want: true,
		},
		"SDK HTTP 404 with empty body": {
			err:  errors.New("404 Not Found: "),
			want: true,
		},
		"no matching secrets": {
			err:  errors.New("no matching secrets"),
			want: true,
		},
		"unrelated error": {
			err:  errors.New("connection refused"),
			want: false,
		},
		"field not found in secret (false positive excluded)": {
			// This error from updateSecret should NOT be treated as not-found.
			err:  fmt.Errorf("field password not found in secret"),
			want: false,
		},
		"field not found in secret template (false positive excluded)": {
			// This error from createSecret should NOT be treated as not-found.
			err:  fmt.Errorf("field username not found in secret template"),
			want: false,
		},
		"wrapped field not found in secret": {
			// Even when wrapped, the false-positive exclusion applies.
			err:  fmt.Errorf("failed to update secret: %w", fmt.Errorf("field password not found in secret")),
			want: false,
		},
		"mixed case Not Found": {
			err:  errors.New("Not Found"),
			want: true,
		},
		"SDK HTTP 401 with not found in body": {
			// Auth error that happens to contain "not found" — still matches
			// the generic substring (this is an acceptable edge case since
			// auth errors should not appear in secret-lookup code paths).
			err:  errors.New("401 Unauthorized: user not found"),
			want: true,
		},
		"SDK HTTP 500 error": {
			err:  errors.New("500 Internal Server Error: something went wrong"),
			want: false,
		},
		"our errMsgNotFound sentinel": {
			// From getSecretByName folder mismatch: errors.New(errMsgNotFound)
			err:  errors.New(errMsgNotFound),
			want: true,
		},
		"errMsgAmbiguousName is not a not-found error": {
			err:  errors.New(errMsgAmbiguousName),
			want: false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got := isNotFoundError(tc.err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestPushSecretInvalidPathKeys tests that PushSecret rejects path-style keys with
// empty final segments (root slash, double slash, etc.) that would produce an empty secret name.
func TestPushSecretInvalidPathKeys(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)

	secret := &corev1.Secret{
		Data: map[string][]byte{
			"my-key": []byte("my-value"),
		},
	}

	metadataJSON := apiextensionsv1.JSON{
		Raw: []byte(`{"apiVersion":"kubernetes.external-secrets.io/v1alpha1","kind":"PushSecretMetadata","spec":{"folderId": 1, "secretTemplateId": 1}}`),
	}

	testCases := map[string]struct {
		remoteKey string
		errMsg    string
	}{
		"root slash only": {
			remoteKey: "/",
			errMsg:    "invalid secret name",
		},
		"double slash": {
			remoteKey: "//",
			errMsg:    "invalid secret name",
		},
		"triple slash": {
			remoteKey: "///",
			errMsg:    "invalid secret name",
		},
		"trailing slash on path": {
			remoteKey: "/Folder/Subfolder/",
			errMsg:    "invalid secret name",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			data := fakePushSecretData{
				remoteKey: tc.remoteKey,
				property:  "username",
				secretKey: "my-key",
				metadata:  &metadataJSON,
			}
			err := c.PushSecret(ctx, secret, data)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.errMsg)
		})
	}
}

// TestParseFolderPrefix tests the parseFolderPrefix helper function.
func TestParseFolderPrefix(t *testing.T) {
	testCases := map[string]struct {
		key              string
		wantFolderID     int
		wantName         string
		wantHasFolderPfx bool
	}{
		"valid prefix": {
			key:              "folderId:73/my-secret",
			wantFolderID:     73,
			wantName:         "my-secret",
			wantHasFolderPfx: true,
		},
		"valid prefix with large folder ID": {
			key:              "folderId:99999/secret-name",
			wantFolderID:     99999,
			wantName:         "secret-name",
			wantHasFolderPfx: true,
		},
		"valid prefix with name containing slashes": {
			key:              "folderId:73/sub/path/secret",
			wantFolderID:     73,
			wantName:         "sub/path/secret",
			wantHasFolderPfx: true,
		},
		"no prefix - plain name": {
			key:              "my-secret",
			wantFolderID:     0,
			wantName:         "my-secret",
			wantHasFolderPfx: false,
		},
		"no prefix - numeric key": {
			key:              "12345",
			wantFolderID:     0,
			wantName:         "12345",
			wantHasFolderPfx: false,
		},
		"no prefix - path key": {
			key:              "/Folder/SecretName",
			wantFolderID:     0,
			wantName:         "/Folder/SecretName",
			wantHasFolderPfx: false,
		},
		"prefix without slash": {
			key:              "folderId:73",
			wantFolderID:     0,
			wantName:         "folderId:73",
			wantHasFolderPfx: false,
		},
		"prefix with empty name": {
			key:              "folderId:73/",
			wantFolderID:     0,
			wantName:         "folderId:73/",
			wantHasFolderPfx: false,
		},
		"prefix with non-numeric ID": {
			key:              "folderId:abc/my-secret",
			wantFolderID:     0,
			wantName:         "folderId:abc/my-secret",
			wantHasFolderPfx: false,
		},
		"prefix with zero ID": {
			key:              "folderId:0/my-secret",
			wantFolderID:     0,
			wantName:         "folderId:0/my-secret",
			wantHasFolderPfx: false,
		},
		"prefix with negative ID": {
			key:              "folderId:-1/my-secret",
			wantFolderID:     0,
			wantName:         "folderId:-1/my-secret",
			wantHasFolderPfx: false,
		},
		"empty key": {
			key:              "",
			wantFolderID:     0,
			wantName:         "",
			wantHasFolderPfx: false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			folderID, secretName, hasFolderPrefix := parseFolderPrefix(tc.key)
			assert.Equal(t, tc.wantFolderID, folderID)
			assert.Equal(t, tc.wantName, secretName)
			assert.Equal(t, tc.wantHasFolderPfx, hasFolderPrefix)
		})
	}
}

// TestPushSecretWithFolderPrefix tests PushSecret with the "folderId:<id>/<name>" key format.
func TestPushSecretWithFolderPrefix(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)

	secret := &corev1.Secret{
		Data: map[string][]byte{
			"my-key": []byte("folder-prefix-value"),
		},
	}

	metadataJSON := apiextensionsv1.JSON{
		Raw: []byte(`{"apiVersion":"kubernetes.external-secrets.io/v1alpha1","kind":"PushSecretMetadata","spec":{"folderId": 5, "secretTemplateId": 1}}`),
	}

	// Update an existing secret using folderId prefix — should target folder 5 (ID 9001)
	dataUpdate := fakePushSecretData{
		remoteKey: "folderId:5/FolderSecretname",
		property:  "password",
		secretKey: "my-key",
		metadata:  &metadataJSON,
	}
	err := c.PushSecret(ctx, secret, dataUpdate)
	assert.NoError(t, err)

	// Verify only the secret in folder 5 was updated
	s9001, _ := c.(*client).api.Secret(9001)
	s9000, _ := c.(*client).api.Secret(9000)
	var s9001PW, s9000PW string
	for _, f := range s9001.Fields {
		if f.Slug == passwordSlug {
			s9001PW = f.ItemValue
		}
	}
	for _, f := range s9000.Fields {
		if f.Slug == passwordSlug {
			s9000PW = f.ItemValue
		}
	}
	assert.Equal(t, "folder-prefix-value", s9001PW)
	assert.Equal(t, "passwordvalue", s9000PW) // Unchanged

	// Create a new secret using folderId prefix
	metadataCreate := apiextensionsv1.JSON{
		Raw: []byte(`{"apiVersion":"kubernetes.external-secrets.io/v1alpha1","kind":"PushSecretMetadata","spec":{"folderId": 42, "secretTemplateId": 1}}`),
	}
	dataCreate := fakePushSecretData{
		remoteKey: "folderId:42/brand-new-secret",
		property:  "username",
		secretKey: "my-key",
		metadata:  &metadataCreate,
	}
	err = c.PushSecret(ctx, secret, dataCreate)
	assert.NoError(t, err)

	// Verify the created secret has the plain name (prefix stripped)
	foundSecrets, _ := c.(*client).api.Secrets("brand-new-secret", "Name")
	assert.Len(t, foundSecrets, 1)
	assert.Equal(t, "brand-new-secret", foundSecrets[0].Name)
	assert.Equal(t, 42, foundSecrets[0].FolderID)

	// Test precedence: remoteKey folderId overrides metadata folderId for lookups.
	// Metadata says folderId:4, but remoteKey says folderId:5 — should target folder 5.
	metadataFolder4 := apiextensionsv1.JSON{
		Raw: []byte(`{"apiVersion":"kubernetes.external-secrets.io/v1alpha1","kind":"PushSecretMetadata","spec":{"folderId": 4, "secretTemplateId": 1}}`),
	}
	dataPrecedence := fakePushSecretData{
		remoteKey: "folderId:5/FolderSecretname",
		property:  "username",
		secretKey: "my-key",
		metadata:  &metadataFolder4,
	}
	err = c.PushSecret(ctx, secret, dataPrecedence)
	assert.NoError(t, err)

	// Verify the secret in folder 5 was updated (not folder 4)
	s9001, _ = c.(*client).api.Secret(9001)
	var s9001User string
	for _, f := range s9001.Fields {
		if f.Slug == usernameSlug {
			s9001User = f.ItemValue
		}
	}
	assert.Equal(t, "folder-prefix-value", s9001User)
}

// TestDeleteSecretWithFolderPrefix tests that DeleteSecret correctly uses the
// folderId prefix in the remote key to target the right secret.
func TestDeleteSecretWithFolderPrefix(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)

	// Both secrets 9000 (folder 4) and 9001 (folder 5) have name "FolderSecretname".
	// Delete only the one in folder 5.
	ref := fakePushSecretRemoteRef{
		remoteKey: "folderId:5/FolderSecretname",
	}

	// Should exist initially
	exists, err := c.SecretExists(ctx, ref)
	assert.NoError(t, err)
	assert.True(t, exists)

	// Delete it
	err = c.DeleteSecret(ctx, ref)
	assert.NoError(t, err)

	// Should not exist now
	exists, err = c.SecretExists(ctx, ref)
	assert.NoError(t, err)
	assert.False(t, exists)

	// The secret in folder 4 should still exist
	refFolder4 := fakePushSecretRemoteRef{
		remoteKey: "folderId:4/FolderSecretname",
	}
	exists, err = c.SecretExists(ctx, refFolder4)
	assert.NoError(t, err)
	assert.True(t, exists)
}

// TestSecretExistsWithFolderPrefix tests that SecretExists correctly uses the
// folderId prefix in the remote key.
func TestSecretExistsWithFolderPrefix(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)

	testCases := map[string]struct {
		ref  esv1.PushSecretRemoteRef
		want bool
	}{
		"existing secret in folder 4": {
			ref:  fakePushSecretRemoteRef{remoteKey: "folderId:4/FolderSecretname"},
			want: true,
		},
		"existing secret in folder 5": {
			ref:  fakePushSecretRemoteRef{remoteKey: "folderId:5/FolderSecretname"},
			want: true,
		},
		"non-existing secret in wrong folder": {
			ref:  fakePushSecretRemoteRef{remoteKey: "folderId:99/FolderSecretname"},
			want: false,
		},
		"non-existing secret name": {
			ref:  fakePushSecretRemoteRef{remoteKey: "folderId:4/does-not-exist"},
			want: false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, err := c.SecretExists(ctx, tc.ref)
			assert.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestDeleteSecretAmbiguousName tests that DeleteSecret returns an error when a
// plain name matches multiple secrets across different folders.
func TestDeleteSecretAmbiguousName(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)

	// "FolderSecretname" exists in both folder 4 (ID 9000) and folder 5 (ID 9001).
	// Using just the plain name should fail with an ambiguous error.
	ref := fakePushSecretRemoteRef{
		remoteKey: "FolderSecretname",
	}

	err := c.DeleteSecret(ctx, ref)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "multiple secrets found with the same name")
	assert.Contains(t, err.Error(), "folderId:")

	// Both secrets should still exist (nothing was deleted).
	s9000, err := c.(*client).api.Secret(9000)
	assert.NoError(t, err)
	assert.NotNil(t, s9000)

	s9001, err := c.(*client).api.Secret(9001)
	assert.NoError(t, err)
	assert.NotNil(t, s9001)
}

// TestSecretExistsAmbiguousName tests that SecretExists returns an error when a
// plain name matches multiple secrets across different folders.
func TestSecretExistsAmbiguousName(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)

	// "FolderSecretname" exists in both folder 4 (ID 9000) and folder 5 (ID 9001).
	// Using just the plain name should fail with an ambiguous error.
	ref := fakePushSecretRemoteRef{
		remoteKey: "FolderSecretname",
	}

	exists, err := c.SecretExists(ctx, ref)
	assert.Error(t, err)
	assert.False(t, exists)
	assert.Contains(t, err.Error(), "multiple secrets found with the same name")
}

// TestDeleteSecretUniqueName tests that DeleteSecret still works with a plain
// name when only one secret has that name (no ambiguity).
func TestDeleteSecretUniqueName(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)

	// "Secretname" is unique (only ID 4000 has this name).
	ref := fakePushSecretRemoteRef{
		remoteKey: "Secretname",
	}

	exists, err := c.SecretExists(ctx, ref)
	assert.NoError(t, err)
	assert.True(t, exists)

	err = c.DeleteSecret(ctx, ref)
	assert.NoError(t, err)

	exists, err = c.SecretExists(ctx, ref)
	assert.NoError(t, err)
	assert.False(t, exists)
}

// TestGetSecretByNameStrict tests the getSecretByNameStrict helper directly.
func TestGetSecretByNameStrict(t *testing.T) {
	c := newTestClient(t).(*client)

	testCases := map[string]struct {
		name    string
		wantErr bool
		errMsg  string
	}{
		"unique name returns secret": {
			name:    "Secretname",
			wantErr: false,
		},
		"duplicate name returns ambiguous error": {
			name:    "FolderSecretname",
			wantErr: true,
			errMsg:  "multiple secrets found with the same name",
		},
		"non-existent name returns no matching secrets": {
			name:    "does-not-exist",
			wantErr: true,
			errMsg:  errMsgNoMatchingSecrets,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			secret, err := c.getSecretByNameStrict(tc.name)
			if tc.wantErr {
				assert.Error(t, err)
				assert.Nil(t, secret)
				assert.Contains(t, err.Error(), tc.errMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, secret)
			}
		})
	}
}

// TestPushSecretEmptyProperty tests PushSecret with an empty property, which
// should target the first field of the secret/template.
func TestPushSecretEmptyProperty(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)

	secret := &corev1.Secret{
		Data: map[string][]byte{
			"my-key": []byte("whole-value"),
		},
	}

	metadataJSON := apiextensionsv1.JSON{
		Raw: []byte(`{"apiVersion":"kubernetes.external-secrets.io/v1alpha1","kind":"PushSecretMetadata","spec":{"folderId": 1, "secretTemplateId": 1}}`),
	}

	// Create new secret with empty property → uses first template field
	data := fakePushSecretData{
		remoteKey: "empty-prop-secret",
		property:  "",
		secretKey: "my-key",
		metadata:  &metadataJSON,
	}
	err := c.PushSecret(ctx, secret, data)
	assert.NoError(t, err)

	// Verify: the first field should have the value
	foundSecrets, _ := c.(*client).api.Secrets("empty-prop-secret", "Name")
	require.Len(t, foundSecrets, 1)
	require.Len(t, foundSecrets[0].Fields, 1)
	assert.Equal(t, "whole-value", foundSecrets[0].Fields[0].ItemValue)

	// Update existing secret with empty property → updates first field
	data2 := fakePushSecretData{
		remoteKey: "4000",
		property:  "",
		secretKey: "my-key",
	}
	err = c.PushSecret(ctx, secret, data2)
	assert.NoError(t, err)

	s4000, _ := c.(*client).api.Secret(4000)
	assert.Equal(t, "whole-value", s4000.Fields[0].ItemValue)
}

// TestPushSecretConflictingFolderIDs tests that when the remoteKey has a folderId
// prefix, it overrides the metadata folderId for both lookup AND creation.
func TestPushSecretConflictingFolderIDs(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)

	secret := &corev1.Secret{
		Data: map[string][]byte{
			"my-key": []byte("prefix-wins"),
		},
	}

	// Metadata says folderId:99, but prefix says folderId:42.
	// The prefix should win for creation.
	metadataJSON := apiextensionsv1.JSON{
		Raw: []byte(`{"apiVersion":"kubernetes.external-secrets.io/v1alpha1","kind":"PushSecretMetadata","spec":{"folderId": 99, "secretTemplateId": 1}}`),
	}

	data := fakePushSecretData{
		remoteKey: "folderId:42/conflict-test",
		property:  "username",
		secretKey: "my-key",
		metadata:  &metadataJSON,
	}
	err := c.PushSecret(ctx, secret, data)
	assert.NoError(t, err)

	// Verify: the secret was created in folder 42, not 99.
	foundSecrets, _ := c.(*client).api.Secrets("conflict-test", "Name")
	require.Len(t, foundSecrets, 1)
	assert.Equal(t, 42, foundSecrets[0].FolderID)
}

// TestPushSecretAmbiguousPlainName tests that PushSecret returns an error when
// a plain name (no prefix, no path, no numeric ID) matches multiple secrets.
func TestPushSecretAmbiguousPlainName(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)

	secret := &corev1.Secret{
		Data: map[string][]byte{
			"my-key": []byte("value"),
		},
	}

	// "FolderSecretname" exists in both folder 4 (ID 9000) and folder 5 (ID 9001).
	// Without a folderId prefix or metadata folderId, this should fail.
	data := fakePushSecretData{
		remoteKey: "FolderSecretname",
		property:  "password",
		secretKey: "my-key",
	}
	err := c.PushSecret(ctx, secret, data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "multiple secrets found with the same name")
}

// TestPushSecretEmptyRemoteKey tests that PushSecret rejects empty remote keys.
func TestPushSecretEmptyRemoteKey(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)

	secret := &corev1.Secret{
		Data: map[string][]byte{
			"my-key": []byte("value"),
		},
	}

	data := fakePushSecretData{
		remoteKey: "",
		property:  "username",
		secretKey: "my-key",
	}
	err := c.PushSecret(ctx, secret, data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "remote key must be defined")
}

// TestCreateSecretFolderPrefixWithSlashes tests that createSecret rejects
// folderId prefixed names that contain slashes after prefix stripping.
func TestCreateSecretFolderPrefixWithSlashes(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)

	secret := &corev1.Secret{
		Data: map[string][]byte{
			"my-key": []byte("value"),
		},
	}

	metadataJSON := apiextensionsv1.JSON{
		Raw: []byte(`{"apiVersion":"kubernetes.external-secrets.io/v1alpha1","kind":"PushSecretMetadata","spec":{"folderId": 73, "secretTemplateId": 1}}`),
	}

	data := fakePushSecretData{
		remoteKey: "folderId:73/sub/path/secret",
		property:  "username",
		secretKey: "my-key",
		metadata:  &metadataJSON,
	}
	err := c.PushSecret(ctx, secret, data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must not contain path separators")
}

// TestCreateSecretEmptyTemplateFields tests createSecret when the template has
// no fields and no property is specified.
func TestCreateSecretEmptyTemplateFields(t *testing.T) {
	// Create a fakeAPI that returns a template with no fields
	fake := &fakeAPI{secrets: []*server.Secret{}}
	// Override SecretTemplate to return empty fields (template ID 888)
	c := &client{api: &emptyTemplateAPI{fakeAPI: fake}}

	err := c.createSecret("test-secret", "", "value", PushSecretMetadataSpec{
		FolderID:         1,
		SecretTemplateID: 888,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "secret template has no fields")
}

// emptyTemplateAPI wraps fakeAPI but returns an empty template for ID 888.
type emptyTemplateAPI struct {
	*fakeAPI
}

func (e *emptyTemplateAPI) SecretTemplate(id int) (*server.SecretTemplate, error) {
	if id == 888 {
		return &server.SecretTemplate{
			ID:     888,
			Name:   "Empty Template",
			Fields: []server.SecretTemplateField{},
		}, nil
	}
	return e.fakeAPI.SecretTemplate(id)
}

// TestGetSecretFieldPriorityOverGjson tests that field slug/name matching takes
// priority over gjson extraction from Fields[0].ItemValue.
func TestGetSecretFieldPriorityOverGjson(t *testing.T) {
	ctx := context.Background()

	// Create a secret where:
	// - Fields[0].ItemValue is JSON containing key "password"
	// - Fields[1] has Slug "password" with a DIFFERENT value
	// Field slug/name should win over gjson.
	s := &server.Secret{
		ID:   100,
		Name: "priority-test",
		Fields: []server.SecretField{
			{
				FieldName: "Data",
				Slug:      "data",
				ItemValue: `{"password": "from-json-blob"}`,
			},
			{
				FieldName: "Password",
				Slug:      "password",
				ItemValue: "from-field-slug",
			},
		},
	}

	c := &client{api: &fakeAPI{secrets: []*server.Secret{s}}}

	got, err := c.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{
		Key:      "100",
		Property: "password",
	})
	assert.NoError(t, err)
	// Field slug match should return "from-field-slug", NOT "from-json-blob"
	assert.Equal(t, []byte("from-field-slug"), got)
}

// TestGetSecretGjsonFallback tests that gjson extraction from Fields[0].ItemValue
// works as a fallback when no field slug/name matches.
func TestGetSecretGjsonFallback(t *testing.T) {
	ctx := context.Background()

	s := &server.Secret{
		ID:   101,
		Name: "gjson-fallback-test",
		Fields: []server.SecretField{
			{
				FieldName: "Data",
				Slug:      "data",
				ItemValue: `{"nested": {"key": "deep-value"}}`,
			},
		},
	}

	c := &client{api: &fakeAPI{secrets: []*server.Secret{s}}}

	// "nested.key" doesn't match any field slug/name, so gjson fallback kicks in
	got, err := c.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{
		Key:      "101",
		Property: "nested.key",
	})
	assert.NoError(t, err)
	assert.Equal(t, []byte("deep-value"), got)
}

// TestLookupSecretNon404Error tests that lookupSecret and lookupSecretStrict
// correctly propagate non-404 API errors instead of falling through.
func TestLookupSecretNon404Error(t *testing.T) {
	// Create an API that returns a non-404 error for Secret()
	fake := &errorAPI{
		fakeAPI:   &fakeAPI{secrets: []*server.Secret{}},
		secretErr: errors.New("500 Internal Server Error: database connection failed"),
	}
	c := &client{api: fake}

	// lookupSecret with numeric key: Secret() returns non-404 error, should propagate
	_, err := c.lookupSecret("42", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database connection failed")

	// lookupSecretStrict with numeric key: same behavior
	_, err = c.lookupSecretStrict("42")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database connection failed")
}

// errorAPI wraps fakeAPI but returns a configurable error for Secret().
type errorAPI struct {
	*fakeAPI
	secretErr error
}

func (e *errorAPI) Secret(_ int) (*server.Secret, error) {
	if e.secretErr != nil {
		return nil, e.secretErr
	}
	return e.fakeAPI.Secret(0)
}

// TestFakeAPISecretsReturnsEmptySlice verifies the fakeAPI mock matches real SDK
// behavior: Secrets() returns ([]Secret{}, nil) for zero matches, not an error.
func TestFakeAPISecretsReturnsEmptySlice(t *testing.T) {
	fake := &fakeAPI{secrets: []*server.Secret{}}

	secrets, err := fake.Secrets("nonexistent", "Name")
	assert.NoError(t, err)
	assert.NotNil(t, secrets)
	assert.Empty(t, secrets)
}

// TestPushSecretMetadataNoFolderID tests that PushSecret requires a folderId
// for creation when the prefix doesn't provide one.
func TestPushSecretMetadataNoFolderID(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)

	secret := &corev1.Secret{
		Data: map[string][]byte{
			"my-key": []byte("value"),
		},
	}

	// Metadata has secretTemplateId but no folderId
	metadataJSON := apiextensionsv1.JSON{
		Raw: []byte(`{"apiVersion":"kubernetes.external-secrets.io/v1alpha1","kind":"PushSecretMetadata","spec":{"folderId": 0, "secretTemplateId": 1}}`),
	}

	data := fakePushSecretData{
		remoteKey: "no-folder-secret",
		property:  "username",
		secretKey: "my-key",
		metadata:  &metadataJSON,
	}
	err := c.PushSecret(ctx, secret, data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "folderId and secretTemplateId must be provided")
}

// TestPushSecretCreateWithFolderPrefixNoMetadataFolder tests that PushSecret can
// create a secret when the folderId comes from the prefix even if metadata has
// folderId: 0, as long as secretTemplateId is provided.
func TestPushSecretCreateWithFolderPrefixNoMetadataFolder(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)

	secret := &corev1.Secret{
		Data: map[string][]byte{
			"my-key": []byte("prefix-folder-value"),
		},
	}

	// Metadata has secretTemplateId but folderId is 0; prefix provides the folder.
	metadataJSON := apiextensionsv1.JSON{
		Raw: []byte(`{"apiVersion":"kubernetes.external-secrets.io/v1alpha1","kind":"PushSecretMetadata","spec":{"folderId": 0, "secretTemplateId": 1}}`),
	}

	data := fakePushSecretData{
		remoteKey: "folderId:55/prefix-only-folder",
		property:  "username",
		secretKey: "my-key",
		metadata:  &metadataJSON,
	}
	err := c.PushSecret(ctx, secret, data)
	assert.NoError(t, err)

	// Verify: created in folder 55
	foundSecrets, _ := c.(*client).api.Secrets("prefix-only-folder", "Name")
	require.Len(t, foundSecrets, 1)
	assert.Equal(t, 55, foundSecrets[0].FolderID)
}

// TestGetSecretByFolderPrefix tests GetSecret with the folderId prefix format.
func TestGetSecretByFolderPrefix(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)

	// Secret 9000 is in folder 4, secret 9001 is in folder 5, both named "FolderSecretname"
	got, err := c.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{
		Key:      "folderId:4/FolderSecretname",
		Property: "username",
	})
	assert.NoError(t, err)
	assert.Equal(t, []byte("usernamevalue"), got)

	// Non-existent folder
	_, err = c.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{
		Key:      "folderId:99/FolderSecretname",
		Property: "username",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestPushSecretUpdateSecretNoFields tests updating a secret that has no fields.
func TestPushSecretUpdateSecretNoFields(t *testing.T) {
	ctx := context.Background()

	// Create a secret with no fields
	s := &server.Secret{
		ID:     200,
		Name:   "no-fields-secret",
		Fields: []server.SecretField{},
	}
	c := &client{api: &fakeAPI{secrets: []*server.Secret{s}}}

	secret := &corev1.Secret{
		Data: map[string][]byte{
			"my-key": []byte("value"),
		},
	}

	// Update with empty property → tries to write to first field, but there are none
	data := fakePushSecretData{
		remoteKey: "200",
		property:  "",
		secretKey: "my-key",
	}
	err := c.PushSecret(ctx, secret, data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "secret has no fields to update")
}

// TestPushSecretNonExistentTemplateField tests creating a secret with a property that
// doesn't match any template field.
func TestPushSecretNonExistentTemplateField(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)

	secret := &corev1.Secret{
		Data: map[string][]byte{
			"my-key": []byte("value"),
		},
	}

	metadataJSON := apiextensionsv1.JSON{
		Raw: []byte(`{"apiVersion":"kubernetes.external-secrets.io/v1alpha1","kind":"PushSecretMetadata","spec":{"folderId": 1, "secretTemplateId": 1}}`),
	}

	data := fakePushSecretData{
		remoteKey: "nonexistent-field-secret",
		property:  "nonexistent-field",
		secretKey: "my-key",
		metadata:  &metadataJSON,
	}
	err := c.PushSecret(ctx, secret, data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "field nonexistent-field not found in secret template")
}
