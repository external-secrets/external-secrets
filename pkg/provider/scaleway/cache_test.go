package scaleway

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCacheMissReturnsFalse(t *testing.T) {

	cache := newCache()

	_, ok := cache.Get("26f72b22-bcae-4131-a26e-b98abb3fa3dd", 1)

	assert.False(t, ok)
}

func TestCachePutThenGet(t *testing.T) {

	cache := newCache()
	secretId := "cfd5dda5-dedb-40eb-b9c4-b9cf8e254727"
	revision := uint32(1)
	expectedValue := []byte("some value")

	cache.Put(secretId, revision, expectedValue)

	value, ok := cache.Get(secretId, revision)
	assert.True(t, ok)
	assert.Equal(t, expectedValue, value)
}

func TestCacheLeastRecentlyUsedIsRemovedFirst(t *testing.T) {

	cache := newCache()
	secretId := "0c82ecf4-d3f7-4960-8301-0def5230eee2"
	maxEntryCount := 500

	for i := 0; i < maxEntryCount; i++ {
		cache.Put(secretId, uint32(i+1), []byte{})
	}

	for i := 0; i < maxEntryCount; i++ {
		cache.Get(secretId, uint32(i+1))
	}

	cache.Put(secretId, uint32(maxEntryCount+2), []byte{})

	_, ok := cache.Get(secretId, 1)
	assert.False(t, ok)

	_, ok = cache.Get(secretId, uint32(maxEntryCount+2))
	assert.True(t, ok)
}
