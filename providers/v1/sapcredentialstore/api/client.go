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

package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// SAPCSClientInterface is the interface the HTTP client implements.
// Defined here so the fake package can implement it independently.
type SAPCSClientInterface interface {
	GetCredential(ctx context.Context, ns, credType, name string) (*Credential, error)
	ListCredentials(ctx context.Context, ns, credType string) ([]CredentialMeta, error)
	PutCredential(ctx context.Context, ns, credType, name string, body *CredentialBody) error
	DeleteCredential(ctx context.Context, ns, credType, name string) error
	CredentialExists(ctx context.Context, ns, credType, name string) (bool, error)
}

// httpClient implements SAPCSClientInterface using a standard net/http.Client.
type httpClient struct {
	baseURL string
	http    *http.Client
}

// NewOAuth2Client creates an httpClient that authenticates via OAuth2 bearer tokens.
// The token source is managed by golang.org/x/oauth2 and tokens are refreshed automatically.
func NewOAuth2Client(baseURL string, transport http.RoundTripper) SAPCSClientInterface {
	return &httpClient{
		baseURL: baseURL,
		http:    &http.Client{Transport: transport},
	}
}

// NewMTLSClient creates an httpClient that authenticates via mutual TLS.
// certPEM and keyPEM must be PEM-encoded certificate and private key bytes.
func NewMTLSClient(baseURL string, certPEM, keyPEM []byte) (SAPCSClientInterface, error) {
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse mTLS key pair: %w", err)
	}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		},
	}
	return &httpClient{
		baseURL: baseURL,
		http:    &http.Client{Transport: transport},
	}, nil
}

func (c *httpClient) credentialURL(ns, credType, name string) string {
	if name == "" {
		return fmt.Sprintf("%s/api/v1/namespaces/%s/credentials?type=%s", c.baseURL, ns, credType)
	}
	return fmt.Sprintf("%s/api/v1/namespaces/%s/credentials/%s/%s", c.baseURL, ns, credType, name)
}

func (c *httpClient) GetCredential(ctx context.Context, ns, credType, name string) (*Credential, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.credentialURL(ns, credType, name), nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET credential %s/%s: %w", credType, name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, &NotFoundError{CredType: credType, Name: name}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET credential %s/%s: unexpected status %d", credType, name, resp.StatusCode)
	}

	var cred Credential
	if err := json.NewDecoder(resp.Body).Decode(&cred); err != nil {
		return nil, fmt.Errorf("decoding credential %s/%s: %w", credType, name, err)
	}
	return &cred, nil
}

func (c *httpClient) ListCredentials(ctx context.Context, ns, credType string) ([]CredentialMeta, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.credentialURL(ns, credType, ""), nil)
	if err != nil {
		return nil, fmt.Errorf("building list request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("LIST credentials type=%s: %w", credType, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LIST credentials type=%s: unexpected status %d", credType, resp.StatusCode)
	}

	var items []CredentialMeta
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("decoding credential list type=%s: %w", credType, err)
	}
	return items, nil
}

func (c *httpClient) PutCredential(ctx context.Context, ns, credType, name string, body *CredentialBody) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling credential body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.credentialURL(ns, credType, name), bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("building PUT request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("PUT credential %s/%s: %w", credType, name, err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) //nolint:errcheck

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("PUT credential %s/%s: unexpected status %d", credType, name, resp.StatusCode)
	}
	return nil
}

func (c *httpClient) DeleteCredential(ctx context.Context, ns, credType, name string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.credentialURL(ns, credType, name), nil)
	if err != nil {
		return fmt.Errorf("building DELETE request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("DELETE credential %s/%s: %w", credType, name, err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) //nolint:errcheck

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("DELETE credential %s/%s: unexpected status %d", credType, name, resp.StatusCode)
	}
	return nil
}

func (c *httpClient) CredentialExists(ctx context.Context, ns, credType, name string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, c.credentialURL(ns, credType, name), nil)
	if err != nil {
		return false, fmt.Errorf("building HEAD request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return false, fmt.Errorf("HEAD credential %s/%s: %w", credType, name, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("HEAD credential %s/%s: unexpected status %d", credType, name, resp.StatusCode)
	}
}

// NotFoundError is returned by GetCredential when the credential does not exist.
type NotFoundError struct {
	CredType string
	Name     string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("credential %s/%s not found", e.CredType, e.Name)
}
