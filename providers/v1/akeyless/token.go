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
	"strconv"
	"time"

	"github.com/external-secrets/external-secrets/runtime/esutils"
)

const (
	// Refresh the cached token this long before JWT expiry so GetSecret
	// does not race the token dying mid-call.
	tokenExpirySkew = 30 * time.Second
	// Used when the token is not a JWT (or has no exp). Kept below the
	// Akeyless Kubernetes auth-method default token-exp of 300s.
	defaultTokenCacheTTL = 4 * time.Minute
)

func (a *Akeyless) cachedOrFreshToken(ctx context.Context) (string, error) {
	a.tokenMu.Lock()
	defer a.tokenMu.Unlock()

	if a.cachedToken != "" && time.Now().Before(a.tokenExpiry) {
		return a.cachedToken, nil
	}

	token, err := a.Client.TokenFromSecretRef(ctx)
	if err != nil {
		a.cachedToken = ""
		a.tokenExpiry = time.Time{}
		return "", err
	}
	a.cachedToken = token
	a.tokenExpiry = tokenCacheExpiry(token, time.Now())
	return token, nil
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
	exp := time.Unix(expUnix, 0).Add(-tokenExpirySkew)
	if !exp.After(now) {
		return now
	}
	return exp
}
