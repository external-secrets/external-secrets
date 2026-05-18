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

package truefoundry

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

const (
	tToken = "tfy-cluster-token-xxxx"
	tFQN   = "truefoundry:test-eso:test"
)

func newTestClient(serverURL string) *Client {
	return newClient(serverURL, tToken, &http.Client{Timeout: 5 * time.Second})
}

// --- doRequest primitives ---------------------------------------------------

func TestDoRequest_AuthHeadersOnEveryAttempt(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		require.Equal(t, "Bearer "+tToken, r.Header.Get("Authorization"))
		require.Equal(t, "application/json", r.Header.Get("Accept"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"value":"ok"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, status, err := c.doRequest(context.Background(), "test", srv.URL+"/x")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, int32(1), atomic.LoadInt32(&calls))
}

func TestDoRequest_RetryOn5xxThenSuccess(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			http.Error(w, "boom", http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"value":"ok"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	body, status, err := c.doRequest(context.Background(), "test", srv.URL+"/x")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, int32(3), atomic.LoadInt32(&calls))
	require.Contains(t, string(body), "ok")
}

func TestDoRequest_RetryExhausted(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		http.Error(w, "still down", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, status, err := c.doRequest(context.Background(), "test", srv.URL+"/x")
	require.Error(t, err)
	require.Equal(t, http.StatusInternalServerError, status)
	require.Equal(t, int32(maxRetryAttempts), atomic.LoadInt32(&calls))
	require.Contains(t, err.Error(), "server error 500")
}

func TestDoRequest_NonRetryable4xxFailsFast(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, status, err := c.doRequest(context.Background(), "test", srv.URL+"/x")
	require.Error(t, err)
	require.Equal(t, http.StatusUnauthorized, status)
	require.Equal(t, int32(1), atomic.LoadInt32(&calls), "401 must not retry")
	require.Contains(t, err.Error(), "status 401")
}

func TestDoRequest_RetryAfter429(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			w.Header().Set("Retry-After", "1")
			http.Error(w, "slow down", http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"value":"ok"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	start := time.Now()
	_, status, err := c.doRequest(context.Background(), "test", srv.URL+"/x")
	elapsed := time.Since(start)

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, int32(2), atomic.LoadInt32(&calls))
	require.GreaterOrEqual(t, elapsed, 900*time.Millisecond, "Retry-After must be honored")
}

func TestDoRequest_TransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	srv.Close()

	c := newTestClient(srv.URL)
	_, _, err := c.doRequest(context.Background(), "test", srv.URL+"/x")
	require.Error(t, err)
}

func TestDoRequest_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := c.doRequest(ctx, "test", srv.URL+"/x")
	require.Error(t, err)
}

func TestParseRetryAfter(t *testing.T) {
	require.Equal(t, time.Duration(0), parseRetryAfter(""))
	require.Equal(t, time.Duration(0), parseRetryAfter("garbage"))
	require.Equal(t, 5*time.Second, parseRetryAfter("5"))
	require.Equal(t, time.Duration(0), parseRetryAfter("Mon, 01 Jan 2001 00:00:00 GMT"))
}

// --- fetchSecret (single endpoint) ------------------------------------------

func TestFetchSecret_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, apiPath, r.URL.Path)
		require.Equal(t, secretRefScheme+tFQN, r.URL.Query().Get("secret_ref"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"value":"hunter2"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	val, err := c.fetchSecret(context.Background(), tFQN)
	require.NoError(t, err)
	require.Equal(t, []byte("hunter2"), val)
}

func TestFetchSecret_RefIsURLEncoded(t *testing.T) {
	var capturedQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"value":"x"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.fetchSecret(context.Background(), "truefoundry:test-eso:test")
	require.NoError(t, err)
	// tfy-secret://truefoundry:test-eso:test → URL-encoded
	require.Contains(t, capturedQuery, "secret_ref=tfy-secret%3A%2F%2Ftruefoundry%3Atest-eso%3Atest")
}

func TestFetchSecret_404IsNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "gone", http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.fetchSecret(context.Background(), tFQN)
	require.ErrorIs(t, err, esv1.NoSecretErr)
}

func TestFetchSecret_401Errors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"err":"bad token"}`, http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.fetchSecret(context.Background(), tFQN)
	require.Error(t, err)
	require.False(t, errors.Is(err, esv1.NoSecretErr))
	require.Contains(t, err.Error(), "status 401")
}

func TestFetchSecret_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<not json>`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.fetchSecret(context.Background(), tFQN)
	require.Error(t, err)
	require.Contains(t, err.Error(), "decode secret response")
	require.Contains(t, err.Error(), tFQN)
}

func TestFetchSecret_EmptyFQNRejected(t *testing.T) {
	c := newTestClient("http://unused")
	_, err := c.fetchSecret(context.Background(), "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "empty secret reference")
}

// --- GetSecret / GetSecretMap (SecretsClient interface) ---------------------

func TestGetSecret_PassesKeyVerbatim(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, secretRefScheme+tFQN, r.URL.Query().Get("secret_ref"))
		_, _ = w.Write([]byte(`{"value":"v"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	val, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: tFQN})
	require.NoError(t, err)
	require.Equal(t, []byte("v"), val)
}

func TestGetSecret_EmptyKeyRejected(t *testing.T) {
	c := newTestClient("http://unused")
	_, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: ""})
	require.Error(t, err)
	require.Contains(t, err.Error(), "remoteRef.key is required")
}

func TestGetSecret_PropertySelector(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"value":"{\"nested\":{\"x\":\"yes\"}}"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	val, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: tFQN, Property: "nested.x"})
	require.NoError(t, err)
	require.Equal(t, []byte("yes"), val)
}

func TestGetSecret_PropertyMissingErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"value":"{\"x\":\"yes\"}"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: tFQN, Property: "missing.path"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "property")
}

func TestGetSecret_404Propagates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "gone", http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: tFQN})
	require.ErrorIs(t, err, esv1.NoSecretErr)
}

func TestGetSecretMap_JSONObject(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// value is a JSON-encoded object as a string
		_, _ = w.Write([]byte(`{"value":"{\"a\":\"x\",\"b\":42}"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	m, err := c.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: tFQN})
	require.NoError(t, err)
	require.Equal(t, []byte("x"), m["a"])
	require.Equal(t, []byte("42"), m["b"])
}

func TestGetSecretMap_NonObjectErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"value":"just a string"}`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	_, err := c.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: tFQN})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unable to unmarshal")
}

// --- not-supported / unimplemented ------------------------------------------

func TestGetAllSecrets_NotSupported(t *testing.T) {
	c := newTestClient("http://unused")
	_, err := c.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not supported")
	require.Contains(t, err.Error(), "FQN")
}

func TestWriteOpsNotImplemented(t *testing.T) {
	c := newTestClient("http://unused")
	require.ErrorIs(t, c.PushSecret(context.Background(), nil, nil), errNotImplemented)
	require.ErrorIs(t, c.DeleteSecret(context.Background(), nil), errNotImplemented)
	ok, err := c.SecretExists(context.Background(), nil)
	require.ErrorIs(t, err, errNotImplemented)
	require.False(t, ok)
}

func TestClose(t *testing.T) {
	c := newTestClient("http://unused")
	require.NoError(t, c.Close(context.Background()))
}

func TestValidate(t *testing.T) {
	t.Run("ready when token accepted (2xx)", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"value":"x"}`))
		}))
		defer srv.Close()
		c := newTestClient(srv.URL)
		r, err := c.Validate()
		require.NoError(t, err)
		require.Equal(t, esv1.ValidationResultReady, r)
	})

	t.Run("ready on 404 (token good, sentinel absent)", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "not found", http.StatusNotFound)
		}))
		defer srv.Close()
		c := newTestClient(srv.URL)
		r, err := c.Validate()
		require.NoError(t, err)
		require.Equal(t, esv1.ValidationResultReady, r)
	})

	t.Run("error on 401", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "no", http.StatusUnauthorized)
		}))
		defer srv.Close()
		c := newTestClient(srv.URL)
		r, err := c.Validate()
		require.Error(t, err)
		require.Equal(t, esv1.ValidationResultError, r)
		require.Contains(t, err.Error(), "rejected")
	})

	t.Run("error on 403", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "no", http.StatusForbidden)
		}))
		defer srv.Close()
		c := newTestClient(srv.URL)
		r, err := c.Validate()
		require.Error(t, err)
		require.Equal(t, esv1.ValidationResultError, r)
	})

	t.Run("unknown on transport error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		srv.Close()
		c := newTestClient(srv.URL)
		r, err := c.Validate()
		require.Error(t, err)
		require.Equal(t, esv1.ValidationResultUnknown, r)
	})

	t.Run("sentinel FQN appears in the request", func(t *testing.T) {
		var captured string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			captured = r.URL.Query().Get("secret_ref")
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()
		c := newTestClient(srv.URL)
		_, _ = c.Validate()
		require.Equal(t, secretRefScheme+validateSentinelFQN, captured)
	})
}
