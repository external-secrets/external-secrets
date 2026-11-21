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
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"

	"github.com/external-secrets/external-secrets/runtime/esutils"
)

const (
	azureIdentityBindingAudience = "api://AKSIdentityBinding"
	azureKubernetesTokenProxy    = "AZURE_KUBERNETES_TOKEN_PROXY"
	azureKubernetesSNIName       = "AZURE_KUBERNETES_SNI_NAME"
	azureKubernetesCAFile        = "AZURE_KUBERNETES_CA_FILE"
	azureKubernetesCAData        = "AZURE_KUBERNETES_CA_DATA"
)

// identityBindingProxy supplies the proxy transport that azidentity currently exposes
// only for file-based WorkloadIdentityCredential, not for ClientAssertionCredential.
type identityBindingProxy struct {
	endpoint *url.URL
	sniName  string
	caFile   string

	mu        sync.Mutex
	caData    string
	transport *http.Transport
}

func newIdentityBindingProxy() (*identityBindingProxy, error) {
	rawURL := os.Getenv(azureKubernetesTokenProxy)
	if rawURL == "" {
		return nil, fmt.Errorf("%s must be set in the controller pod", azureKubernetesTokenProxy)
	}
	endpoint, err := url.Parse(rawURL)
	if err != nil || endpoint.Scheme != "https" || endpoint.Hostname() == "" ||
		endpoint.User != nil || endpoint.ForceQuery || endpoint.RawQuery != "" || endpoint.Fragment != "" {
		return nil, fmt.Errorf("%s must be an HTTPS URL without credentials, a query, or a fragment", azureKubernetesTokenProxy)
	}

	p := &identityBindingProxy{
		endpoint: endpoint,
		sniName:  os.Getenv(azureKubernetesSNIName),
		caFile:   os.Getenv(azureKubernetesCAFile),
		caData:   os.Getenv(azureKubernetesCAData),
	}
	if p.caFile != "" && p.caData != "" {
		return nil, fmt.Errorf("only one of %s and %s may be set", azureKubernetesCAFile, azureKubernetesCAData)
	}
	if _, err := p.loadTransport(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *identityBindingProxy) loadTransport() (*http.Transport, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	caData := p.caData
	if p.caFile != "" {
		data, err := os.ReadFile(p.caFile)
		if err != nil {
			return nil, fmt.Errorf("read AKS identity binding proxy CA file: %w", err)
		}
		if len(data) == 0 {
			return nil, errors.New("AKS identity binding proxy CA file is empty")
		}
		caData = string(data)
	}
	if p.transport != nil && p.caData == caData {
		return p.transport, nil
	}

	var roots *x509.CertPool
	if caData != "" {
		roots = x509.NewCertPool()
		if !roots.AppendCertsFromPEM([]byte(caData)) {
			return nil, errors.New("AKS identity binding proxy CA contains no valid PEM certificates")
		}
	}
	transport, err := esutils.CloneDefaultHTTPTransport()
	if err != nil {
		return nil, err
	}
	transport.TLSClientConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    roots,
		ServerName: p.sniName,
	}
	if p.transport != nil {
		p.transport.CloseIdleConnections()
	}
	p.transport = transport
	p.caData = caData
	return p.transport, nil
}

// Do mirrors the Azure SDK's proxy routing: preserve the authority Host header,
// path, and query while connecting to the configured AKS endpoint for every request.
func (p *identityBindingProxy) Do(req *http.Request) (*http.Response, error) {
	transport, err := p.loadTransport()
	if err != nil {
		return nil, err
	}
	proxied := req.Clone(req.Context())
	proxied.URL = p.endpoint.JoinPath(req.URL.EscapedPath())
	proxied.URL.RawQuery = req.URL.RawQuery
	proxied.URL.ForceQuery = req.URL.ForceQuery
	// RoundTrip does not follow redirects to other endpoints.
	return transport.RoundTrip(proxied)
}
