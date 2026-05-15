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

package truefoundry

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// TestSmoke_RealAPI exercises Validate, GetSecret, and GetSecretMap against a
// real TrueFoundry control plane. It is skipped by default. Run it manually
// from this submodule directory (it has its own go.mod):
//
//	cd providers/v1/truefoundry
//	TFY_BASE_URL=https://app.truefoundry.com \
//	TFY_TENANT=my-tenant \
//	TFY_API_KEY=tfy-pat-xxxx \
//	TFY_GROUP=my-secret-group \
//	TFY_KEY=SOME_KEY \                 # optional; if set, exercises the group/key path
//	go test -v -run TestSmoke_RealAPI ./...
func TestSmoke_RealAPI(t *testing.T) {
	baseURL := os.Getenv("TFY_BASE_URL")
	tenant := os.Getenv("TFY_TENANT")
	apiKey := os.Getenv("TFY_API_KEY")
	group := os.Getenv("TFY_GROUP")
	if baseURL == "" || tenant == "" || apiKey == "" || group == "" {
		t.Skip("set TFY_BASE_URL, TFY_TENANT, TFY_API_KEY, TFY_GROUP to run this smoke test")
	}
	key := os.Getenv("TFY_KEY") // optional

	c := newClient(baseURL, tenant, apiKey, &http.Client{Timeout: 15 * time.Second})

	t.Run("Validate", func(t *testing.T) {
		res, err := c.Validate()
		t.Logf("Validate → %v err=%v", res, err)
		if err != nil {
			t.Fatalf("Validate failed: %v", err)
		}
	})

	t.Run("GetSecretMap(whole group)", func(t *testing.T) {
		m, err := c.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: group})
		if err != nil {
			t.Fatalf("GetSecretMap %q: %v", group, err)
		}
		t.Logf("GetSecretMap %s → %d keys", group, len(m))
		for k, v := range m {
			t.Logf("  %s = %q", k, string(v))
		}
	})

	if key == "" {
		return
	}
	t.Run("GetSecret(group/key)", func(t *testing.T) {
		v, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: group + "/" + key})
		if err != nil {
			t.Fatalf("GetSecret %s/%s: %v", group, key, err)
		}
		t.Logf("GetSecret %s/%s → %q", group, key, string(v))
	})
}
