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

package dvls

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Devolutions/go-dvls"
	"github.com/spf13/pflag"
	"golang.org/x/sync/singleflight"

	"github.com/external-secrets/external-secrets/runtime/cache"
	"github.com/external-secrets/external-secrets/runtime/feature"
)

const defaultCacheSize = 1 << 10

var (
	enableCache bool
	clientCache *cache.Cache[*sharedClient]
	// logins makes concurrent cache misses for the same credentials wait for a single login.
	logins singleflight.Group
	// loginTimeout bounds a shared login, so a server that never answers does not hold every store waiting on it.
	loginTimeout = 30 * time.Second
	// callTimeout bounds each call on a cached client, so a request that never gets an answer releases the lock.
	callTimeout = time.Minute
	// keySecret changes with each process, so a cache key cannot confirm a guessed secret.
	keySecret = []byte(rand.Text())
)

// callLock serializes calls on a go-dvls client. Unlike a mutex, waiting for it stops when the context is done.
type callLock chan struct{}

func newCallLock() callLock {
	return make(callLock, 1)
}

func (l callLock) acquire(ctx context.Context) (func(), error) {
	if l == nil {
		return func() {
			// A nil lock has nothing to release.
		}, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("waiting for the DVLS client: %w", err)
	}
	select {
	case l <- struct{}{}:
		return func() { <-l }, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("waiting for the DVLS client: %w", ctx.Err())
	}
}

// sharedClient is a go-dvls client that may serve several stores and reconciliations.
// go-dvls renews its token on demand without guarding it, so calls are serialized.
type sharedClient struct {
	lock   callLock
	client *dvls.Client
	// cached is true when the client outlives a reconciliation, so its session can expire mid-use.
	cached bool
}

// checkSession reports whether DVLS answers and the session is valid or can be renewed:
// go-dvls checks the session, and logs in again if it expired, before any request.
func (s *sharedClient) checkSession(ctx context.Context) error {
	_, err := run(ctx, s.lock, s.cached, func(callCtx context.Context) (dvls.Server, error) {
		return s.client.GetPublicServerInfoWithContext(callCtx)
	})
	return err
}

// run serializes call on a shared client. On a cached client, the call runs under callTimeout.
func run[T any](ctx context.Context, lock callLock, cached bool, call func(context.Context) (T, error)) (T, error) {
	unlock, err := lock.acquire(ctx)
	if err != nil {
		var zero T
		return zero, err
	}
	defer unlock()
	if !cached {
		return call(ctx)
	}
	callCtx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	return call(callCtx)
}

// runRead is run for reads: on a cached client, a read rejected with 401 runs once more.
func runRead[T any](ctx context.Context, lock callLock, cached bool, call func(context.Context) (T, error)) (T, error) {
	return run(ctx, lock, cached, func(callCtx context.Context) (T, error) {
		return retryUnauthorized(cached, func() (T, error) {
			return call(callCtx)
		})
	})
}

// retryUnauthorized runs call a second time when DVLS answers 401. go-dvls checks the
// session before each request, so a cached session can expire between the check and
// the request. The second attempt checks the session again and logs in if it is gone.
func retryUnauthorized[T any](enabled bool, call func() (T, error)) (T, error) {
	v, err := call()
	if !enabled || !isUnauthorized(err) {
		return v, err
	}
	return call()
}

func isUnauthorized(err error) bool {
	var reqErr *dvls.RequestError
	return errors.As(err, &reqErr) && reqErr.StatusCode == http.StatusUnauthorized
}

// credentialsKey identifies a DVLS session. Stores that use the same server and
// application credentials share one client, and a rotated secret yields a new key.
func credentialsKey(serverURL, appID, appSecret string) cache.Key {
	mac := hmac.New(sha256.New, keySecret)
	mac.Write([]byte(serverURL + "\x00" + appID + "\x00" + appSecret))
	return cache.Key{Name: hex.EncodeToString(mac.Sum(nil))}
}

// getClient returns the cached client for the credentials, or logs in and caches a new one.
func getClient(ctx context.Context, serverURL, appID, appSecret string) (*sharedClient, error) {
	clients := clientCache
	if !enableCache || clients == nil {
		return newSharedClient(ctx, serverURL, appID, appSecret, false)
	}

	key := credentialsKey(serverURL, appID, appSecret)
	if c, ok := clients.Get("", key); ok {
		return c, nil
	}

	login := logins.DoChan(key.Name, func() (any, error) {
		if c, ok := clients.Get("", key); ok {
			return c, nil
		}
		// Every waiter shares this login, so it gets its own deadline instead of the first caller's context.
		loginCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), loginTimeout)
		defer cancel()
		c, err := newSharedClient(loginCtx, serverURL, appID, appSecret, true)
		if err != nil {
			return nil, err
		}
		clients.Add("", key, c)
		return c, nil
	})

	select {
	case res := <-login:
		if res.Err != nil {
			return nil, res.Err
		}
		c, ok := res.Val.(*sharedClient)
		if !ok {
			return nil, fmt.Errorf("unexpected DVLS client type %T", res.Val)
		}
		return c, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("waiting for the DVLS login: %w", ctx.Err())
	}
}

func newSharedClient(ctx context.Context, serverURL, appID, appSecret string, cached bool) (*sharedClient, error) {
	client, err := dvls.NewClientWithContext(ctx, appID, appSecret, serverURL)
	if err != nil {
		return nil, err
	}
	return &sharedClient{lock: newCallLock(), client: &client, cached: cached}, nil
}

func initCache(size int) {
	// go-dvls has no logout, so an evicted session simply expires on the server.
	clientCache = cache.Must[*sharedClient](size, nil)
}

type cacheFlags struct {
	enable bool
	size   int
}

// apply leaves the cache unallocated when it is disabled, so the size is only read when it is used.
func (f *cacheFlags) apply() {
	// Initialize cannot return an error, and the controller must not start with an invalid size.
	if f.enable && f.size <= 0 {
		panic(fmt.Sprintf("--dvls-client-cache-size must be greater than 0, got %d", f.size))
	}
	enableCache = f.enable
	if f.enable {
		initCache(f.size)
	}
}

func registerCacheFlags(errorHandling pflag.ErrorHandling) (*pflag.FlagSet, *cacheFlags) {
	flags := &cacheFlags{}
	fs := pflag.NewFlagSet("dvls", errorHandling)
	fs.BoolVar(
		&flags.enable,
		"enable-dvls-client-cache",
		false,
		"Enable DVLS client cache. Stores that share a server and application credentials reuse one DVLS session instead of logging in on each reconciliation.",
	)
	fs.IntVar(
		&flags.size,
		"dvls-client-cache-size",
		defaultCacheSize,
		"Maximum size of the DVLS client cache. Only used if --enable-dvls-client-cache is set.",
	)
	return fs, flags
}

func init() {
	fs, flags := registerCacheFlags(pflag.ExitOnError)
	feature.Register(feature.Feature{
		Flags:      fs,
		Initialize: flags.apply,
	})
}
