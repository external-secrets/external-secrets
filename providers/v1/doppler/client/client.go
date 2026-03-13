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

// Package client provides the client implementation for interacting with Doppler's API.
package client

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DopplerClient represents a client for interacting with Doppler's API.
type DopplerClient struct {
	baseURL      *url.URL
	DopplerToken string
	VerifyTLS    bool
	UserAgent    string
}

type queryParams map[string]string

type headers map[string]string

type httpRequestBody []byte

// Secrets represents a map of secret names to their values.
type Secrets map[string]string

// Change represents a request to modify a secret in Doppler.
type Change struct {
	Name         string  `json:"name"`
	OriginalName string  `json:"originalName"`
	Value        *string `json:"value"`
	ShouldDelete bool    `json:"shouldDelete,omitempty"`
}

// APIError represents an error returned by the Doppler API.
type APIError struct {
	Err     error
	Message string
	Data    string
}

type apiResponse struct {
	HTTPResponse *http.Response
	Body         []byte
}

type apiErrorResponse struct {
	Messages []string
	Success  bool
}

// SecretRequest represents a request to retrieve a single secret.
type SecretRequest struct {
	Name    string
	Project string
	Config  string
	ETag    string
}

// SecretsRequest represents a request to retrieve multiple secrets.
type SecretsRequest struct {
	Project         string
	Config          string
	NameTransformer string
	Format          string
	ETag            string
}

// UpdateSecretsRequest represents a request to update secrets in Doppler.
type UpdateSecretsRequest struct {
	Secrets        Secrets  `json:"secrets,omitempty"`
	ChangeRequests []Change `json:"change_requests,omitempty"`
	Project        string   `json:"project,omitempty"`
	Config         string   `json:"config,omitempty"`
}

// SecretResponse represents the response from retrieving a secret.
type SecretResponse struct {
	Name     string
	Value    string
	Modified bool
	ETag     string
}

// SecretsResponse represents the response from retrieving multiple secrets.
type SecretsResponse struct {
	Secrets  Secrets
	Body     []byte
	Modified bool
	ETag     string
}

// NewDopplerClient creates a new Doppler API client.
func NewDopplerClient(dopplerToken string) (*DopplerClient, error) {
	client := &DopplerClient{
		DopplerToken: dopplerToken,
		VerifyTLS:    true,
		UserAgent:    "doppler-external-secrets",
	}

	if err := client.SetBaseURL("https://api.doppler.com"); err != nil {
		return nil, &APIError{Err: err, Message: "setting base URL failed"}
	}

	return client, nil
}

// BaseURL returns the base URL of the Doppler API.
func (c *DopplerClient) BaseURL() *url.URL {
	u := *c.baseURL
	return &u
}

// SetBaseURL sets the base URL for the Doppler API.
func (c *DopplerClient) SetBaseURL(urlStr string) error {
	baseURL, err := url.Parse(strings.TrimSuffix(urlStr, "/"))

	if err != nil {
		return err
	}

	if baseURL.Scheme == "" {
		baseURL.Scheme = "https"
	}

	c.baseURL = baseURL
	return nil
}

// Authenticate validates the authentication credentials.
func (c *DopplerClient) Authenticate() error {
	//  Choose projects as a lightweight endpoint for testing authentication
	if _, err := c.performRequest("/v3/projects", "GET", headers{}, queryParams{}, httpRequestBody{}); err != nil {
		return err
	}

	return nil
}

// GetSecret retrieves a secret from Doppler.
func (c *DopplerClient) GetSecret(request SecretRequest) (*SecretResponse, error) {
	hdrs := headers{}
	if request.ETag != "" {
		hdrs["if-none-match"] = request.ETag
	}

	params := queryParams{}
	if request.Project != "" {
		params["project"] = request.Project
	}
	if request.Config != "" {
		params["config"] = request.Config
	}
	params["secrets"] = request.Name

	response, err := c.performRequest("/v3/configs/config/secrets/download", "GET", hdrs, params, httpRequestBody{})
	if err != nil {
		return nil, err
	}

	if response.HTTPResponse.StatusCode == 304 {
		return &SecretResponse{Modified: false, ETag: request.ETag}, nil
	}

	eTag := response.HTTPResponse.Header.Get("etag")

	var secrets Secrets
	if err := json.Unmarshal(response.Body, &secrets); err != nil {
		return nil, &APIError{Err: err, Message: "unable to unmarshal secret payload", Data: string(response.Body)}
	}

	value, ok := secrets[request.Name]
	if !ok {
		return nil, &APIError{Message: fmt.Sprintf("secret '%s' not found", request.Name)}
	}

	return &SecretResponse{Name: request.Name, Value: value, Modified: true, ETag: eTag}, nil
}

// GetSecrets should only have an ETag supplied if Secrets are cached as SecretsResponse.Secrets will be nil if 304 (not modified) returned.
func (c *DopplerClient) GetSecrets(request SecretsRequest) (*SecretsResponse, error) {
	headers := headers{}
	if request.ETag != "" {
		headers["if-none-match"] = request.ETag
	}
	if request.Format != "" && request.Format != "json" {
		headers["accept"] = "text/plain"
	}

	params := request.buildQueryParams()
	response, apiErr := c.performRequest("/v3/configs/config/secrets/download", "GET", headers, params, httpRequestBody{})
	if apiErr != nil {
		return nil, apiErr
	}

	if response.HTTPResponse.StatusCode == 304 {
		return &SecretsResponse{Modified: false, Secrets: nil, ETag: request.ETag}, nil
	}

	eTag := response.HTTPResponse.Header.Get("etag")

	// Format defeats JSON parsing
	if request.Format != "" {
		return &SecretsResponse{Modified: true, Body: response.Body, ETag: eTag}, nil
	}

	var secrets Secrets
	if err := json.Unmarshal(response.Body, &secrets); err != nil {
		return nil, &APIError{Err: err, Message: "unable to unmarshal secrets payload"}
	}
	return &SecretsResponse{Modified: true, Secrets: secrets, Body: response.Body, ETag: eTag}, nil
}

// UpdateSecrets updates secrets in Doppler.
func (c *DopplerClient) UpdateSecrets(request UpdateSecretsRequest) error {
	body, jsonErr := json.Marshal(request)
	if jsonErr != nil {
		return &APIError{Err: jsonErr, Message: "unable to unmarshal update secrets payload"}
	}
	_, err := c.performRequest("/v3/configs/config/secrets", "POST", headers{}, queryParams{}, body)
	if err != nil {
		return err
	}
	return nil
}

func (r *SecretsRequest) buildQueryParams() queryParams {
	params := queryParams{}

	if r.Project != "" {
		params["project"] = r.Project
	}

	if r.Config != "" {
		params["config"] = r.Config
	}

	if r.NameTransformer != "" {
		params["name_transformer"] = r.NameTransformer
	}

	if r.Format != "" {
		params["format"] = r.Format
	}

	return params
}

func (c *DopplerClient) performRequest(path, method string, headers headers, params queryParams, body httpRequestBody) (*apiResponse, error) {
	urlStr := c.BaseURL().String() + path
	reqURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, &APIError{Err: err, Message: fmt.Sprintf("invalid API URL: %s", urlStr)}
	}

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	} else {
		bodyReader = http.NoBody
	}

	req, err := http.NewRequest(method, reqURL.String(), bodyReader)
	if err != nil {
		return nil, &APIError{Err: err, Message: "unable to form HTTP request"}
	}

	if method == "POST" && req.Header.Get("content-type") == "" {
		req.Header.Set("content-type", "application/json")
	}

	if req.Header.Get("accept") == "" {
		req.Header.Set("accept", "application/json")
	}
	req.Header.Set("user-agent", c.UserAgent)
	req.SetBasicAuth(c.DopplerToken, "")

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	query := req.URL.Query()
	for key, value := range params {
		query.Add(key, value)
	}
	req.URL.RawQuery = query.Encode()

	httpClient := &http.Client{Timeout: 10 * time.Second}

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	if !c.VerifyTLS {
		tlsConfig.InsecureSkipVerify = true
	}

	httpClient.Transport = &http.Transport{
		DisableKeepAlives: true,
		TLSClientConfig:   tlsConfig,
	}

	r, err := httpClient.Do(req)
	if err != nil {
		return nil, &APIError{Err: err, Message: "unable to load response"}
	}
	defer func() {
		_ = r.Body.Close()
	}()

	bodyResponse, err := io.ReadAll(r.Body)
	if err != nil {
		return &apiResponse{HTTPResponse: r, Body: nil}, &APIError{Err: err, Message: "unable to read entire response body"}
	}

	response := &apiResponse{HTTPResponse: r, Body: bodyResponse}
	success := isSuccess(r.StatusCode)

	if !success {
		if contentType := r.Header.Get("content-type"); strings.HasPrefix(contentType, "application/json") {
			var errResponse apiErrorResponse
			err := json.Unmarshal(bodyResponse, &errResponse)
			if err != nil {
				return response, &APIError{Err: err, Message: "unable to unmarshal error JSON payload"}
			}
			return response, &APIError{Err: nil, Message: strings.Join(errResponse.Messages, "\n")}
		}
		return nil, &APIError{Err: fmt.Errorf("%d status code; %d bytes", r.StatusCode, len(bodyResponse)), Message: "unable to load response"}
	}

	if success && err != nil {
		return nil, &APIError{Err: err, Message: "unable to load data from successful response"}
	}
	return response, nil
}

func isSuccess(statusCode int) bool {
	return (statusCode >= 200 && statusCode <= 299) || (statusCode >= 300 && statusCode <= 399)
}

func (e *APIError) Error() string {
	message := fmt.Sprintf("Doppler API Client Error: %s", e.Message)
	if underlyingError := e.Err; underlyingError != nil {
		message = fmt.Sprintf("%s\n%s", message, underlyingError.Error())
	}
	if e.Data != "" {
		message = fmt.Sprintf("%s\nData: %s", message, e.Data)
	}
	return message
}
