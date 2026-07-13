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
	"context"
	"crypto/sha256"
	"fmt"
	"sync"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// tokenCacheMap is a process-level cache of oauth2.TokenSource instances, keyed by credential identity.
// Using sync.Map because the cache is read-heavy (one read per reconcile) with rare writes (token refresh).
var tokenCacheMap sync.Map

// cacheKey returns a non-secret stable identifier for a (tokenURL, clientID) pair.
func cacheKey(tokenURL, clientID string) string {
	h := sha256.Sum256([]byte(tokenURL + "\x00" + clientID))
	return fmt.Sprintf("%x", h)
}

// GetOrCreateTokenSource returns a cached oauth2.TokenSource for the given credentials, creating
// one on first call. The returned ReuseTokenSource automatically refreshes the token before expiry.
// Safe for concurrent use.
func GetOrCreateTokenSource(tokenURL, clientID, clientSecret string) oauth2.TokenSource {
	key := cacheKey(tokenURL, clientID)
	if cached, ok := tokenCacheMap.Load(key); ok {
		return cached.(oauth2.TokenSource)
	}

	cfg := clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     tokenURL,
	}
	base := cfg.TokenSource(context.Background())
	rts := oauth2.ReuseTokenSource(nil, base)

	// LoadOrStore handles the race where two goroutines both miss the cache simultaneously;
	// only one entry is stored and both callers use the same source.
	actual, _ := tokenCacheMap.LoadOrStore(key, rts)
	return actual.(oauth2.TokenSource)
}
