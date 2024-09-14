//Copyright External Secrets Inc. All Rights Reserved

package scaleway

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCacheMissReturnsFalse(t *testing.T) {
	cache := newCache()

	_, ok := cache.Get("26f72b22-bcae-4131-a26e-b98abb3fa3dd", 1)

	assert.False(t, ok)
}

func TestCachePutThenGet(t *testing.T) {
	cache := newCache()
	secretID := "cfd5dda5-dedb-40eb-b9c4-b9cf8e254727"
	revision := uint32(1)
	expectedValue := []byte("some value")

	cache.Put(secretID, revision, expectedValue)

	value, ok := cache.Get(secretID, revision)
	assert.True(t, ok)
	assert.Equal(t, expectedValue, value)
}

func TestCacheLeastRecentlyUsedIsRemovedFirst(t *testing.T) {
	cache := newCache()
	secretID := "0c82ecf4-d3f7-4960-8301-0def5230eee2"
	maxEntryCount := 500

	for i := 0; i < maxEntryCount; i++ {
		cache.Put(secretID, uint32(i+1), []byte{})
	}

	for i := 0; i < maxEntryCount; i++ {
		cache.Get(secretID, uint32(i+1))
	}

	cache.Put(secretID, uint32(maxEntryCount+2), []byte{})

	_, ok := cache.Get(secretID, 1)
	assert.False(t, ok)

	_, ok = cache.Get(secretID, uint32(maxEntryCount+2))
	assert.True(t, ok)
}
