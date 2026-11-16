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

package mysterybox

import (
	"context"
	"fmt"

	lru "github.com/hashicorp/golang-lru"
	"golang.org/x/sync/singleflight"
	"k8s.io/utils/clock"

	"github.com/external-secrets/external-secrets/providers/v1/nebius/common/auth"
	"github.com/external-secrets/external-secrets/providers/v1/nebius/common/sdk/iam"
)

// TokenGetter is an interface for generating and retrieving authentication tokens.
type TokenGetter interface {
	// GetToken returns an IAM token for the provided credentials request.
	GetToken(ctx context.Context, apiDomain string, credentialsRequest *auth.CredentialRequest, caCert []byte) (string, error)
}

// CachedTokenGetter is responsible for managing Nebius IAM token caching and token exchange processes.
type CachedTokenGetter struct {
	TokenExchanger iam.TokenExchanger
	Clock          clock.Clock
	tokenCache     *lru.Cache
	sf             singleflight.Group
}

// NewCachedTokenGetter initializes a CachedTokenGetter with the specified cache size, token exchanger, and clock.
// Returns a CachedTokenGetter instance and an error if LRU cache creation fails.
func NewCachedTokenGetter(cacheSize int, tokenExchanger iam.TokenExchanger, clock clock.Clock) (*CachedTokenGetter, error) {
	cache, err := lru.New(cacheSize)
	if err != nil {
		return nil, err
	}
	return &CachedTokenGetter{
		tokenCache:     cache,
		TokenExchanger: tokenExchanger,
		Clock:          clock,
	}, nil
}

func isTokenExpired(token *iam.Token, clk clock.Clock) bool {
	now := clk.Now()
	if token.ExpiresAt.After(now) {
		total := token.ExpiresAt.Sub(token.IssuedAt)
		remaining := token.ExpiresAt.Sub(now)
		if remaining > total/10 {
			return false
		}
	}
	return true
}

// GetToken retrieves an IAM token for the given API domain and subject credentials, using a cache to optimize requests.
// It exchanges credentials for a new token if no valid cached token exists or the cached token is nearing expiration.
func (c *CachedTokenGetter) GetToken(ctx context.Context, apiDomain string, credentialsRequest *auth.CredentialRequest, caCert []byte) (string, error) {
	cacheKey := credentialsRequest.CacheKey()

	value, ok := c.tokenCache.Get(cacheKey)
	if ok {
		token := value.(*iam.Token)
		tokenExpired := isTokenExpired(token, c.Clock)
		if !tokenExpired {
			return token.Token, nil
		}
	}

	token, err, _ := c.sf.Do(cacheKey, func() (any, error) {
		if v, ok := c.tokenCache.Get(cacheKey); ok {
			tok := v.(*iam.Token)
			if !isTokenExpired(tok, c.Clock) {
				return tok.Token, nil
			}
		}
		newToken, err := c.exchangeToken(ctx, apiDomain, credentialsRequest, caCert)
		if err != nil {
			return "", err
		}

		c.tokenCache.Add(cacheKey, newToken)
		return newToken.Token, nil
	})

	if err != nil {
		return "", err
	}
	return token.(string), nil
}

func (c *CachedTokenGetter) exchangeToken(ctx context.Context, apiDomain string, credentialsRequest *auth.CredentialRequest, caCert []byte) (*iam.Token, error) {
	resolvedCreds, err := credentialsRequest.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	token, err := c.TokenExchanger.ExchangeIamToken(ctx, apiDomain, resolvedCreds, c.Clock.Now(), caCert)
	if err != nil {
		return nil, fmt.Errorf("could not exchange creds to iam token: %w", MapGrpcErrors("create token", err))
	}
	return token, nil
}

var _ TokenGetter = &CachedTokenGetter{}
