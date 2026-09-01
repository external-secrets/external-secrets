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

package cache

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

type client struct {
	id int
}

var cacheKey = Key{Name: "foo"}

func TestCacheAdd(t *testing.T) {
	c, err := New[client](1, nil)
	if err != nil {
		t.Fail()
	}

	cl := client{}
	c.Add("", cacheKey, cl)
	cachedVal, _ := c.Get("", cacheKey)

	assert.EqualValues(t, cl, cachedVal)
}

func TestCacheContains(t *testing.T) {
	c, err := New[client](1, nil)
	if err != nil {
		t.Fail()
	}

	cl := client{}
	c.Add("", cacheKey, cl)
	exists := c.Contains(cacheKey)
	notExists := c.Contains(Key{Name: "does not exist"})

	assert.True(t, exists)
	assert.False(t, notExists)
	assert.Nil(t, err)
}

func TestCacheContainsOrAdd(t *testing.T) {
	c := Must[client](1, nil)
	first := client{id: 1}
	second := client{id: 2}

	assert.False(t, c.ContainsOrAdd("v1", cacheKey, first))
	assert.True(t, c.ContainsOrAdd("v1", cacheKey, second))

	cached, ok := c.Get("v1", cacheKey)
	assert.True(t, ok)
	assert.Equal(t, first, cached)
}

func TestCacheContainsOrAddVersionMismatch(t *testing.T) {
	c := Must[client](1, nil)
	first := client{id: 1}

	c.Add("v1", cacheKey, first)
	assert.True(t, c.ContainsOrAdd("v2", cacheKey, client{id: 2}))

	cached, ok := c.Get("v1", cacheKey)
	assert.True(t, ok)
	assert.Equal(t, first, cached)
}

func TestCacheContainsOrAddDoesNotCleanUpRejectedValue(t *testing.T) {
	var cleaned []client
	c := Must(1, func(value client) {
		cleaned = append(cleaned, value)
	})
	first := client{id: 1}

	assert.False(t, c.ContainsOrAdd("v1", cacheKey, first))
	assert.True(t, c.ContainsOrAdd("v1", cacheKey, client{id: 2}))
	assert.Empty(t, cleaned)

	assert.False(t, c.ContainsOrAdd("v1", Key{Name: "bar"}, client{id: 3}))
	assert.Equal(t, []client{first}, cleaned)
}

func TestCacheContainsOrAddConcurrent(t *testing.T) {
	const workers = 64
	c := Must[client](workers, nil)
	start := make(chan struct{})
	var inserted atomic.Int32
	var wg sync.WaitGroup

	for i := range workers {
		wg.Go(func() {
			<-start
			if !c.ContainsOrAdd("v1", cacheKey, client{id: i}) {
				inserted.Add(1)
			}
		})
	}

	close(start)
	wg.Wait()

	assert.EqualValues(t, 1, inserted.Load())
	_, ok := c.Get("v1", cacheKey)
	assert.True(t, ok)
}

func TestCacheGet(t *testing.T) {
	c, err := New[*client](1, nil)
	if err != nil {
		t.Fail()
	}
	cachedVal, ok := c.Get("", cacheKey)

	assert.Nil(t, cachedVal)
	assert.False(t, ok)
}

func TestCacheGetInvalidVersion(t *testing.T) {
	var cleanupCalled bool
	c, err := New(1, func(*client) {
		cleanupCalled = true
	})
	if err != nil {
		t.Fail()
	}
	cl := &client{}
	c.Add("", cacheKey, cl)
	cachedVal, ok := c.Get("invalid", cacheKey)

	assert.Nil(t, cachedVal)
	assert.False(t, ok)
	assert.True(t, cleanupCalled)
}

func TestCacheEvict(t *testing.T) {
	var cleanupCalled bool
	c, err := New(1, func(client) {
		cleanupCalled = true
	})
	if err != nil {
		t.Fail()
	}

	// add first version
	c.Add("", Key{Name: "foo"}, client{})
	assert.False(t, cleanupCalled)

	// adding a second version should evict old one
	c.Add("", Key{Name: "bar"}, client{})
	assert.True(t, cleanupCalled)
}
