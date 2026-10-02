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
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"sync"
	"time"

	infisicalSdk "github.com/infisical/go-sdk"
	sdkErrors "github.com/infisical/go-sdk/packages/errors"
	"github.com/spf13/pflag"
	"golang.org/x/sync/singleflight"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/providers/v1/infisical/constants"
	"github.com/external-secrets/external-secrets/runtime/feature"
	"github.com/external-secrets/external-secrets/runtime/metrics"
)

// maxRefreshMargin caps how long before its expiry a cached token is replaced.
// A reconcile lasts seconds, so a token is never used past its expiry.
const maxRefreshMargin = 5 * time.Minute

// accessTokenCache shares machine-identity access tokens across reconciles, so
// a resync makes one login per identity instead of one per ExternalSecret.
// Cached tokens are never revoked: other clients may still use them, and they
// expire on their own.
var accessTokenCache = newTokenCache(time.Now)

var enableTokenCache bool

func init() {
	fs := pflag.NewFlagSet("infisical", pflag.ExitOnError)
	fs.BoolVar(
		&enableTokenCache,
		"enable-infisical-token-cache",
		false,
		"Enable Infisical access token cache. External secrets will reuse the access token until it expires, instead of a login on each request.",
	)
	feature.Register(feature.Feature{Flags: fs})
}

type cachedToken struct {
	accessToken string
	refreshAt   time.Time
}

type tokenCache struct {
	mu      sync.Mutex
	entries map[string]cachedToken
	// owned holds the tokens that are not cached. Their client revokes them.
	owned  map[string]struct{}
	logins singleflight.Group
	now    func() time.Time
}

func newTokenCache(now func() time.Time) *tokenCache {
	return &tokenCache{entries: map[string]cachedToken{}, owned: map[string]struct{}{}, now: now}
}

// cachedLogin authenticates sdkClient with a cached token, or runs login and
// caches its token. The key covers every login input, so a store can use a
// cached token only with the same credentials against the same host. Without
// --enable-infisical-token-cache, it only runs login.
func cachedLogin(
	sdkClient infisicalSdk.InfisicalClientInterface,
	infisicalSpec *esv1.InfisicalProvider,
	authMethod string,
	inputs []string,
	login func() (infisicalSdk.MachineIdentityCredential, error),
) error {
	organizationSlug := infisicalSpec.SecretsScope.OrganizationSlug
	auth := sdkClient.Auth().WithOrganizationSlug(organizationSlug)
	observedLogin := func() (infisicalSdk.MachineIdentityCredential, error) {
		credential, err := login()
		metrics.ObserveAPICall(constants.ProviderName, authMethod, err)
		return credential, err
	}
	if !enableTokenCache {
		_, err := observedLogin()
		return err
	}

	key := tokenCacheKey(append([]string{infisicalSpec.HostAPI, organizationSlug, authMethod}, inputs...)...)
	accessToken, cached, err := accessTokenCache.login(key, observedLogin)
	if err == nil && cached {
		auth.SetAccessToken(accessToken)
	}
	return err
}

func tokenCacheKey(parts ...string) string {
	h := sha256.New()
	for _, part := range parts {
		h.Write([]byte(part))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func (c *tokenCache) get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[key]
	if !ok || !c.now().Before(entry.refreshAt) {
		return "", false
	}
	return entry.accessToken, true
}

// store keeps the token until a margin before its expiry and drops the expired
// entries. A token without a TTL is not cached: it is recorded as owned.
func (c *tokenCache) store(key string, credential infisicalSdk.MachineIdentityCredential, issuedAt time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if credential.ExpiresIn <= 0 {
		c.owned[credential.AccessToken] = struct{}{}
		return false
	}
	ttl := time.Duration(credential.ExpiresIn) * time.Second
	now := c.now()
	for k, entry := range c.entries {
		if !now.Before(entry.refreshAt) {
			delete(c.entries, k)
		}
	}
	c.entries[key] = cachedToken{
		accessToken: credential.AccessToken,
		refreshAt:   issuedAt.Add(ttl - min(ttl/5, maxRefreshMargin)),
	}
	return true
}

// login returns a cached token for key, or runs loginFn once for all
// concurrent callers with the same key.
func (c *tokenCache) login(key string, loginFn func() (infisicalSdk.MachineIdentityCredential, error)) (accessToken string, cached bool, err error) {
	if token, ok := c.get(key); ok {
		return token, true, nil
	}

	type result struct {
		accessToken string
		cached      bool
	}
	ownLogin := false
	v, err, _ := c.logins.Do(key, func() (any, error) {
		if token, ok := c.get(key); ok {
			return result{accessToken: token, cached: true}, nil
		}
		ownLogin = true
		issuedAt := c.now()
		credential, err := loginFn()
		if err != nil {
			return nil, err
		}
		return result{accessToken: credential.AccessToken, cached: c.store(key, credential, issuedAt)}, nil
	})
	if err != nil {
		return "", false, err
	}

	r := v.(result)
	if r.cached || ownLogin {
		return r.accessToken, r.cached, nil
	}
	// Another caller owns this uncached token and revokes it on Close.
	credential, err := loginFn()
	if err != nil {
		return "", false, err
	}
	c.store(key, credential, c.now())
	return credential.AccessToken, false, nil
}

// invalidateToken drops the entry that holds accessToken. A token that another
// client stored after a new login stays in the cache.
func (c *tokenCache) invalidateToken(accessToken string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for k, entry := range c.entries {
		if entry.accessToken == accessToken {
			delete(c.entries, k)
		}
	}
}

// releaseOwned reports whether the caller owns accessToken and must revoke it.
func (c *tokenCache) releaseOwned(accessToken string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, ok := c.owned[accessToken]
	delete(c.owned, accessToken)
	return ok
}

// observeAPICall records the call metric. On a 401 it drops the cached token,
// so the next reconcile logs in again instead of reusing a revoked token.
func (p *Provider) observeAPICall(call string, err error) {
	metrics.ObserveAPICall(constants.ProviderName, call, err)
	if isUnauthorizedError(err) {
		accessTokenCache.invalidateToken(p.sdkClient.Auth().GetAccessToken())
	}
}

func isUnauthorizedError(err error) bool {
	var apiErr *sdkErrors.APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusUnauthorized
}
