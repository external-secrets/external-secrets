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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/providers/v1/infisical/api"
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

// TestGetAllSecretsPreservesCrossFolderDuplicates pins the fix for issue #6230.
//
// When a project has multiple folders that share the same secret name(s) and
// the ClusterSecretStore is configured with `recursive: true`, the Infisical
// Go SDK's `EnsureUniqueSecretsByKey` would (by default) dedupe the response
// by SecretKey alone before returning it to the provider, collapsing all
// same-named entries to a single one — last-write-wins. The provider's
// downstream path filter (HasPrefix on SecretPath) would then either keep
// the wrong folder's entry or drop the key entirely, depending on the
// response order. The fix passes SkipUniqueValidation: true so the SDK
// dedupes by (path, key) composite and preserves all entries.
func TestGetAllSecretsPreservesCrossFolderDuplicates(t *testing.T) {
	// Same key name "FOOBAR" lives in two folders with distinct values.
	// /app/FOOBAR  = "from-app"   ← we want this one
	// /crons/FOOBAR = "from-crons"
	sdkClient, closeFunc := api.NewMockClient(200, api.GetSecretsV3Response{
		Secrets: []api.SecretsV3{
			{SecretKey: "FOOBAR", SecretValue: "from-app", SecretPath: "/app"},
			{SecretKey: "FOOBAR", SecretValue: "from-crons", SecretPath: "/crons"},
			{SecretKey: "ONLY_IN_APP", SecretValue: "app-only", SecretPath: "/app"},
		},
	})
	defer closeFunc()

	p := &Provider{
		sdkClient: sdkClient,
		apiScope: &ClientScope{
			ProjectSlug:     "test-project",
			EnvironmentSlug: "test",
			SecretPath:      "/",
			Recursive:       true,
		},
	}

	appPath := "/app"
	result, err := p.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{
		Path: &appPath,
	})
	assert.NoError(t, err)

	// Both /app keys must be present; the /crons value must NOT shadow /app's FOOBAR.
	assert.Equal(t, []byte("from-app"), result["FOOBAR"],
		"FOOBAR should resolve to the /app folder's value, not /crons. "+
			"If you see 'from-crons' here, the SDK dedupe-by-key has eaten the "+
			"/app entry — see issue #6230.")
	assert.Equal(t, []byte("app-only"), result["ONLY_IN_APP"])
	assert.Len(t, result, 2, "result should contain only /app keys")

	// Sanity-check the other direction with the same fixture and a /crons filter.
	cronsPath := "/crons"
	result, err = p.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{
		Path: &cronsPath,
	})
	assert.NoError(t, err)
	assert.Equal(t, []byte("from-crons"), result["FOOBAR"])
	assert.NotContains(t, result, "ONLY_IN_APP")
	assert.Len(t, result, 1)
}
