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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// TLSConfig holds TLS configuration for connecting to a provider.
type TLSConfig struct {
	CACert     []byte
	ClientCert []byte
	ClientKey  []byte
	// ServerName is optional. If empty, hostname verification is skipped
	// but certificate chain validation is still performed (safe for mTLS).
	ServerName string
}

// LoadClientTLSConfig loads TLS configuration for connecting to a provider.
// It fetches the provider's certificate secret from Kubernetes.
func LoadClientTLSConfig(
	ctx context.Context,
	k8sClient ctrlclient.Client,
	address string,
	namespace string,
) (*TLSConfig, error) {
	// parse the address to get the hostname, it is a url
	hostname, _, err := net.SplitHostPort(address)
	if err != nil {
		hostname = address
	}

	secretName := "external-secrets-provider-tls"

	// Fetch secret
	var secret corev1.Secret
	key := types.NamespacedName{
		Name:      secretName,
		Namespace: namespace,
	}

	if err := k8sClient.Get(ctx, key, &secret); err != nil {
		return nil, fmt.Errorf("failed to get provider TLS secret %s: %w", secretName, err)
	}

	// Validate secret data
	caCert, ok := secret.Data["ca.crt"]
	if !ok || len(caCert) == 0 {
		return nil, fmt.Errorf("ca.crt not found or empty in secret %s", secretName)
	}

	clientCert, ok := secret.Data["client.crt"]
	if !ok || len(clientCert) == 0 {
		return nil, fmt.Errorf("client.crt not found or empty in secret %s", secretName)
	}

	clientKey, ok := secret.Data["client.key"]
	if !ok || len(clientKey) == 0 {
		return nil, fmt.Errorf("client.key not found or empty in secret %s", secretName)
	}

	return &TLSConfig{
		CACert:     caCert,
		ClientCert: clientCert,
		ClientKey:  clientKey,
		ServerName: hostname,
	}, nil
}

// ToGRPCTLSConfig converts TLSConfig to a *tls.Config suitable for gRPC.
func (t *TLSConfig) ToGRPCTLSConfig() (*tls.Config, error) {
	// Load client certificate
	cert, err := tls.X509KeyPair(t.ClientCert, t.ClientKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}

	// Load CA certificate
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(t.CACert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
		MinVersion:   tls.VersionTLS12, // TLS 1.2 minimum for compatibility
		ServerName:   t.ServerName,
	}

	return tlsConfig, nil
}
