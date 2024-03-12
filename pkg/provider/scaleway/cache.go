/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
