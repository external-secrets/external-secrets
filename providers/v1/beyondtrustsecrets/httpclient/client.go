/*
Copyright © 2025 ESO Maintainer Team

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

// Package httpclient provides an HTTP client for interacting with BeyondTrust Secrets Manager API.
package httpclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	btsutil "github.com/external-secrets/external-secrets/providers/v1/beyondtrustsecrets/util"
)

const (
	// API version header for BeyondTrust Secrets Manager.
	apiVersionHeader = "bt-secrets-api-version"
	apiVersion       = "2026-02-16"

	// Default timeout for HTTP requests.
	defaultTimeout = 30 * time.Second
)

// Client represents a client for interacting with BeyondTrust Secrets Manager API.
type Client struct {
	httpClient *http.Client
	baseURL    *url.URL
	token      string
}

// NewClient creates a new BeyondTrust Secrets Manager HTTP client.
func NewClient(serverURL, token string) (*Client, error) {
	if err := validateServerURL(serverURL); err != nil {
		return nil, err
	}

	parsedURL, err := url.Parse(strings.TrimSuffix(serverURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse server URL %q: %w", serverURL, err)
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		baseURL: parsedURL,
		token:   token,
	}, nil
}

// NewClientWithCustomCA creates a client using the provided PEM-encoded CA bundle.
func NewClientWithCustomCA(serverURL, token string, caBundlePEM []byte) (*Client, error) {
	if err := validateServerURL(serverURL); err != nil {
		return nil, err
	}

	parsedURL, err := url.Parse(strings.TrimSuffix(serverURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse server URL %q: %w", serverURL, err)
	}

	httpClient := &http.Client{
		Timeout: defaultTimeout,
	}

	if len(caBundlePEM) > 0 {
		roots := x509.NewCertPool()
		if !roots.AppendCertsFromPEM(caBundlePEM) {
			return nil, fmt.Errorf("failed to parse CA bundle PEM")
		}
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:    roots,
				MinVersion: tls.VersionTLS12,
			},
		}
	}

	return &Client{
		httpClient: httpClient,
		baseURL:    parsedURL,
		token:      token,
	}, nil
}

// BaseURL returns the base URL of the API.
func (c *Client) BaseURL() *url.URL {
	if c.baseURL == nil {
		return nil
	}
	u := *c.baseURL
	return &u
}

// SetBaseURL sets the base URL for the API.
func (c *Client) SetBaseURL(urlStr string) error {
	baseURL, err := url.Parse(strings.TrimSuffix(urlStr, "/"))
	if err != nil {
		return fmt.Errorf("failed to parse base URL %q: %w", urlStr, err)
	}

	c.baseURL = baseURL
	return nil
}

// GetSecret fetches a single secret by name from the specified folder path.
func (c *Client) GetSecret(ctx context.Context, name string, folderPath *string) (*btsutil.KV, error) {
	path := formatPath(folderPath)

	endpoint := fmt.Sprintf("%s/static/%s", c.baseURL.String(), url.PathEscape(name))

	// Add folder query parameter if specified
	if folderPath != nil && *folderPath != "" {
		endpoint += fmt.Sprintf("?folder=%s", url.QueryEscape(*folderPath))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch secret %q at %q: %w", name, path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode == http.StatusOK {
		var kv btsutil.KV
		if err := json.Unmarshal(body, &kv); err != nil {
			return nil, fmt.Errorf("failed to unmarshal secret response: %w", err)
		}
		return &kv, nil
	}

	return nil, parseError(body, resp.StatusCode, fmt.Sprintf("%s/%s", path, name))
}

// GetSecrets fetches a list of secrets at the specified folder path.
func (c *Client) GetSecrets(ctx context.Context, folderPath *string) ([]btsutil.KVListItem, error) {
	path := formatPath(folderPath)

	endpoint := fmt.Sprintf("%s/static", c.baseURL.String())

	// Add path query parameter if specified
	if folderPath != nil && *folderPath != "" {
		endpoint += fmt.Sprintf("?path=%s", url.QueryEscape(*folderPath))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode == http.StatusOK {
		var listResp struct {
			Data  []btsutil.KVListItem `json:"data"`
			Error string               `json:"error,omitempty"`
		}
		if err := json.Unmarshal(body, &listResp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal list response: %w", err)
		}

		// Check for API error in response body even with 200 status
		if listResp.Error != "" {
			return nil, fmt.Errorf("beyondtrust API error: %s", listResp.Error)
		}

		// Empty folder is valid - return empty list
		if len(listResp.Data) == 0 {
			return []btsutil.KVListItem{}, nil
		}

		return listResp.Data, nil
	}

	return nil, parseError(body, resp.StatusCode, path)
}

// GenerateDynamicSecret calls the dynamic secret generation endpoint.
func (c *Client) GenerateDynamicSecret(ctx context.Context, secretName string, folderPath *string) (*btsutil.GeneratedSecret, error) {
	path := formatPath(folderPath)

	endpoint := fmt.Sprintf("%s/dynamic/%s/generate", c.baseURL.String(), url.PathEscape(secretName))

	// Add folder query parameter if specified
	if folderPath != nil && *folderPath != "" {
		endpoint += fmt.Sprintf("?folder=%s", url.QueryEscape(*folderPath))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate dynamic secret: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
		var wrapped struct {
			Secret btsutil.GeneratedSecret `json:"secret"`
		}
		if err := json.Unmarshal(body, &wrapped); err != nil {
			return nil, fmt.Errorf("failed to unmarshal generated secret response: %w", err)
		}
		return &wrapped.Secret, nil
	}

	return nil, parseError(body, resp.StatusCode, path)
}

// setHeaders adds required headers to the HTTP request.
func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	req.Header.Set(apiVersionHeader, apiVersion)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
}

// validateServerURL checks if the provided server URL is valid.
func validateServerURL(server string) error {
	server = strings.TrimSpace(server)
	if server == "" {
		return fmt.Errorf("server URL is required")
	}

	if _, err := url.ParseRequestURI(server); err != nil {
		return fmt.Errorf("invalid server URL %q: %w", server, err)
	}

	return nil
}

// formatPath returns the string value of the given path pointer.
func formatPath(pathPtr *string) string {
	if pathPtr == nil || *pathPtr == "" {
		return "/"
	}
	return *pathPtr
}

// parseError attempts to parse an error response from the API.
func parseError(body []byte, statusCode int, path string) error {
	var errResp errorResponse

	// Try to parse structured error response
	if err := json.Unmarshal(body, &errResp); err == nil {
		var msg strings.Builder

		if errResp.Error != "" {
			msg.WriteString(errResp.Error)
		}

		if errResp.Message != "" {
			if msg.Len() > 0 {
				msg.WriteString(": ")
			}
			msg.WriteString(errResp.Message)
		}

		// Include details if present
		if len(errResp.Details) > 0 {
			detailsJSON, _ := json.Marshal(errResp.Details)
			if msg.Len() > 0 {
				msg.WriteString(" ")
			}
			msg.WriteString(fmt.Sprintf("(details: %s)", string(detailsJSON)))
		}

		if msg.Len() > 0 {
			return &APIError{
				StatusCode: statusCode,
				Message:    fmt.Sprintf("API error (HTTP %d): %s at path %q", statusCode, msg.String(), path),
				Path:       path,
			}
		}
	}

	// Fallback error
	return &APIError{
		StatusCode: statusCode,
		Message:    fmt.Sprintf("API error (HTTP %d): unexpected response at path %q", statusCode, path),
		Path:       path,
	}
}

// Ensure Client implements btsutil.Client interface.
var _ btsutil.Client = (*Client)(nil)

// NewBeyondtrustSecretsClient is a wrapper for backward compatibility.
func NewBeyondtrustSecretsClient(server, token string) (btsutil.Client, error) {
	return NewClient(server, token)
}

// NewBeyondtrustSecretsClientWithCustomCA is a wrapper for backward compatibility.
func NewBeyondtrustSecretsClientWithCustomCA(server, token string, caBundlePEM []byte) (btsutil.Client, error) {
	return NewClientWithCustomCA(server, token, caBundlePEM)
}
