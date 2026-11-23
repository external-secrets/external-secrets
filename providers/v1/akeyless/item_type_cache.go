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

package akeyless

import (
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

// Item type almost never changes; ESO only needs it to pick get-secret vs
// get-dynamic / rotated / certificate. Keep this shorter than a typical
// ExternalSecret refreshInterval so a recreate-as-other-type heals quickly.
const itemTypeCacheTTL = 10 * time.Minute

// Cap growth from dataFrom.find over changing names on a long-lived client.
const itemTypeCacheMaxEntries = 1024

type itemTypeEntry struct {
	itemType string
	expiry   time.Time
}

// itemTypeCache checks expiry on read instead of using expirable.LRU, whose
// cleanup goroutine never exits and would leak with every discarded client.
type itemTypeCache struct {
	entries *lru.Cache[string, itemTypeEntry]
	now     func() time.Time
	ttl     time.Duration
}

func newItemTypeCache() *itemTypeCache {
	return newItemTypeCacheWithSize(itemTypeCacheMaxEntries)
}

func newItemTypeCacheWithSize(size int) *itemTypeCache {
	entries, err := lru.New[string, itemTypeEntry](size)
	if err != nil {
		panic(err)
	}
	return &itemTypeCache{
		entries: entries,
		now:     time.Now,
		ttl:     itemTypeCacheTTL,
	}
}

func (c *itemTypeCache) get(name string) (string, bool) {
	if c == nil {
		return "", false
	}
	ent, ok := c.entries.Get(name)
	if !ok {
		return "", false
	}
	if !c.now().Before(ent.expiry) {
		c.entries.Remove(name)
		return "", false
	}
	return ent.itemType, true
}

func (c *itemTypeCache) set(name, itemType string) {
	if c == nil || name == "" || itemType == "" {
		return
	}
	c.entries.Add(name, itemTypeEntry{itemType: itemType, expiry: c.now().Add(c.ttl)})
}

func (c *itemTypeCache) invalidate(name string) {
	if c == nil {
		return
	}
	c.entries.Remove(name)
}
