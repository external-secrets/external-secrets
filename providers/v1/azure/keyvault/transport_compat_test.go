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

package keyvault

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
)

func newLegacyCompatibilityClient(t *testing.T) (keyvault.BaseClient, *http.Transport) {
	t.Helper()
	sender := newLegacyKeyVaultHTTPClient()
	t.Cleanup(sender.CloseIdleConnections)
	transport, ok := sender.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("unexpected transport type %T", sender.Transport)
	}
	client := keyvault.New()
	client.Sender = sender
	client.RetryDuration = 0
	return client, transport
}

func legacyCompatibilitySecret(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, `{"value":"test-value"}`)
}

func TestLegacyKeyVaultTransportTLS(t *testing.T) {
	for _, tc := range []struct {
		name    string
		version uint16
		trust   bool
		wantErr bool
	}{
		{name: "trusted TLS 1.2", version: tls.VersionTLS12, trust: true},
		{name: "untrusted certificate", version: tls.VersionTLS12, wantErr: true},
		{name: "reject TLS 1.1", version: tls.VersionTLS11, trust: true, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewUnstartedServer(http.HandlerFunc(legacyCompatibilitySecret))
			server.Config.ErrorLog = log.New(io.Discard, "", 0)
			// Include obsolete TLS to verify the client's minimum version.
			server.TLS = &tls.Config{MinVersion: tc.version, MaxVersion: tc.version}
			server.StartTLS()
			t.Cleanup(server.Close)

			client, transport := newLegacyCompatibilityClient(t)
			if tc.trust {
				transport.TLSClientConfig.RootCAs = x509.NewCertPool()
				transport.TLSClientConfig.RootCAs.AddCert(server.Certificate())
			}
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			secret, err := client.GetSecret(ctx, server.URL, "test-secret", "")
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected TLS handshake failure")
				}
				if errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("request timed out instead of rejecting TLS: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetSecret: %v", err)
			}
			if secret.Value == nil || *secret.Value != "test-value" {
				t.Fatalf("unexpected secret: %v", secret.Value)
			}
		})
	}
}

func TestLegacyKeyVaultTransportRetry(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = io.WriteString(w, `{"error":{"code":"ServiceUnavailable","message":"retry"}}`)
			return
		}
		legacyCompatibilitySecret(w, r)
	}))
	t.Cleanup(server.Close)
	client, _ := newLegacyCompatibilityClient(t)
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	secret, err := client.GetSecret(ctx, server.URL, "test-secret", "")
	if err != nil {
		t.Fatalf("GetSecret after retry: %v", err)
	}
	if got := requests.Load(); got != 2 {
		t.Fatalf("expected one retry, got %d requests", got)
	}
	if secret.Value == nil || *secret.Value != "test-value" {
		t.Fatalf("unexpected secret: %v", secret.Value)
	}
}

func TestLegacyKeyVaultTransportCancellation(t *testing.T) {
	started := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		select {
		case started <- struct{}{}:
		default:
		}
		<-r.Context().Done()
	}))
	t.Cleanup(server.Close)
	client, _ := newLegacyCompatibilityClient(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	result := make(chan error, 1)
	go func() {
		_, err := client.GetSecret(ctx, server.URL, "test-secret", "")
		result <- err
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("request did not reach the server")
	}
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context cancellation, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("GetSecret did not stop after context cancellation")
	}
}

func TestLegacyKeyVaultTransportProxy(t *testing.T) {
	// ProxyFromEnvironment caches its environment on first use. A subprocess
	// keeps this test independent of requests made by other provider tests.
	const subprocessEnv = "ESO_AZURE_TRANSPORT_PROXY_TEST"
	if os.Getenv(subprocessEnv) != "1" {
		executable, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, executable, "-test.run=^TestLegacyKeyVaultTransportProxy$", "-test.count=1")
		for _, env := range os.Environ() {
			key, _, _ := strings.Cut(env, "=")
			switch strings.ToUpper(key) {
			case "HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY", "ALL_PROXY", subprocessEnv:
				continue
			}
			cmd.Env = append(cmd.Env, env)
		}
		cmd.Env = append(cmd.Env, subprocessEnv+"=1")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("proxy subprocess: %v\n%s", err, output)
		}
		return
	}

	server := httptest.NewTLSServer(http.HandlerFunc(legacyCompatibilitySecret))
	t.Cleanup(server.Close)
	var connections atomic.Int32
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodConnect || r.Host != "example.com:443" {
			http.Error(w, "unexpected proxy request", http.StatusBadRequest)
			return
		}
		upstream, err := net.DialTimeout("tcp", server.Listener.Addr().String(), 5*time.Second)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer upstream.Close()
		downstream, buffered, err := w.(http.Hijacker).Hijack()
		if err != nil {
			return
		}
		defer downstream.Close()
		connections.Add(1)
		if _, err := buffered.WriteString("HTTP/1.1 200 Connection Established\r\n\r\n"); err != nil {
			return
		}
		if err := buffered.Flush(); err != nil {
			return
		}
		go func() {
			_, _ = io.Copy(upstream, buffered)
			_ = upstream.Close()
		}()
		_, _ = io.Copy(downstream, upstream)
	}))
	t.Cleanup(proxy.Close)
	t.Setenv("HTTPS_PROXY", proxy.URL)
	t.Setenv("NO_PROXY", "direct.example")
	client, transport := newLegacyCompatibilityClient(t)
	transport.TLSClientConfig.RootCAs = x509.NewCertPool()
	transport.TLSClientConfig.RootCAs.AddCert(server.Certificate())
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	if _, err := client.GetSecret(ctx, "https://example.com", "test-secret", ""); err != nil {
		t.Fatalf("GetSecret through HTTPS proxy: %v", err)
	}
	if connections.Load() != 1 {
		t.Fatalf("expected one proxy tunnel, got %d", connections.Load())
	}
	direct := &http.Request{URL: &url.URL{Scheme: "https", Host: "direct.example"}}
	if proxyURL, err := transport.Proxy(direct); err != nil || proxyURL != nil {
		t.Fatalf("NO_PROXY was not respected: proxy=%v, error=%v", proxyURL, err)
	}
}
