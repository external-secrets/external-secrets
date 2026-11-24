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
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/akeylesslabs/akeyless-go/v4"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	fakeakeyless "github.com/external-secrets/external-secrets/providers/v1/akeyless/fake"
	testingfake "github.com/external-secrets/external-secrets/runtime/testing/fake"
)

func newStatusServer(t *testing.T, status int) *akeylessBase {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"error":"canned"}`))
	}))
	t.Cleanup(srv.Close)
	return &akeylessBase{
		RestAPI: akeyless.NewAPIClient(&akeyless.Configuration{
			Servers: []akeyless.ServerConfiguration{{URL: srv.URL}},
		}).V2Api,
	}
}

func writeCalls(base *akeylessBase) map[string]func(context.Context) error {
	return map[string]func(context.Context) error{
		"CreateSecret": func(ctx context.Context) error { return base.CreateSecret(ctx, "/k", "v") },
		"UpdateSecret": func(ctx context.Context) error { return base.UpdateSecret(ctx, "/k", "v") },
		"DeleteSecret": func(ctx context.Context) error { return base.DeleteSecret(ctx, "/k") },
	}
}

func TestWriteCallsMapAPIStatus(t *testing.T) {
	tests := []struct {
		status           int
		wantErr          bool
		wantUnauthorized bool
	}{
		{status: http.StatusOK},
		{status: http.StatusUnauthorized, wantErr: true, wantUnauthorized: true},
		{status: http.StatusForbidden, wantErr: true, wantUnauthorized: true},
		{status: http.StatusInternalServerError, wantErr: true},
	}
	for _, tt := range tests {
		for name, call := range writeCalls(newStatusServer(t, tt.status)) {
			t.Run(name+"/"+http.StatusText(tt.status), func(t *testing.T) {
				err := call(describeItemCtx())
				if !tt.wantErr {
					require.NoError(t, err)
					return
				}
				require.Error(t, err)
				require.Equal(t, tt.wantUnauthorized, errors.Is(err, ErrUnauthorized))
			})
		}
	}
}

func TestWriteCallsWithoutResponseDoNotPanic(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()
	base := &akeylessBase{
		RestAPI: akeyless.NewAPIClient(&akeyless.Configuration{
			Servers: []akeyless.ServerConfiguration{{URL: srv.URL}},
		}).V2Api,
	}
	for name, call := range writeCalls(base) {
		t.Run(name, func(t *testing.T) {
			var err error
			require.NotPanics(t, func() { err = call(describeItemCtx()) })
			require.Error(t, err)
			require.NotErrorIs(t, err, ErrUnauthorized)
		})
	}
}

// numberedTokenClient issues t-1, t-2, ... and otherwise behaves like the mock.
type numberedTokenClient struct {
	*fakeakeyless.AkeylessMockClient
	authCalls atomic.Int32
}

func (c *numberedTokenClient) TokenFromSecretRef(_ context.Context) (string, error) {
	return fmt.Sprintf("t-%d", c.authCalls.Add(1)), nil
}

func TestPushSecretRetriesOnceWhenCachedTokenRejectedOnWrite(t *testing.T) {
	var seen []string
	mock := &numberedTokenClient{
		AkeylessMockClient: fakeakeyless.New().
			SetGetSecretFn(func(_ string, _ int32) (string, error) { return "", ErrItemNotExists }).
			SetCreateSecretFn(func(ctx context.Context, _, _ string) error {
				token, _ := ctx.Value(aKeylessToken).(string)
				seen = append(seen, token)
				if token == "t-cached" {
					return &unauthorizedError{err: errors.New("token expired")}
				}
				return nil
			}),
	}
	client := &Akeyless{Client: mock, cachedToken: "t-cached", tokenExpiry: time.Now().Add(time.Hour)}

	err := client.PushSecret(context.Background(),
		&corev1.Secret{Data: map[string][]byte{"k": []byte("v")}},
		&testingfake.PushSecretData{RemoteKey: "/k"})

	require.NoError(t, err)
	require.Equal(t, []string{"t-cached", "t-1"}, seen)
	require.Equal(t, "t-1", client.cachedToken)
	require.Equal(t, int32(1), mock.authCalls.Load())
}
