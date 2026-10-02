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

package dvls

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/Devolutions/go-dvls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newPaginatedDVLSServer(t *testing.T, vaults []dvls.Vault) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/login", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"TokenId":"test-token"}`))
	})
	mux.HandleFunc("/api/is-logged", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("true"))
	})
	mux.HandleFunc("/api/v1/vault", func(w http.ResponseWriter, r *http.Request) {
		pageSize := 25
		if v, err := strconv.Atoi(r.URL.Query().Get("pageSize")); err == nil && v > 0 {
			pageSize = v
		}
		pageNumber := 1
		if v, err := strconv.Atoi(r.URL.Query().Get("pageNumber")); err == nil && v > 0 {
			pageNumber = v
		}

		start := min((pageNumber-1)*pageSize, len(vaults))
		end := min(start+pageSize, len(vaults))
		totalPage := (len(vaults) + pageSize - 1) / pageSize

		_ = json.NewEncoder(w).Encode(map[string]any{
			"data":        vaults[start:end],
			"currentPage": pageNumber,
			"pageSize":    pageSize,
			"totalCount":  len(vaults),
			"totalPage":   totalPage,
		})
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return server
}

func TestResolveVaultRef_RealClientAcrossPages(t *testing.T) {
	vaults := make([]dvls.Vault, 30)
	for i := range vaults {
		vaults[i] = dvls.Vault{
			Id:   fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1),
			Name: fmt.Sprintf("vault-%02d", i+1),
		}
	}

	server := newPaginatedDVLSServer(t, vaults)

	client, err := dvls.NewClient("app-id", "app-secret", server.URL)
	require.NoError(t, err)

	vc := &realVaultClient{vaults: client.Vaults}

	tests := []struct {
		name   string
		vault  string
		wantID string
	}{
		{name: "vault on the first page", vault: "vault-01", wantID: vaults[0].Id},
		{name: "vault on the second page", vault: "vault-30", wantID: vaults[29].Id},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := resolveVaultRef(context.Background(), tt.vault, vc)
			require.NoError(t, err)
			assert.Equal(t, tt.wantID, id)
		})
	}
}
