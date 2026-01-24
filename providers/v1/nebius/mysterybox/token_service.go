// /*
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package mysterybox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	lru "github.com/hashicorp/golang-lru"
	"github.com/nebius/gosdk/auth"
	"k8s.io/utils/clock"

	"github.com/external-secrets/external-secrets/providers/v1/nebius/common/sdk/iam"
)

const (
	errInvalidSubjectCreds = "invalid subject credentials: malformed JSON"
)

// TokenService is an interface for generating and retrieving authentication tokens.
type TokenService interface {
	GetToken(ctx context.Context, apiDomain, subjectCreds string, caCert []byte) (string, error)
}

type tokenCacheKey struct {
	APIDomain        string
	PublicKeyID      string
	ServiceAccountID string
	PrivateKeyHash   string
}

// TokenCacheService is responsible for managing Nebius IAM token caching and token exchange processes.
type TokenCacheService struct {
	TokenExchanger iam.TokenExchangerClient
	Clock          clock.Clock
	tokenCache     *lru.Cache
}

// NewTokenCacheService initializes a TokenCacheService with the specified cache size, token exchanger client, and clock.
// Returns a TokenCacheService instance and an error if LRU cache creation fails.
func NewTokenCacheService(cacheSize int, client iam.TokenExchangerClient, clock clock.Clock) (*TokenCacheService, error) {
	cache, err := lru.New(cacheSize)
	if err != nil {
		return nil, err
	}
	return &TokenCacheService{
		tokenCache:     cache,
		TokenExchanger: client,
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
func (c *TokenCacheService) GetToken(ctx context.Context, apiDomain, subjectCreds string, caCert []byte) (string, error) {
	byteCreds := []byte(subjectCreds)
	cacheKey, err := buildTokenCacheKey(byteCreds, apiDomain)

	if err != nil {
		return "", err
	}
	value, ok := c.tokenCache.Get(*cacheKey)
	if ok {
		token := value.(*iam.Token)
		tokenExpired := isTokenExpired(token, c.Clock)
		if !tokenExpired {
			return token.Token, nil
		}
	}

	newToken, err := c.TokenExchanger.NewIamToken(ctx, apiDomain, subjectCreds, c.Clock.Now(), caCert)
	if err != nil {
		return "", fmt.Errorf("could not exchange creds to iam token: %w", MapGrpcErrors("create token", err))
	}
	c.tokenCache.Add(*cacheKey, newToken)
	return newToken.Token, nil
}

func buildTokenCacheKey(subjectCreds []byte, apiDomain string) (*tokenCacheKey, error) {
	parsedSubjectCreds := &auth.ServiceAccountCredentials{}
	err := json.Unmarshal(subjectCreds, parsedSubjectCreds)
	if err != nil {
		return nil, errors.New(errInvalidSubjectCreds)
	}
	return &tokenCacheKey{
		APIDomain:        apiDomain,
		PublicKeyID:      parsedSubjectCreds.SubjectCredentials.KeyID,
		ServiceAccountID: parsedSubjectCreds.SubjectCredentials.Subject,
		PrivateKeyHash:   HashBytes([]byte(parsedSubjectCreds.SubjectCredentials.PrivateKey)),
	}, nil
}

var _ TokenService = &TokenCacheService{}
