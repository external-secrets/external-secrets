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
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	"github.com/Azure/go-autorest/autorest"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

func TestLegacyKeyVaultConnectionReuse(t *testing.T) {
	const concurrency, rounds = 20, 90
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	var accepted, closed, requests atomic.Int64
	barriers := make([]chan struct{}, rounds)
	for i := range barriers {
		barriers[i] = make(chan struct{})
	}
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := requests.Add(1) - 1
		if (n+1)%concurrency == 0 {
			close(barriers[n/concurrency])
		}
		// All workers must occupy a connection before any can return to the pool.
		select {
		case <-barriers[n/concurrency]:
		case <-r.Context().Done():
			return
		}
		secret := strings.TrimPrefix(strings.TrimSuffix(r.URL.Path, "/"), "/secrets/")
		if r.Header.Get("Authorization") != "Bearer "+secret {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"value":"test-value"}`)
	}))
	server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		switch state {
		case http.StateNew:
			accepted.Add(1)
		case http.StateClosed:
			closed.Add(1)
		case http.StateActive, http.StateIdle, http.StateHijacked:
		}
	}
	// Keep HTTP/1.1: HTTP/2 multiplexing would hide the per-host idle limit.
	server.StartTLS()
	defer server.Close()

	sender := newLegacyKeyVaultHTTPClient()
	transport := sender.Transport.(*http.Transport)
	transport.TLSClientConfig.RootCAs = x509.NewCertPool()
	transport.TLSClientConfig.RootCAs.AddCert(server.Certificate())
	defer transport.CloseIdleConnections()

	started := time.Now()
	for round := range rounds {
		results := make(chan error, concurrency)
		for worker := range concurrency {
			go func() {
				// Each reconcile gets a fresh SDK client and authorizer, sharing only HTTP state.
				secret := fmt.Sprintf("secret-%d-%d", round, worker)
				cl := keyvault.New()
				cl.Sender = sender
				cl.Authorizer = autorest.NewBearerAuthorizer(&tokenProvider{accessToken: secret})
				cl.RetryAttempts = 1
				value, err := cl.GetSecret(ctx, server.URL, secret, "")
				if err == nil && (value.Value == nil || *value.Value != "test-value") {
					err = fmt.Errorf("unexpected secret value: %v", value.Value)
				}
				az := Azure{baseClient: &cl}
				if closeErr := az.Close(ctx); err == nil {
					err = closeErr
				}
				results <- err
			}()
		}
		for range concurrency {
			require.NoError(t, <-results)
		}
	}
	t.Logf("requests=%d new_connections=%d closed_connections=%d elapsed=%s",
		requests.Load(), accepted.Load(), closed.Load(), time.Since(started))
	require.EqualValues(t, concurrency*rounds, requests.Load())
	require.EqualValues(t, concurrency, accepted.Load(), "connections should survive between reconcile waves")
	require.Zero(t, closed.Load(), "no idle connection should be evicted during these bursts")
}

func TestInitializeLegacyClientSharesSender(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "credentials", Namespace: "test"},
		Data:       map[string][]byte{"client-id": []byte("client"), "client-secret": []byte("secret")},
	}
	kube := fake.NewClientBuilder().WithObjects(secret).Build()
	var previous *keyvault.BaseClient
	for _, name := range []string{"first", "second"} {
		store := &esv1.SecretStore{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "test"}}
		az := &Azure{
			store:     store,
			namespace: "test",
			crClient:  kube,
			provider: &esv1.AzureKVProvider{
				AuthType: new(esv1.AzureServicePrincipal),
				TenantID: new("tenant"),
				AuthSecretRef: &esv1.AzureKVAuth{
					ClientID:     &esmeta.SecretKeySelector{Name: "credentials", Key: "client-id"},
					ClientSecret: &esmeta.SecretKeySelector{Name: "credentials", Key: "client-secret"},
				},
			},
		}
		require.NoError(t, initializeLegacyClient(t.Context(), az))
		cl := az.baseClient.(*keyvault.BaseClient)
		require.Same(t, legacyKeyVaultSender(), cl.Sender)
		if previous != nil {
			require.NotSame(t, previous.Authorizer, cl.Authorizer)
		}
		require.NoError(t, az.Close(t.Context()))
		previous = cl
	}
}
