//Copyright External Secrets Inc. All Rights Reserved

package scaleway

import (
	"container/list"
	"fmt"
	"sync"
)

// cache is for caching values of the secrets. Secret versions are immutable, thus there is no need
// for time-based expiration.
type cache interface {
	Get(secretID string, revision uint32) ([]byte, bool)
	Put(secretID string, revision uint32, value []byte)
}

type cacheEntry struct {
	value []byte
	elem  *list.Element
}

type cacheImpl struct {
	mutex                sync.Mutex
	entries              map[string]cacheEntry
	entryKeysByLastUsage list.List
	maxEntryCount        int
}

func newCache() cache {
	return &cacheImpl{
		entries:       map[string]cacheEntry{},
		maxEntryCount: 500,
	}
}

func (c *cacheImpl) Get(secretID string, revision uint32) ([]byte, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	key := c.key(secretID, revision)

	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}

	c.entryKeysByLastUsage.MoveToFront(entry.elem)

	return entry.value, true
}

func (c *cacheImpl) Put(secretID string, revision uint32, value []byte) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	key := c.key(secretID, revision)

	_, alreadyPresent := c.entries[key]
	if alreadyPresent {
		return
	}

	if len(c.entries) == c.maxEntryCount {
		c.evictLeastRecentlyUsed()
	}

	entry := c.entryKeysByLastUsage.PushFront(key)

	c.entries[key] = cacheEntry{
		value: value,
		elem:  entry,
	}
}

func (c *cacheImpl) evictLeastRecentlyUsed() {
	elem := c.entryKeysByLastUsage.Back()

	delete(c.entries, elem.Value.(string))

	c.entryKeysByLastUsage.Remove(elem)
}

func (c *cacheImpl) key(secretID string, revision uint32) string {
	return fmt.Sprintf("%s/%d", secretID, revision)
}
