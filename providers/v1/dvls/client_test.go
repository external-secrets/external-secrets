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

package dvls

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/Devolutions/go-dvls"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

const (
	testVaultUUID               = "00000000-0000-0000-0000-000000000001"
	testEntryUUID               = "00000000-0000-0000-0000-000000000002"
	testEntryUUID3              = "00000000-0000-0000-0000-000000000003"
	testEntryUUID4              = "00000000-0000-0000-0000-000000000004"
	testEntryUUID5              = "00000000-0000-0000-0000-000000000005"
	testEntryName               = "my-entry"
	testVaultName               = "my-vault"
	testSecretName              = "my-secret"
	testNonExistName            = "some-name"
	testCreatedUUIDPrefix       = "00000000-0000-0000-0000-0000000000"
	testCreatedFolderUUIDPrefix = "00000000-0000-0000-0000-0000000001"
)

// --- Mock credential client ---

type mockCredentialClient struct {
	entries       map[string]dvls.Entry
	getErr        error
	getEntriesErr error
	newErr        error
	updateErr     error
	deleteErr     error
	lastCreated   dvls.Entry
	lastUpdated   dvls.Entry
	lastDeleted   string
	created       int
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

func (m *mockCredentialClient) GetEntries(_ context.Context, _ string, opts dvls.GetEntriesOptions) ([]dvls.Entry, error) {
	if m.getEntriesErr != nil {
		return nil, m.getEntriesErr
	}
	if opts.Name == nil {
		return nil, nil
	}
	var matches []dvls.Entry
	for _, e := range m.entries {
		if e.Name == *opts.Name {
			if opts.Path != nil && e.Path != *opts.Path {
				continue
			}
			matches = append(matches, e)
		}
	}
	return matches, nil
}

func (m *mockCredentialClient) New(_ context.Context, entry dvls.Entry) (string, error) {
	if m.newErr != nil {
		return "", m.newErr
	}
	if entry.Id == "" {
		// Distinct ids, so a test that creates more than once can tell the
		// entries apart instead of silently overwriting the first.
		entry.Id = fmt.Sprintf("%s%02d", testCreatedUUIDPrefix, m.created)
	}
	m.created++
	m.entries[entry.Id] = entry
	m.lastCreated = entry
	return entry.Id, nil
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

// --- Mock folder client ---

// mockFolderClient mirrors go-dvls: the server's path filter also matches
// everything nested under the path, so a caller that wants one exact folder has
// to check the match itself.
type mockFolderClient struct {
	folders       []dvls.Entry
	getEntriesErr error
	newErr        error
	created       []dvls.Entry
	lookups       int
}

func newMockFolderClient(folders ...dvls.Entry) *mockFolderClient {
	return &mockFolderClient{folders: folders}
}

func (m *mockFolderClient) GetEntries(_ context.Context, _ string, opts dvls.GetEntriesOptions) ([]dvls.Entry, error) {
	m.lookups++
	if m.getEntriesErr != nil {
		return nil, m.getEntriesErr
	}

	var matches []dvls.Entry
	for _, f := range m.folders {
		if opts.Name != nil && f.Name != *opts.Name {
			continue
		}
		if opts.Path != nil && f.Path != *opts.Path && !strings.HasPrefix(f.Path, *opts.Path+`\`) {
			continue
		}
		matches = append(matches, f)
	}
	return matches, nil
}

func (m *mockFolderClient) New(_ context.Context, entry dvls.Entry) (string, error) {
	if m.newErr != nil {
		return "", m.newErr
	}
	entry.Id = fmt.Sprintf("%s%02d", testCreatedFolderUUIDPrefix, len(m.created))
	// What was submitted: Path names the parent the folder goes under.
	m.created = append(m.created, entry)
	// What a later read returns: Path is the folder's own full path.
	stored := entry
	stored.Path = folderPath(entry.Name, entry.Path)
	m.folders = append(m.folders, stored)
	return entry.Id, nil
}

// folderPath is where a folder of this name under this parent ends up, which is
// what the server reports as the folder's own Path.
func folderPath(name, parent string) string {
	if parent == "" {
		return name
	}
	return parent + `\` + name
}

// folder builds a folder entry sitting directly under parent, as the server
// hands it back: Path is the folder's own full path, not the parent's.
func folder(name, parent string) dvls.Entry {
	return dvls.Entry{
		Id:      name + "@" + parent,
		Name:    name,
		Path:    folderPath(name, parent),
		Type:    dvls.EntryFolderType,
		SubType: dvls.EntryFolderSubTypeFolder,
		Data:    &dvls.EntryFolderData{},
	}
}

// --- Mock vault client ---

type mockVaultClient struct {
	vaults map[string]dvls.Vault
	getErr error
}

func newMockVaultClient(vaults map[string]dvls.Vault) *mockVaultClient {
	if vaults == nil {
		vaults = make(map[string]dvls.Vault)
	}
	return &mockVaultClient{vaults: vaults}
}

func (m *mockVaultClient) GetByName(_ context.Context, name string) (dvls.Vault, error) {
	if m.getErr != nil {
		return dvls.Vault{}, m.getErr
	}
	if v, ok := m.vaults[name]; ok {
		return v, nil
	}
	return dvls.Vault{}, dvls.ErrVaultNotFound
}

// --- Test stubs ---

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

// --- Helper to create a test client ---

func newTestClient(entries map[string]dvls.Entry) (*Client, *mockCredentialClient) {
	c, cred, _ := newTestClientWithFolders(entries)
	return c, cred
}

// newTestClientWithFolders also hands back the folder mock, for the tests that
// care about which folders a push creates. folders seeds the ones that already
// exist in the vault.
func newTestClientWithFolders(entries map[string]dvls.Entry, folders ...dvls.Entry) (*Client, *mockCredentialClient, *mockFolderClient) {
	mockCred := newMockCredentialClient(entries)
	mockFolder := newMockFolderClient(folders...)
	return NewClient(mockCred, mockFolder, testVaultUUID), mockCred, mockFolder
}

// --- Tests: parseEntryRef ---

func TestParseEntryRef(t *testing.T) {
	t.Run("name only", func(t *testing.T) {
		name, path := parseEntryRef(testEntryName)
		assert.Equal(t, testEntryName, name)
		assert.Equal(t, "", path)
	})

	t.Run("forward slash path", func(t *testing.T) {
		name, path := parseEntryRef("folder/my-entry")
		assert.Equal(t, testEntryName, name)
		assert.Equal(t, "folder", path)
	})

	t.Run("backslash path", func(t *testing.T) {
		name, path := parseEntryRef(`folder\my-entry`)
		assert.Equal(t, testEntryName, name)
		assert.Equal(t, "folder", path)
	})

	t.Run("nested forward slashes", func(t *testing.T) {
		name, path := parseEntryRef("go-dvls/folders/server/123")
		assert.Equal(t, "123", name)
		assert.Equal(t, `go-dvls\folders\server`, path)
	})

	t.Run("nested backslashes", func(t *testing.T) {
		name, path := parseEntryRef(`go-dvls\folders\server\123`)
		assert.Equal(t, "123", name)
		assert.Equal(t, `go-dvls\folders\server`, path)
	})

	t.Run("mixed separators", func(t *testing.T) {
		name, path := parseEntryRef(`go-dvls/folders\server/123`)
		assert.Equal(t, "123", name)
		assert.Equal(t, `go-dvls\folders\server`, path)
	})

	t.Run("trailing separator", func(t *testing.T) {
		name, path := parseEntryRef("folder/")
		assert.Equal(t, "", name)
		assert.Equal(t, "folder", path)
	})
}

// --- Tests: isUUID ---

func TestIsUUID(t *testing.T) {
	t.Run("valid UUID", func(t *testing.T) {
		assert.True(t, isUUID("00000000-0000-0000-0000-000000000001"))
	})

	t.Run("valid UUID v4", func(t *testing.T) {
		assert.True(t, isUUID("550e8400-e29b-41d4-a716-446655440000"))
	})

	t.Run("name string", func(t *testing.T) {
		assert.False(t, isUUID("my-vault-name"))
	})

	t.Run("empty string", func(t *testing.T) {
		assert.False(t, isUUID(""))
	})

	t.Run("malformed UUID", func(t *testing.T) {
		assert.False(t, isUUID("00000000-0000-0000-000000000001"))
	})
}

// --- Tests: resolveVaultRef ---

func TestResolveVaultRef(t *testing.T) {
	t.Run("UUID passthrough", func(t *testing.T) {
		id, err := resolveVaultRef(context.Background(), testVaultUUID, newMockVaultClient(nil))
		assert.NoError(t, err)
		assert.Equal(t, testVaultUUID, id)
	})

	t.Run("name resolved", func(t *testing.T) {
		mockVault := newMockVaultClient(map[string]dvls.Vault{
			testVaultName: {Id: testVaultUUID, Name: testVaultName},
		})
		id, err := resolveVaultRef(context.Background(), testVaultName, mockVault)
		assert.NoError(t, err)
		assert.Equal(t, testVaultUUID, id)
	})

	t.Run("name not found", func(t *testing.T) {
		_, err := resolveVaultRef(context.Background(), "nonexistent", newMockVaultClient(nil))
		assert.Error(t, err)
		assert.ErrorIs(t, err, dvls.ErrVaultNotFound)
	})
}

// --- Tests: resolveEntryRef ---

func TestResolveEntryRef(t *testing.T) {
	entry := dvls.Entry{
		Id:      testEntryUUID,
		Name:    testEntryName,
		Type:    dvls.EntryCredentialType,
		SubType: dvls.EntryCredentialSubTypeDefault,
	}

	t.Run("UUID passthrough", func(t *testing.T) {
		c, _ := newTestClient(nil)
		entryID, err := c.resolveEntryRef(context.Background(), testEntryUUID)
		assert.NoError(t, err)
		assert.Equal(t, testEntryUUID, entryID)
	})

	t.Run("name resolved", func(t *testing.T) {
		c, _ := newTestClient(map[string]dvls.Entry{testEntryUUID: entry})
		entryID, err := c.resolveEntryRef(context.Background(), testEntryName)
		assert.NoError(t, err)
		assert.Equal(t, testEntryUUID, entryID)
	})

	t.Run("name with path resolved", func(t *testing.T) {
		entryInPath := dvls.Entry{
			Id:      testEntryUUID,
			Name:    testEntryName,
			Path:    `go-dvls\folders`,
			Type:    dvls.EntryCredentialType,
			SubType: dvls.EntryCredentialSubTypeDefault,
		}
		c, _ := newTestClient(map[string]dvls.Entry{testEntryUUID: entryInPath})
		entryID, err := c.resolveEntryRef(context.Background(), "go-dvls/folders/my-entry")
		assert.NoError(t, err)
		assert.Equal(t, testEntryUUID, entryID)
	})

	t.Run("path filters out other paths", func(t *testing.T) {
		entryA := dvls.Entry{Id: testEntryUUID, Name: "db", Path: `prod`, Type: dvls.EntryCredentialType, SubType: dvls.EntryCredentialSubTypeDefault}
		entryB := dvls.Entry{Id: testEntryUUID5, Name: "db", Path: `staging`, Type: dvls.EntryCredentialType, SubType: dvls.EntryCredentialSubTypeDefault}
		c, _ := newTestClient(map[string]dvls.Entry{testEntryUUID: entryA, testEntryUUID5: entryB})
		entryID, err := c.resolveEntryRef(context.Background(), "prod/db")
		assert.NoError(t, err)
		assert.Equal(t, testEntryUUID, entryID)
	})

	t.Run("name not found", func(t *testing.T) {
		c, _ := newTestClient(nil)
		_, err := c.resolveEntryRef(context.Background(), "nonexistent")
		assert.Error(t, err)
		assert.ErrorIs(t, err, dvls.ErrEntryNotFound)
	})

	t.Run("multiple entries found", func(t *testing.T) {
		entry2 := dvls.Entry{Id: testEntryUUID3, Name: "dup", Type: dvls.EntryCredentialType, SubType: dvls.EntryCredentialSubTypeDefault}
		entry3 := dvls.Entry{Id: testEntryUUID4, Name: "dup", Type: dvls.EntryCredentialType, SubType: dvls.EntryCredentialSubTypeConnectionString}
		c, _ := newTestClient(map[string]dvls.Entry{
			testEntryUUID3: entry2,
			testEntryUUID4: entry3,
		})
		_, err := c.resolveEntryRef(context.Background(), "dup")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "found 2 credential entries")
		assert.Contains(t, err.Error(), testEntryUUID3)
		assert.Contains(t, err.Error(), testEntryUUID4)
	})

	t.Run("empty key", func(t *testing.T) {
		c, _ := newTestClient(nil)
		_, err := c.resolveEntryRef(context.Background(), "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("trailing separator produces empty name", func(t *testing.T) {
		c, _ := newTestClient(nil)
		_, err := c.resolveEntryRef(context.Background(), "folder/")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "entry name cannot be empty")
	})

	t.Run("GetEntries API error", func(t *testing.T) {
		c, mockCred := newTestClient(nil)
		mockCred.getEntriesErr = errors.New("network error")
		_, err := c.resolveEntryRef(context.Background(), testNonExistName)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to resolve entry")
		assert.Contains(t, err.Error(), "network error")
	})

	t.Run("vault not found during name resolution", func(t *testing.T) {
		c, mockCred := newTestClient(nil)
		mockCred.getEntriesErr = dvls.ErrVaultNotFound
		_, err := c.resolveEntryRef(context.Background(), testNonExistName)
		assert.Error(t, err)
		assert.ErrorIs(t, err, dvls.ErrVaultNotFound)
	})

	t.Run("cache hit avoids second GetEntries call", func(t *testing.T) {
		entry := dvls.Entry{
			Id:      testEntryUUID,
			Name:    testEntryName,
			Type:    dvls.EntryCredentialType,
			SubType: dvls.EntryCredentialSubTypeDefault,
		}
		c, mockCred := newTestClient(map[string]dvls.Entry{testEntryUUID: entry})

		// First call populates the cache.
		id1, err := c.resolveEntryRef(context.Background(), testEntryName)
		assert.NoError(t, err)
		assert.Equal(t, testEntryUUID, id1)

		// Remove entries from mock so only the cache can satisfy the lookup.
		mockCred.entries = map[string]dvls.Entry{}

		id2, err := c.resolveEntryRef(context.Background(), testEntryName)
		assert.NoError(t, err)
		assert.Equal(t, testEntryUUID, id2)
	})
}

// --- Tests: parseLegacyRef ---

func TestParseLegacyRef(t *testing.T) {
	t.Run("valid format", func(t *testing.T) {
		vaultID, entryID, err := parseLegacyRef(testVaultUUID + "/" + testEntryUUID)
		assert.NoError(t, err)
		assert.Equal(t, testVaultUUID, vaultID)
		assert.Equal(t, testEntryUUID, entryID)
	})

	t.Run("no separator", func(t *testing.T) {
		_, _, err := parseLegacyRef(testEntryUUID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid key format")
	})

	t.Run("empty vault", func(t *testing.T) {
		_, _, err := parseLegacyRef("/" + testEntryUUID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "vault ID cannot be empty")
	})

	t.Run("empty entry", func(t *testing.T) {
		_, _, err := parseLegacyRef(testVaultUUID + "/")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "entry ID cannot be empty")
	})

	t.Run("invalid vault UUID", func(t *testing.T) {
		_, _, err := parseLegacyRef("not-a-uuid/" + testEntryUUID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid vault UUID")
	})

	t.Run("invalid entry UUID", func(t *testing.T) {
		_, _, err := parseLegacyRef(testVaultUUID + "/not-a-uuid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid entry UUID")
	})
}

// --- Tests: resolveRef legacy mode ---

func TestResolveRef_LegacyMode(t *testing.T) {
	entry := dvls.Entry{
		Id: testEntryUUID, Name: "test",
		Type: dvls.EntryCredentialType, SubType: dvls.EntryCredentialSubTypeDefault,
		Data: &dvls.EntryCredentialDefaultData{Password: "pass"},
	}

	t.Run("legacy format when vaultID is empty", func(t *testing.T) {
		mockCred := newMockCredentialClient(map[string]dvls.Entry{testEntryUUID: entry})
		c := NewClient(mockCred, newMockFolderClient(), "")
		vaultID, entryID, err := c.resolveRef(context.Background(), testVaultUUID+"/"+testEntryUUID)
		assert.NoError(t, err)
		assert.Equal(t, testVaultUUID, vaultID)
		assert.Equal(t, testEntryUUID, entryID)
	})

	t.Run("new format with name when vaultID is set", func(t *testing.T) {
		c, _ := newTestClient(map[string]dvls.Entry{testEntryUUID: entry})
		vaultID, entryID, err := c.resolveRef(context.Background(), "test")
		assert.NoError(t, err)
		assert.Equal(t, testVaultUUID, vaultID)
		assert.Equal(t, testEntryUUID, entryID)
	})

	t.Run("new format when vaultID is set", func(t *testing.T) {
		c, _ := newTestClient(map[string]dvls.Entry{testEntryUUID: entry})
		vaultID, entryID, err := c.resolveRef(context.Background(), testEntryUUID)
		assert.NoError(t, err)
		assert.Equal(t, testVaultUUID, vaultID)
		assert.Equal(t, testEntryUUID, entryID)
	})
}

// --- Tests: Validate ---

func TestClient_Validate(t *testing.T) {
	t.Run("nil cred client", func(t *testing.T) {
		c := &Client{cred: nil, vaultID: testVaultUUID}
		result, err := c.Validate()
		assert.Error(t, err)
		assert.Equal(t, esv1.ValidationResultError, result)
	})

	t.Run("empty vault ID is valid (legacy mode)", func(t *testing.T) {
		c := &Client{cred: newMockCredentialClient(nil), vaultID: ""}
		result, err := c.Validate()
		assert.NoError(t, err)
		assert.Equal(t, esv1.ValidationResultReady, result)
	})

	t.Run("initialized client", func(t *testing.T) {
		c := NewClient(newMockCredentialClient(nil), newMockFolderClient(), testVaultUUID)
		result, err := c.Validate()
		assert.NoError(t, err)
		assert.Equal(t, esv1.ValidationResultReady, result)
	})
}

func TestNewClient(t *testing.T) {
	c := NewClient(nil, nil, "")
	assert.NotNil(t, c)
	assert.Nil(t, c.cred)
	assert.Empty(t, c.vaultID)
}

// --- Tests: entryToSecretMap ---

func TestEntryToSecretMap(t *testing.T) {
	t.Run("Default credential type", func(t *testing.T) {
		entry := dvls.Entry{
			Id: "entry-id-123", Name: "test-entry",
			Type: dvls.EntryCredentialType, SubType: dvls.EntryCredentialSubTypeDefault,
			Data: &dvls.EntryCredentialDefaultData{Username: "testuser", Password: "testpass", Domain: "testdomain"},
		}
		secretMap, err := entryToSecretMap(entry)
		assert.NoError(t, err)
		assert.Equal(t, "testuser", string(secretMap["username"]))
		assert.Equal(t, "testpass", string(secretMap["password"]))
		assert.Equal(t, "testdomain", string(secretMap["domain"]))
	})

	t.Run("AccessCode credential type", func(t *testing.T) {
		entry := dvls.Entry{
			Id: "entry-id-456", Name: "access-code-entry",
			Type: dvls.EntryCredentialType, SubType: dvls.EntryCredentialSubTypeAccessCode,
			Data: &dvls.EntryCredentialAccessCodeData{Password: "access-code-123"},
		}
		secretMap, err := entryToSecretMap(entry)
		assert.NoError(t, err)
		assert.Equal(t, "access-code-123", string(secretMap["password"]))
	})

	t.Run("ApiKey credential type", func(t *testing.T) {
		entry := dvls.Entry{
			Id: "entry-id-789", Name: "api-key-entry",
			Type: dvls.EntryCredentialType, SubType: dvls.EntryCredentialSubTypeApiKey,
			Data: &dvls.EntryCredentialApiKeyData{ApiId: "api-id-123", ApiKey: "api-key-secret", TenantId: "tenant-123"},
		}
		secretMap, err := entryToSecretMap(entry)
		assert.NoError(t, err)
		assert.Equal(t, "api-id-123", string(secretMap["api-id"]))
		assert.Equal(t, "api-key-secret", string(secretMap["api-key"]))
	})

	t.Run("AzureServicePrincipal credential type", func(t *testing.T) {
		entry := dvls.Entry{
			Id: "entry-id-azure", Name: "azure-sp-entry",
			Type: dvls.EntryCredentialType, SubType: dvls.EntryCredentialSubTypeAzureServicePrincipal,
			Data: &dvls.EntryCredentialAzureServicePrincipalData{ClientId: "client-id-123", ClientSecret: "client-secret-456", TenantId: "tenant-id-789"},
		}
		secretMap, err := entryToSecretMap(entry)
		assert.NoError(t, err)
		assert.Equal(t, "client-id-123", string(secretMap["client-id"]))
		assert.Equal(t, "client-secret-456", string(secretMap["client-secret"]))
		assert.Equal(t, "tenant-id-789", string(secretMap["tenant-id"]))
	})

	t.Run("ConnectionString credential type", func(t *testing.T) {
		entry := dvls.Entry{
			Id: "entry-id-conn", Name: "connection-string-entry",
			Type: dvls.EntryCredentialType, SubType: dvls.EntryCredentialSubTypeConnectionString,
			Data: &dvls.EntryCredentialConnectionStringData{ConnectionString: "Server=localhost;Database=mydb"},
		}
		secretMap, err := entryToSecretMap(entry)
		assert.NoError(t, err)
		assert.Equal(t, "Server=localhost;Database=mydb", string(secretMap["connection-string"]))
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
				PrivateKey: "-----BEGIN RSA PRIVATE KEY-----",
				PublicKey:  "ssh-rsa AAAA",
				Passphrase: "my-passphrase",
			},
		}
		secretMap, err := entryToSecretMap(entry)
		assert.NoError(t, err)
		assert.Equal(t, "ssh-user", string(secretMap["username"]))
		assert.Equal(t, "key-password", string(secretMap["password"]))
		assert.Equal(t, "-----BEGIN RSA PRIVATE KEY-----", string(secretMap["private-key"]))
		assert.Equal(t, "ssh-rsa AAAA", string(secretMap["public-key"]))
		assert.Equal(t, "my-passphrase", string(secretMap["passphrase"]))
	})

	t.Run("Default credential with partial data", func(t *testing.T) {
		entry := dvls.Entry{
			Id: "entry-id-partial", Name: "partial-entry",
			Type: dvls.EntryCredentialType, SubType: dvls.EntryCredentialSubTypeDefault,
			Data: &dvls.EntryCredentialDefaultData{Username: "onlyuser"},
		}
		secretMap, err := entryToSecretMap(entry)
		assert.NoError(t, err)
		assert.Equal(t, "onlyuser", string(secretMap["username"]))
		_, hasPassword := secretMap["password"]
		_, hasDomain := secretMap["domain"]
		assert.False(t, hasPassword)
		assert.False(t, hasDomain)
	})

	t.Run("Unsupported credential type", func(t *testing.T) {
		entry := dvls.Entry{Id: "x", Name: "x", Type: dvls.EntryCredentialType, SubType: "UnknownType"}
		_, err := entryToSecretMap(entry)
		assert.Error(t, err)
	})
}

// --- Tests: GetSecret ---

func TestClient_GetSecret_NotFound(t *testing.T) {
	c, _ := newTestClient(nil)
	_, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: testEntryUUID})
	assert.ErrorIs(t, err, esv1.NoSecretErr)
}

func TestClient_GetSecret_Success(t *testing.T) {
	entry := dvls.Entry{
		Id: testEntryUUID, Name: "test-entry",
		Type: dvls.EntryCredentialType, SubType: dvls.EntryCredentialSubTypeDefault,
		Data: &dvls.EntryCredentialDefaultData{Password: "super-secret"},
	}
	c, _ := newTestClient(map[string]dvls.Entry{testEntryUUID: entry})

	val, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: testEntryUUID, Property: "password"})
	assert.NoError(t, err)
	assert.Equal(t, "super-secret", string(val))
}

func TestClient_GetSecret_ByName(t *testing.T) {
	entry := dvls.Entry{
		Id: testEntryUUID, Name: testSecretName,
		Type: dvls.EntryCredentialType, SubType: dvls.EntryCredentialSubTypeDefault,
		Data: &dvls.EntryCredentialDefaultData{Password: "name-resolved"},
	}
	c, _ := newTestClient(map[string]dvls.Entry{testEntryUUID: entry})

	val, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: testSecretName, Property: "password"})
	assert.NoError(t, err)
	assert.Equal(t, "name-resolved", string(val))
}

func TestClient_GetSecret_ByNameNotFound(t *testing.T) {
	c, _ := newTestClient(nil)
	_, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "nonexistent"})
	assert.ErrorIs(t, err, esv1.NoSecretErr)
}

func TestClient_GetSecret_WithPath(t *testing.T) {
	entry := dvls.Entry{
		Id: testEntryUUID, Name: "db", Path: `prod\databases`,
		Type: dvls.EntryCredentialType, SubType: dvls.EntryCredentialSubTypeDefault,
		Data: &dvls.EntryCredentialDefaultData{Password: "prod-pass"},
	}
	c, _ := newTestClient(map[string]dvls.Entry{testEntryUUID: entry})

	val, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "prod/databases/db", Property: "password"})
	assert.NoError(t, err)
	assert.Equal(t, "prod-pass", string(val))
}

func TestClient_GetSecret_VaultNotFound(t *testing.T) {
	c, mockCred := newTestClient(nil)
	mockCred.getErr = dvls.ErrVaultNotFound
	// UUID key bypasses name resolution, so GetByID is called directly.
	_, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: testEntryUUID})
	assert.Error(t, err)
	assert.ErrorIs(t, err, dvls.ErrVaultNotFound)
}

func TestClient_GetSecret_VaultNotFoundDuringNameResolution(t *testing.T) {
	c, mockCred := newTestClient(nil)
	mockCred.getEntriesErr = dvls.ErrVaultNotFound
	_, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: testNonExistName})
	assert.Error(t, err)
	assert.ErrorIs(t, err, dvls.ErrVaultNotFound)
}

// --- Tests: GetSecretMap ---

func TestClient_GetSecretMap_ByName(t *testing.T) {
	entry := dvls.Entry{
		Id: testEntryUUID, Name: testSecretName,
		Type: dvls.EntryCredentialType, SubType: dvls.EntryCredentialSubTypeDefault,
		Data: &dvls.EntryCredentialDefaultData{Username: "user", Password: "pass"},
	}
	c, _ := newTestClient(map[string]dvls.Entry{testEntryUUID: entry})

	secretMap, err := c.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: testSecretName})
	assert.NoError(t, err)
	assert.Equal(t, "user", string(secretMap["username"]))
	assert.Equal(t, "pass", string(secretMap["password"]))
}

func TestClient_GetSecretMap_NotFoundByUUID(t *testing.T) {
	c, _ := newTestClient(nil)
	_, err := c.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: testEntryUUID})
	assert.ErrorIs(t, err, esv1.NoSecretErr)
}

func TestClient_GetSecretMap_NotFoundByName(t *testing.T) {
	c, _ := newTestClient(nil)
	_, err := c.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "nonexistent"})
	assert.ErrorIs(t, err, esv1.NoSecretErr)
}

// --- Tests: SecretExists ---

func TestClient_SecretExists(t *testing.T) {
	c, mockCred := newTestClient(nil)

	exists, err := c.SecretExists(context.Background(), pushSecretRemoteRefStub{remoteKey: testEntryUUID})
	assert.NoError(t, err)
	assert.False(t, exists)

	mockCred.entries[testEntryUUID] = dvls.Entry{Id: testEntryUUID, Type: dvls.EntryCredentialType, SubType: dvls.EntryCredentialSubTypeDefault}

	exists, err = c.SecretExists(context.Background(), pushSecretRemoteRefStub{remoteKey: testEntryUUID})
	assert.NoError(t, err)
	assert.True(t, exists)

	mockCred.getErr = errors.New("boom")
	_, err = c.SecretExists(context.Background(), pushSecretRemoteRefStub{remoteKey: testEntryUUID})
	assert.Error(t, err)
}

func TestClient_SecretExists_ByName(t *testing.T) {
	entry := dvls.Entry{Id: testEntryUUID, Name: testEntryName, Type: dvls.EntryCredentialType, SubType: dvls.EntryCredentialSubTypeDefault}
	c, _ := newTestClient(map[string]dvls.Entry{testEntryUUID: entry})

	exists, err := c.SecretExists(context.Background(), pushSecretRemoteRefStub{remoteKey: testEntryName})
	assert.NoError(t, err)
	assert.True(t, exists)

	exists, err = c.SecretExists(context.Background(), pushSecretRemoteRefStub{remoteKey: "nonexistent"})
	assert.NoError(t, err)
	assert.False(t, exists)
}

// --- Tests: DeleteSecret ---

func TestClient_DeleteSecret(t *testing.T) {
	c, mockCred := newTestClient(map[string]dvls.Entry{testEntryUUID: {Id: testEntryUUID, Type: dvls.EntryCredentialType, SubType: dvls.EntryCredentialSubTypeAccessCode}})

	err := c.DeleteSecret(context.Background(), pushSecretRemoteRefStub{remoteKey: testEntryUUID})
	assert.NoError(t, err)
	assert.Equal(t, testEntryUUID, mockCred.lastDeleted)
}

func TestClient_DeleteSecret_ByName(t *testing.T) {
	entry := dvls.Entry{Id: testEntryUUID, Name: testEntryName, Type: dvls.EntryCredentialType, SubType: dvls.EntryCredentialSubTypeDefault}
	c, mockCred := newTestClient(map[string]dvls.Entry{testEntryUUID: entry})

	err := c.DeleteSecret(context.Background(), pushSecretRemoteRefStub{remoteKey: testEntryName})
	assert.NoError(t, err)
	assert.Equal(t, testEntryUUID, mockCred.lastDeleted)
}

func TestClient_DeleteSecret_ByNameNotFound(t *testing.T) {
	c, _ := newTestClient(nil)
	err := c.DeleteSecret(context.Background(), pushSecretRemoteRefStub{remoteKey: "nonexistent"})
	assert.NoError(t, err)
}

// --- Tests: PushSecret ---

func TestClient_PushSecret_UpdateDefault(t *testing.T) {
	c, mockCred := newTestClient(map[string]dvls.Entry{
		testEntryUUID: {Id: testEntryUUID, Type: dvls.EntryCredentialType, SubType: dvls.EntryCredentialSubTypeDefault},
	})
	secret := &corev1.Secret{Data: map[string][]byte{"password": []byte("new-value")}}
	data := pushSecretDataStub{remoteKey: testEntryUUID, secretKey: "password"}

	err := c.PushSecret(context.Background(), secret, data)
	assert.NoError(t, err)

	credData, ok := mockCred.entries[testEntryUUID].Data.(*dvls.EntryCredentialDefaultData)
	assert.True(t, ok)
	assert.Equal(t, "new-value", credData.Password)
}

func TestClient_PushSecret_ByName(t *testing.T) {
	entry := dvls.Entry{Id: testEntryUUID, Name: testEntryName, Type: dvls.EntryCredentialType, SubType: dvls.EntryCredentialSubTypeDefault}
	c, mockCred := newTestClient(map[string]dvls.Entry{testEntryUUID: entry})
	secret := &corev1.Secret{Data: map[string][]byte{"password": []byte("pushed-via-name")}}
	data := pushSecretDataStub{remoteKey: testEntryName, secretKey: "password"}

	err := c.PushSecret(context.Background(), secret, data)
	assert.NoError(t, err)

	credData, ok := mockCred.entries[testEntryUUID].Data.(*dvls.EntryCredentialDefaultData)
	assert.True(t, ok)
	assert.Equal(t, "pushed-via-name", credData.Password)
}

func TestClient_PushSecret_UpdateAccessCode(t *testing.T) {
	c, mockCred := newTestClient(map[string]dvls.Entry{
		testEntryUUID: {Id: testEntryUUID, Type: dvls.EntryCredentialType, SubType: dvls.EntryCredentialSubTypeAccessCode},
	})
	secret := &corev1.Secret{Data: map[string][]byte{"code": []byte("code-value")}}
	data := pushSecretDataStub{remoteKey: testEntryUUID, secretKey: "code"}

	err := c.PushSecret(context.Background(), secret, data)
	assert.NoError(t, err)

	credData, ok := mockCred.entries[testEntryUUID].Data.(*dvls.EntryCredentialAccessCodeData)
	assert.True(t, ok)
	assert.Equal(t, "code-value", credData.Password)
}

func TestClient_PushSecret_UnsupportedSubtype(t *testing.T) {
	c, _ := newTestClient(map[string]dvls.Entry{
		testEntryUUID: {Id: testEntryUUID, Type: dvls.EntryCredentialType, SubType: dvls.EntryCredentialSubTypeApiKey},
	})
	secret := &corev1.Secret{Data: map[string][]byte{"password": []byte("pw")}}
	data := pushSecretDataStub{remoteKey: testEntryUUID, secretKey: "password"}

	err := c.PushSecret(context.Background(), secret, data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot set secret for credential subtype")
}

func TestClient_PushSecret_UUIDNotFound(t *testing.T) {
	// A UUID key names one specific entry. If the vault no longer has it there
	// is nothing to create it as, so this stays an error.
	c, _ := newTestClient(nil)
	secret := &corev1.Secret{Data: map[string][]byte{"password": []byte("pw")}}
	data := pushSecretDataStub{remoteKey: "00000000-0000-0000-0000-000000000099", secretKey: "password"}

	err := c.PushSecret(context.Background(), secret, data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no longer exists")
}

func TestClient_PushSecret_VaultNotFoundDuringNameResolution(t *testing.T) {
	c, mockCred := newTestClient(nil)
	mockCred.getEntriesErr = dvls.ErrVaultNotFound
	secret := &corev1.Secret{Data: map[string][]byte{"password": []byte("pw")}}
	data := pushSecretDataStub{remoteKey: testNonExistName, secretKey: "password"}

	err := c.PushSecret(context.Background(), secret, data)
	assert.Error(t, err)
	assert.ErrorIs(t, err, dvls.ErrVaultNotFound)
}

func TestClient_PushSecret_CreatesMissingEntry(t *testing.T) {
	c, mockCred := newTestClient(nil)
	secret := &corev1.Secret{Data: map[string][]byte{"password": []byte("created-value")}}
	data := pushSecretDataStub{remoteKey: "nonexistent-entry", secretKey: "password"}

	err := c.PushSecret(context.Background(), secret, data)
	assert.NoError(t, err)

	created := mockCred.lastCreated
	assert.Equal(t, "nonexistent-entry", created.Name)
	assert.Equal(t, "", created.Path)
	assert.Equal(t, testVaultUUID, created.VaultId)
	assert.Equal(t, dvls.EntryCredentialType, created.Type)
	assert.Equal(t, dvls.EntryCredentialSubTypeAccessCode, created.SubType)

	credData, ok := created.Data.(*dvls.EntryCredentialAccessCodeData)
	assert.True(t, ok)
	assert.Equal(t, "created-value", credData.Password)
}

func TestClient_PushSecret_CreatesMissingEntryUnderPath(t *testing.T) {
	c, mockCred := newTestClient(nil)
	secret := &corev1.Secret{Data: map[string][]byte{"password": []byte("pw")}}
	data := pushSecretDataStub{remoteKey: `folder\sub\my-new-entry`, secretKey: "password"}

	err := c.PushSecret(context.Background(), secret, data)
	assert.NoError(t, err)
	assert.Equal(t, "my-new-entry", mockCred.lastCreated.Name)
	assert.Equal(t, `folder\sub`, mockCred.lastCreated.Path)
}

func TestClient_PushSecret_CreatesMissingEntryForwardSlashPath(t *testing.T) {
	c, mockCred := newTestClient(nil)
	secret := &corev1.Secret{Data: map[string][]byte{"password": []byte("pw")}}
	data := pushSecretDataStub{remoteKey: "folder/sub/my-new-entry", secretKey: "password"}

	err := c.PushSecret(context.Background(), secret, data)
	assert.NoError(t, err)
	assert.Equal(t, "my-new-entry", mockCred.lastCreated.Name)
	assert.Equal(t, `folder\sub`, mockCred.lastCreated.Path)
}

func TestClient_PushSecret_CreateIsReadableByGetSecret(t *testing.T) {
	// The created subtype must be the one GetSecret reads back with no
	// property set, or a push would produce an entry its own ExternalSecret
	// could not consume.
	c, _ := newTestClient(nil)
	secret := &corev1.Secret{Data: map[string][]byte{"password": []byte("round-trip")}}
	data := pushSecretDataStub{remoteKey: testNonExistName, secretKey: "password"}

	assert.NoError(t, c.PushSecret(context.Background(), secret, data))

	got, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: testNonExistName})
	assert.NoError(t, err)
	assert.Equal(t, []byte("round-trip"), got)
}

func TestClient_PushSecret_CreateRequiresVault(t *testing.T) {
	mockCred := newMockCredentialClient(nil)
	c := NewClient(mockCred, newMockFolderClient(), "")
	secret := &corev1.Secret{Data: map[string][]byte{"password": []byte("pw")}}
	data := pushSecretDataStub{remoteKey: "some-vault-id/some-entry-id", secretKey: "password"}

	err := c.PushSecret(context.Background(), secret, data)
	assert.Error(t, err)
}

func TestClient_PushSecret_CreatesEveryFolderLevel(t *testing.T) {
	// A nested path names folders that are entries in their own right, so each
	// level has to be created, from the root down, before the entry itself.
	c, mockCred, mockFolder := newTestClientWithFolders(nil)
	secret := &corev1.Secret{Data: map[string][]byte{"password": []byte("pw")}}
	data := pushSecretDataStub{remoteKey: "prod/db/postgres", secretKey: "password"}

	assert.NoError(t, c.PushSecret(context.Background(), secret, data))

	assert.Len(t, mockFolder.created, 2)
	assert.Equal(t, "prod", mockFolder.created[0].Name)
	assert.Equal(t, "", mockFolder.created[0].Path)
	assert.Equal(t, dvls.EntryFolderType, mockFolder.created[0].Type)
	assert.Equal(t, dvls.EntryFolderSubTypeFolder, mockFolder.created[0].SubType)
	assert.Equal(t, "db", mockFolder.created[1].Name)
	assert.Equal(t, "prod", mockFolder.created[1].Path)

	assert.Equal(t, "postgres", mockCred.lastCreated.Name)
	assert.Equal(t, `prod\db`, mockCred.lastCreated.Path)
}

func TestClient_PushSecret_ReusesExistingFolders(t *testing.T) {
	c, _, mockFolder := newTestClientWithFolders(nil, folder("prod", ""), folder("db", "prod"))
	secret := &corev1.Secret{Data: map[string][]byte{"password": []byte("pw")}}
	data := pushSecretDataStub{remoteKey: "prod/db/postgres", secretKey: "password"}

	assert.NoError(t, c.PushSecret(context.Background(), secret, data))
	assert.Empty(t, mockFolder.created)
}

func TestClient_PushSecret_CreatesOnlyTheMissingFolderLevels(t *testing.T) {
	c, _, mockFolder := newTestClientWithFolders(nil, folder("prod", ""))
	secret := &corev1.Secret{Data: map[string][]byte{"password": []byte("pw")}}
	data := pushSecretDataStub{remoteKey: "prod/db/postgres", secretKey: "password"}

	assert.NoError(t, c.PushSecret(context.Background(), secret, data))
	assert.Len(t, mockFolder.created, 1)
	assert.Equal(t, "db", mockFolder.created[0].Name)
	assert.Equal(t, "prod", mockFolder.created[0].Path)
}

func TestClient_PushSecret_FolderElsewhereDoesNotCount(t *testing.T) {
	// The server's path filter matches sub-paths too, so a folder of the same
	// name nested deeper, or sitting under a different parent, must not be
	// mistaken for the one the path asks for.
	c, _, mockFolder := newTestClientWithFolders(nil,
		folder("prod", ""),
		folder("db", "staging"),
		folder("db", `prod\legacy`),
	)
	secret := &corev1.Secret{Data: map[string][]byte{"password": []byte("pw")}}
	data := pushSecretDataStub{remoteKey: "prod/db/postgres", secretKey: "password"}

	assert.NoError(t, c.PushSecret(context.Background(), secret, data))
	assert.Len(t, mockFolder.created, 1)
	assert.Equal(t, "db", mockFolder.created[0].Name)
	assert.Equal(t, "prod", mockFolder.created[0].Path)
}

func TestClient_PushSecret_CreateUnderPathIsReadableByGetSecret(t *testing.T) {
	// The whole point of creating the folders: the entry has to be resolvable
	// again by the same remote key on the next reconciliation, or every push
	// would create another copy of it.
	c, mockCred, _ := newTestClientWithFolders(nil)
	secret := &corev1.Secret{Data: map[string][]byte{"password": []byte("round-trip")}}
	data := pushSecretDataStub{remoteKey: "prod/db/postgres", secretKey: "password"}

	assert.NoError(t, c.PushSecret(context.Background(), secret, data))

	// A fresh client, because the name cache does not outlive a reconciliation.
	next := NewClient(mockCred, newMockFolderClient(), testVaultUUID)
	got, err := next.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "prod/db/postgres"})
	assert.NoError(t, err)
	assert.Equal(t, []byte("round-trip"), got)

	// And the second push updates that entry instead of creating a second one.
	assert.NoError(t, next.PushSecret(context.Background(), secret, data))
	assert.Equal(t, 1, mockCred.created)
}

func TestClient_PushSecret_CachesFolderLookups(t *testing.T) {
	c, _, mockFolder := newTestClientWithFolders(nil)
	secret := &corev1.Secret{Data: map[string][]byte{"password": []byte("pw")}}

	assert.NoError(t, c.PushSecret(context.Background(), secret, pushSecretDataStub{remoteKey: "prod/db/first", secretKey: "password"}))
	assert.NoError(t, c.PushSecret(context.Background(), secret, pushSecretDataStub{remoteKey: "prod/db/second", secretKey: "password"}))

	// One lookup per level, on the first push only.
	assert.Equal(t, 2, mockFolder.lookups)
	assert.Len(t, mockFolder.created, 2)
}

func TestClient_PushSecret_FolderCreateError(t *testing.T) {
	c, mockCred, mockFolder := newTestClientWithFolders(nil)
	mockFolder.newErr = errors.New("boom")
	secret := &corev1.Secret{Data: map[string][]byte{"password": []byte("pw")}}
	data := pushSecretDataStub{remoteKey: "prod/db/postgres", secretKey: "password"}

	err := c.PushSecret(context.Background(), secret, data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), `failed to create folder "prod"`)
	// The entry is not created into a tree that is not there.
	assert.Equal(t, 0, mockCred.created)
}

func TestClient_PushSecret_FolderLookupError(t *testing.T) {
	c, mockCred, mockFolder := newTestClientWithFolders(nil)
	mockFolder.getEntriesErr = errors.New("boom")
	secret := &corev1.Secret{Data: map[string][]byte{"password": []byte("pw")}}
	data := pushSecretDataStub{remoteKey: "prod/db/postgres", secretKey: "password"}

	err := c.PushSecret(context.Background(), secret, data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), `failed to look up folder "prod"`)
	assert.Equal(t, 0, mockCred.created)
}

func TestClient_PushSecret_RejectsEmptyFolderSegment(t *testing.T) {
	c, mockCred, _ := newTestClientWithFolders(nil)
	secret := &corev1.Secret{Data: map[string][]byte{"password": []byte("pw")}}
	data := pushSecretDataStub{remoteKey: "prod//db/postgres", secretKey: "password"}

	err := c.PushSecret(context.Background(), secret, data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "a path segment cannot be empty")
	assert.Equal(t, 0, mockCred.created)
}

func TestClient_PushSecret_CreateError(t *testing.T) {
	c, mockCred := newTestClient(nil)
	mockCred.newErr = errors.New("boom")
	secret := &corev1.Secret{Data: map[string][]byte{"password": []byte("pw")}}
	data := pushSecretDataStub{remoteKey: testNonExistName, secretKey: "password"}

	err := c.PushSecret(context.Background(), secret, data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create entry")
}

// --- Tests: isNotFoundError ---

func TestIsNotFoundError(t *testing.T) {
	assert.False(t, isNotFoundError(nil))
	assert.True(t, isNotFoundError(&dvls.RequestError{Err: fmt.Errorf("not found"), StatusCode: http.StatusNotFound}))
	assert.True(t, isNotFoundError(dvls.ErrEntryNotFound))
	assert.True(t, isNotFoundError(fmt.Errorf("wrapped: %w", dvls.ErrEntryNotFound)))
	assert.False(t, isNotFoundError(dvls.ErrMultipleEntriesFound))
	assert.False(t, isNotFoundError(errors.New("some other error")))
}

func TestIsVaultNotFoundError(t *testing.T) {
	assert.False(t, isVaultNotFoundError(nil))
	assert.True(t, isVaultNotFoundError(dvls.ErrVaultNotFound))
	assert.True(t, isVaultNotFoundError(fmt.Errorf("wrapped: %w", dvls.ErrVaultNotFound)))
	assert.False(t, isVaultNotFoundError(errors.New("some other error")))
}
