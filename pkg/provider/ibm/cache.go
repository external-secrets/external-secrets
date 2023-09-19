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
package ibm

import (
	"time"

	"github.com/golang/groupcache/lru"
)

type (
	cacheIntf interface {
		GetData(key string) (bool, []byte)
		PutData(key string, value []byte)
		DeleteData(key string)
	}

	lruCache struct {
		lru *lru.Cache
		ttl time.Duration
	}

	cacheObject struct {
		timeExpires time.Time
		value       []byte
	}
)

func NewCache(maxEntries int, ttl time.Duration) cacheIntf {
	lruCache := &lruCache{
		lru: lru.New(maxEntries),
		ttl: ttl,
	}

	return lruCache
}

func (c *lruCache) GetData(key string) (bool, []byte) {
	v, ok := c.lru.Get(key)
	if !ok {
		return false, nil
	}
	returnedObj := v.(cacheObject)
	if time.Now().After(returnedObj.timeExpires) && c.ttl > 0 {
		c.DeleteData(key)
		return false, nil
	}
	return true, returnedObj.value
}

func (c *lruCache) PutData(key string, value []byte) {
	obj := cacheObject{
		timeExpires: time.Now().Add(c.ttl),
		value:       value,
	}
	c.lru.Add(key, obj)
}

func (c *lruCache) DeleteData(key string) {
	c.lru.Remove(key)
}
