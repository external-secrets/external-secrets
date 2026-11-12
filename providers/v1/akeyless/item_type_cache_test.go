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

package akeyless

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/akeylesslabs/akeyless-go/v4"
	"github.com/stretchr/testify/require"
)

type cannedResponse struct {
	status int
	body   string
}

// newRoutedAkeylessBase serves canned responses per API path, counts the
// requests per path, and returns an akeylessBase with an item-type cache.
func newRoutedAkeylessBase(t *testing.T, routes map[string]cannedResponse) (*akeylessBase, func(path string) int) {
	t.Helper()
	var mu sync.Mutex
	calls := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls[r.URL.Path]++
		mu.Unlock()
		resp, ok := routes[r.URL.Path]
		if !ok {
			resp = cannedResponse{status: http.StatusInternalServerError, body: `{"error":"unexpected path"}`}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.status)
		_, _ = w.Write([]byte(resp.body))
	}))
	t.Cleanup(srv.Close)
	base := &akeylessBase{
		RestAPI: akeyless.NewAPIClient(&akeyless.Configuration{
			Servers: []akeyless.ServerConfiguration{{URL: srv.URL}},
		}).V2Api,
		itemTypes: newItemTypeCache(),
	}
	count := func(path string) int {
		mu.Lock()
		defer mu.Unlock()
		return calls[path]
	}
	return base, count
}

func TestItemTypeCacheHitAndExpiry(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	c := newItemTypeCache()
	c.now = func() time.Time { return now }

	_, ok := c.get("/prod/db")
	require.False(t, ok)

	c.set("/prod/db", "STATIC_SECRET")
	got, ok := c.get("/prod/db")
	require.True(t, ok)
	require.Equal(t, "STATIC_SECRET", got)

	c.now = func() time.Time { return now.Add(itemTypeCacheTTL + time.Second) }
	_, ok = c.get("/prod/db")
	require.False(t, ok)
	require.Equal(t, 0, c.entries.Len())
}

func TestItemTypeCacheInvalidate(t *testing.T) {
	c := newItemTypeCache()
	c.set("/prod/db", "STATIC_SECRET")
	c.invalidate("/prod/db")
	_, ok := c.get("/prod/db")
	require.False(t, ok)
}

func TestItemTypeCacheEnforcesMaxEntries(t *testing.T) {
	c := newItemTypeCacheWithSize(2)
	c.set("/a", "STATIC_SECRET")
	c.set("/b", "STATIC_SECRET")
	c.set("/c", "STATIC_SECRET")

	require.Equal(t, 2, c.entries.Len())
	_, ok := c.get("/a")
	require.False(t, ok, "least recently used entry should be evicted")
	_, ok = c.get("/c")
	require.True(t, ok)
}

func TestSecretTypeReturnsCachedTypeWithoutDescribe(t *testing.T) {
	base := &akeylessBase{
		itemTypes: newItemTypeCache(),
	}
	base.itemTypes.set("/prod/db", "ROTATED_SECRET")

	got, cached, err := base.secretType(t.Context(), "/prod/db")
	require.NoError(t, err)
	require.True(t, cached)
	require.Equal(t, "ROTATED_SECRET", got)
}

func TestSecretTypeIgnoreCacheDoesNotReadCache(t *testing.T) {
	base := &akeylessBase{
		ignoreCache: true,
		itemTypes:   newItemTypeCache(),
	}
	base.itemTypes.set("/prod/db", "STATIC_SECRET")
	// With the cache skipped, secretType goes to DescribeItem, which fails
	// here because the context carries no token.
	_, _, err := base.secretType(t.Context(), "/prod/db")
	require.ErrorIs(t, err, ErrTokenNotExists)
}

func TestGetSecretByTypeCachedTypeDeletedItemReturnsNotExists(t *testing.T) {
	base, count := newRoutedAkeylessBase(t, map[string]cannedResponse{
		"/get-secret-value": {status: http.StatusNotFound, body: `{"error":"item not found"}`},
		"/describe-item":    {status: http.StatusNotFound, body: `{"error":"item not found"}`},
	})
	base.itemTypes.set("/gone", "STATIC_SECRET")

	_, err := base.GetSecretByType(describeItemCtx(), "/gone", 0)

	require.ErrorIs(t, err, ErrItemNotExists)
	require.Equal(t, 1, count("/describe-item"))
	_, ok := base.itemTypes.get("/gone")
	require.False(t, ok)
}

func TestGetSecretByTypeCachedTypeRecreatedAsOtherKind(t *testing.T) {
	base, count := newRoutedAkeylessBase(t, map[string]cannedResponse{
		"/get-secret-value":         {status: http.StatusBadRequest, body: `{"error":"not a static secret"}`},
		"/describe-item":            {status: http.StatusOK, body: `{"item_name":"/db","item_type":"DYNAMIC_SECRET"}`},
		"/get-dynamic-secret-value": {status: http.StatusOK, body: `{"user":"u1"}`},
	})
	base.itemTypes.set("/db", "STATIC_SECRET")

	got, err := base.GetSecretByType(describeItemCtx(), "/db", 0)

	require.NoError(t, err)
	require.JSONEq(t, `{"user":"u1"}`, got)
	require.Equal(t, 1, count("/get-secret-value"))
	require.Equal(t, 1, count("/get-dynamic-secret-value"))
	itemType, ok := base.itemTypes.get("/db")
	require.True(t, ok)
	require.Equal(t, "DYNAMIC_SECRET", itemType)
}

func TestGetSecretByTypeCachedTypeUnchangedReturnsOriginalError(t *testing.T) {
	base, count := newRoutedAkeylessBase(t, map[string]cannedResponse{
		"/get-secret-value": {status: http.StatusInternalServerError, body: `{"error":"temporary failure"}`},
		"/describe-item":    {status: http.StatusOK, body: `{"item_name":"/db","item_type":"STATIC_SECRET"}`},
	})
	base.itemTypes.set("/db", "STATIC_SECRET")

	_, err := base.GetSecretByType(describeItemCtx(), "/db", 0)

	require.ErrorContains(t, err, "temporary failure")
	require.Equal(t, 1, count("/get-secret-value"), "same type must not be fetched twice")
}

func TestGetSecretByTypeCachedTypeUnauthorizedSkipsDescribe(t *testing.T) {
	base, count := newRoutedAkeylessBase(t, map[string]cannedResponse{
		"/get-secret-value": {status: http.StatusUnauthorized, body: `{"error":"UnauthorizedAccess"}`},
	})
	base.itemTypes.set("/db", "STATIC_SECRET")

	_, err := base.GetSecretByType(describeItemCtx(), "/db", 0)

	require.ErrorIs(t, err, ErrUnauthorized)
	require.ErrorContains(t, err, "can't get secret value: ")
	require.Equal(t, 0, count("/describe-item"))
}
