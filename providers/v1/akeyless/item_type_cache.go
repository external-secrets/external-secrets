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
	"sync"
	"time"
)

// Item type almost never changes; ESO only needs it to pick get-secret vs
// get-dynamic / rotated / certificate. Keep this shorter than a typical
// ExternalSecret refreshInterval so a recreate-as-other-type heals quickly.
const itemTypeCacheTTL = 10 * time.Minute

type itemTypeEntry struct {
	itemType string
	expiry   time.Time
}

type itemTypeCache struct {
	mu      sync.Mutex
	entries map[string]itemTypeEntry
	now     func() time.Time
	ttl     time.Duration
}

func newItemTypeCache() *itemTypeCache {
	return &itemTypeCache{
		entries: make(map[string]itemTypeEntry),
		now:     time.Now,
		ttl:     itemTypeCacheTTL,
	}
}

func (c *itemTypeCache) get(name string) (string, bool) {
	if c == nil {
		return "", false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	ent, ok := c.entries[name]
	if !ok || !c.now().Before(ent.expiry) {
		if ok {
			delete(c.entries, name)
		}
		return "", false
	}
	return ent.itemType, true
}

func (c *itemTypeCache) set(name, itemType string) {
	if c == nil || name == "" || itemType == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[name] = itemTypeEntry{itemType: itemType, expiry: c.now().Add(c.ttl)}
}

func (c *itemTypeCache) invalidate(name string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, name)
}
