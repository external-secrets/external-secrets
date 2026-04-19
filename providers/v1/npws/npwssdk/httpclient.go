// /*
// Copyright © The ESO Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package npwssdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// HTTPClient handles HTTP communication with the NPWS server.
type HTTPClient struct {
	client      *http.Client
	baseURL     string
	token2      *AuthenticationToken2
	apiKeyToken string
}

// NewHTTPClient creates a new HTTP client for the given base URL.
func NewHTTPClient(baseURL string) *HTTPClient {
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	return &HTTPClient{
		client:  &http.Client{},
		baseURL: baseURL,
	}
}

// SetHTTPClient replaces the underlying http.Client (e.g. for custom TLS settings).
func (c *HTTPClient) SetHTTPClient(hc *http.Client) {
	c.client = hc
}

// SetAuth sets the authentication headers for subsequent requests.
func (c *HTTPClient) SetAuth(token2 *AuthenticationToken2, apiKeyToken string) {
	c.token2 = token2
	c.apiKeyToken = apiKeyToken
}

// ClearAuth removes authentication headers.
func (c *HTTPClient) ClearAuth() {
	c.token2 = nil
	c.apiKeyToken = ""
}

// Post sends a POST request with JSON body and deserializes the response.
func (c *HTTPClient) Post(ctx context.Context, path string, body, result interface{}) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("HTTP POST %s: marshal: %w", path, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("HTTP POST %s: %w", path, err)
	}

	req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	c.applyAuthHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP POST %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	return c.handleResponse(resp, path, result)
}

// Get sends a GET request with optional query parameters.
func (c *HTTPClient) Get(ctx context.Context, path string, params map[string]string, result interface{}) error {
	reqURL := c.baseURL + path
	if len(params) > 0 {
		q := url.Values{}
		for k, v := range params {
			q.Set(k, v)
		}
		reqURL += "?" + q.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("HTTP GET %s: %w", path, err)
	}

	c.applyAuthHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP GET %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	return c.handleResponse(resp, path, result)
}

// applyAuthHeaders adds token2 and ApiKeyToken headers if set.
func (c *HTTPClient) applyAuthHeaders(req *http.Request) {
	if c.token2 != nil {
		serialized, err := c.token2.Serialize()
		if err == nil {
			req.Header.Set("token2", serialized)
		}
	}
	if c.apiKeyToken != "" {
		req.Header.Set("ApiKeyToken", c.apiKeyToken)
	}
}

// handleResponse checks the HTTP status and optionally deserializes JSON.
func (c *HTTPClient) handleResponse(resp *http.Response, path string, result interface{}) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("HTTP %s: reading body: %w", path, err)
	}

	// Strip UTF-8 BOM if present (common with .NET servers)
	body = bytes.TrimPrefix(body, []byte("\xef\xbb\xbf"))

	switch resp.StatusCode {
	case http.StatusOK:
		if result != nil && len(body) > 0 {
			if err := json.Unmarshal(body, result); err != nil {
				return fmt.Errorf("HTTP %s: unmarshal: %w", path, err)
			}
		}
		return nil
	case http.StatusUnauthorized:
		return fmt.Errorf("HTTP %s: 401 Unauthorized: %s", path, string(body))
	case http.StatusForbidden:
		return fmt.Errorf("HTTP %s: 403 Forbidden: %s", path, string(body))
	default:
		return fmt.Errorf("HTTP %s: status %d: %s", path, resp.StatusCode, string(body))
	}
}
