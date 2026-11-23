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

package esutils

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestCloneDefaultHTTPTransport(t *testing.T) {
	defaultTransport, ok := http.DefaultTransport.(*http.Transport)
	require.True(t, ok)

	tests := []struct {
		name            string
		setup           func(t *testing.T) (restore func())
		wantErr         error
		assertTransport func(t *testing.T, transport *http.Transport)
	}{
		{
			name: "clones default transport",
			assertTransport: func(t *testing.T, transport *http.Transport) {
				require.NotSame(t, defaultTransport, transport)
				require.False(t, transport.ForceAttemptHTTP2)
				if transport.TLSClientConfig != nil {
					require.Empty(t, transport.TLSClientConfig.NextProtos)
				}
				require.Equal(t, defaultTransport.MaxIdleConns, transport.MaxIdleConns)
			},
		},
		{
			name: "custom default transport returns error",
			setup: func(t *testing.T) func() {
				original := http.DefaultTransport
				http.DefaultTransport = roundTripperFunc(func(*http.Request) (*http.Response, error) {
					return nil, errors.New("custom transport")
				})
				return func() {
					http.DefaultTransport = original
				}
			},
			wantErr: ErrUnexpectedDefaultTransport,
			assertTransport: func(t *testing.T, transport *http.Transport) {
				require.Nil(t, transport)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				restore := tt.setup(t)
				t.Cleanup(restore)
			}

			transport, err := CloneDefaultHTTPTransport()
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			if tt.assertTransport != nil {
				tt.assertTransport(t, transport)
			}
		})
	}
}

func TestCloneDefaultHTTPTransportTLSProtocols(t *testing.T) {
	newTLSServer := func(t *testing.T, enableHTTP2 bool) (*httptest.Server, *x509.CertPool) {
		t.Helper()
		srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		srv.EnableHTTP2 = enableHTTP2
		srv.StartTLS()
		t.Cleanup(srv.Close)

		pool := x509.NewCertPool()
		pool.AddCert(srv.Certificate())
		return srv, pool
	}

	setRootCAs := func(tr *http.Transport, pool *x509.CertPool) {
		if tr.TLSClientConfig == nil {
			tr.TLSClientConfig = &tls.Config{RootCAs: pool}
			return
		}
		tr.TLSClientConfig.RootCAs = pool
	}

	doRequest := func(t *testing.T, tr *http.Transport, url string) (int, string) {
		t.Helper()
		t.Cleanup(tr.CloseIdleConnections)
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, http.NoBody)
		require.NoError(t, err)
		resp, err := (&http.Client{Transport: tr}).Do(req)
		require.NoError(t, err)
		defer func() {
			_ = resp.Body.Close()
		}()
		return resp.StatusCode, resp.Proto
	}

	t.Run("speaks HTTP/1.1 against HTTP/2-enabled TLS server without replacing TLSClientConfig", func(t *testing.T) {
		srv, pool := newTLSServer(t, true)

		transport, err := CloneDefaultHTTPTransport()
		require.NoError(t, err)
		setRootCAs(transport, pool)

		statusCode, proto := doRequest(t, transport, srv.URL)
		require.Equal(t, http.StatusOK, statusCode)
		require.Equal(t, "HTTP/1.1", proto)

		// DefaultTransport itself must still advertise h2 for other callers.
		defaultTransport, ok := http.DefaultTransport.(*http.Transport)
		require.True(t, ok)
		require.NotNil(t, defaultTransport.TLSClientConfig)
		require.Contains(t, defaultTransport.TLSClientConfig.NextProtos, "h2")
	})

	t.Run("speaks HTTP/1.1 against HTTP/1.1-only TLS server", func(t *testing.T) {
		srv, pool := newTLSServer(t, false)

		transport, err := CloneDefaultHTTPTransport()
		require.NoError(t, err)
		setRootCAs(transport, pool)

		statusCode, proto := doRequest(t, transport, srv.URL)
		require.Equal(t, http.StatusOK, statusCode)
		require.Equal(t, "HTTP/1.1", proto)
	})

	t.Run("preserves custom TLSClientConfig fields from DefaultTransport while clearing NextProtos", func(t *testing.T) {
		srv, pool := newTLSServer(t, true)

		original := http.DefaultTransport
		origTransport, ok := original.(*http.Transport)
		require.True(t, ok)
		customDefault := origTransport.Clone()
		customDefault.TLSClientConfig = &tls.Config{RootCAs: pool}
		customDefault.ForceAttemptHTTP2 = true
		http.DefaultTransport = customDefault
		t.Cleanup(func() {
			http.DefaultTransport = original
		})

		transport, err := CloneDefaultHTTPTransport()
		require.NoError(t, err)

		// No TLSClientConfig mutation on the clone is needed: RootCAs is preserved
		// and h2 ALPN was cleared.
		statusCode, proto := doRequest(t, transport, srv.URL)
		require.Equal(t, http.StatusOK, statusCode)
		require.Equal(t, "HTTP/1.1", proto)
		require.Contains(t, customDefault.TLSClientConfig.NextProtos, "h2")
	})

	t.Run("allows callers to opt back into HTTP/2 via ForceAttemptHTTP2", func(t *testing.T) {
		srv, pool := newTLSServer(t, true)

		transport, err := CloneDefaultHTTPTransport()
		require.NoError(t, err)
		transport.ForceAttemptHTTP2 = true
		setRootCAs(transport, pool)

		statusCode, proto := doRequest(t, transport, srv.URL)
		require.Equal(t, http.StatusOK, statusCode)
		require.Equal(t, "HTTP/2.0", proto)
	})
}
