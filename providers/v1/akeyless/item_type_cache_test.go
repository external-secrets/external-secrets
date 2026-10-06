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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

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
}

func TestItemTypeCacheInvalidate(t *testing.T) {
	c := newItemTypeCache()
	c.set("/prod/db", "STATIC_SECRET")
	c.invalidate("/prod/db")
	_, ok := c.get("/prod/db")
	require.False(t, ok)
}

func TestSecretTypeReturnsCachedTypeWithoutDescribe(t *testing.T) {
	base := &akeylessBase{
		itemTypes: newItemTypeCache(),
	}
	base.itemTypes.set("/prod/db", "ROTATED_SECRET")

	got, err := base.secretType(t.Context(), "/prod/db")
	require.NoError(t, err)
	require.Equal(t, "ROTATED_SECRET", got)
}

func TestGetSecretByTypeInvalidatesCachedTypeOnFetchError(t *testing.T) {
	base := &akeylessBase{itemTypes: newItemTypeCache()}
	base.itemTypes.set("/gone", "NOT_A_REAL_TYPE")

	_, err := base.GetSecretByType(t.Context(), "/gone", 0)
	require.ErrorContains(t, err, "invalid item type")
	_, ok := base.itemTypes.get("/gone")
	require.False(t, ok)
}

func TestSecretTypeIgnoreCacheDoesNotReadCache(t *testing.T) {
	base := &akeylessBase{
		ignoreCache: true,
		itemTypes:   newItemTypeCache(),
	}
	base.itemTypes.set("/prod/db", "STATIC_SECRET")
	// ignoreCache must not use the map; DescribeItem is the next step and
	// needs RestAPI. Assert the cache is skipped by checking get would hit
	// but secretType still tries Describe (token missing).
	_, err := base.secretType(t.Context(), "/prod/db")
	require.ErrorIs(t, err, ErrTokenNotExists)
}
