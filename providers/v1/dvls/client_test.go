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
	"sync"
	"testing"
	"time"

	"github.com/Devolutions/go-dvls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

const (
	testVaultUUID    = "00000000-0000-0000-0000-000000000001"
	testVaultUUID2   = "00000000-0000-0000-0000-0000000000a1"
	testVaultName2   = "other-vault"
	testEntryUUID    = "00000000-0000-0000-0000-000000000002"
	testEntryUUID3   = "00000000-0000-0000-0000-000000000003"
	testEntryUUID4   = "00000000-0000-0000-0000-000000000004"
	testEntryUUID5   = "00000000-0000-0000-0000-000000000005"
	testEntryName    = "my-entry"
	testVaultName    = "my-vault"
	testSecretName   = "my-secret"
	testNonExistName = "some-name"
)

// --- Mock credential client ---

type mockCredentialClient struct {
	entries       map[string]dvls.Entry
	getErr        error
	getEntriesErr error
	updateErr     error
	deleteErr     error
	lastUpdated   dvls.Entry
	lastDeleted   string

	mu           sync.Mutex
	entriesCalls map[string]int
}

func (m *mockCredentialClient) countEntriesCall(vaultID string) {
	m.mu.Lock()
	m.entriesCalls[vaultID]++
	m.mu.Unlock()
}

func newMockCredentialClient(entries map[string]dvls.Entry) *mockCredentialClient {
	if entries == nil {
		entries = make(map[string]dvls.Entry)
	}
	return &mockCredentialClient{entries: entries, entriesCalls: make(map[string]int)}
}

// entryMatchesVault scopes lookups to a vault so wrong-vault bugs are visible. Entries with
// no VaultId belong to every vault, keeping single-vault fixtures terse.
func entryMatchesVault(e dvls.Entry, vaultID string) bool {
	return e.VaultId == "" || e.VaultId == vaultID
}

func (m *mockCredentialClient) GetByID(_ context.Context, vaultID, entryID string) (dvls.Entry, error) {
	if m.getErr != nil {
		return dvls.Entry{}, m.getErr
	}

	if entry, ok := m.entries[entryID]; ok && entryMatchesVault(entry, vaultID) {
		return entry, nil
	}

	return dvls.Entry{}, &dvls.RequestError{Err: fmt.Errorf("unexpected status code %d", http.StatusNotFound), Url: entryID, StatusCode: http.StatusNotFound}
}

func (m *mockCredentialClient) GetEntries(_ context.Context, vaultID string, opts dvls.GetEntriesOptions) ([]dvls.Entry, error) {
	m.countEntriesCall(vaultID)
	if m.getEntriesErr != nil {
		return nil, m.getEntriesErr
	}
	if opts.Name == nil {
		return nil, nil
	}
	var matches []dvls.Entry
	for _, e := range m.entries {
		if e.Name == *opts.Name && entryMatchesVault(e, vaultID) {
			if opts.Path != nil && e.Path != *opts.Path {
				continue
			}
			matches = append(matches, e)
		}
	}
	return matches, nil
}

func (m *mockCredentialClient) Update(_ context.Context, entry dvls.Entry) (dvls.Entry, error) {
	if m.updateErr != nil {
		return entry, m.updateErr
	}
	m.entries[entry.Id] = entry
	m.lastUpdated = entry
	return entry, nil
}

func (m *mockCredentialClient) DeleteByID(_ context.Context, vaultID, entryID string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}

	// Mirror the API: deleting an entry the vault does not hold is a 404.
	if entry, ok := m.entries[entryID]; !ok || !entryMatchesVault(entry, vaultID) {
		return &dvls.RequestError{Err: fmt.Errorf("unexpected status code %d", http.StatusNotFound), Url: entryID, StatusCode: http.StatusNotFound}
	}

	delete(m.entries, entryID)
	m.lastDeleted = entryID
	return nil
}

// --- Mock vault client ---

type mockVaultClient struct {
	vaults     map[string]dvls.Vault
	getErr     error
	getByIDErr error
	listErr    error
	duplicate  string
	listGate   chan struct{}

	mu        sync.Mutex
	listCalls int
	getCalls  int
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

// Get reports the vault as reachable unless getByIDErr is set, so tests that expect a
// plain missing entry keep working and only vault-outage tests opt in.
func (m *mockVaultClient) Get(_ context.Context, id string) (dvls.Vault, error) {
	m.mu.Lock()
	m.getCalls++
	m.mu.Unlock()
	if m.getByIDErr != nil {
		return dvls.Vault{}, m.getByIDErr
	}
	return dvls.Vault{Id: id}, nil
}

func (m *mockVaultClient) List(_ context.Context) ([]dvls.Vault, error) {
	m.mu.Lock()
	m.listCalls++
	m.mu.Unlock()
	// listGate lets a test hold List open to prove concurrent callers share one enumeration.
	if m.listGate != nil {
		<-m.listGate
	}
	if m.listErr != nil {
		return nil, m.listErr
	}
	list := make([]dvls.Vault, 0, len(m.vaults)+1)
	for name, v := range m.vaults {
		v.Name = name
		list = append(list, v)
	}
	if m.duplicate != "" {
		list = append(list, dvls.Vault{Id: testVaultUUID2, Name: m.duplicate})
	}
	return list, nil
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

// --- Helpers to create a test client ---

// newTestClient builds a client with the vault pinned by the store.
func newTestClient(entries map[string]dvls.Entry) (*Client, *mockCredentialClient) {
	mockCred := newMockCredentialClient(entries)
	c := NewClient(mockCred, newMockVaultClient(nil), testVaultUUID, true)
	return c, mockCred
}

// newDynamicTestClient builds a client with no pinned vault, so keys carry the vault.
func newDynamicTestClient(entries map[string]dvls.Entry, vaults map[string]dvls.Vault) (*Client, *mockCredentialClient, *mockVaultClient) {
	mockCred := newMockCredentialClient(entries)
	mockVault := newMockVaultClient(vaults)
	c := NewClient(mockCred, mockVault, "", false)
	return c, mockCred, mockVault
}

func defaultTestVaults() map[string]dvls.Vault {
	return map[string]dvls.Vault{
		testVaultName:  {Id: testVaultUUID, Name: testVaultName},
		testVaultName2: {Id: testVaultUUID2, Name: testVaultName2},
	}
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
		entryID, err := c.resolveEntryRef(context.Background(), testVaultUUID, testEntryUUID)
		assert.NoError(t, err)
		assert.Equal(t, testEntryUUID, entryID)
	})

	t.Run("name resolved", func(t *testing.T) {
		c, _ := newTestClient(map[string]dvls.Entry{testEntryUUID: entry})
		entryID, err := c.resolveEntryRef(context.Background(), testVaultUUID, testEntryName)
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
		entryID, err := c.resolveEntryRef(context.Background(), testVaultUUID, "go-dvls/folders/my-entry")
		assert.NoError(t, err)
		assert.Equal(t, testEntryUUID, entryID)
	})

	t.Run("path filters out other paths", func(t *testing.T) {
		entryA := dvls.Entry{Id: testEntryUUID, Name: "db", Path: `prod`, Type: dvls.EntryCredentialType, SubType: dvls.EntryCredentialSubTypeDefault}
		entryB := dvls.Entry{Id: testEntryUUID5, Name: "db", Path: `staging`, Type: dvls.EntryCredentialType, SubType: dvls.EntryCredentialSubTypeDefault}
		c, _ := newTestClient(map[string]dvls.Entry{testEntryUUID: entryA, testEntryUUID5: entryB})
		entryID, err := c.resolveEntryRef(context.Background(), testVaultUUID, "prod/db")
		assert.NoError(t, err)
		assert.Equal(t, testEntryUUID, entryID)
	})

	t.Run("name not found", func(t *testing.T) {
		c, _ := newTestClient(nil)
		_, err := c.resolveEntryRef(context.Background(), testVaultUUID, "nonexistent")
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
		_, err := c.resolveEntryRef(context.Background(), testVaultUUID, "dup")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "found 2 credential entries")
		assert.Contains(t, err.Error(), testEntryUUID3)
		assert.Contains(t, err.Error(), testEntryUUID4)
	})

	t.Run("empty key", func(t *testing.T) {
		c, _ := newTestClient(nil)
		_, err := c.resolveEntryRef(context.Background(), testVaultUUID, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("trailing separator produces empty name", func(t *testing.T) {
		c, _ := newTestClient(nil)
		_, err := c.resolveEntryRef(context.Background(), testVaultUUID, "folder/")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "entry name cannot be empty")
	})

	// A non-404 failure is not a vault verdict, so it keeps the plain message and its chain.
	t.Run("GetEntries API error", func(t *testing.T) {
		c, mockCred := newTestClient(nil)
		wrapped := errors.New("network error")
		mockCred.getEntriesErr = wrapped
		_, err := c.resolveEntryRef(context.Background(), testVaultUUID, testNonExistName)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to resolve entry")
		assert.ErrorIs(t, err, wrapped)
		assert.False(t, isNotFoundError(err))
	})

	t.Run("vault not found during name resolution", func(t *testing.T) {
		c, mockCred := newTestClient(nil)
		mockCred.getEntriesErr = dvls.ErrVaultNotFound
		_, err := c.resolveEntryRef(context.Background(), testVaultUUID, testNonExistName)
		assert.Error(t, err)
		assert.ErrorIs(t, err, dvls.ErrVaultNotFound)
		assert.False(t, isNotFoundError(err))
	})

	// A 404 from GetEntries is a vault/transport failure, never a missing entry: a missing
	// name returns an empty list. It must not reach isNotFoundError or GetSecret would
	// report NoSecretErr and deletionPolicy: Delete would prune a live Secret.
	t.Run("GetEntries 404 is not treated as not-found", func(t *testing.T) {
		c, mockCred := newTestClient(nil)
		mockCred.getEntriesErr = &dvls.RequestError{Err: errors.New("unexpected status code 404"), StatusCode: http.StatusNotFound}
		_, err := c.resolveEntryRef(context.Background(), testVaultUUID, testNonExistName)
		assert.Error(t, err)
		assert.False(t, isNotFoundError(err))
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
		id1, err := c.resolveEntryRef(context.Background(), testVaultUUID, testEntryName)
		assert.NoError(t, err)
		assert.Equal(t, testEntryUUID, id1)

		// Remove entries from mock so only the cache can satisfy the lookup.
		mockCred.entries = map[string]dvls.Entry{}

		id2, err := c.resolveEntryRef(context.Background(), testVaultUUID, testEntryName)
		assert.NoError(t, err)
		assert.Equal(t, testEntryUUID, id2)
	})
}

// --- Tests: splitVaultRef ---

func TestSplitVaultRef(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		wantVault string
		wantEntry string
		wantErr   string
	}{
		{name: "legacy UUID pair", key: testVaultUUID + "/" + testEntryUUID, wantVault: testVaultUUID, wantEntry: testEntryUUID},
		{name: "vault name and entry name", key: "my-vault/db-creds", wantVault: "my-vault", wantEntry: "db-creds"},
		{name: "vault name with folder path", key: "my-vault/infra/dbs/db-creds", wantVault: "my-vault", wantEntry: "infra/dbs/db-creds"},
		{name: "backslash separator", key: `my-vault\db-creds`, wantVault: "my-vault", wantEntry: "db-creds"},
		{name: "vault UUID with entry name", key: testVaultUUID + "/db-creds", wantVault: testVaultUUID, wantEntry: "db-creds"},
		{name: "surrounding whitespace", key: "  my-vault/db-creds  ", wantVault: "my-vault", wantEntry: "db-creds"},
		{name: "uppercase UUIDs", key: strings.ToUpper(testVaultUUID) + "/" + strings.ToUpper(testEntryUUID), wantVault: strings.ToUpper(testVaultUUID), wantEntry: strings.ToUpper(testEntryUUID)},

		{name: "no separator", key: testEntryUUID, wantErr: "invalid key format"},
		{name: "empty vault", key: "/" + testEntryUUID, wantErr: "vault reference cannot be empty"},
		{name: "empty entry", key: testVaultUUID + "/", wantErr: "entry reference cannot be empty"},
		// These two fail locally today and must keep failing: PushSecret and DeleteSecret
		// share this parser, so accepting them would let an old broken key mutate a real entry.
		{name: "doubled separator", key: testVaultUUID + "//" + testEntryUUID, wantErr: "empty path segment"},
		{name: "mixed doubled separator", key: `my-vault/\db-creds`, wantErr: "empty path segment"},
		{name: "entry UUID with trailing segment", key: testVaultUUID + "/" + testEntryUUID + "/extra", wantErr: "must be the whole entry reference"},
		{name: "trailing separator on path", key: "my-vault/folder/", wantErr: "empty path segment"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vaultRef, entryRef, err := splitVaultRef(tt.key)
			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantVault, vaultRef)
			assert.Equal(t, tt.wantEntry, entryRef)
		})
	}
}

// --- Tests: resolveRef mode switching ---

func TestResolveRef_PinnedVault(t *testing.T) {
	entry := dvls.Entry{
		Id: testEntryUUID, Name: "test",
		Type: dvls.EntryCredentialType, SubType: dvls.EntryCredentialSubTypeDefault,
		Data: &dvls.EntryCredentialDefaultData{Password: "pass"},
	}

	t.Run("name resolved in pinned vault", func(t *testing.T) {
		c, _ := newTestClient(map[string]dvls.Entry{testEntryUUID: entry})
		vaultID, entryID, err := c.resolveRef(context.Background(), "test")
		assert.NoError(t, err)
		assert.Equal(t, testVaultUUID, vaultID)
		assert.Equal(t, testEntryUUID, entryID)
	})

	t.Run("UUID resolved in pinned vault", func(t *testing.T) {
		c, _ := newTestClient(map[string]dvls.Entry{testEntryUUID: entry})
		vaultID, entryID, err := c.resolveRef(context.Background(), testEntryUUID)
		assert.NoError(t, err)
		assert.Equal(t, testVaultUUID, vaultID)
		assert.Equal(t, testEntryUUID, entryID)
	})

	// A pinned vault keeps '/' meaning folder path, so a key that looks like
	// "<vault>/<entry>" must not be reinterpreted as a vault reference.
	t.Run("slash stays a folder path", func(t *testing.T) {
		inFolder := dvls.Entry{
			Id: testEntryUUID3, Name: "db", Path: "prod",
			Type: dvls.EntryCredentialType, SubType: dvls.EntryCredentialSubTypeDefault,
		}
		c, _ := newTestClient(map[string]dvls.Entry{testEntryUUID3: inFolder})
		vaultID, entryID, err := c.resolveRef(context.Background(), "prod/db")
		assert.NoError(t, err)
		assert.Equal(t, testVaultUUID, vaultID)
		assert.Equal(t, testEntryUUID3, entryID)
	})
}

func TestResolveRef_VaultInKey(t *testing.T) {
	entry := dvls.Entry{
		Id: testEntryUUID, Name: "test", VaultId: testVaultUUID,
		Type: dvls.EntryCredentialType, SubType: dvls.EntryCredentialSubTypeDefault,
	}

	t.Run("legacy UUID pair needs no vault lookup", func(t *testing.T) {
		c, _, mockVault := newDynamicTestClient(map[string]dvls.Entry{testEntryUUID: entry}, defaultTestVaults())
		vaultID, entryID, err := c.resolveRef(context.Background(), testVaultUUID+"/"+testEntryUUID)
		assert.NoError(t, err)
		assert.Equal(t, testVaultUUID, vaultID)
		assert.Equal(t, testEntryUUID, entryID)
		assert.Equal(t, 0, mockVault.listCalls)
	})

	t.Run("vault name and entry name", func(t *testing.T) {
		c, _, _ := newDynamicTestClient(map[string]dvls.Entry{testEntryUUID: entry}, defaultTestVaults())
		vaultID, entryID, err := c.resolveRef(context.Background(), testVaultName+"/test")
		assert.NoError(t, err)
		assert.Equal(t, testVaultUUID, vaultID)
		assert.Equal(t, testEntryUUID, entryID)
	})

	t.Run("vault name with folder path", func(t *testing.T) {
		inFolder := dvls.Entry{
			Id: testEntryUUID3, Name: "db", Path: `infra\dbs`, VaultId: testVaultUUID,
			Type: dvls.EntryCredentialType, SubType: dvls.EntryCredentialSubTypeDefault,
		}
		c, _, _ := newDynamicTestClient(map[string]dvls.Entry{testEntryUUID3: inFolder}, defaultTestVaults())
		vaultID, entryID, err := c.resolveRef(context.Background(), testVaultName+"/infra/dbs/db")
		assert.NoError(t, err)
		assert.Equal(t, testVaultUUID, vaultID)
		assert.Equal(t, testEntryUUID3, entryID)
	})

	t.Run("single segment key is rejected", func(t *testing.T) {
		c, _, _ := newDynamicTestClient(nil, defaultTestVaults())
		_, _, err := c.resolveRef(context.Background(), "db-creds")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid key format")
	})

	t.Run("unknown vault name", func(t *testing.T) {
		c, _, _ := newDynamicTestClient(nil, defaultTestVaults())
		_, _, err := c.resolveRef(context.Background(), "nope/db-creds")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), `vault "nope"`)
		assert.False(t, isNotFoundError(err))
	})

	// The vault index is built once per client: GetByName enumerates every vault per call,
	// so resolving N names must not cost N enumerations.
	t.Run("vault index built once", func(t *testing.T) {
		other := dvls.Entry{
			Id: testEntryUUID4, Name: "test", VaultId: testVaultUUID2,
			Type: dvls.EntryCredentialType, SubType: dvls.EntryCredentialSubTypeDefault,
		}
		c, _, mockVault := newDynamicTestClient(map[string]dvls.Entry{testEntryUUID: entry, testEntryUUID4: other}, defaultTestVaults())

		_, _, err := c.resolveRef(context.Background(), testVaultName+"/test")
		assert.NoError(t, err)
		_, _, err = c.resolveRef(context.Background(), testVaultName2+"/test")
		assert.NoError(t, err)
		assert.Equal(t, 1, mockVault.listCalls)
	})

	// nameCache is keyed by vault: one client now serves several, so an unscoped cache
	// would return the first vault's entry for every later vault. Repeating each lookup
	// also pins the cache itself — a removed cache would issue a second GetEntries.
	t.Run("same entry name resolves per vault", func(t *testing.T) {
		other := dvls.Entry{
			Id: testEntryUUID4, Name: "test", VaultId: testVaultUUID2,
			Type: dvls.EntryCredentialType, SubType: dvls.EntryCredentialSubTypeDefault,
		}
		c, mockCred, _ := newDynamicTestClient(map[string]dvls.Entry{testEntryUUID: entry, testEntryUUID4: other}, defaultTestVaults())

		for range 2 {
			vaultID, entryID, err := c.resolveRef(context.Background(), testVaultName+"/test")
			assert.NoError(t, err)
			assert.Equal(t, testVaultUUID, vaultID)
			assert.Equal(t, testEntryUUID, entryID)

			vaultID, entryID, err = c.resolveRef(context.Background(), testVaultName2+"/test")
			assert.NoError(t, err)
			assert.Equal(t, testVaultUUID2, vaultID)
			assert.Equal(t, testEntryUUID4, entryID)
		}

		// One lookup per vault, not per reference.
		assert.Equal(t, map[string]int{testVaultUUID: 1, testVaultUUID2: 1}, mockCred.entriesCalls)
	})

	t.Run("duplicate vault names error", func(t *testing.T) {
		c, _, mockVault := newDynamicTestClient(nil, defaultTestVaults())
		mockVault.duplicate = testVaultName
		_, _, err := c.resolveRef(context.Background(), testVaultName+"/test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "multiple vaults named")
	})

	t.Run("vault list error is not a not-found", func(t *testing.T) {
		c, _, mockVault := newDynamicTestClient(nil, defaultTestVaults())
		mockVault.listErr = &dvls.RequestError{Err: errors.New("unexpected status code 404"), StatusCode: http.StatusNotFound}
		_, _, err := c.resolveRef(context.Background(), testVaultName+"/test")
		assert.Error(t, err)
		assert.False(t, isNotFoundError(err))
	})

	// Holds List open so every other goroutine is in flight while the first enumerates.
	// Without the lock spanning List, the waiters would each start their own enumeration.
	t.Run("concurrent callers share one enumeration", func(t *testing.T) {
		other := dvls.Entry{
			Id: testEntryUUID4, Name: "test", VaultId: testVaultUUID2,
			Type: dvls.EntryCredentialType, SubType: dvls.EntryCredentialSubTypeDefault,
		}
		c, _, mockVault := newDynamicTestClient(map[string]dvls.Entry{testEntryUUID: entry, testEntryUUID4: other}, defaultTestVaults())
		mockVault.listGate = make(chan struct{})

		var wg sync.WaitGroup
		for i := range 20 {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				name := testVaultName
				if i%2 == 0 {
					name = testVaultName2
				}
				_, _, err := c.resolveRef(context.Background(), name+"/test")
				assert.NoError(t, err)
			}(i)
		}

		// Give every goroutine time to pile up, then confirm only one reached List.
		for range 50 {
			time.Sleep(time.Millisecond)
			mockVault.mu.Lock()
			calls := mockVault.listCalls
			mockVault.mu.Unlock()
			require.LessOrEqual(t, calls, 1, "more than one goroutine entered List")
		}

		close(mockVault.listGate)
		wg.Wait()
		assert.Equal(t, 1, mockVault.listCalls)
	})

	t.Run("invalid vault UUID from List is rejected", func(t *testing.T) {
		c, _, _ := newDynamicTestClient(nil, map[string]dvls.Vault{
			testVaultName: {Id: "not-a-uuid", Name: testVaultName},
		})
		_, _, err := c.resolveRef(context.Background(), testVaultName+"/test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid UUID")
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

	t.Run("pinned vault without vault client is valid", func(t *testing.T) {
		c := &Client{cred: newMockCredentialClient(nil), vaultID: testVaultUUID, pinnedVault: true}
		result, err := c.Validate()
		assert.NoError(t, err)
		assert.Equal(t, esv1.ValidationResultReady, result)
	})

	t.Run("vault in key requires a vault client", func(t *testing.T) {
		c := &Client{cred: newMockCredentialClient(nil)}
		result, err := c.Validate()
		assert.Error(t, err)
		assert.Equal(t, esv1.ValidationResultError, result)
	})

	t.Run("initialized client", func(t *testing.T) {
		c := NewClient(newMockCredentialClient(nil), newMockVaultClient(nil), testVaultUUID, true)
		result, err := c.Validate()
		assert.NoError(t, err)
		assert.Equal(t, esv1.ValidationResultReady, result)
	})
}

func TestNewClient(t *testing.T) {
	c := NewClient(nil, nil, "", false)
	assert.NotNil(t, c)
	assert.Nil(t, c.cred)
	assert.Empty(t, c.vaultID)
	assert.False(t, c.pinnedVault)
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
	assert.NotErrorIs(t, err, esv1.NoSecretErr)
}

// An unreachable vault must surface as an error, never as NoSecretErr: the reconciler
// treats NoSecretErr as "the secret is gone" and deletionPolicy: Delete acts on it.
func TestClient_GetSecret_VaultErrorIsNotNoSecretErr(t *testing.T) {
	notFound := &dvls.RequestError{Err: errors.New("unexpected status code 404"), StatusCode: http.StatusNotFound}

	t.Run("vault list 404", func(t *testing.T) {
		c, _, mockVault := newDynamicTestClient(nil, defaultTestVaults())
		mockVault.listErr = notFound
		_, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: testVaultName + "/db"})
		assert.Error(t, err)
		assert.NotErrorIs(t, err, esv1.NoSecretErr)
	})

	t.Run("unknown vault name", func(t *testing.T) {
		c, _, _ := newDynamicTestClient(nil, defaultTestVaults())
		_, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "nope/db"})
		assert.Error(t, err)
		assert.NotErrorIs(t, err, esv1.NoSecretErr)
	})

	t.Run("GetEntries 404", func(t *testing.T) {
		c, mockCred := newTestClient(nil)
		mockCred.getEntriesErr = notFound
		_, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: testNonExistName})
		assert.Error(t, err)
		assert.NotErrorIs(t, err, esv1.NoSecretErr)
	})

	t.Run("GetSecretMap vault list 404", func(t *testing.T) {
		c, _, mockVault := newDynamicTestClient(nil, defaultTestVaults())
		mockVault.listErr = notFound
		_, err := c.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: testVaultName + "/db"})
		assert.Error(t, err)
		assert.NotErrorIs(t, err, esv1.NoSecretErr)
	})
}

// The SDK addresses entries as /vault/{vaultID}/entry/{entryID}, so a direct 404 cannot
// distinguish a deleted entry from an unreachable vault. Legacy "<vault-uuid>/<entry-uuid>"
// keys hit this path with no name resolution at all, so the vault must be confirmed before
// the entry is reported absent.
func TestClient_DirectNotFound_DisambiguatesVault(t *testing.T) {
	notFound := &dvls.RequestError{Err: errors.New("unexpected status code 404"), StatusCode: http.StatusNotFound}
	legacyKey := testVaultUUID + "/" + testEntryUUID

	newClient := func(vaultReachable bool) (*Client, *mockVaultClient) {
		c, _, mockVault := newDynamicTestClient(nil, defaultTestVaults())
		if !vaultReachable {
			mockVault.getByIDErr = notFound
		}
		return c, mockVault
	}

	t.Run("GetSecret reports absent when vault is reachable", func(t *testing.T) {
		c, mockVault := newClient(true)
		_, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: legacyKey})
		assert.ErrorIs(t, err, esv1.NoSecretErr)
		assert.Equal(t, 1, mockVault.getCalls)
	})

	t.Run("GetSecret errors when vault is unreachable", func(t *testing.T) {
		c, _ := newClient(false)
		_, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: legacyKey})
		assert.Error(t, err)
		assert.NotErrorIs(t, err, esv1.NoSecretErr)
	})

	t.Run("GetSecretMap errors when vault is unreachable", func(t *testing.T) {
		c, _ := newClient(false)
		_, err := c.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: legacyKey})
		assert.Error(t, err)
		assert.NotErrorIs(t, err, esv1.NoSecretErr)
	})

	t.Run("DeleteSecret is idempotent only when vault is reachable", func(t *testing.T) {
		c, _ := newClient(true)
		assert.NoError(t, c.DeleteSecret(context.Background(), pushSecretRemoteRefStub{remoteKey: legacyKey}))

		c, _ = newClient(false)
		assert.Error(t, c.DeleteSecret(context.Background(), pushSecretRemoteRefStub{remoteKey: legacyKey}))
	})

	t.Run("SecretExists reports false only when vault is reachable", func(t *testing.T) {
		c, _ := newClient(true)
		exists, err := c.SecretExists(context.Background(), pushSecretRemoteRefStub{remoteKey: legacyKey})
		assert.NoError(t, err)
		assert.False(t, exists)

		c, _ = newClient(false)
		exists, err = c.SecretExists(context.Background(), pushSecretRemoteRefStub{remoteKey: legacyKey})
		assert.Error(t, err)
		assert.False(t, exists)
	})

	t.Run("PushSecret errors on vault outage rather than blaming the entry", func(t *testing.T) {
		secret := &corev1.Secret{Data: map[string][]byte{"key": []byte("val")}}
		data := pushSecretDataStub{remoteKey: legacyKey, secretKey: "key"}

		c, _ := newClient(false)
		err := c.PushSecret(context.Background(), secret, data)
		assert.Error(t, err)
		assert.NotContains(t, err.Error(), "must exist before pushing")

		c, _ = newClient(true)
		err = c.PushSecret(context.Background(), secret, data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must exist before pushing")
	})
}

// Ordinary transport failures must keep their error chain: a reconciler canceling its
// context needs errors.Is(err, context.Canceled) to keep working.
func TestClient_TransportErrorsPreserveChain(t *testing.T) {
	c, mockCred := newTestClient(nil)
	mockCred.getEntriesErr = fmt.Errorf("dialing: %w", context.Canceled)

	_, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: testNonExistName})
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.NotErrorIs(t, err, esv1.NoSecretErr)
}

// DeleteSecret and SecretExists must not report success / "absent" when the vault is
// unreachable, or a push flow would silently skip real work.
func TestClient_MutatingPaths_VaultErrorSurfaces(t *testing.T) {
	notFound := &dvls.RequestError{Err: errors.New("unexpected status code 404"), StatusCode: http.StatusNotFound}

	t.Run("DeleteSecret", func(t *testing.T) {
		c, _, mockVault := newDynamicTestClient(nil, defaultTestVaults())
		mockVault.listErr = notFound
		err := c.DeleteSecret(context.Background(), pushSecretRemoteRefStub{remoteKey: testVaultName + "/db"})
		assert.Error(t, err)
	})

	t.Run("SecretExists", func(t *testing.T) {
		c, _, mockVault := newDynamicTestClient(nil, defaultTestVaults())
		mockVault.listErr = notFound
		exists, err := c.SecretExists(context.Background(), pushSecretRemoteRefStub{remoteKey: testVaultName + "/db"})
		assert.Error(t, err)
		assert.False(t, exists)
	})

	t.Run("PushSecret names the vault reference", func(t *testing.T) {
		c, _, _ := newDynamicTestClient(nil, defaultTestVaults())
		secret := &corev1.Secret{Data: map[string][]byte{"key": []byte("val")}}
		err := c.PushSecret(context.Background(), secret, pushSecretDataStub{remoteKey: "nope/db", secretKey: "key"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), `vault "nope"`)
	})
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

func TestClient_PushSecret_NotFound(t *testing.T) {
	c, _ := newTestClient(nil)
	secret := &corev1.Secret{Data: map[string][]byte{"password": []byte("pw")}}
	data := pushSecretDataStub{remoteKey: "00000000-0000-0000-0000-000000000099", secretKey: "password"}

	err := c.PushSecret(context.Background(), secret, data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
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

func TestClient_PushSecret_ByNameNotFound(t *testing.T) {
	c, _ := newTestClient(nil)
	secret := &corev1.Secret{Data: map[string][]byte{"password": []byte("pw")}}
	data := pushSecretDataStub{remoteKey: "nonexistent-entry", secretKey: "password"}

	err := c.PushSecret(context.Background(), secret, data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "entry must exist before pushing secrets")
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
