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
	"testing"

	"github.com/stretchr/testify/assert"
)

type client struct{}

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
	c, err := New(1, func(client *client) {
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
	c, err := New(1, func(client client) {
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
