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
	"encoding/json"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/nebius/gosdk/auth"
	tassert "github.com/stretchr/testify/assert"
	trequire "github.com/stretchr/testify/require"
	clocktesting "k8s.io/utils/clock/testing"

	"github.com/external-secrets/external-secrets/providers/v1/nebius/common/sdk/iam"
)

type tokenTestEnv struct {
	ctx                context.Context
	clk                *clocktesting.FakeClock
	fakeTokenExchanger *iam.FakeTokenExchanger
	cachedTokenGetter  *CachedTokenGetter
}

func newTokenTestEnv(t *testing.T) *tokenTestEnv {
	t.Helper()
	clk := clocktesting.NewFakeClock(time.Unix(0, 0))
	ex := &iam.FakeTokenExchanger{}
	svc, err := NewCachedTokenGetter(10, ex, clk)
	trequire.NoError(t, err)
	return &tokenTestEnv{ctx: context.Background(), clk: clk, fakeTokenExchanger: ex, cachedTokenGetter: svc}
}

func buildSubjectCredsJSON(t *testing.T, privateKey, keyID, subject string) string {
	t.Helper()
	b, err := json.Marshal(&auth.ServiceAccountCredentials{
		SubjectCredentials: auth.SubjectCredentials{
			PrivateKey: privateKey,
			KeyID:      keyID,
			Subject:    subject,
			Issuer:     subject,
		},
	})
	trequire.NoError(t, err)
	return string(b)
}

func serviceAccountCredsTokenRequest(apiDomain, subjectCreds string) *iam.TokenRequest {
	return &iam.TokenRequest{
		APIDomain:    apiDomain,
		AuthType:     iam.TokenAuthTypeServiceAccountCreds,
		SubjectCreds: subjectCreds,
	}
}

func federatedServiceAccountTokenRequest(apiDomain, subjectToken, namespace, name string, audiences ...string) *iam.TokenRequest {
	return &iam.TokenRequest{
		APIDomain:               apiDomain,
		AuthType:                iam.TokenAuthTypeFederatedServiceAccount,
		SubjectToken:            subjectToken,
		ServiceAccountNamespace: namespace,
		ServiceAccountName:      name,
		ServiceAccountAudiences: audiences,
	}
}

func TestGetToken_CachesUntilTenPercentLeft(t *testing.T) {
	t.Parallel()
	env := newTokenTestEnv(t)

	ctx := env.ctx
	creds := buildSubjectCredsJSON(t, "priv-A", "kid-A", "sa-A")

	token1, err := env.cachedTokenGetter.GetToken(ctx, serviceAccountCredsTokenRequest("api.example", creds), nil)
	tassert.NoError(t, err)
	tassert.Equal(t, "token-1", token1)
	tassert.Equal(t, int64(1), env.fakeTokenExchanger.Calls.Load())

	// add 5 seconds (remaining > 10%)
	addSecondsToClock(env.clk, 5)
	token2, err := env.cachedTokenGetter.GetToken(ctx, serviceAccountCredsTokenRequest("api.example", creds), nil)
	tassert.NoError(t, err)
	tassert.Equal(t, token1, token2)
	tassert.Equal(t, int64(1), env.fakeTokenExchanger.Calls.Load())

	// after >90% elapsed -> should refresh
	addSecondsToClock(env.clk, 91) // total 96s
	token3, err := env.cachedTokenGetter.GetToken(ctx, serviceAccountCredsTokenRequest("api.example", creds), nil)
	tassert.NoError(t, err)
	tassert.NotEqual(t, token1, token3)
	tassert.Equal(t, int64(2), env.fakeTokenExchanger.Calls.Load())
}

func TestGetToken_SeparateCacheEntriesPerSubjectCreds(t *testing.T) {
	t.Parallel()
	env := newTokenTestEnv(t)

	ctx := env.ctx
	credsA := buildSubjectCredsJSON(t, "priv-A", "kid-A", "sa-A")
	credsB := buildSubjectCredsJSON(t, "priv-B", "kid-B", "sa-B")

	tokenA1, err := env.cachedTokenGetter.GetToken(ctx, serviceAccountCredsTokenRequest("api.example", credsA), nil)
	tassert.NoError(t, err)
	tassert.Equal(t, "token-1", tokenA1)

	tokenB1, err := env.cachedTokenGetter.GetToken(ctx, serviceAccountCredsTokenRequest("api.example", credsB), nil)
	tassert.NoError(t, err)
	tassert.Equal(t, "token-2", tokenB1)

	tassert.Equal(t, int64(2), env.fakeTokenExchanger.Calls.Load())

	// check token cached
	addSecondsToClock(env.clk, 1)
	tokA2, err := env.cachedTokenGetter.GetToken(ctx, serviceAccountCredsTokenRequest("api.example", credsA), nil)
	tassert.NoError(t, err)
	tassert.Equal(t, tokenA1, tokA2)
	tassert.Equal(t, int64(2), env.fakeTokenExchanger.Calls.Load())
}

func TestGetToken_InvalidSubjectCreds_ReturnsError(t *testing.T) {
	t.Parallel()
	env := newTokenTestEnv(t)

	_, err := env.cachedTokenGetter.GetToken(env.ctx, serviceAccountCredsTokenRequest("api.example", "not a json"), nil)
	tassert.Error(t, err)
}

func addSecondsToClock(clk *clocktesting.FakeClock, second time.Duration) {
	clk.SetTime(clk.Now().Add(second * time.Second))
}

func TestGetToken_SeparateCacheEntriesPerApiDomain(t *testing.T) {
	t.Parallel()
	env := newTokenTestEnv(t)
	ctx := env.ctx
	creds := buildSubjectCredsJSON(t, "priv-A", "kid-A", "sa-A")

	tokA1, err := env.cachedTokenGetter.GetToken(ctx, serviceAccountCredsTokenRequest("api.one", creds), nil)
	tassert.NoError(t, err)
	tassert.Equal(t, "token-1", tokA1)

	tokB1, err := env.cachedTokenGetter.GetToken(ctx, serviceAccountCredsTokenRequest("api.two", creds), nil)
	tassert.NoError(t, err)
	tassert.Equal(t, "token-2", tokB1)
	tassert.NotEqual(t, tokA1, tokB1)
	tassert.Equal(t, int64(2), env.fakeTokenExchanger.Calls.Load())

	tokA2, err := env.cachedTokenGetter.GetToken(ctx, serviceAccountCredsTokenRequest("api.one", creds), nil)
	tassert.NoError(t, err)
	tassert.Equal(t, tokA1, tokA2)
	tassert.Equal(t, int64(2), env.fakeTokenExchanger.Calls.Load())
}

func TestGetToken_LRUEviction(t *testing.T) {
	t.Parallel()
	clk := clocktesting.NewFakeClock(time.Unix(0, 0))
	ex := &iam.FakeTokenExchanger{}
	svc, err := NewCachedTokenGetter(2, ex, clk)
	tassert.NoError(t, err)
	ctx := context.Background()
	creds := buildSubjectCredsJSON(t, "priv-A", "kid-A", "sa-A")

	tok1, err := svc.GetToken(ctx, serviceAccountCredsTokenRequest("api.first", creds), nil)
	tassert.NoError(t, err)
	tassert.Equal(t, "token-1", tok1)

	tok2, err := svc.GetToken(ctx, serviceAccountCredsTokenRequest("api.second", creds), nil)
	tassert.NoError(t, err)
	tassert.Equal(t, "token-2", tok2)
	tassert.Equal(t, int64(2), ex.Calls.Load())

	tok1again, err := svc.GetToken(ctx, serviceAccountCredsTokenRequest("api.first", creds), nil)
	tassert.NoError(t, err)
	tassert.Equal(t, tok1, tok1again)
	tassert.Equal(t, int64(2), ex.Calls.Load())

	tok3, err := svc.GetToken(ctx, serviceAccountCredsTokenRequest("api.third", creds), nil)
	tassert.NoError(t, err)
	tassert.Equal(t, "token-3", tok3)
	tassert.Equal(t, int64(3), ex.Calls.Load())

	secondAgain, err := svc.GetToken(ctx, serviceAccountCredsTokenRequest("api.second", creds), nil)
	tassert.NoError(t, err)
	tassert.Equal(t, "token-4", secondAgain)
	tassert.Equal(t, int64(4), ex.Calls.Load())
}

func TestGetToken_AfterExpiration_Refreshes(t *testing.T) {
	t.Parallel()
	env := newTokenTestEnv(t)
	ctx := env.ctx
	creds := buildSubjectCredsJSON(t, "priv-A", "kid-A", "sa-A")

	_, err := env.cachedTokenGetter.GetToken(ctx, serviceAccountCredsTokenRequest("api.example", creds), nil)
	tassert.NoError(t, err)
	addSecondsToClock(env.clk, 101)

	tok2, err := env.cachedTokenGetter.GetToken(ctx, serviceAccountCredsTokenRequest("api.example", creds), nil)
	tassert.NoError(t, err)
	tassert.Equal(t, int64(2), env.fakeTokenExchanger.Calls.Load())
	tassert.Equal(t, "token-2", tok2)
}

func TestGetToken_CacheKeyChangesOnKeyRotation(t *testing.T) {
	t.Parallel()
	env := newTokenTestEnv(t)
	ctx := env.ctx

	base := buildSubjectCredsJSON(t, "priv-A", "kid-A", "sa-A")
	rotatedKeyID := buildSubjectCredsJSON(t, "priv-A", "kid-B", "sa-A")
	rotatedPriv := buildSubjectCredsJSON(t, "priv-B", "kid-A", "sa-A")
	rotatedSubject := buildSubjectCredsJSON(t, "priv-A", "kid-A", "sa-B")

	t1, _ := env.cachedTokenGetter.GetToken(ctx, serviceAccountCredsTokenRequest("api", base), nil)
	t2, _ := env.cachedTokenGetter.GetToken(ctx, serviceAccountCredsTokenRequest("api", rotatedKeyID), nil)
	t3, _ := env.cachedTokenGetter.GetToken(ctx, serviceAccountCredsTokenRequest("api", rotatedPriv), nil)
	t4, _ := env.cachedTokenGetter.GetToken(ctx, serviceAccountCredsTokenRequest("api", rotatedSubject), nil)

	tassert.NotEqual(t, t1, t2)
	tassert.NotEqual(t, t1, t3)
	tassert.NotEqual(t, t1, t4)
	tassert.Equal(t, int64(4), env.fakeTokenExchanger.Calls.Load())
}

func TestGetToken_ExchangerErrorIsWrapped(t *testing.T) {
	t.Parallel()
	clk := clocktesting.NewFakeClock(time.Unix(0, 0))
	svc, err := NewCachedTokenGetter(10, &iam.FakeTokenExchanger{ReturnError: true}, clk)
	trequire.NoError(t, err)

	_, err = svc.GetToken(context.Background(), serviceAccountCredsTokenRequest("api", buildSubjectCredsJSON(t, "p", "k", "s")), nil)
	tassert.Error(t, err)
	tassert.Contains(t, err.Error(), "could not exchange creds to iam token:")
}

func TestGetToken_Singleflight_DedupesConcurrentSameKey(t *testing.T) {
	t.Parallel()
	clk := clocktesting.NewFakeClock(time.Unix(0, 0))
	ex := &iam.FakeTokenExchanger{}
	svc, err := NewCachedTokenGetter(10, ex, clk)
	trequire.NoError(t, err)

	creds := buildSubjectCredsJSON(t, "priv-A", "kid-A", "sa-A")

	const n = 50
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(n)

	tokens := make([]string, n)
	errs := make([]error, n)

	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			<-start
			tok, err := svc.GetToken(context.Background(), serviceAccountCredsTokenRequest("api.example", creds), nil)
			tokens[i] = tok
			errs[i] = err
		}()
	}

	close(start)
	wg.Wait()

	for i := 0; i < n; i++ {
		tassert.NoError(t, errs[i])
		tassert.Equal(t, tokens[0], tokens[i])
	}
	tassert.Equal(t, int64(1), ex.Calls.Load())
}

func TestGetToken_ConcurrentDifferentKeys_NoRaceAndWorks(t *testing.T) {
	t.Parallel()
	clk := clocktesting.NewFakeClock(time.Unix(0, 0))
	ex := &iam.FakeTokenExchanger{}
	svc, err := NewCachedTokenGetter(2, ex, clk)
	trequire.NoError(t, err)

	creds := buildSubjectCredsJSON(t, "priv-A", "kid-A", "sa-A")

	const n = 50
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			<-start
			domain := "api." + strconv.Itoa(i%5)
			_, err := svc.GetToken(context.Background(), serviceAccountCredsTokenRequest(domain, creds), nil)
			tassert.NoError(t, err)
		}()
	}

	close(start)
	wg.Wait()

	tassert.GreaterOrEqual(t, ex.Calls.Load(), int64(1)) // lru cache is small, no guarantees
}

func TestGetToken_FederatedServiceAccountCacheKeyIgnoresRawJWT(t *testing.T) {
	t.Parallel()
	env := newTokenTestEnv(t)

	reqA := federatedServiceAccountTokenRequest("api.example", "jwt-1", "ns-a", "sa-a", "aud-a")
	reqB := federatedServiceAccountTokenRequest("api.example", "jwt-2", "ns-a", "sa-a", "aud-a")

	token1, err := env.cachedTokenGetter.GetToken(env.ctx, reqA, nil)
	tassert.NoError(t, err)
	token2, err := env.cachedTokenGetter.GetToken(env.ctx, reqB, nil)
	tassert.NoError(t, err)

	tassert.Equal(t, token1, token2)
	tassert.Equal(t, int64(1), env.fakeTokenExchanger.Calls.Load())
}

func TestGetToken_FederatedServiceAccountCacheKeySeparatesLogicalSources(t *testing.T) {
	t.Parallel()
	env := newTokenTestEnv(t)

	reqA := federatedServiceAccountTokenRequest("api.example", "jwt-1", "ns-a", "sa-a", "aud-a")
	reqDifferentNamespace := federatedServiceAccountTokenRequest("api.example", "jwt-1", "ns-b", "sa-a", "aud-a")
	reqDifferentName := federatedServiceAccountTokenRequest("api.example", "jwt-1", "ns-a", "sa-b", "aud-a")
	reqDifferentAudience := federatedServiceAccountTokenRequest("api.example", "jwt-1", "ns-a", "sa-a", "aud-b")

	tokenA, err := env.cachedTokenGetter.GetToken(env.ctx, reqA, nil)
	tassert.NoError(t, err)
	tokenNS, err := env.cachedTokenGetter.GetToken(env.ctx, reqDifferentNamespace, nil)
	tassert.NoError(t, err)
	tokenName, err := env.cachedTokenGetter.GetToken(env.ctx, reqDifferentName, nil)
	tassert.NoError(t, err)
	tokenAud, err := env.cachedTokenGetter.GetToken(env.ctx, reqDifferentAudience, nil)
	tassert.NoError(t, err)

	tassert.NotEqual(t, tokenA, tokenNS)
	tassert.NotEqual(t, tokenA, tokenName)
	tassert.NotEqual(t, tokenA, tokenAud)
	tassert.Equal(t, int64(4), env.fakeTokenExchanger.Calls.Load())
}
