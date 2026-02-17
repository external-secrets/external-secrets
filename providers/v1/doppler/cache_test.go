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

package doppler

import (
	"bytes"
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/providers/v1/doppler/client"
	"github.com/external-secrets/external-secrets/providers/v1/doppler/fake"
	"github.com/external-secrets/external-secrets/runtime/cache"
)

const testETagValue = "etag-123"

// testCacheSize is used in tests to create caches with sufficient capacity.
const testCacheSize = 100

const testAPIKeyValue = "secret-value"
const testDBPassValue = "password"

// testStore is a default store identity used in tests.
var testStore = storeIdentity{
	namespace: "test-namespace",
	name:      "test-store",
	kind:      "SecretStore",
}

func TestCacheKey(t *testing.T) {
	store := storeIdentity{namespace: "ns", name: "store", kind: "SecretStore"}

	tests := []struct {
		store      storeIdentity
		secretName string
		expected   cache.Key
	}{
		{store, "", cache.Key{Name: "store", Namespace: "ns", Kind: "SecretStore"}},
		{store, "API_KEY", cache.Key{Name: "store|API_KEY", Namespace: "ns", Kind: "SecretStore"}},
		{store, "DB_PASS", cache.Key{Name: "store|DB_PASS", Namespace: "ns", Kind: "SecretStore"}},
	}

	for _, tt := range tests {
		result := cacheKey(tt.store, tt.secretName)
		if result != tt.expected {
			t.Errorf("cacheKey(%v, %q) = %v, want %v", tt.store, tt.secretName, result, tt.expected)
		}
	}
}

func TestSecretsCacheGetSet(t *testing.T) {
	c := newSecretsCache(testCacheSize)

	entry, found := c.get(testStore, "")
	if found || entry != nil {
		t.Error("expected empty cache to return nil, false")
	}

	testEntry := &cacheEntry{
		etag:    "test-etag",
		secrets: client.Secrets{"KEY": "value"},
		body:    []byte("test body"),
	}
	c.set(testStore, "", testEntry)

	entry, found = c.get(testStore, "")
	if !found {
		t.Error("expected cache hit after set")
	}
	if entry.etag != testEntry.etag {
		t.Errorf("expected etag %q, got %q", testEntry.etag, entry.etag)
	}
	if !cmp.Equal(entry.secrets, testEntry.secrets) {
		t.Errorf("expected secrets %v, got %v", testEntry.secrets, entry.secrets)
	}

	// Different secret name should miss
	entry, found = c.get(testStore, "API_KEY")
	if found || entry != nil {
		t.Error("expected cache miss for different secret name")
	}

	// Different store should not see the entry
	otherStore := storeIdentity{namespace: "other-ns", name: "other-store", kind: "SecretStore"}
	entry, found = c.get(otherStore, "")
	if found || entry != nil {
		t.Error("expected cache miss for different store")
	}
}

func TestSecretsCacheInvalidate(t *testing.T) {
	c := newSecretsCache(testCacheSize)

	testEntry := &cacheEntry{
		etag:    "test-etag",
		secrets: client.Secrets{"KEY": "value"},
	}
	c.set(testStore, "", testEntry)
	c.set(testStore, "API_KEY", testEntry)
	c.set(testStore, "DB_PASS", testEntry)

	_, found := c.get(testStore, "")
	if !found {
		t.Error("expected cache hit before invalidate")
	}
	_, found = c.get(testStore, "API_KEY")
	if !found {
		t.Error("expected cache hit for API_KEY before invalidate")
	}

	c.invalidate(testStore)

	_, found = c.get(testStore, "")
	if found {
		t.Error("expected cache miss after invalidate")
	}
	_, found = c.get(testStore, "API_KEY")
	if found {
		t.Error("expected cache miss for API_KEY after invalidate")
	}
	_, found = c.get(testStore, "DB_PASS")
	if found {
		t.Error("expected cache miss for DB_PASS after invalidate")
	}
}

func TestSecretsCacheConcurrency(t *testing.T) {
	c := newSecretsCache(testCacheSize)
	const numGoroutines = 100
	const numIterations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()
			for j := range numIterations {
				entry := &cacheEntry{
					etag:    "etag",
					secrets: client.Secrets{"KEY": "value"},
				}
				c.set(testStore, "", entry)
				c.get(testStore, "")
				if j%10 == 0 {
					c.invalidate(testStore)
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestGetAllSecretsUsesCache(t *testing.T) {
	etagCache = newSecretsCache(testCacheSize)

	fakeClient := &fake.DopplerClient{}

	var callCount atomic.Int32
	testSecrets := client.Secrets{"API_KEY": testAPIKeyValue, "DB_PASS": testDBPassValue}
	testETag := testETagValue

	fakeClient.WithSecretsFunc(func(request client.SecretsRequest) (*client.SecretsResponse, error) {
		count := callCount.Add(1)

		if request.ETag == "" {
			return &client.SecretsResponse{
				Modified: true,
				Secrets:  testSecrets,
				ETag:     testETag,
			}, nil
		}

		if request.ETag == testETag {
			return &client.SecretsResponse{
				Modified: false,
				Secrets:  nil,
				ETag:     testETag,
			}, nil
		}

		t.Errorf("unexpected call %d with ETag %q", count, request.ETag)
		return nil, nil
	})

	c := &Client{
		doppler:   fakeClient,
		project:   "test-project",
		config:    "test-config",
		namespace: "test-namespace",
		storeName: "test-store",
		storeKind: "SecretStore",
	}

	secrets, err := c.secrets(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(secrets) != 2 {
		t.Errorf("expected 2 secrets, got %d", len(secrets))
	}
	if string(secrets["API_KEY"]) != testAPIKeyValue {
		t.Errorf("expected API_KEY=%s, got %s", testAPIKeyValue, secrets["API_KEY"])
	}

	secrets, err = c.secrets(context.Background())
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}
	if len(secrets) != 2 {
		t.Errorf("expected 2 secrets on second call, got %d", len(secrets))
	}

	if callCount.Load() != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount.Load())
	}
}

func TestGetSecretUsesCache(t *testing.T) {
	etagCache = newSecretsCache(testCacheSize)

	fakeClient := &fake.DopplerClient{}

	var callCount atomic.Int32
	apiKeyETag := "etag-api-key"
	dbPassETag := "etag-db-pass"

	fakeClient.WithSecretFunc(func(request client.SecretRequest) (*client.SecretResponse, error) {
		callCount.Add(1)

		secretName := request.Name
		var expectedETag string
		var secretValue string

		switch secretName {
		case "API_KEY":
			expectedETag = apiKeyETag
			secretValue = testAPIKeyValue
		case "DB_PASS":
			expectedETag = dbPassETag
			secretValue = testDBPassValue
		default:
			t.Errorf("unexpected secret requested: %s", secretName)
			return nil, nil
		}

		if request.ETag == expectedETag {
			return &client.SecretResponse{Modified: false, ETag: expectedETag}, nil
		}

		return &client.SecretResponse{
			Name:     secretName,
			Value:    secretValue,
			Modified: true,
			ETag:     expectedETag,
		}, nil
	})

	c := &Client{
		doppler:   fakeClient,
		project:   "test-project",
		config:    "test-config",
		namespace: "test-namespace",
		storeName: "test-store",
		storeKind: "SecretStore",
	}

	secret, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "API_KEY"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(secret) != testAPIKeyValue {
		t.Errorf("expected %s, got %s", testAPIKeyValue, secret)
	}

	secret, err = c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "API_KEY"})
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}
	if string(secret) != testAPIKeyValue {
		t.Errorf("expected %s on second call, got %s", testAPIKeyValue, secret)
	}

	secret, err = c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "DB_PASS"})
	if err != nil {
		t.Fatalf("unexpected error for DB_PASS: %v", err)
	}
	if string(secret) != testDBPassValue {
		t.Errorf("expected %s, got %s", testDBPassValue, secret)
	}

	secret, err = c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "DB_PASS"})
	if err != nil {
		t.Fatalf("unexpected error on second DB_PASS call: %v", err)
	}
	if string(secret) != testDBPassValue {
		t.Errorf("expected %s on second call, got %s", testDBPassValue, secret)
	}

	if callCount.Load() != 4 {
		t.Errorf("expected 4 API calls, got %d", callCount.Load())
	}
}

func TestCacheInvalidationOnPushSecret(t *testing.T) {
	etagCache = newSecretsCache(testCacheSize)

	fakeClient := &fake.DopplerClient{}

	var secretsCallCount atomic.Int32
	testSecrets := client.Secrets{"API_KEY": "original-value"}
	updatedSecrets := client.Secrets{"API_KEY": "updated-value"}
	testETag := testETagValue
	newETag := "etag-456"

	fakeClient.WithSecretsFunc(func(request client.SecretsRequest) (*client.SecretsResponse, error) {
		count := secretsCallCount.Add(1)

		switch count {
		case 1:
			return &client.SecretsResponse{
				Modified: true,
				Secrets:  testSecrets,
				ETag:     testETag,
			}, nil
		case 2:
			if request.ETag != "" {
				t.Errorf("expected no ETag after cache invalidation, got %q", request.ETag)
			}
			return &client.SecretsResponse{
				Modified: true,
				Secrets:  updatedSecrets,
				ETag:     newETag,
			}, nil
		default:
			t.Errorf("unexpected call %d", count)
			return nil, nil
		}
	})

	fakeClient.WithUpdateValue(client.UpdateSecretsRequest{
		Secrets: client.Secrets{validRemoteKey: validSecretValue},
		Project: "test-project",
		Config:  "test-config",
	}, nil)

	c := &Client{
		doppler:   fakeClient,
		project:   "test-project",
		config:    "test-config",
		namespace: "test-namespace",
		storeName: "test-store",
		storeKind: "SecretStore",
	}

	_, err := c.secrets(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	storeID := c.storeIdentity()
	entry, found := etagCache.get(storeID, "")
	if !found {
		t.Error("expected cache to be populated after first call")
	}
	if entry.etag != testETag {
		t.Errorf("expected ETag %q, got %q", testETag, entry.etag)
	}

	secret := &corev1.Secret{
		Data: map[string][]byte{
			validSecretName: []byte(validSecretValue),
		},
	}
	secretData := esv1alpha1.PushSecretData{
		Match: esv1alpha1.PushSecretMatch{
			SecretKey: validSecretName,
			RemoteRef: esv1alpha1.PushSecretRemoteRef{
				RemoteKey: validRemoteKey,
			},
		},
	}
	err = c.PushSecret(context.Background(), secret, secretData)
	if err != nil {
		t.Fatalf("unexpected error pushing secret: %v", err)
	}

	_, found = etagCache.get(storeID, "")
	if found {
		t.Error("expected cache to be invalidated after push")
	}

	_, err = c.secrets(context.Background())
	if err != nil {
		t.Fatalf("unexpected error after push: %v", err)
	}

	if secretsCallCount.Load() != 2 {
		t.Errorf("expected 2 secrets API calls, got %d", secretsCallCount.Load())
	}
}

func TestCacheInvalidationOnDeleteSecret(t *testing.T) {
	etagCache = newSecretsCache(testCacheSize)

	fakeClient := &fake.DopplerClient{}

	testSecrets := client.Secrets{"API_KEY": "value"}
	testETag := testETagValue

	fakeClient.WithSecretsFunc(func(_ client.SecretsRequest) (*client.SecretsResponse, error) {
		return &client.SecretsResponse{
			Modified: true,
			Secrets:  testSecrets,
			ETag:     testETag,
		}, nil
	})

	fakeClient.WithUpdateValue(client.UpdateSecretsRequest{
		ChangeRequests: []client.Change{
			{
				Name:         validRemoteKey,
				OriginalName: validRemoteKey,
				ShouldDelete: true,
			},
		},
		Project: "test-project",
		Config:  "test-config",
	}, nil)

	c := &Client{
		doppler:   fakeClient,
		project:   "test-project",
		config:    "test-config",
		namespace: "test-namespace",
		storeName: "test-store",
		storeKind: "SecretStore",
	}

	_, err := c.secrets(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	storeID := c.storeIdentity()
	_, found := etagCache.get(storeID, "")
	if !found {
		t.Error("expected cache to be populated")
	}

	remoteRef := &esv1alpha1.PushSecretRemoteRef{RemoteKey: validRemoteKey}
	err = c.DeleteSecret(context.Background(), remoteRef)
	if err != nil {
		t.Fatalf("unexpected error deleting secret: %v", err)
	}

	_, found = etagCache.get(storeID, "")
	if found {
		t.Error("expected cache to be invalidated after delete")
	}
}

func TestCacheWithFormat(t *testing.T) {
	etagCache = newSecretsCache(testCacheSize)

	fakeClient := &fake.DopplerClient{}

	var callCount atomic.Int32
	testBody := []byte("KEY=value\nDB_PASS=password")
	testETag := "etag-format-123"

	fakeClient.WithSecretsFunc(func(request client.SecretsRequest) (*client.SecretsResponse, error) {
		count := callCount.Add(1)

		if request.ETag == "" {
			return &client.SecretsResponse{
				Modified: true,
				Body:     testBody,
				ETag:     testETag,
			}, nil
		}

		if request.ETag == testETag {
			return &client.SecretsResponse{
				Modified: false,
				Body:     nil,
				ETag:     testETag,
			}, nil
		}

		t.Errorf("unexpected call %d with ETag %q", count, request.ETag)
		return nil, nil
	})

	c := &Client{
		doppler:   fakeClient,
		project:   "test-project",
		config:    "test-config",
		format:    "env",
		namespace: "test-namespace",
		storeName: "test-store",
		storeKind: "SecretStore",
	}

	secrets, err := c.secrets(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(secrets["DOPPLER_SECRETS_FILE"], testBody) {
		t.Errorf("expected body %q, got %q", testBody, secrets["DOPPLER_SECRETS_FILE"])
	}

	secrets, err = c.secrets(context.Background())
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}
	if !bytes.Equal(secrets["DOPPLER_SECRETS_FILE"], testBody) {
		t.Errorf("expected cached body %q, got %q", testBody, secrets["DOPPLER_SECRETS_FILE"])
	}

	if callCount.Load() != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount.Load())
	}
}

func TestSecretsCacheDisabled(t *testing.T) {
	// When cache size is 0, caching should be disabled
	c := newSecretsCache(0)
	if c != nil {
		t.Error("expected nil cache when size is 0")
	}

	// Operations on nil cache should be no-ops (not panic)
	var nilCache *secretsCache
	entry, found := nilCache.get(testStore, "")
	if found || entry != nil {
		t.Error("expected nil cache get to return nil, false")
	}

	// set should be a no-op
	nilCache.set(testStore, "", &cacheEntry{etag: "test"})

	// invalidate should be a no-op
	nilCache.invalidate(testStore)
}

func TestDisabledCacheDoesNotCacheSecrets(t *testing.T) {
	// Test that when cache is disabled, secrets are fetched on every call
	etagCache = nil // Disabled cache

	fakeClient := &fake.DopplerClient{}

	var callCount atomic.Int32
	testSecrets := client.Secrets{"API_KEY": testAPIKeyValue}

	fakeClient.WithSecretsFunc(func(_ client.SecretsRequest) (*client.SecretsResponse, error) {
		callCount.Add(1)
		return &client.SecretsResponse{
			Modified: true,
			Secrets:  testSecrets,
			ETag:     "etag-123",
		}, nil
	})

	c := &Client{
		doppler:   fakeClient,
		project:   "test-project",
		config:    "test-config",
		namespace: "test-namespace",
		storeName: "test-store",
		storeKind: "SecretStore",
	}

	// First call
	_, err := c.secrets(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Second call - should still fetch because cache is disabled
	_, err = c.secrets(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Third call
	_, err = c.secrets(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All three calls should have been made to the API
	if callCount.Load() != 3 {
		t.Errorf("expected 3 API calls with disabled cache, got %d", callCount.Load())
	}
}

func TestCacheIsolationBetweenStores(t *testing.T) {
	// Test that different stores don't share cache entries even with same project/config
	c := newSecretsCache(testCacheSize)

	storeA := storeIdentity{namespace: "ns-a", name: "store-a", kind: "SecretStore"}
	storeB := storeIdentity{namespace: "ns-b", name: "store-b", kind: "SecretStore"}

	entryA := &cacheEntry{etag: "etag-a", secrets: client.Secrets{"KEY": "value-a"}}
	entryB := &cacheEntry{etag: "etag-b", secrets: client.Secrets{"KEY": "value-b"}}

	// Both stores use same project/config but should have separate cache entries
	c.set(storeA, "", entryA)
	c.set(storeB, "", entryB)

	// Each store should get its own entry
	gotA, foundA := c.get(storeA, "")
	gotB, foundB := c.get(storeB, "")

	if !foundA {
		t.Error("expected cache hit for store A")
	}
	if !foundB {
		t.Error("expected cache hit for store B")
	}

	if gotA.etag != "etag-a" {
		t.Errorf("store A got wrong etag: %q, want %q", gotA.etag, "etag-a")
	}
	if gotB.etag != "etag-b" {
		t.Errorf("store B got wrong etag: %q, want %q", gotB.etag, "etag-b")
	}

	// Invalidating store A should not affect store B
	c.invalidate(storeA)

	_, foundA = c.get(storeA, "")
	_, foundB = c.get(storeB, "")

	if foundA {
		t.Error("expected cache miss for store A after invalidation")
	}
	if !foundB {
		t.Error("expected cache hit for store B after store A invalidation")
	}
}
