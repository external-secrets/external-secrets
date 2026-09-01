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
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tassert "github.com/stretchr/testify/assert"
	trequire "github.com/stretchr/testify/require"
	clocktesting "k8s.io/utils/clock/testing"

	"github.com/external-secrets/external-secrets/providers/v1/nebius/common/auth"
	"github.com/external-secrets/external-secrets/providers/v1/nebius/common/sdk/iam"
)

func dummyResolveFunc(_ context.Context) (auth.TokenExchangeCredentials, error) {
	return &auth.ResolvedServiceAccountCreds{}, nil
}

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

func TestGetToken_CachesUntilTenPercentLeft(t *testing.T) {
	t.Parallel()
	env := newTokenTestEnv(t)

	var calls atomic.Int64
	resolveFunc := func(_ context.Context) (auth.TokenExchangeCredentials, error) {
		calls.Add(1)
		return &auth.ResolvedServiceAccountCreds{}, nil
	}

	ctx := env.ctx
	creds := auth.NewCredentialRequest("key", resolveFunc)

	token1, err := env.cachedTokenGetter.GetToken(ctx, "api.example", creds, nil)
	tassert.NoError(t, err)
	tassert.Equal(t, "token-1", token1)
	tassert.Equal(t, int64(1), env.fakeTokenExchanger.Calls.Load())
	tassert.Equal(t, int64(1), calls.Load())

	// add 5 seconds (remaining > 10%)
	addSecondsToClock(env.clk, 5)
	token2, err := env.cachedTokenGetter.GetToken(ctx, "api.example", creds, nil)
	tassert.NoError(t, err)
	tassert.Equal(t, token1, token2)
	tassert.Equal(t, int64(1), env.fakeTokenExchanger.Calls.Load())
	tassert.Equal(t, int64(1), calls.Load())

	// after >90% elapsed -> should refresh
	addSecondsToClock(env.clk, 91) // total 96s
	token3, err := env.cachedTokenGetter.GetToken(ctx, "api.example", creds, nil)
	tassert.NoError(t, err)
	tassert.NotEqual(t, token1, token3)
	tassert.Equal(t, int64(2), env.fakeTokenExchanger.Calls.Load())

	tassert.Equal(t, int64(2), calls.Load())
}

func TestGetToken_SeparateCacheEntriesPerKey(t *testing.T) {
	t.Parallel()
	env := newTokenTestEnv(t)

	var callsA atomic.Int64
	var callsB atomic.Int64

	ctx := env.ctx
	credsA := auth.NewCredentialRequest("key-a", func(ctx context.Context) (auth.TokenExchangeCredentials, error) {
		callsA.Add(1)
		return &auth.ResolvedServiceAccountCreds{}, nil
	})
	credsB := auth.NewCredentialRequest("key-b", func(ctx context.Context) (auth.TokenExchangeCredentials, error) {
		callsB.Add(1)
		return &auth.ResolvedServiceAccountCreds{}, nil
	})

	tokenA1, err := env.cachedTokenGetter.GetToken(ctx, "api.example", credsA, nil)
	tassert.NoError(t, err)
	tassert.Equal(t, "token-1", tokenA1)
	tassert.Equal(t, int64(1), callsA.Load())
	tassert.Equal(t, int64(0), callsB.Load())

	tokenB1, err := env.cachedTokenGetter.GetToken(ctx, "api.example", credsB, nil)
	tassert.NoError(t, err)
	tassert.Equal(t, "token-2", tokenB1)
	tassert.Equal(t, int64(1), callsA.Load())
	tassert.Equal(t, int64(1), callsB.Load())

	tassert.Equal(t, int64(2), env.fakeTokenExchanger.Calls.Load())

	// check token cached
	addSecondsToClock(env.clk, 1)
	tokA2, err := env.cachedTokenGetter.GetToken(ctx, "api.example", credsA, nil)
	tassert.NoError(t, err)
	tassert.Equal(t, tokenA1, tokA2)
	tassert.Equal(t, int64(2), env.fakeTokenExchanger.Calls.Load())
	tassert.Equal(t, int64(1), callsA.Load())
	tassert.Equal(t, int64(1), callsB.Load())
}

func addSecondsToClock(clk *clocktesting.FakeClock, second time.Duration) {
	clk.SetTime(clk.Now().Add(second * time.Second))
}

func TestGetToken_LRUEviction(t *testing.T) {
	t.Parallel()
	clk := clocktesting.NewFakeClock(time.Unix(0, 0))
	ex := &iam.FakeTokenExchanger{}
	svc, err := NewCachedTokenGetter(2, ex, clk)
	tassert.NoError(t, err)
	ctx := context.Background()
	creds1 := auth.NewCredentialRequest("key-1", dummyResolveFunc)
	creds2 := auth.NewCredentialRequest("key-2", dummyResolveFunc)
	creds3 := auth.NewCredentialRequest("key-3", dummyResolveFunc)

	tok1, err := svc.GetToken(ctx, "api.example", creds1, nil)
	tassert.NoError(t, err)
	tassert.Equal(t, "token-1", tok1)

	tok2, err := svc.GetToken(ctx, "api.example", creds2, nil)
	tassert.NoError(t, err)
	tassert.Equal(t, "token-2", tok2)
	tassert.Equal(t, int64(2), ex.Calls.Load())

	tok1again, err := svc.GetToken(ctx, "api.example", creds1, nil)
	tassert.NoError(t, err)
	tassert.Equal(t, tok1, tok1again)
	tassert.Equal(t, int64(2), ex.Calls.Load())

	tok3, err := svc.GetToken(ctx, "api.example", creds3, nil)
	tassert.NoError(t, err)
	tassert.Equal(t, "token-3", tok3)
	tassert.Equal(t, int64(3), ex.Calls.Load())

	secondAgain, err := svc.GetToken(ctx, "api.example", creds2, nil)
	tassert.NoError(t, err)
	tassert.Equal(t, "token-4", secondAgain)
	tassert.Equal(t, int64(4), ex.Calls.Load())
}

func TestGetToken_AfterExpiration_Refreshes(t *testing.T) {
	t.Parallel()
	env := newTokenTestEnv(t)
	ctx := env.ctx
	var resolveCalls atomic.Int64
	creds := auth.NewCredentialRequest("key", func(ctx context.Context) (auth.TokenExchangeCredentials, error) {
		resolveCalls.Add(1)
		return &auth.ResolvedServiceAccountCreds{}, nil
	})
	_, err := env.cachedTokenGetter.GetToken(ctx, "api.example", creds, nil)
	tassert.NoError(t, err)
	addSecondsToClock(env.clk, 101)
	tassert.Equal(t, int64(1), resolveCalls.Load())

	tok2, err := env.cachedTokenGetter.GetToken(ctx, "api.example", creds, nil)
	tassert.NoError(t, err)
	tassert.Equal(t, int64(2), env.fakeTokenExchanger.Calls.Load())
	tassert.Equal(t, int64(2), resolveCalls.Load())
	tassert.Equal(t, "token-2", tok2)
}

func TestGetToken_ExchangerErrorIsWrapped(t *testing.T) {
	t.Parallel()
	clk := clocktesting.NewFakeClock(time.Unix(0, 0))
	svc, err := NewCachedTokenGetter(10, &iam.FakeTokenExchanger{ReturnError: true}, clk)
	trequire.NoError(t, err)

	var resolveCalls atomic.Int64
	creds := auth.NewCredentialRequest("key", func(ctx context.Context) (auth.TokenExchangeCredentials, error) {
		resolveCalls.Add(1)
		return &auth.ResolvedServiceAccountCreds{}, nil
	})
	_, err = svc.GetToken(context.Background(), "api", creds, nil)
	tassert.Error(t, err)
	tassert.Equal(t, int64(1), resolveCalls.Load())
	tassert.Contains(t, err.Error(), "could not exchange creds to iam token:")
}

func TestGetToken_ResolveErrorIsNotCached(t *testing.T) {
	t.Parallel()
	env := newTokenTestEnv(t)

	resolveErr := errors.New("resolve credentials")
	var resolveCalls atomic.Int64
	credentialsRequest := auth.NewCredentialRequest("key", func(_ context.Context) (auth.TokenExchangeCredentials, error) {
		if resolveCalls.Add(1) == 1 {
			return nil, resolveErr
		}
		return &auth.ResolvedServiceAccountCreds{}, nil
	})

	_, err := env.cachedTokenGetter.GetToken(env.ctx, "api.example", credentialsRequest, nil)
	trequire.ErrorIs(t, err, resolveErr)
	tassert.Equal(t, int64(1), resolveCalls.Load())
	tassert.Equal(t, int64(0), env.fakeTokenExchanger.Calls.Load())

	token, err := env.cachedTokenGetter.GetToken(env.ctx, "api.example", credentialsRequest, nil)
	trequire.NoError(t, err)
	tassert.Equal(t, "token-1", token)
	tassert.Equal(t, int64(2), resolveCalls.Load())
	tassert.Equal(t, int64(1), env.fakeTokenExchanger.Calls.Load())
}

func TestGetToken_Singleflight_DedupesConcurrentSameKey(t *testing.T) {
	t.Parallel()
	clk := clocktesting.NewFakeClock(time.Unix(0, 0))
	ex := &iam.FakeTokenExchanger{}
	svc, err := NewCachedTokenGetter(10, ex, clk)
	trequire.NoError(t, err)

	var resolveCalls atomic.Int64
	creds := auth.NewCredentialRequest("key", func(ctx context.Context) (auth.TokenExchangeCredentials, error) {
		resolveCalls.Add(1)
		return &auth.ResolvedServiceAccountCreds{}, nil
	})

	const n = 50
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(n)

	tokens := make([]string, n)
	errs := make([]error, n)

	for i := range n {
		go func() {
			defer wg.Done()
			<-start
			tok, err := svc.GetToken(context.Background(), "api.example", creds, nil)
			tokens[i] = tok
			errs[i] = err
		}()
	}

	close(start)
	wg.Wait()

	for i := range n {
		tassert.NoError(t, errs[i])
		tassert.Equal(t, tokens[0], tokens[i])
	}
	tassert.Equal(t, int64(1), ex.Calls.Load())
	tassert.Equal(t, int64(1), resolveCalls.Load())
}

func TestGetToken_ConcurrentDifferentKeys_NoRaceAndWorks(t *testing.T) {
	t.Parallel()
	clk := clocktesting.NewFakeClock(time.Unix(0, 0))
	ex := &iam.FakeTokenExchanger{}
	svc, err := NewCachedTokenGetter(2, ex, clk)
	trequire.NoError(t, err)

	const n = 50
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(n)

	for i := range n {
		go func() {
			defer wg.Done()
			<-start
			creds := auth.NewCredentialRequest("key-"+strconv.Itoa(i%5), dummyResolveFunc)
			_, err := svc.GetToken(context.Background(), "api.example", creds, nil)
			tassert.NoError(t, err)
		}()
	}

	close(start)
	wg.Wait()

	tassert.GreaterOrEqual(t, ex.Calls.Load(), int64(1)) // lru cache is small, no guarantees
}
