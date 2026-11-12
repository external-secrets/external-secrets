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
	"errors"
	"strconv"
	"time"

	"github.com/external-secrets/external-secrets/runtime/esutils"
)

const (
	// Refresh the cached token this long before it expires so GetSecret
	// does not race the token dying mid-call.
	tokenExpirySkew = 30 * time.Second
	// Used when Auth returns no expiry and the token is not a JWT. Kept below
	// the Akeyless Kubernetes auth-method default token-exp of 300s.
	defaultTokenCacheTTL = 4 * time.Minute
)

// expiringTokenSource is implemented by clients that can report the token
// expiry returned by Akeyless Auth.
type expiringTokenSource interface {
	tokenWithExpiry(ctx context.Context) (string, time.Time, error)
}

// withToken runs fn with an Akeyless token in ctx. If a cached token is
// rejected, it is dropped and fn is retried once with a fresh token.
func withToken[T any](ctx context.Context, a *Akeyless, fn func(context.Context) (T, error)) (T, error) {
	if ctx.Value(aKeylessToken) != nil {
		return fn(ctx)
	}
	var zero T
	token, cached, err := a.cachedOrFreshToken(ctx)
	if err != nil {
		return zero, err
	}
	out, err := fn(context.WithValue(ctx, aKeylessToken, token))
	if !cached || !errors.Is(err, ErrUnauthorized) {
		return out, err
	}
	a.invalidateToken(token)
	token, _, err = a.cachedOrFreshToken(ctx)
	if err != nil {
		return zero, err
	}
	return fn(context.WithValue(ctx, aKeylessToken, token))
}

func (a *Akeyless) cachedOrFreshToken(ctx context.Context) (token string, cached bool, err error) {
	a.tokenMu.Lock()
	defer a.tokenMu.Unlock()

	if a.cachedToken != "" && time.Now().Before(a.tokenExpiry) {
		return a.cachedToken, true, nil
	}

	token, expiry, err := a.freshToken(ctx)
	if err != nil {
		a.cachedToken = ""
		a.tokenExpiry = time.Time{}
		return "", false, err
	}
	a.cachedToken = token
	a.tokenExpiry = expiry
	return token, false, nil
}

// invalidateToken drops the cached token only if it is still the rejected one,
// so a token refreshed concurrently by another reconcile is kept.
func (a *Akeyless) invalidateToken(token string) {
	a.tokenMu.Lock()
	defer a.tokenMu.Unlock()
	if a.cachedToken == token {
		a.cachedToken = ""
		a.tokenExpiry = time.Time{}
	}
}

func (a *Akeyless) freshToken(ctx context.Context) (string, time.Time, error) {
	if src, ok := a.Client.(expiringTokenSource); ok {
		token, exp, err := src.tokenWithExpiry(ctx)
		if err != nil {
			return "", time.Time{}, err
		}
		now := time.Now()
		if !exp.IsZero() {
			return token, expiryWithSkew(exp, now), nil
		}
		return token, tokenCacheExpiry(token, now), nil
	}
	token, err := a.Client.TokenFromSecretRef(ctx)
	if err != nil {
		return "", time.Time{}, err
	}
	return token, tokenCacheExpiry(token, time.Now()), nil
}

func tokenCacheExpiry(token string, now time.Time) time.Time {
	expStr, err := esutils.ExtractJWTExpiration(token)
	if err != nil {
		return now.Add(defaultTokenCacheTTL)
	}
	expUnix, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil {
		return now.Add(defaultTokenCacheTTL)
	}
	return expiryWithSkew(time.Unix(expUnix, 0), now)
}

func expiryWithSkew(exp, now time.Time) time.Time {
	exp = exp.Add(-tokenExpirySkew)
	if !exp.After(now) {
		return now
	}
	return exp
}

// authCredsExpiry converts the Auth creds expiry to a time. The SDK only
// documents it as an expiration date, so both Unix seconds and milliseconds
// are accepted.
func authCredsExpiry(expiry int64) time.Time {
	if expiry <= 0 {
		return time.Time{}
	}
	if expiry > 1e12 {
		return time.UnixMilli(expiry)
	}
	return time.Unix(expiry, 0)
}
