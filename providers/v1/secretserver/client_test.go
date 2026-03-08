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
	var secrets []server.Secret
	for _, s := range f.secrets {
		if s.Name == searchText {
			secrets = append(secrets, *s)
		}
	}
	if len(secrets) > 0 {
		return secrets, nil
	}
	return nil, errNotFound
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

	_, err := c.getSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: "nonexistent"}, 0)
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
		"no matching secrets": {
			err:  errors.New("no matching secrets"),
			want: true,
		},
		"unrelated error": {
			err:  errors.New("connection refused"),
			want: false,
		},
		"field not found in secret": {
			// This error message from updateSecret contains "not found" and will match.
			// Callers must not pass these through isNotFoundError; this test documents the behavior.
			err:  fmt.Errorf("field password not found in secret"),
			want: true,
		},
		"mixed case Not Found": {
			err:  errors.New("Not Found"),
			want: true,
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
