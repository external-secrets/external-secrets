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

package vault

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResolveVaultCacheConfig covers the flag-reconciliation logic that was
// previously inlined in init() and untested. The key regression (issue #6733) is
// that the deprecated experimental cache flags must only take effect when the user
// explicitly sets them, so the supported --vault-token-cache-size is honored and no
// spurious deprecation warning fires on a vanilla install.
func TestResolveVaultCacheConfig(t *testing.T) {
	cases := []struct {
		name       string
		args       []string
		wantEnable bool
		wantSize   int
	}{
		{"nothing set", nil, false, defaultCacheSize},
		{"supported only", []string{"--enable-vault-token-cache", "--vault-token-cache-size=1000"}, true, 1000},
		{"deprecated only", []string{"--experimental-enable-vault-token-cache", "--experimental-vault-token-cache-size=5000"}, true, 5000},
		{"both set: deprecated overrides", []string{"--vault-token-cache-size=1000", "--experimental-vault-token-cache-size=5000"}, false, 5000},
		{"explicit disable of deprecated bool overrides supported", []string{"--enable-vault-token-cache", "--experimental-enable-vault-token-cache=false"}, false, defaultCacheSize},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
			var (
				enableCache bool
				expEnable   bool
				size        int
				expSize     int
			)
			fs.BoolVar(&enableCache, "enable-vault-token-cache", false, "")
			fs.IntVar(&size, "vault-token-cache-size", defaultCacheSize, "")
			fs.BoolVar(&expEnable, "experimental-enable-vault-token-cache", false, "")
			fs.IntVar(&expSize, "experimental-vault-token-cache-size", defaultCacheSize, "")
			require.NoError(t, fs.Parse(tc.args))

			gotEnable, gotSize := resolveVaultCacheConfig(fs, enableCache, expEnable, size, expSize)
			assert.Equal(t, tc.wantEnable, gotEnable)
			assert.Equal(t, tc.wantSize, gotSize)
		})
	}
}
