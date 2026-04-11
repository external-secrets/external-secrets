/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package grpc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestNamespaceFromAddress(t *testing.T) {
	testCases := []struct {
		name     string
		address  string
		fallback string
		expected string
	}{
		{
			name:     "service_dns_with_port",
			address:  "provider.team-a.svc:9443",
			fallback: "fallback",
			expected: "team-a",
		},
		{
			name:     "service_dns_cluster_local",
			address:  "provider.team-b.svc.cluster.local:9443",
			fallback: "fallback",
			expected: "team-b",
		},
		{
			name:     "non_service_address_uses_fallback",
			address:  "127.0.0.1:9443",
			fallback: "tenant-a",
			expected: "tenant-a",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := NamespaceFromAddress(tc.address, tc.fallback); got != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestResolveTLSSecretNamespace(t *testing.T) {
	testCases := []struct {
		name                 string
		address              string
		authNamespace        string
		resourceNamespace    string
		providerRefNamespace string
		expected             string
	}{
		{
			name:                 "service_dns_takes_precedence",
			address:              "provider.service-ns.svc:9443",
			authNamespace:        "auth-ns",
			resourceNamespace:    "resource-ns",
			providerRefNamespace: "provider-ref-ns",
			expected:             "service-ns",
		},
		{
			name:                 "auth_namespace_used_before_other_fallbacks",
			address:              "127.0.0.1:9443",
			authNamespace:        "auth-ns",
			resourceNamespace:    "resource-ns",
			providerRefNamespace: "provider-ref-ns",
			expected:             "auth-ns",
		},
		{
			name:                 "resource_namespace_used_before_provider_ref_namespace",
			address:              "127.0.0.1:9443",
			resourceNamespace:    "resource-ns",
			providerRefNamespace: "provider-ref-ns",
			expected:             "resource-ns",
		},
		{
			name:                 "provider_ref_namespace_is_final_fallback",
			address:              "127.0.0.1:9443",
			providerRefNamespace: "provider-ref-ns",
			expected:             "provider-ref-ns",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ResolveTLSSecretNamespace(tc.address, tc.authNamespace, tc.resourceNamespace, tc.providerRefNamespace); got != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestLoadClientTLSConfig(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		serverName := "127.0.0.1"
		_, _, clientCertPEM, clientKeyPEM, caCertPEM := newTLSArtifactsForTest(t, serverName)
		kubeClient := newTLSSecretClient(t, map[string][]byte{
			"ca.crt":     caCertPEM,
			"client.crt": clientCertPEM,
			"client.key": clientKeyPEM,
		})

		cfg, err := LoadClientTLSConfig(context.Background(), kubeClient, "127.0.0.1:9443", "tenant-a")
		if err != nil {
			t.Fatalf("LoadClientTLSConfig() error = %v", err)
		}

		if string(cfg.CACert) != string(caCertPEM) || string(cfg.ClientCert) != string(clientCertPEM) || string(cfg.ClientKey) != string(clientKeyPEM) {
			t.Fatalf("unexpected tls config: %#v", cfg)
		}
		if cfg.ServerName != serverName {
			t.Fatalf("expected server name %q, got %q", serverName, cfg.ServerName)
		}
	})

	t.Run("missing_secret", func(t *testing.T) {
		scheme := runtime.NewScheme()
		utilruntime.Must(clientgoscheme.AddToScheme(scheme))

		kubeClient := fakeclient.NewClientBuilder().
			WithScheme(scheme).
			Build()

		_, err := LoadClientTLSConfig(context.Background(), kubeClient, "127.0.0.1:9443", "tenant-a")
		if err == nil || err.Error() == "" {
			t.Fatalf("expected missing secret error, got %v", err)
		}
	})

	t.Run("missing_secret_data", func(t *testing.T) {
		kubeClient := newTLSSecretClient(t, map[string][]byte{
			"ca.crt": []byte("ca"),
		})

		_, err := LoadClientTLSConfig(context.Background(), kubeClient, "127.0.0.1:9443", "tenant-a")
		if err == nil || err.Error() != "client.crt not found or empty in secret external-secrets-provider-tls" {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestTLSConfigToGRPCTLSConfig(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		serverName := "127.0.0.1"
		_, _, clientCertPEM, clientKeyPEM, caCertPEM := newTLSArtifactsForTest(t, serverName)

		cfg, err := (&TLSConfig{
			CACert:     caCertPEM,
			ClientCert: clientCertPEM,
			ClientKey:  clientKeyPEM,
			ServerName: serverName,
		}).ToGRPCTLSConfig()
		if err != nil {
			t.Fatalf("ToGRPCTLSConfig() error = %v", err)
		}

		if cfg.MinVersion != tls.VersionTLS12 {
			t.Fatalf("expected min version %v, got %v", tls.VersionTLS12, cfg.MinVersion)
		}
		if cfg.ServerName != serverName {
			t.Fatalf("expected server name %q, got %q", serverName, cfg.ServerName)
		}
		if len(cfg.Certificates) != 1 {
			t.Fatalf("expected one certificate, got %d", len(cfg.Certificates))
		}
		if cfg.RootCAs == nil {
			t.Fatal("expected root CAs to be set")
		}
	})

	t.Run("invalid_keypair", func(t *testing.T) {
		_, err := (&TLSConfig{
			CACert:     []byte("not-a-ca"),
			ClientCert: []byte("not-a-cert"),
			ClientKey:  []byte("not-a-key"),
		}).ToGRPCTLSConfig()
		if err == nil {
			t.Fatal("expected invalid keypair to fail")
		}
	})

	t.Run("invalid_ca", func(t *testing.T) {
		serverName := "127.0.0.1"
		_, _, clientCertPEM, clientKeyPEM, _ := newTLSArtifactsForTest(t, serverName)

		_, err := (&TLSConfig{
			CACert:     []byte("not-a-ca"),
			ClientCert: clientCertPEM,
			ClientKey:  clientKeyPEM,
			ServerName: serverName,
		}).ToGRPCTLSConfig()
		if err == nil || err.Error() != "failed to parse CA certificate" {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func newTLSSecretClient(t *testing.T, data map[string][]byte) ctrlclient.Client {
	t.Helper()

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	return fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(&corev1.Secret{
			ObjectMeta: metav1ForTLS("external-secrets-provider-tls", "tenant-a"),
			Data:       data,
		}).
		Build()
}

func metav1ForTLS(name, namespace string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
	}
}

func newTLSArtifactsForTest(t *testing.T, host string) (serverCertPEM, serverKeyPEM, clientCertPEM, clientKeyPEM, caCertPEM []byte) {
	t.Helper()

	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "grpc-test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("CreateCertificate() error = %v", err)
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatalf("ParseCertificate() error = %v", err)
	}

	serverCertPEM, serverKeyPEM = newSignedTLSCertForTest(t, caCert, caKey, 2, host, []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth})
	clientCertPEM, clientKeyPEM = newSignedTLSCertForTest(t, caCert, caKey, 3, host, []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth})
	caCertPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	return serverCertPEM, serverKeyPEM, clientCertPEM, clientKeyPEM, caCertPEM
}

func newSignedTLSCertForTest(t *testing.T, caCert *x509.Certificate, caKey *rsa.PrivateKey, serial int64, host string, usages []x509.ExtKeyUsage) ([]byte, []byte) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(serial),
		Subject:      pkix.Name{CommonName: host},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  usages,
	}

	if ip := net.ParseIP(host); ip != nil {
		template.IPAddresses = []net.IP{ip}
	} else {
		template.DNSNames = []string{host}
	}

	der, err := x509.CreateCertificate(rand.Reader, template, caCert, &key.PublicKey, caKey)
	if err != nil {
		t.Fatalf("CreateCertificate() error = %v", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	return certPEM, keyPEM
}
