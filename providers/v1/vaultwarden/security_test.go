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
	"crypto/x509"
	"encoding/pem"
	"net/http/httptest"
	"strings"
	"testing"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
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

// Ensure the x509 import is used even if the helpers above are refactored.
var _ = x509.NewCertPool
