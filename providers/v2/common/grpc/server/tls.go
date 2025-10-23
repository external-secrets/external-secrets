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

package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

const (
	// Default paths for certificate files
	DefaultCertDir    = "/etc/provider/certs"
	DefaultCACertFile = "ca.crt"
	DefaultCertFile   = "tls.crt"
	DefaultKeyFile    = "tls.key"
)

// TLSConfig holds configuration for provider server TLS.
type TLSConfig struct {
	CertDir    string
	CACertFile string
	CertFile   string
	KeyFile    string
}

// DefaultTLSConfig returns a TLSConfig with default values.
// Values can be overridden via environment variables:
// - TLS_CERT_DIR
// - TLS_CA_CERT_FILE
// - TLS_CERT_FILE
// - TLS_KEY_FILE
func DefaultTLSConfig() *TLSConfig {
	return &TLSConfig{
		CertDir:    getEnvOrDefault("TLS_CERT_DIR", DefaultCertDir),
		CACertFile: getEnvOrDefault("TLS_CA_CERT_FILE", DefaultCACertFile),
		CertFile:   getEnvOrDefault("TLS_CERT_FILE", DefaultCertFile),
		KeyFile:    getEnvOrDefault("TLS_KEY_FILE", DefaultKeyFile),
	}
}

// LoadTLSConfig loads TLS configuration for a provider server.
// This enables mTLS, requiring and verifying client certificates.
func LoadTLSConfig(config *TLSConfig) (*tls.Config, error) {
	// Load server certificate and key
	certPath := fmt.Sprintf("%s/%s", config.CertDir, config.CertFile)
	keyPath := fmt.Sprintf("%s/%s", config.CertDir, config.KeyFile)

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load server certificate: %w", err)
	}

	// Load CA certificate for client verification
	caPath := fmt.Sprintf("%s/%s", config.CertDir, config.CACertFile)
	caCert, err := os.ReadFile(caPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load CA certificate: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caCertPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS12, // TLS 1.2 minimum for compatibility
	}, nil
}

// TLSVersionName returns a human-readable name for a TLS version.
func TLSVersionName(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return "Unknown"
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
