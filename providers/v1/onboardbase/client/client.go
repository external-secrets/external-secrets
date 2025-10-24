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

// Package client implements an HTTP client for interacting with the Onboardbase API,
// providing functionality to securely retrieve and manage secrets.
package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	aesdecrypt "github.com/Onboardbase/go-cryptojs-aes-decrypt/decrypt"
)

const (
	// HTTPTimeoutDuration defines the default timeout for HTTP requests.
	HTTPTimeoutDuration = 20 * time.Second

	// ObbSecretsEndpointPath defines the endpoint path for secrets API.
	ObbSecretsEndpointPath = "/secrets"

	errUnableToDecrtypt = "unable to decrypt secret payload"
)

// OnboardbaseClient defines the interface for interacting with Onboardbase API.
type OnboardbaseClient struct {
	baseURL             *url.URL
	OnboardbaseAPIKey   string
	VerifyTLS           bool
	UserAgent           string
	OnboardbasePassCode string
	httpClient          *http.Client
}

type queryParams map[string]string

type headers map[string]string

// DeleteSecretsRequest represents a request to delete secrets from Onboardbase.
type DeleteSecretsRequest struct {
	SecretID string `json:"secretId,omitempty"`
}

type httpRequestBody []byte

// Secrets represents a map of secret key-value pairs.
type Secrets map[string]string

// RawSecret represents a raw secret from Onboardbase.
type RawSecret struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

// RawSecrets represents a collection of raw secrets.
type RawSecrets []RawSecret

// APIError represents an error response from the Onboardbase API.
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

// SecretRequest represents a request for a single secret.
type SecretRequest struct {
	Environment string
	Project     string
	Name        string
}

// SecretsRequest represents a request for multiple secrets.
type SecretsRequest struct {
	Environment string
	Project     string
}

type secretResponseBodyObject struct {
	Title string `json:"title,omitempty"`
	ID    string `json:"id,omitempty"`
}

type secretResponseSecrets struct {
	ID    string `json:"id"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

type secretResponseBodyData struct {
	Project     secretResponseBodyObject `json:"project,omitempty"`
	Environment secretResponseBodyObject `json:"environment,omitempty"`
	Team        secretResponseBodyObject `json:"team,omitempty"`
	Secrets     []secretResponseSecrets  `json:"secrets,omitempty"`
	Status      string                   `json:"status"`
	Message     string                   `json:"string"`
}

type secretResponseBody struct {
	Data    secretResponseBodyData `json:"data,omitempty"`
	Message string                 `json:"message,omitempty"`
	Status  string                 `json:"status,omitempty"`
}

// SecretResponse represents a single secret response from Onboardbase.
type SecretResponse struct {
	Name  string
	Value string
}

// SecretsResponse represents a collection of secrets from Onboardbase.
type SecretsResponse struct {
	Secrets Secrets
	Body    []byte
}

// NewOnboardbaseClient creates a new client for interacting with Onboardbase API.
// It requires an API key and passcode for authentication.
func NewOnboardbaseClient(onboardbaseAPIKey, onboardbasePasscode string) (*OnboardbaseClient, error) {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	httpTransport := &http.Transport{
		DisableKeepAlives: true,
		TLSClientConfig:   tlsConfig,
	}
	client := &OnboardbaseClient{
		OnboardbaseAPIKey:   onboardbaseAPIKey,
		OnboardbasePassCode: onboardbasePasscode,
		VerifyTLS:           true,
		UserAgent:           "onboardbase-external-secrets",
		httpClient: &http.Client{
			Timeout:   HTTPTimeoutDuration,
			Transport: httpTransport,
		},
	}

	if err := client.SetBaseURL("https://public.onboardbase.com/api/v1/"); err != nil {
		return nil, &APIError{Err: err, Message: "setting base URL failed"}
	}

	return client, nil
}

// BaseURL returns the base URL of the Onboardbase API.
func (c *OnboardbaseClient) BaseURL() *url.URL {
	u := *c.baseURL
	return &u
}

// SetBaseURL updates the base URL for the Onboardbase API client.
func (c *OnboardbaseClient) SetBaseURL(urlStr string) error {
	baseURL, err := url.Parse(strings.TrimSuffix(urlStr, "/"))

	if err != nil {
		return err
	}
	c.baseURL = baseURL
	return nil
}

// Authenticate verifies the API credentials with Onboardbase.
func (c *OnboardbaseClient) Authenticate() error {
	_, err := c.performRequest(
		&performRequestConfig{
			path:    "/team/members",
			method:  "GET",
			headers: headers{},
			params:  queryParams{},
			body:    httpRequestBody{},
		})

	if err != nil {
		return err
	}

	return nil
}

func (c *OnboardbaseClient) getSecretsFromPayload(data secretResponseBodyData) (map[string]string, error) {
	kv := make(map[string]string)
	for _, secret := range data.Secrets {
		passphrase := c.OnboardbasePassCode
		key, err := aesdecrypt.Run(secret.Key, passphrase)
		if err != nil {
			return nil, &APIError{Err: err, Message: errUnableToDecrtypt, Data: secret.Key}
		}
		value, err := aesdecrypt.Run(secret.Value, passphrase)
		if err != nil {
			return nil, &APIError{Err: err, Message: errUnableToDecrtypt, Data: secret.Value}
		}
		kv[key] = value
	}
	return kv, nil
}

func (c *OnboardbaseClient) mapSecretsByPlainKey(data secretResponseBodyData) (map[string]secretResponseSecrets, error) {
	kv := make(map[string]secretResponseSecrets)
	for _, secret := range data.Secrets {
		passphrase := c.OnboardbasePassCode
		key, err := aesdecrypt.Run(secret.Key, passphrase)
		if err != nil {
			return nil, &APIError{Err: err, Message: errUnableToDecrtypt, Data: secret.Key}
		}
		kv[key] = secret
	}
	return kv, nil
}

// GetSecret retrieves a specific secret from Onboardbase.
func (c *OnboardbaseClient) GetSecret(request SecretRequest) (*SecretResponse, error) {
	response, err := c.performRequest(
		&performRequestConfig{
			path:    ObbSecretsEndpointPath,
			method:  "GET",
			headers: headers{},
			params:  request.buildQueryParams(),
			body:    httpRequestBody{},
		})
	if err != nil {
		return nil, err
	}

	var data secretResponseBody
	if err := json.Unmarshal(response.Body, &data); err != nil {
		return nil, &APIError{Err: err, Message: "unable to unmarshal secret payload", Data: string(response.Body)}
	}

	secrets, _ := c.getSecretsFromPayload(data.Data)
	secret := secrets[request.Name]

	if secret == "" {
		return nil, &APIError{Message: fmt.Sprintf("secret %s for project '%s' and environment '%s' not found", request.Name, request.Project, request.Environment)}
	}

	return &SecretResponse{Name: request.Name, Value: secrets[request.Name]}, nil
}

// DeleteSecret removes a secret from Onboardbase.
func (c *OnboardbaseClient) DeleteSecret(request SecretRequest) error {
	secretsrequest := SecretsRequest{
		Project:     request.Project,
		Environment: request.Environment,
	}

	secretsData, _, err := c.makeGetSecretsRequest(secretsrequest)
	if err != nil {
		return err
	}
	secrets, err := c.mapSecretsByPlainKey(secretsData.Data)
	if err != nil {
		return err
	}
	secret, ok := secrets[request.Name]
	if !ok || secret.ID == "" {
		return nil
	}

	params := request.buildQueryParams()
	deleteSecretDto := &DeleteSecretsRequest{
		SecretID: secret.ID,
	}
	body, jsonErr := json.Marshal(deleteSecretDto)
	if jsonErr != nil {
		return &APIError{Err: jsonErr, Message: "unable to unmarshal delete secrets payload"}
	}
	_, err = c.performRequest(&performRequestConfig{
		path:    ObbSecretsEndpointPath,
		method:  "DELETE",
		headers: headers{},
		params:  params,
		body:    body,
	})
	if err != nil {
		return err
	}

	return nil
}

func (c *OnboardbaseClient) makeGetSecretsRequest(request SecretsRequest) (*secretResponseBody, *apiResponse, error) {
	response, apiErr := c.performRequest(&performRequestConfig{
		path:    ObbSecretsEndpointPath,
		method:  "GET",
		headers: headers{},
		params:  request.buildQueryParams(),
		body:    httpRequestBody{},
	})
	if apiErr != nil {
		return nil, nil, apiErr
	}

	var data *secretResponseBody
	if err := json.Unmarshal(response.Body, &data); err != nil {
		return nil, nil, &APIError{Err: err, Message: "unable to unmarshal secret payload", Data: string(response.Body)}
	}
	return data, response, nil
}

// GetSecrets retrieves multiple secrets from Onboardbase.
func (c *OnboardbaseClient) GetSecrets(request SecretsRequest) (*SecretsResponse, error) {
	data, response, err := c.makeGetSecretsRequest(request)
	if err != nil {
		return nil, err
	}

	secrets, _ := c.getSecretsFromPayload(data.Data)
	return &SecretsResponse{Secrets: secrets, Body: response.Body}, nil
}

func (r *SecretsRequest) buildQueryParams() queryParams {
	params := queryParams{}

	if r.Project != "" {
		params["project"] = r.Project
	}

	if r.Environment != "" {
		params["environment"] = r.Environment
	}

	return params
}

func (r *SecretRequest) buildQueryParams() queryParams {
	params := queryParams{}

	if r.Project != "" {
		params["project"] = r.Project
	}

	if r.Environment != "" {
		params["environment"] = r.Environment
	}

	return params
}

type performRequestConfig struct {
	path    string
	method  string
	headers headers
	params  queryParams
	body    httpRequestBody
}

func (c *OnboardbaseClient) performRequest(config *performRequestConfig) (*apiResponse, error) {
	urlStr := c.BaseURL().String() + config.path
	reqURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, &APIError{Err: err, Message: fmt.Sprintf("invalid API URL: %s", urlStr)}
	}

	var bodyReader io.Reader
	if config.body != nil {
		bodyReader = bytes.NewReader(config.body)
	} else {
		bodyReader = http.NoBody
	}

	// timeout this request after 20 seconds
	ctx, cancel := context.WithTimeout(context.Background(), HTTPTimeoutDuration)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, config.method, reqURL.String(), bodyReader)
	if err != nil {
		return nil, &APIError{Err: err, Message: "unable to form HTTP request"}
	}

	req.Header.Set("content-type", "application/json")
	req.Header.Set("user-agent", c.UserAgent)
	req.Header.Set("api_key", c.OnboardbaseAPIKey)

	for key, value := range config.headers {
		req.Header.Set(key, value)
	}

	query := req.URL.Query()
	for key, value := range config.params {
		query.Add(key, value)
	}
	req.URL.RawQuery = query.Encode()

	r, err := c.httpClient.Do(req)

	if err != nil {
		return nil, &APIError{Err: err, Message: "unable to load response"}
	}
	defer func() {
		_ = r.Body.Close()
	}()

	bodyResponse, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, &APIError{Err: err, Message: "unable to read entire response body"}
	}

	response := &apiResponse{HTTPResponse: r, Body: bodyResponse}
	success := isSuccess(r.StatusCode)

	if !success {
		return handlePerformRequestFailure(response)
	}

	if success && err != nil {
		return nil, &APIError{Err: err, Message: "unable to load data from successful response"}
	}
	return response, nil
}

func handlePerformRequestFailure(response *apiResponse) (*apiResponse, *APIError) {
	if contentType := response.HTTPResponse.Header.Get("content-type"); strings.HasPrefix(contentType, "application/json") {
		var errResponse apiErrorResponse
		err := json.Unmarshal(response.Body, &errResponse)
		if err != nil {
			return response, &APIError{Err: err, Message: "unable to unmarshal error JSON payload"}
		}
		return response, &APIError{Err: nil, Message: strings.Join(errResponse.Messages, "\n")}
	}
	return nil, &APIError{Err: fmt.Errorf("%d status code; %d bytes", response.HTTPResponse.StatusCode, len(response.Body)), Message: "unable to load response"}
}

func isSuccess(statusCode int) bool {
	return (statusCode >= 200 && statusCode <= 299) || (statusCode >= 300 && statusCode <= 399)
}

func (e *APIError) Error() string {
	message := fmt.Sprintf("Onboardbase API Client Error: %s", e.Message)
	if underlyingError := e.Err; underlyingError != nil {
		message = fmt.Sprintf("%s\n%s", message, underlyingError.Error())
	}
	if e.Data != "" {
		message = fmt.Sprintf("%s\nData: %s", message, e.Data)
	}
	return message
}
