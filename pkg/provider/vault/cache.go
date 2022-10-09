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

package vault

import (
	"context"
	"errors"
	"fmt"
	"sync"

	lru "github.com/hashicorp/golang-lru"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

type clientCache struct {
	cache       *lru.Cache
	Size        int
	initialized bool
	mu          sync.Mutex
}

type clientCacheKey struct {
	Name      string
	Namespace string
	Kind      string
}

type clientCacheValue struct {
	ResourceVersion string
	Client          Client
}

func (c *clientCache) initialize() error {
	if !c.initialized {
		var err error
		c.cache, err = lru.New(c.Size)
		if err != nil {
			return fmt.Errorf(errVaultCacheCreate, err)
		}
		c.initialized = true
	}
	return nil
}

func (c *clientCache) get(ctx context.Context, store esv1beta1.GenericStore, key clientCacheKey) (Client, bool, error) {
	value, ok := c.cache.Get(key)
	if ok {
		cachedClient := value.(clientCacheValue)
		if cachedClient.ResourceVersion == store.GetObjectMeta().ResourceVersion {
			return cachedClient.Client, true, nil
		}
		// revoke token and clear old item from cache if resource has been updated
		err := revokeTokenIfValid(ctx, cachedClient.Client)
		if err != nil {
			return nil, false, err
		}
		c.cache.Remove(key)
	}
	return nil, false, nil
}

func (c *clientCache) add(ctx context.Context, store esv1beta1.GenericStore, key clientCacheKey, client Client) error {
	// don't let the LRU cache evict items
	// remove the oldest item manually when needed so we can do some cleanup
	for c.cache.Len() >= c.Size {
		_, value, ok := c.cache.RemoveOldest()
		if !ok {
			return errors.New(errVaultCacheRemove)
		}
		cachedClient := value.(clientCacheValue)
		err := revokeTokenIfValid(ctx, cachedClient.Client)
		if err != nil {
			return fmt.Errorf(errVaultRevokeToken, err)
		}
	}
	evicted := c.cache.Add(key, clientCacheValue{ResourceVersion: store.GetObjectMeta().ResourceVersion, Client: client})
	if evicted {
		return errors.New(errVaultCacheEviction)
	}
	return nil
}

func (c *clientCache) contains(key clientCacheKey) bool {
	return c.cache.Contains(key)
}

func (c *clientCache) lock() {
	c.mu.Lock()
}

func (c *clientCache) unlock() {
	c.mu.Unlock()
}
