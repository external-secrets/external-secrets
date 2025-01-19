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

package infisical

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSecretAddress(t *testing.T) {
	path, key := getSecretAddress("/", "foo")
	assert.Equal(t, path, "/")
	assert.Equal(t, key, "foo")

	path, key = getSecretAddress("/", "foo/bar")
	assert.Equal(t, path, "/foo")
	assert.Equal(t, key, "bar")

	path, key = getSecretAddress("/", "foo/bar/baz")
	assert.Equal(t, path, "/foo/bar")
	assert.Equal(t, key, "baz")

	path, key = getSecretAddress("/foo", "bar/baz")
	assert.Equal(t, path, "/foo/bar")
	assert.Equal(t, key, "baz")
}
