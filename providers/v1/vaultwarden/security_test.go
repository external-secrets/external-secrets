//go:build vaultwarden || all_providers

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

package vaultwarden

import (
	"context"
	"encoding/pem"
	"net/http/httptest"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// TestTLSRejectsSelfSignedWithoutCABundle verifies that the HTTP client built
// by initHTTPClient rejects a self-signed certificate when no CABundle is
// configured (i.e. InsecureSkipVerify must NOT be set).
func TestTLSRejectsSelfSignedWithoutCABundle(t *testing.T) {
	srv := httptest.NewTLSServer(nil)
	defer srv.Close()

	c := &Client{
		provider: &esv1.VaultwardenProvider{
			URL: srv.URL,
		},
	}
	if err := c.initHTTPClient(); err != nil {
		t.Fatalf("initHTTPClient: %v", err)
	}

	_, err := c.listCiphersWithToken(context.Background(), "x")
	if err == nil {
		t.Fatalf("expected TLS verification failure, got nil error")
	}
	if !strings.Contains(err.Error(), "x509") &&
		!strings.Contains(err.Error(), "certificate") {
		t.Fatalf("expected TLS error, got: %v", err)
	}
}

// TestTLSAcceptsSelfSignedWithCABundle verifies that when the server's
// certificate is explicitly trusted via CABundle, the TLS handshake succeeds.
func TestTLSAcceptsSelfSignedWithCABundle(t *testing.T) {
	srv := httptest.NewTLSServer(nil)
	defer srv.Close()

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: srv.Certificate().Raw,
	})

	c := &Client{
		provider: &esv1.VaultwardenProvider{
			URL:      srv.URL,
			CABundle: certPEM,
		},
	}
	if err := c.initHTTPClient(); err != nil {
		t.Fatalf("initHTTPClient: %v", err)
	}

	// httptest.NewTLSServer returns an empty body for unrecognised paths;
	// this will fail JSON decoding. We only care that the TLS dial succeeds
	// (no x509/certificate error in the chain).
	_, err := c.listCiphersWithToken(context.Background(), "x")
	if err != nil &&
		(strings.Contains(err.Error(), "x509") ||
			strings.Contains(err.Error(), "certificate")) {
		t.Fatalf("expected TLS to succeed with caBundle, got TLS error: %v", err)
	}
}

// TestLogLeakResistance verifies that secret values (clientSecret, masterPassword)
// do not appear in error strings returned from the auth path. We use a fake K8s
// client so no real cluster is needed, and an unreachable URL so fetchToken fails.
func TestLogLeakResistance(t *testing.T) {
	const (
		knownClientSecret  = "super-secret-client-secret"
		knownMasterPassword = "super-secret-master-pw"
		secretName         = "vw-creds"
		namespace          = "default"
	)

	// Build a fake K8s secret containing the credential values we want to
	// ensure never appear in error output.
	k8sSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"clientID":       []byte("user.fake"),
			"clientSecret":   []byte(knownClientSecret),
			"masterPassword": []byte(knownMasterPassword),
		},
	}

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme: %v", err)
	}

	kubeClient := clientfake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(k8sSecret).
		Build()

	store := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vw-test-store",
			Namespace: namespace,
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				Vaultwarden: &esv1.VaultwardenProvider{
					URL: "https://nonexistent.invalid.tld",
					Auth: esv1.VaultwardenAuth{
						SecretRef: esv1.VaultwardenSecretRef{
							ClientID: esmeta.SecretKeySelector{
								Name: secretName,
								Key:  "clientID",
							},
							ClientSecret: esmeta.SecretKeySelector{
								Name: secretName,
								Key:  "clientSecret",
							},
							MasterPassword: esmeta.SecretKeySelector{
								Name: secretName,
								Key:  "masterPassword",
							},
						},
					},
				},
			},
		},
	}

	c := &Client{
		provider:  store.Spec.Provider.Vaultwarden,
		crClient:  kubeClient,
		namespace: namespace,
		store:     store,
	}
	if err := c.initHTTPClient(); err != nil {
		t.Fatalf("initHTTPClient: %v", err)
	}

	// Trigger the full auth path — fetchToken will fail at DNS/TLS for the
	// unreachable host. We expect an error; we just need to check it for leaks.
	_, _, err := c.getToken(context.Background())
	if err == nil {
		t.Fatalf("expected auth failure against unreachable host, got nil error")
	}

	errStr := err.Error()
	for _, secret := range []string{knownClientSecret, knownMasterPassword} {
		if strings.Contains(errStr, secret) {
			t.Fatalf("secret value leaked in error string: %q found in: %s", secret, errStr)
		}
	}
}
