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

package cache

import (
	"fmt"

	lru "github.com/hashicorp/golang-lru"
)

// Cache is a generic lru cache that allows you to
// lookup values using a key and a version.
// By design, this cache allows access to only a single version of a given key.
// A version mismatch is considered a cache miss and the key gets evicted if it exists.
// When a key is evicted a optional cleanup function is called.
type Cache[T any] struct {
	lru         *lru.Cache
	size        int
	cleanupFunc cleanupFunc[T]
}

// Key is the cache lookup key.
type Key struct {
	Name      string
	Namespace string
	Kind      string
}

type value[T any] struct {
	Version string
	Client  T
}

type cleanupFunc[T any] func(client T)

// New constructs a new lru cache with the desired size and cleanup func.
func New[T any](size int, cleanup cleanupFunc[T]) (*Cache[T], error) {
	lruCache, err := lru.NewWithEvict(size, func(_, val any) {
		if cleanup == nil {
			return
		}
		cleanup(val.(value[T]).Client)
	})
	if err != nil {
		return nil, fmt.Errorf("unable to create lru: %w", err)
	}
	return &Cache[T]{
		lru:         lruCache,
		size:        size,
		cleanupFunc: cleanup,
	}, nil
}

// Must creates a new lru cache with the desired size and cleanup func
// This function panics if a error occurrs.
func Must[T any](size int, cleanup cleanupFunc[T]) *Cache[T] {
	c, err := New(size, cleanup)
	if err != nil {
		panic(err)
	}
	return c
}

// Get retrieves the desired value using the key and
// compares the version. If there is a mismatch
// it is considered a cache miss and the existing key is purged.
func (c *Cache[T]) Get(version string, key Key) (T, bool) {
	val, ok := c.lru.Get(key)
	if ok {
		cachedClient := val.(value[T])
		if cachedClient.Version == version {
			return cachedClient.Client, true
		}
		c.lru.Remove(key)
	}
	return value[T]{}.Client, false
}

// Add adds a new value for the given key/version.
func (c *Cache[T]) Add(version string, key Key, client T) {
	c.lru.Add(key, value[T]{Version: version, Client: client})
}

// Contains returns true if a value with the given key exists.
func (c *Cache[T]) Contains(key Key) bool {
	return c.lru.Contains(key)
}
