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

package infisical

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSecretAddress(t *testing.T) {
	t.Run("when the key is not addressing a path and uses the default path", func(t *testing.T) {
		path, key := getSecretAddress("/", "foo")
		assert.Equal(t, "/", path)
		assert.Equal(t, "foo", key)

		path, key = getSecretAddress("/foo", "bar")
		assert.Equal(t, "/foo", path)
		assert.Equal(t, "bar", key)
	})

	t.Run("when the key is addressing a path", func(t *testing.T) {
		path, key := getSecretAddress("/", "/foo/bar")
		assert.Equal(t, path, "/foo")
		assert.Equal(t, key, "bar")
	})

	t.Run("when the key is addressing a path and ignores the default path", func(t *testing.T) {
		path, key := getSecretAddress("/foo", "/bar/baz")
		assert.Equal(t, "/bar", path)
		assert.Equal(t, "baz", key)
	})

	t.Run("works with a nested directory", func(t *testing.T) {
		path, key := getSecretAddress("/", "/foo/bar/baz")
		assert.Equal(t, "/foo/bar", path)
		assert.Equal(t, "baz", key, "baz")
	})

	t.Run("relative key joins onto the default path", func(t *testing.T) {
		path, key := getSecretAddress("/secrets/mysql-core", "azure/admin-users")
		assert.Equal(t, "/secrets/mysql-core/azure", path)
		assert.Equal(t, "admin-users", key)
	})

	t.Run("relative key with default root path", func(t *testing.T) {
		path, key := getSecretAddress("/", "azure/admin-users")
		assert.Equal(t, "/azure", path)
		assert.Equal(t, "admin-users", key)
	})

	t.Run("relative key with nested folders", func(t *testing.T) {
		path, key := getSecretAddress("/scope", "a/b/c/name")
		assert.Equal(t, "/scope/a/b/c", path)
		assert.Equal(t, "name", key)
	})
}
