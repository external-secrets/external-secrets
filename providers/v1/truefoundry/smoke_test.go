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

// TestSmoke_RealAPI exercises GetSecret against a real TrueFoundry
// control-plane endpoint. It is skipped by default. Run it manually from
// this submodule directory (it has its own go.mod):
//
//	cd providers/v1/truefoundry
//	TFY_BASE_URL=https://<cluster>.tfy-usea1-ctl.example.com \
//	TFY_CLUSTER_TOKEN=<bearer-token> \
//	TFY_FQN=truefoundry:test-eso:test \
//	go test -v -run TestSmoke_RealAPI ./...
func TestSmoke_RealAPI(t *testing.T) {
	baseURL := os.Getenv("TFY_BASE_URL")
	token := os.Getenv("TFY_CLUSTER_TOKEN")
	fqn := os.Getenv("TFY_FQN")
	if baseURL == "" || token == "" || fqn == "" {
		t.Skip("set TFY_BASE_URL, TFY_CLUSTER_TOKEN, TFY_FQN to run this smoke test")
	}

	c := newClient(baseURL, token, &http.Client{Timeout: 15 * time.Second})

	t.Run("GetSecret", func(t *testing.T) {
		val, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: fqn})
		if err != nil {
			t.Fatalf("GetSecret %q: %v", fqn, err)
		}
		t.Logf("GetSecret %s → %q (%d bytes)", fqn, string(val), len(val))
	})
}
