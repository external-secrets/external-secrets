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

package dvls

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/Devolutions/go-dvls"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

type mockCredentialClient struct {
	entries     map[string]dvls.Entry
	getErr      error
	updateErr   error
	deleteErr   error
	lastUpdated dvls.Entry
	lastDeleted string
}

func newMockCredentialClient(entries map[string]dvls.Entry) *mockCredentialClient {
	if entries == nil {
		entries = make(map[string]dvls.Entry)
	}
	return &mockCredentialClient{entries: entries}
}

func (m *mockCredentialClient) GetByID(_ context.Context, _, entryID string) (dvls.Entry, error) {
	if m.getErr != nil {
		return dvls.Entry{}, m.getErr
	}

	if entry, ok := m.entries[entryID]; ok {
		return entry, nil
	}

	return dvls.Entry{}, &dvls.RequestError{Err: fmt.Errorf("unexpected status code %d", http.StatusNotFound), Url: entryID, StatusCode: http.StatusNotFound}
}

func (m *mockCredentialClient) Update(_ context.Context, entry dvls.Entry) (dvls.Entry, error) {
	if m.updateErr != nil {
		return entry, m.updateErr
	}
	m.entries[entry.Id] = entry
	m.lastUpdated = entry
	return entry, nil
}

func (m *mockCredentialClient) DeleteByID(_ context.Context, _, entryID string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}

	delete(m.entries, entryID)
	m.lastDeleted = entryID
	return nil
}

type pushSecretDataStub struct {
	remoteKey string
	secretKey string
	property  string
}

func (p pushSecretDataStub) GetMetadata() *apiextensionsv1.JSON { return nil }
func (p pushSecretDataStub) GetSecretKey() string               { return p.secretKey }
func (p pushSecretDataStub) GetRemoteKey() string               { return p.remoteKey }
func (p pushSecretDataStub) GetProperty() string                { return p.property }

type pushSecretRemoteRefStub struct {
	remoteKey string
	property  string
}

func (p pushSecretRemoteRefStub) GetRemoteKey() string { return p.remoteKey }
func (p pushSecretRemoteRefStub) GetProperty() string  { return p.property }

func TestClient_parseSecretRef(t *testing.T) {
	c := &Client{}

	t.Run("case 1: valid key format", func(t *testing.T) {
		vaultID, entryID, err := c.parseSecretRef("vault-123/entry-456")
		assert.NoError(t, err)
		assert.Equal(t, "vault-123", vaultID)
		assert.Equal(t, "entry-456", entryID)
	})

	t.Run("case 2: invalid key format - no separator", func(t *testing.T) {
		_, _, err := c.parseSecretRef("invalid-key")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid key format")
	})

	t.Run("case 3: invalid key format - empty vault ID", func(t *testing.T) {
		_, _, err := c.parseSecretRef("/entry-456")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "vault ID cannot be empty")
	})

	t.Run("case 4: invalid key format - empty entry ID", func(t *testing.T) {
		_, _, err := c.parseSecretRef("vault-123/")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "entry ID cannot be empty")
	})

	t.Run("case 5: key with spaces", func(t *testing.T) {
		vaultID, entryID, err := c.parseSecretRef(" vault-123 / entry-456 ")
		assert.NoError(t, err)
		assert.Equal(t, "vault-123", vaultID)
		assert.Equal(t, "entry-456", entryID)
	})
}

func TestClient_Validate(t *testing.T) {
	t.Run("case 1: nil client", func(t *testing.T) {
		c := &Client{dvls: nil}
		result, err := c.Validate()
		assert.Error(t, err)
		assert.Equal(t, esv1.ValidationResultError, result)
	})

	t.Run("case 2: initialized client", func(t *testing.T) {
		c := &Client{dvls: newMockCredentialClient(nil)}
		result, err := c.Validate()
		assert.NoError(t, err)
		assert.Equal(t, esv1.ValidationResultReady, result)
	})
}

func TestNewClient(t *testing.T) {
	// Test that NewClient returns a non-nil client
	c := NewClient(nil)
	assert.NotNil(t, c)
	assert.Nil(t, c.dvls)
}

func TestClient_entryToSecretMap(t *testing.T) {
	c := &Client{}

	t.Run("Default credential type", func(t *testing.T) {
		entry := dvls.Entry{
			Id:      "entry-id-123",
			Name:    "test-entry",
			Type:    dvls.EntryCredentialType,
			SubType: dvls.EntryCredentialSubTypeDefault,
			Data: &dvls.EntryCredentialDefaultData{
				Username: "testuser",
				Password: "testpass",
				Domain:   "testdomain",
			},
		}

		secretMap, err := c.entryToSecretMap(entry)
		assert.NoError(t, err)
		assert.Equal(t, "entry-id-123", string(secretMap["entry-id"]))
		assert.Equal(t, "test-entry", string(secretMap["entry-name"]))
		assert.Equal(t, "testuser", string(secretMap["username"]))
		assert.Equal(t, "testpass", string(secretMap["password"]))
		assert.Equal(t, "testdomain", string(secretMap["domain"]))
	})

	t.Run("AccessCode credential type", func(t *testing.T) {
		entry := dvls.Entry{
			Id:      "entry-id-456",
			Name:    "access-code-entry",
			Type:    dvls.EntryCredentialType,
			SubType: dvls.EntryCredentialSubTypeAccessCode,
			Data: &dvls.EntryCredentialAccessCodeData{
				Password: "access-code-123",
			},
		}

		secretMap, err := c.entryToSecretMap(entry)
		assert.NoError(t, err)
		assert.Equal(t, "entry-id-456", string(secretMap["entry-id"]))
		assert.Equal(t, "access-code-entry", string(secretMap["entry-name"]))
		assert.Equal(t, "access-code-123", string(secretMap["password"]))
	})

	t.Run("ApiKey credential type", func(t *testing.T) {
		entry := dvls.Entry{
			Id:      "entry-id-789",
			Name:    "api-key-entry",
			Type:    dvls.EntryCredentialType,
			SubType: dvls.EntryCredentialSubTypeApiKey,
			Data: &dvls.EntryCredentialApiKeyData{
				ApiId:    "api-id-123",
				ApiKey:   "api-key-secret",
				TenantId: "tenant-123",
			},
		}

		secretMap, err := c.entryToSecretMap(entry)
		assert.NoError(t, err)
		assert.Equal(t, "entry-id-789", string(secretMap["entry-id"]))
		assert.Equal(t, "api-key-entry", string(secretMap["entry-name"]))
		assert.Equal(t, "api-id-123", string(secretMap["api-id"]))
		assert.Equal(t, "api-key-secret", string(secretMap["api-key"]))
		assert.Equal(t, "tenant-123", string(secretMap["tenant-id"]))
	})

	t.Run("AzureServicePrincipal credential type", func(t *testing.T) {
		entry := dvls.Entry{
			Id:      "entry-id-azure",
			Name:    "azure-sp-entry",
			Type:    dvls.EntryCredentialType,
			SubType: dvls.EntryCredentialSubTypeAzureServicePrincipal,
			Data: &dvls.EntryCredentialAzureServicePrincipalData{
				ClientId:     "client-id-123",
				ClientSecret: "client-secret-456",
				TenantId:     "tenant-id-789",
			},
		}

		secretMap, err := c.entryToSecretMap(entry)
		assert.NoError(t, err)
		assert.Equal(t, "entry-id-azure", string(secretMap["entry-id"]))
		assert.Equal(t, "azure-sp-entry", string(secretMap["entry-name"]))
		assert.Equal(t, "client-id-123", string(secretMap["client-id"]))
		assert.Equal(t, "client-secret-456", string(secretMap["client-secret"]))
		assert.Equal(t, "tenant-id-789", string(secretMap["tenant-id"]))
	})

	t.Run("ConnectionString credential type", func(t *testing.T) {
		entry := dvls.Entry{
			Id:      "entry-id-conn",
			Name:    "connection-string-entry",
			Type:    dvls.EntryCredentialType,
			SubType: dvls.EntryCredentialSubTypeConnectionString,
			Data: &dvls.EntryCredentialConnectionStringData{
				ConnectionString: "Server=localhost;Database=mydb;User=admin;Password=secret",
			},
		}

		secretMap, err := c.entryToSecretMap(entry)
		assert.NoError(t, err)
		assert.Equal(t, "entry-id-conn", string(secretMap["entry-id"]))
		assert.Equal(t, "connection-string-entry", string(secretMap["entry-name"]))
		assert.Equal(t, "Server=localhost;Database=mydb;User=admin;Password=secret", string(secretMap["connection-string"]))
	})

	t.Run("PrivateKey credential type", func(t *testing.T) {
		entry := dvls.Entry{
			Id:      "entry-id-pk",
			Name:    "private-key-entry",
			Type:    dvls.EntryCredentialType,
			SubType: dvls.EntryCredentialSubTypePrivateKey,
			Data: &dvls.EntryCredentialPrivateKeyData{
				Username:   "ssh-user",
				Password:   "key-password",
				PrivateKey: "-----BEGIN RSA PRIVATE KEY-----\nMIIE...",
				PublicKey:  "ssh-rsa AAAA...",
				Passphrase: "my-passphrase",
			},
		}

		secretMap, err := c.entryToSecretMap(entry)
		assert.NoError(t, err)
		assert.Equal(t, "entry-id-pk", string(secretMap["entry-id"]))
		assert.Equal(t, "private-key-entry", string(secretMap["entry-name"]))
		assert.Equal(t, "ssh-user", string(secretMap["username"]))
		assert.Equal(t, "key-password", string(secretMap["password"]))
		assert.Equal(t, "-----BEGIN RSA PRIVATE KEY-----\nMIIE...", string(secretMap["private-key"]))
		assert.Equal(t, "ssh-rsa AAAA...", string(secretMap["public-key"]))
		assert.Equal(t, "my-passphrase", string(secretMap["passphrase"]))
	})

	t.Run("Unsupported credential type", func(t *testing.T) {
		entry := dvls.Entry{
			Id:      "entry-id-unknown",
			Name:    "unknown-entry",
			Type:    dvls.EntryCredentialType,
			SubType: "UnknownType",
		}

		_, err := c.entryToSecretMap(entry)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported credential subtype")
	})

	t.Run("Default credential with partial data", func(t *testing.T) {
		entry := dvls.Entry{
			Id:      "entry-id-partial",
			Name:    "partial-entry",
			Type:    dvls.EntryCredentialType,
			SubType: dvls.EntryCredentialSubTypeDefault,
			Data: &dvls.EntryCredentialDefaultData{
				Username: "onlyuser",
				// Password and Domain are empty
			},
		}

		secretMap, err := c.entryToSecretMap(entry)
		assert.NoError(t, err)
		assert.Equal(t, "entry-id-partial", string(secretMap["entry-id"]))
		assert.Equal(t, "partial-entry", string(secretMap["entry-name"]))
		assert.Equal(t, "onlyuser", string(secretMap["username"]))
		// Empty fields should not be included
		_, hasPassword := secretMap["password"]
		_, hasDomain := secretMap["domain"]
		assert.False(t, hasPassword)
		assert.False(t, hasDomain)
	})
}

func TestClient_GetSecret_NotFound(t *testing.T) {
	c := NewClient(newMockCredentialClient(nil))

	_, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "vault/entry"})
	assert.ErrorIs(t, err, esv1.NoSecretErr)
}

func TestClient_GetSecretAndMap_Success(t *testing.T) {
	entry := dvls.Entry{
		Id:      "entry-1",
		Name:    "test-entry",
		Type:    dvls.EntryCredentialType,
		SubType: dvls.EntryCredentialSubTypeDefault,
		Data: &dvls.EntryCredentialDefaultData{
			Password: "super-secret",
		},
	}

	mockClient := newMockCredentialClient(map[string]dvls.Entry{"entry-1": entry})
	c := NewClient(mockClient)

	val, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "vault-1/entry-1", Property: "password"})
	assert.NoError(t, err)
	assert.Equal(t, "super-secret", string(val))

	secretMap, err := c.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "vault-1/entry-1"})
	assert.NoError(t, err)
	assert.Equal(t, "super-secret", string(secretMap["password"]))
	assert.Equal(t, "test-entry", string(secretMap["entry-name"]))
}

func TestClient_SecretExists(t *testing.T) {
	mockClient := newMockCredentialClient(nil)
	c := NewClient(mockClient)

	exists, err := c.SecretExists(context.Background(), pushSecretRemoteRefStub{remoteKey: "vault/entry"})
	assert.NoError(t, err)
	assert.False(t, exists)

	mockClient.entries["entry"] = dvls.Entry{Id: "entry", Type: dvls.EntryCredentialType, SubType: dvls.EntryCredentialSubTypeDefault}

	exists, err = c.SecretExists(context.Background(), pushSecretRemoteRefStub{remoteKey: "vault/entry"})
	assert.NoError(t, err)
	assert.True(t, exists)

	mockClient.getErr = errors.New("boom")
	_, err = c.SecretExists(context.Background(), pushSecretRemoteRefStub{remoteKey: "vault/entry"})
	assert.Error(t, err)
}

func TestClient_DeleteSecret(t *testing.T) {
	mockClient := newMockCredentialClient(map[string]dvls.Entry{"entry": {Id: "entry", Type: dvls.EntryCredentialType, SubType: dvls.EntryCredentialSubTypeAccessCode}})
	c := NewClient(mockClient)

	err := c.DeleteSecret(context.Background(), pushSecretRemoteRefStub{remoteKey: "vault/entry"})
	assert.NoError(t, err)
	assert.Equal(t, "entry", mockClient.lastDeleted)
}

func TestClient_PushSecret_UpdateDefault(t *testing.T) {
	mockClient := newMockCredentialClient(map[string]dvls.Entry{
		"entry": {Id: "entry", Type: dvls.EntryCredentialType, SubType: dvls.EntryCredentialSubTypeDefault},
	})
	c := NewClient(mockClient)

	secret := &corev1.Secret{
		Data: map[string][]byte{
			"password": []byte("new-value"),
		},
	}

	data := pushSecretDataStub{remoteKey: "vault/entry", secretKey: "password"}

	err := c.PushSecret(context.Background(), secret, data)
	assert.NoError(t, err)

	updatedEntry := mockClient.entries["entry"]
	credData, ok := updatedEntry.Data.(*dvls.EntryCredentialDefaultData)
	assert.True(t, ok)
	assert.Equal(t, "new-value", credData.Password)
}

func TestClient_PushSecret_UpdateAccessCode(t *testing.T) {
	mockClient := newMockCredentialClient(map[string]dvls.Entry{
		"entry": {Id: "entry", Type: dvls.EntryCredentialType, SubType: dvls.EntryCredentialSubTypeAccessCode},
	})
	c := NewClient(mockClient)

	secret := &corev1.Secret{
		Data: map[string][]byte{
			"code": []byte("code-value"),
		},
	}

	data := pushSecretDataStub{remoteKey: "vault/entry", secretKey: "code"}

	err := c.PushSecret(context.Background(), secret, data)
	assert.NoError(t, err)

	updatedEntry := mockClient.entries["entry"]
	credData, ok := updatedEntry.Data.(*dvls.EntryCredentialAccessCodeData)
	assert.True(t, ok)
	assert.Equal(t, "code-value", credData.Password)
}

func TestClient_PushSecret_NotFound(t *testing.T) {
	c := NewClient(newMockCredentialClient(nil))
	secret := &corev1.Secret{Data: map[string][]byte{"password": []byte("pw")}}
	data := pushSecretDataStub{remoteKey: "vault/missing", secretKey: "password"}

	err := c.PushSecret(context.Background(), secret, data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestClient_PushSecret_UnsupportedSubtype(t *testing.T) {
	mockClient := newMockCredentialClient(map[string]dvls.Entry{
		"entry": {Id: "entry", Type: dvls.EntryCredentialType, SubType: dvls.EntryCredentialSubTypeApiKey},
	})
	c := NewClient(mockClient)
	secret := &corev1.Secret{Data: map[string][]byte{"password": []byte("pw")}}
	data := pushSecretDataStub{remoteKey: "vault/entry", secretKey: "password"}

	err := c.PushSecret(context.Background(), secret, data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot set secret for credential subtype")
}
