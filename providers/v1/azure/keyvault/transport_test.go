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
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

func TestLegacyHTTPFlags(t *testing.T) {
	for _, tc := range []struct {
		name    string
		args    []string
		want    int
		wantErr bool
	}{
		{name: "default", want: 2},
		{name: "lower", args: []string{"--azure-kv-max-idle-connections-per-host=1"}, want: 1},
		{name: "tuned", args: []string{"--azure-kv-max-idle-connections-per-host=20"}, want: 20},
		{name: "higher", args: []string{"--azure-kv-max-idle-connections-per-host=40"}, want: 40},
		{name: "above total", args: []string{"--azure-kv-max-idle-connections-per-host=200"}, want: 200},
		{name: "zero", args: []string{"--azure-kv-max-idle-connections-per-host=0"}, wantErr: true},
		{name: "negative", args: []string{"--azure-kv-max-idle-connections-per-host=-1"}, wantErr: true},
		{name: "text", args: []string{"--azure-kv-max-idle-connections-per-host=many"}, wantErr: true},
		{name: "fraction", args: []string{"--azure-kv-max-idle-connections-per-host=1.5"}, wantErr: true},
		{name: "overflow", args: []string{"--azure-kv-max-idle-connections-per-host=999999999999999999999"}, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, getClient := newLegacyHTTPFeature()
			f.Flags.Init(f.Flags.Name(), pflag.ContinueOnError)
			f.Flags.SetOutput(io.Discard)
			err := f.Flags.Parse(tc.args)
			if tc.wantErr {
				require.ErrorContains(t, err, "greater than zero")
				return
			}
			require.NoError(t, err)
			f.Initialize()
			client := getClient()
			t.Cleanup(client.CloseIdleConnections)
			require.Same(t, client, getClient())
			transport := client.Transport.(*http.Transport)
			require.Equal(t, tc.want, transport.MaxIdleConnsPerHost)
			require.Equal(t, max(100, tc.want), transport.MaxIdleConns)
		})
	}
}

func TestLegacyHTTPDefaultsIsolatedFromAutorest(t *testing.T) {
	shared := autorest.CreateSender().(*http.Client)
	sharedTransport := shared.Transport.(*http.Transport)
	oldLimit := sharedTransport.MaxIdleConnsPerHost
	client := newLegacyHTTPClient(defaultMaxIdleConnsPerHost)
	t.Cleanup(client.CloseIdleConnections)
	transport := client.Transport.(*http.Transport)

	require.NotSame(t, shared, client)
	require.NotSame(t, sharedTransport, transport)
	require.Equal(t, oldLimit, sharedTransport.MaxIdleConnsPerHost)
	legacyLimit := oldLimit
	if legacyLimit == 0 {
		legacyLimit = http.DefaultMaxIdleConnsPerHost
	}
	require.Equal(t, legacyLimit, transport.MaxIdleConnsPerHost)
	require.Equal(t, sharedTransport.MaxIdleConns, transport.MaxIdleConns)
	require.Equal(t, sharedTransport.IdleConnTimeout, transport.IdleConnTimeout)
	require.Equal(t, sharedTransport.TLSHandshakeTimeout, transport.TLSHandshakeTimeout)
	require.Equal(t, sharedTransport.ExpectContinueTimeout, transport.ExpectContinueTimeout)
	require.Equal(t, sharedTransport.ForceAttemptHTTP2, transport.ForceAttemptHTTP2)
	require.Equal(t, sharedTransport.TLSClientConfig.MinVersion, transport.TLSClientConfig.MinVersion)
	require.Equal(t, sharedTransport.TLSClientConfig.Renegotiation, transport.TLSClientConfig.Renegotiation)
	require.False(t, transport.TLSClientConfig.InsecureSkipVerify)
	require.NotNil(t, transport.Proxy)
	require.NotNil(t, transport.DialContext)
	require.Equal(t, shared.Timeout, client.Timeout)
	require.NotNil(t, client.Jar)
}

func TestLegacyClientsShareTransport(t *testing.T) {
	clients := make([]*Azure, 0, 2)
	for i := range 2 {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("credentials-%d", i), Namespace: "default"},
			Data: map[string][]byte{
				"client-id":     fmt.Appendf(nil, "client-%d", i),
				"client-secret": fmt.Appendf(nil, "secret-%d", i),
			},
		}
		az := &Azure{
			crClient:  fake.NewClientBuilder().WithObjects(secret).Build(),
			namespace: "default",
			store:     &esv1.SecretStore{},
			provider: &esv1.AzureKVProvider{
				AuthType: new(esv1.AzureServicePrincipal),
				TenantID: new("tenant"),
				VaultURL: new("https://test.vault.azure.net"),
				AuthSecretRef: &esv1.AzureKVAuth{
					ClientID:     &esmeta.SecretKeySelector{Name: secret.Name, Key: "client-id"},
					ClientSecret: &esmeta.SecretKeySelector{Name: secret.Name, Key: "client-secret"},
				},
			},
		}
		require.NoError(t, initializeLegacyClient(t.Context(), az))
		clients = append(clients, az)
	}
	first := clients[0].baseClient.(*keyvault.BaseClient)
	second := clients[1].baseClient.(*keyvault.BaseClient)
	require.Same(t, getLegacyHTTPClient(), first.Sender)
	require.Same(t, first.Sender, second.Sender)
	require.NotSame(t, first.Authorizer, second.Authorizer)
	require.NoError(t, clients[0].Close(t.Context()))
	require.Same(t, first.Sender, second.Sender)
}

func TestLegacyHTTPBurstReuse(t *testing.T) {
	const concurrency, bursts = 20, 3
	for _, limit := range []int{2, 20, 40} {
		t.Run(fmt.Sprintf("limit-%d", limit), func(t *testing.T) {
			arrived := make(chan struct{}, concurrency)
			release := make(chan struct{}, concurrency)
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				arrived <- struct{}{}
				select {
				case <-release:
				case <-r.Context().Done():
					return
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"value":"fixture"}`)
			}))
			defer server.Close()
			f, getClient := newLegacyHTTPFeature()
			if limit != defaultMaxIdleConnsPerHost {
				require.NoError(t, f.Flags.Set("azure-kv-max-idle-connections-per-host", fmt.Sprint(limit)))
			}
			f.Initialize()
			httpClient := getClient()
			defer httpClient.CloseIdleConnections()
			transport := trustPoolServers(httpClient, server)
			var connections atomic.Int64
			dial := transport.DialContext
			transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				conn, err := dial(ctx, network, addr)
				if err == nil {
					connections.Add(1)
				}
				return conn, err
			}
			ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
			defer cancel()
			for burst := range bursts {
				results := make(chan error, concurrency)
				for range concurrency {
					go func() {
						cl := poolTestSDKClient(getClient(), "fixture-token")
						result, err := cl.GetSecret(ctx, server.URL, "fixture", "")
						if err == nil && (result.Value == nil || *result.Value != "fixture") {
							err = fmt.Errorf("unexpected secret value: %v", result.Value)
						}
						results <- err
					}()
				}
				for range concurrency {
					select {
					case <-arrived:
					case <-ctx.Done():
						t.Fatal("requests did not reach the server concurrently")
					}
				}
				for range concurrency {
					release <- struct{}{}
				}
				for range concurrency {
					require.NoError(t, <-results)
				}
				want := concurrency + burst*max(0, concurrency-limit)
				require.EqualValues(t, want, connections.Load(), "burst %d", burst)
			}
			t.Logf("%d SDK clients, %d TCP connections with idle limit %d", concurrency*bursts, connections.Load(), limit)
		})
	}
}

func TestLegacyHTTPProtocolsAndCredentials(t *testing.T) {
	for _, http2 := range []bool{false, true} {
		t.Run(fmt.Sprintf("http2-%t", http2), func(t *testing.T) {
			var connections atomic.Int64
			servers := make([]*httptest.Server, 0, 2)
			for range 2 {
				server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					wantProto := 1
					if http2 {
						wantProto = 2
					}
					if r.ProtoMajor != wantProto {
						t.Errorf("expected HTTP/%d, got %s", wantProto, r.Proto)
					}
					w.Header().Set("Content-Type", "application/json")
					fmt.Fprintf(w, `{"value":%q}`, r.Header.Get("Authorization"))
				}))
				server.EnableHTTP2 = http2
				server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
					if state == http.StateNew {
						connections.Add(1)
					}
				}
				server.StartTLS()
				t.Cleanup(server.Close)
				servers = append(servers, server)
			}
			client := newLegacyHTTPClient(defaultMaxIdleConnsPerHost)
			defer client.CloseIdleConnections()
			trustPoolServers(client, servers...)
			for i := range 8 {
				for _, server := range servers {
					token := fmt.Sprintf("store-%d", i%2)
					cl := poolTestSDKClient(client, token)
					az := &Azure{baseClient: &cl}
					result, err := cl.GetSecret(t.Context(), server.URL, "fixture", "")
					require.NoError(t, err)
					require.NotNil(t, result.Value)
					require.Equal(t, "Bearer "+token, *result.Value)
					require.NoError(t, az.Close(t.Context()))
				}
			}
			require.EqualValues(t, len(servers), connections.Load(), "clients should reuse one connection per host")
		})
	}
}

func TestLegacyHTTPRetryAndCancellation(t *testing.T) {
	var requests atomic.Int64
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if requests.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprint(w, `{"error":{"code":"ServiceUnavailable","message":"retry"}}`)
			return
		}
		fmt.Fprint(w, `{"value":"fixture"}`)
	}))
	defer server.Close()
	client := newLegacyHTTPClient(defaultMaxIdleConnsPerHost)
	defer client.CloseIdleConnections()
	trustPoolServers(client, server)
	cl := poolTestSDKClient(client, "fixture-token")
	cl.RetryAttempts = 1
	cl.RetryDuration = time.Millisecond
	result, err := cl.GetSecret(t.Context(), server.URL, "fixture", "")
	require.NoError(t, err)
	require.Equal(t, "fixture", *result.Value)
	require.EqualValues(t, 2, requests.Load())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = cl.GetSecret(ctx, server.URL, "fixture", "")
	require.ErrorIs(t, err, context.Canceled)
	require.EqualValues(t, 2, requests.Load())
}

func trustPoolServers(client *http.Client, servers ...*httptest.Server) *http.Transport {
	transport := client.Transport.(*http.Transport)
	transport.TLSClientConfig.RootCAs = x509.NewCertPool()
	for _, server := range servers {
		transport.TLSClientConfig.RootCAs.AddCert(server.Certificate())
	}
	return transport
}

func poolTestSDKClient(sender autorest.Sender, token string) keyvault.BaseClient {
	cl := keyvault.New()
	cl.Sender = sender
	cl.Authorizer = autorest.NewBearerAuthorizer(&adal.Token{AccessToken: token})
	return cl
}
