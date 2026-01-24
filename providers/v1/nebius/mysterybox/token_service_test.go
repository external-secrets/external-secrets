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
	TokenCacheService  *TokenCacheService
}

func newTokenTestEnv(t *testing.T) *tokenTestEnv {
	t.Helper()
	clk := clocktesting.NewFakeClock(time.Unix(0, 0))
	ex := &iam.FakeTokenExchanger{}
	svc, err := NewTokenCacheService(10, ex, clk)
	trequire.NoError(t, err)
	return &tokenTestEnv{ctx: context.Background(), clk: clk, fakeTokenExchanger: ex, TokenCacheService: svc}
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

func TestGetToken_CachesUntilTenPercentLeft(t *testing.T) {
	env := newTokenTestEnv(t)

	ctx := env.ctx
	creds := buildSubjectCredsJSON(t, "priv-A", "kid-A", "sa-A")

	token1, err := env.TokenCacheService.GetToken(ctx, "api.example", creds, nil)
	tassert.NoError(t, err)
	tassert.Equal(t, "token-1", token1)
	tassert.Equal(t, 1, env.fakeTokenExchanger.Calls)

	// add 5 seconds (remaining > 10%)
	addSecondsToClock(env.clk, 5)
	token2, err := env.TokenCacheService.GetToken(ctx, "api.example", creds, nil)
	tassert.NoError(t, err)
	tassert.Equal(t, token1, token2)
	tassert.Equal(t, 1, env.fakeTokenExchanger.Calls)

	// after >90% elapsed -> should refresh
	addSecondsToClock(env.clk, 91) // total 96s
	token3, err := env.TokenCacheService.GetToken(ctx, "api.example", creds, nil)
	tassert.NoError(t, err)
	tassert.NotEqual(t, token1, token3)
	tassert.Equal(t, 2, env.fakeTokenExchanger.Calls)
}

func TestGetToken_SeparateCacheEntriesPerSubjectCreds(t *testing.T) {
	env := newTokenTestEnv(t)

	ctx := env.ctx
	credsA := buildSubjectCredsJSON(t, "priv-A", "kid-A", "sa-A")
	credsB := buildSubjectCredsJSON(t, "priv-B", "kid-B", "sa-B")

	tokenA1, err := env.TokenCacheService.GetToken(ctx, "api.example", credsA, nil)
	tassert.NoError(t, err)
	tassert.Equal(t, "token-1", tokenA1)

	tokenB1, err := env.TokenCacheService.GetToken(ctx, "api.example", credsB, nil)
	tassert.NoError(t, err)
	tassert.Equal(t, "token-2", tokenB1)

	tassert.Equal(t, 2, env.fakeTokenExchanger.Calls)

	// check token cached
	addSecondsToClock(env.clk, 1)
	tokA2, err := env.TokenCacheService.GetToken(ctx, "api.example", credsA, nil)
	tassert.NoError(t, err)
	tassert.Equal(t, tokenA1, tokA2)
	tassert.Equal(t, 2, env.fakeTokenExchanger.Calls)
}

func TestGetToken_InvalidSubjectCreds_ReturnsError(t *testing.T) {
	env := newTokenTestEnv(t)

	_, err := env.TokenCacheService.GetToken(env.ctx, "api.example", "not a json", nil)
	tassert.Error(t, err)
}

func addSecondsToClock(clk *clocktesting.FakeClock, second time.Duration) {
	clk.SetTime(clk.Now().Add(second * time.Second))
}

func TestGetToken_SeparateCacheEntriesPerApiDomain(t *testing.T) {
	env := newTokenTestEnv(t)
	ctx := env.ctx
	creds := buildSubjectCredsJSON(t, "priv-A", "kid-A", "sa-A")

	tokA1, err := env.TokenCacheService.GetToken(ctx, "api.one", creds, nil)
	tassert.NoError(t, err)
	tassert.Equal(t, "token-1", tokA1)

	tokB1, err := env.TokenCacheService.GetToken(ctx, "api.two", creds, nil)
	tassert.NoError(t, err)
	tassert.Equal(t, "token-2", tokB1)
	tassert.NotEqual(t, tokA1, tokB1)
	tassert.Equal(t, 2, env.fakeTokenExchanger.Calls)

	tokA2, err := env.TokenCacheService.GetToken(ctx, "api.one", creds, nil)
	tassert.NoError(t, err)
	tassert.Equal(t, tokA1, tokA2)
	tassert.Equal(t, 2, env.fakeTokenExchanger.Calls)
}

func TestGetToken_LRUEviction(t *testing.T) {
	clk := clocktesting.NewFakeClock(time.Unix(0, 0))
	ex := &iam.FakeTokenExchanger{}
	svc, err := NewTokenCacheService(2, ex, clk)
	tassert.NoError(t, err)
	ctx := context.Background()
	creds := buildSubjectCredsJSON(t, "priv-A", "kid-A", "sa-A")

	tok1, err := svc.GetToken(ctx, "api.first", creds, nil)
	tassert.NoError(t, err)
	tassert.Equal(t, "token-1", tok1)

	tok2, err := svc.GetToken(ctx, "api.second", creds, nil)
	tassert.NoError(t, err)
	tassert.Equal(t, "token-2", tok2)
	tassert.Equal(t, 2, ex.Calls)

	tok1again, err := svc.GetToken(ctx, "api.first", creds, nil)
	tassert.NoError(t, err)
	tassert.Equal(t, tok1, tok1again)
	tassert.Equal(t, 2, ex.Calls)

	tok3, err := svc.GetToken(ctx, "api.third", creds, nil)
	tassert.NoError(t, err)
	tassert.Equal(t, "token-3", tok3)
	tassert.Equal(t, 3, ex.Calls)

	secondAgain, err := svc.GetToken(ctx, "api.second", creds, nil)
	tassert.NoError(t, err)
	tassert.Equal(t, "token-4", secondAgain)
	tassert.Equal(t, 4, ex.Calls)
}

func TestGetToken_AfterExpiration_Refreshes(t *testing.T) {
	env := newTokenTestEnv(t)
	ctx := env.ctx
	creds := buildSubjectCredsJSON(t, "priv-A", "kid-A", "sa-A")

	_, _ = env.TokenCacheService.GetToken(ctx, "api.example", creds, nil)
	addSecondsToClock(env.clk, 101)

	tok2, err := env.TokenCacheService.GetToken(ctx, "api.example", creds, nil)
	tassert.NoError(t, err)
	tassert.Equal(t, 2, env.fakeTokenExchanger.Calls)
	tassert.Equal(t, "token-2", tok2)
}

func TestGetToken_CacheKeyChangesOnKeyRotation(t *testing.T) {
	env := newTokenTestEnv(t)
	ctx := env.ctx

	base := buildSubjectCredsJSON(t, "priv-A", "kid-A", "sa-A")
	rotatedKeyID := buildSubjectCredsJSON(t, "priv-A", "kid-B", "sa-A")
	rotatedPriv := buildSubjectCredsJSON(t, "priv-B", "kid-A", "sa-A")
	rotatedSubject := buildSubjectCredsJSON(t, "priv-A", "kid-A", "sa-B")

	t1, _ := env.TokenCacheService.GetToken(ctx, "api", base, nil)
	t2, _ := env.TokenCacheService.GetToken(ctx, "api", rotatedKeyID, nil)
	t3, _ := env.TokenCacheService.GetToken(ctx, "api", rotatedPriv, nil)
	t4, _ := env.TokenCacheService.GetToken(ctx, "api", rotatedSubject, nil)

	tassert.NotEqual(t, t1, t2)
	tassert.NotEqual(t, t1, t3)
	tassert.NotEqual(t, t1, t4)
	tassert.Equal(t, 4, env.fakeTokenExchanger.Calls)
}

func TestGetToken_ExchangerErrorIsWrapped(t *testing.T) {
	clk := clocktesting.NewFakeClock(time.Unix(0, 0))
	svc, err := NewTokenCacheService(10, &iam.FakeTokenExchanger{ReturnError: true}, clk)
	trequire.NoError(t, err)

	_, err = svc.GetToken(context.Background(), "api", buildSubjectCredsJSON(t, "p", "k", "s"), nil)
	tassert.Error(t, err)
	tassert.Contains(t, err.Error(), "could not exchange creds to iam token:")
}
