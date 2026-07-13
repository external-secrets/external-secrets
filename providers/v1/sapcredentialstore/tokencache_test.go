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

package sapcredentialstore

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// staticSource is a TokenSource that returns a pre-built token.
type staticSource struct {
	token *oauth2.Token
}

func (s *staticSource) Token() (*oauth2.Token, error) {
	return s.token, nil
}

// resetCache clears the package-level token cache between test runs.
func resetCache() {
	tokenCacheMap.Range(func(k, _ any) bool {
		tokenCacheMap.Delete(k)
		return true
	})
}

// T019: cache miss on first call, cache hit on second call with same credentials.
func TestTokenCache_MissAndHit(t *testing.T) {
	resetCache()

	ts1 := GetOrCreateTokenSource("https://auth.example.com/oauth/token", "client-a", "secret-a")
	require.NotNil(t, ts1)

	// Second call with identical credentials must return the same source object.
	ts2 := GetOrCreateTokenSource("https://auth.example.com/oauth/token", "client-a", "secret-a")
	assert.Same(t, ts1, ts2, "cache hit must return the identical TokenSource")
}

// T019: different credential identity produces a different source.
func TestTokenCache_DifferentIdentityDifferentSource(t *testing.T) {
	resetCache()

	ts1 := GetOrCreateTokenSource("https://auth.example.com/oauth/token", "client-a", "secret-a")
	ts2 := GetOrCreateTokenSource("https://auth.example.com/oauth/token", "client-b", "secret-b")
	assert.NotSame(t, ts1, ts2, "different clientID must produce a distinct TokenSource")
}

// T019: concurrent calls with the same identity do not race and all get the same source.
func TestTokenCache_ConcurrentSameIdentity(t *testing.T) {
	resetCache()

	const goroutines = 50
	results := make([]oauth2.TokenSource, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := range goroutines {
		go func(idx int) {
			defer wg.Done()
			results[idx] = GetOrCreateTokenSource("https://auth.example.com/oauth/token", "client-c", "secret-c")
		}(i)
	}
	wg.Wait()

	first := results[0]
	for i, ts := range results {
		assert.Same(t, first, ts, "goroutine %d got a different TokenSource", i)
	}
}

// T020: ReuseTokenSource serves a still-valid token without calling the base source again.
func TestTokenCache_ReuseValidToken(t *testing.T) {
	resetCache()

	callCount := 0
	base := &countingSource{
		token: &oauth2.Token{
			AccessToken: "valid-token",
			Expiry:      time.Now().Add(10 * time.Minute),
		},
		onToken: func() { callCount++ },
	}
	rts := oauth2.ReuseTokenSource(nil, base)

	tok1, err := rts.Token()
	require.NoError(t, err)
	assert.Equal(t, "valid-token", tok1.AccessToken)

	tok2, err := rts.Token()
	require.NoError(t, err)
	assert.Equal(t, "valid-token", tok2.AccessToken)

	// Base source must have been called exactly once; subsequent calls use the cached token.
	assert.Equal(t, 1, callCount, "base source should be called only once for a valid cached token")
}

type countingSource struct {
	token   *oauth2.Token
	onToken func()
}

func (c *countingSource) Token() (*oauth2.Token, error) {
	c.onToken()
	return c.token, nil
}
