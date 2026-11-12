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
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	fakeakeyless "github.com/external-secrets/external-secrets/providers/v1/akeyless/fake"
)

func jwtWithExp(t *testing.T, exp time.Time) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload, err := json.Marshal(map[string]any{"exp": float64(exp.Unix())})
	require.NoError(t, err)
	return header + "." + base64.RawURLEncoding.EncodeToString(payload) + ".sig"
}

func TestTokenCacheExpiryFromJWT(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	exp := now.Add(5 * time.Minute)
	got := tokenCacheExpiry(jwtWithExp(t, exp), now)
	require.Equal(t, exp.Add(-tokenExpirySkew), got)
}

func TestTokenCacheExpirySkewTreatsSoonExpiryAsStale(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	got := tokenCacheExpiry(jwtWithExp(t, now.Add(10*time.Second)), now)
	require.Equal(t, now, got)
}

func TestTokenCacheExpiryNonJWTUsesDefaultTTL(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	got := tokenCacheExpiry("not-a-jwt", now)
	require.Equal(t, now.Add(defaultTokenCacheTTL), got)
}

type countingTokenClient struct {
	*fakeakeyless.AkeylessMockClient
	calls atomic.Int32
	token string
}

func (c *countingTokenClient) TokenFromSecretRef(_ context.Context) (string, error) {
	c.calls.Add(1)
	return c.token, nil
}

func TestCachedOrFreshTokenReusesUntilExpiry(t *testing.T) {
	token := jwtWithExp(t, time.Now().Add(time.Hour))
	mock := &countingTokenClient{
		AkeylessMockClient: fakeakeyless.New(),
		token:              token,
	}
	client := &Akeyless{Client: mock}

	first, cached, err := client.cachedOrFreshToken(context.Background())
	require.NoError(t, err)
	require.False(t, cached)
	require.Equal(t, token, first)

	second, cached, err := client.cachedOrFreshToken(context.Background())
	require.NoError(t, err)
	require.True(t, cached)
	require.Equal(t, token, second)
	require.Equal(t, int32(1), mock.calls.Load(), "expected a single Auth/token fetch")
}

func TestCachedOrFreshTokenRefreshesWhenExpired(t *testing.T) {
	mock := &countingTokenClient{
		AkeylessMockClient: fakeakeyless.New(),
		token:              jwtWithExp(t, time.Now().Add(time.Hour)),
	}
	client := &Akeyless{
		Client:      mock,
		cachedToken: "old",
		tokenExpiry: time.Now().Add(-time.Second),
	}

	got, _, err := client.cachedOrFreshToken(context.Background())
	require.NoError(t, err)
	require.Equal(t, mock.token, got)
	require.Equal(t, int32(1), mock.calls.Load())
}

func TestAuthCredsExpiry(t *testing.T) {
	require.True(t, authCredsExpiry(0).IsZero())
	require.Equal(t, time.Unix(1_700_000_000, 0), authCredsExpiry(1_700_000_000))
	require.Equal(t, time.UnixMilli(1_700_000_000_123), authCredsExpiry(1_700_000_000_123))
}

type expiringTokenClient struct {
	*fakeakeyless.AkeylessMockClient
	expiry time.Time
}

func (c *expiringTokenClient) tokenWithExpiry(_ context.Context) (string, time.Time, error) {
	return "t-opaque", c.expiry, nil
}

func TestCachedOrFreshTokenUsesAuthExpiry(t *testing.T) {
	expiry := time.Now().Add(time.Hour).Truncate(time.Second)
	client := &Akeyless{Client: &expiringTokenClient{AkeylessMockClient: fakeakeyless.New(), expiry: expiry}}

	_, _, err := client.cachedOrFreshToken(context.Background())
	require.NoError(t, err)
	require.Equal(t, expiry.Add(-tokenExpirySkew), client.tokenExpiry)
}

func TestCachedOrFreshTokenFallsBackWithoutAuthExpiry(t *testing.T) {
	client := &Akeyless{Client: &expiringTokenClient{AkeylessMockClient: fakeakeyless.New()}}

	before := time.Now()
	_, _, err := client.cachedOrFreshToken(context.Background())
	require.NoError(t, err)
	require.WithinRange(t, client.tokenExpiry, before.Add(defaultTokenCacheTTL), time.Now().Add(defaultTokenCacheTTL))
}

// rejectingClient issues numbered tokens and rejects the ones listed in rejected.
type rejectingClient struct {
	*fakeakeyless.AkeylessMockClient
	authCalls atomic.Int32
	rejected  map[string]bool
	seen      []string
}

func (c *rejectingClient) TokenFromSecretRef(_ context.Context) (string, error) {
	return fmt.Sprintf("t-%d", c.authCalls.Add(1)), nil
}

func (c *rejectingClient) GetSecretByType(ctx context.Context, _ string, _ int32) (string, error) {
	token, _ := ctx.Value(aKeylessToken).(string)
	c.seen = append(c.seen, token)
	if c.rejected[token] {
		return "", &unauthorizedError{err: errors.New("can't get secret value: token expired")}
	}
	return "secret-val", nil
}

func TestGetSecretRetriesOnceWhenCachedTokenRejected(t *testing.T) {
	mock := &rejectingClient{AkeylessMockClient: fakeakeyless.New(), rejected: map[string]bool{"t-cached": true}}
	client := &Akeyless{Client: mock, cachedToken: "t-cached", tokenExpiry: time.Now().Add(time.Hour)}

	out, err := client.GetSecret(context.Background(), *makeValidRef())

	require.NoError(t, err)
	require.Equal(t, "secret-val", string(out))
	require.Equal(t, []string{"t-cached", "t-1"}, mock.seen)
	require.Equal(t, "t-1", client.cachedToken)
}

func TestGetSecretDoesNotRetryWithFreshToken(t *testing.T) {
	mock := &rejectingClient{AkeylessMockClient: fakeakeyless.New(), rejected: map[string]bool{"t-1": true}}
	client := &Akeyless{Client: mock}

	_, err := client.GetSecret(context.Background(), *makeValidRef())

	require.ErrorIs(t, err, ErrUnauthorized)
	require.Equal(t, int32(1), mock.authCalls.Load())
	require.Equal(t, []string{"t-1"}, mock.seen)
}

func TestInvalidateTokenKeepsConcurrentlyRefreshedToken(t *testing.T) {
	client := &Akeyless{cachedToken: "t-new", tokenExpiry: time.Now().Add(time.Hour)}
	client.invalidateToken("t-old")
	require.Equal(t, "t-new", client.cachedToken)

	client.invalidateToken("t-new")
	require.Empty(t, client.cachedToken)
}

func TestGetSecretReusesTokenAcrossCalls(t *testing.T) {
	token := jwtWithExp(t, time.Now().Add(time.Hour))
	mock := &countingTokenClient{
		AkeylessMockClient: fakeakeyless.New().SetGetSecretFn(func(_ string, _ int32) (string, error) {
			return "secret-val", nil
		}),
		token: token,
	}
	client := &Akeyless{Client: mock}

	for i := range 3 {
		out, err := client.GetSecret(context.Background(), *makeValidRef())
		require.NoError(t, err, fmt.Sprintf("call %d", i))
		require.Equal(t, "secret-val", string(out))
	}
	require.Equal(t, int32(1), mock.calls.Load())
}
