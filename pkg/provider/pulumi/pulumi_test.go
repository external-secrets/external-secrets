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
package pulumi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	esc2 "github.com/pulumi/esc"
	esc "github.com/pulumi/esc/cmd/esc/cli/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

func newTestClient(t *testing.T, _, _ string, handler func(w http.ResponseWriter, r *http.Request)) *client {
	const userAgent = "test-user-agent"
	const token = "test-token"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "token "+token, r.Header.Get("Authorization"))
		handler(w, r)
	}))
	t.Cleanup(server.Close)

	return &client{
		escClient:    esc.New(userAgent, server.URL, token, true),
		organization: "foo",
		environment:  "bar",
	}
}

func TestGetSecret(t *testing.T) {
	ctx := context.Background()
	expected := esc2.NewValue("world")

	client := newTestClient(t, http.MethodGet, "/api/preview/environments/foo/bar/open/session", func(w http.ResponseWriter, r *http.Request) {
		err := json.NewEncoder(w).Encode(expected)
		require.NoError(t, err)
	})

	testCases := map[string]struct {
		ref  esv1beta1.ExternalSecretDataRemoteRef
		want []byte
		err  error
	}{
		"querying for the key returns the value": {
			ref: esv1beta1.ExternalSecretDataRemoteRef{
				Key: "b",
			},
			want: []byte(`world`),
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, err := client.GetSecret(ctx, tc.ref)
			if tc.err == nil {
				assert.NoError(t, err)
				assert.Equal(t, tc.want, got)
			} else {
				assert.Nil(t, got)
				assert.ErrorIs(t, err, tc.err)
				assert.Equal(t, tc.err, err)
			}
		})
	}
}
