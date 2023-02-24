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

	aesdecrypt "github.com/Onboardbase/go-cryptojs-aes-decrypt/decrypt"
)

type OnboardbaseClient struct {
	baseURL      *url.URL
	OnboardbaseAPIKey string
	VerifyTLS    bool
	UserAgent    string
	OnboardbasePassCode string
}

type queryParams map[string]string

type headers map[string]string

type httpRequestBody []byte

type Secrets map[string]string

type RawSecret struct {
	Key interface{} `json:"key,omitempty"`
	Value interface{} `json:"value,omitempty"`
}

type RawSecrets []RawSecret

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

type SecretsRequest struct {
	Environment    string
	Project string
}

type UpdateSecretsRequest struct {
	Secrets RawSecrets `json:"secrets,omitempty"`
	Project string     `json:"project,omitempty"`
	Config  string     `json:"config,omitempty"`
}

type secretResponseBodyObject struct {
	Title string `json:"title,omitempty"`
	Id string `json:"id,omitempty"`
}

type secretResponseBodyData struct {
	Project secretResponseBodyObject `json:"project,omitempty"`
	Environment secretResponseBodyObject `json:"environment,omitempty"`
	Team secretResponseBodyObject `json:"team,omitempty"`
	Secrets []string `json:"secrets,omitempty"`
}

type secretResponseBody struct {
	Data secretResponseBodyData `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
	Status string `json:"status,omitempty"`
}

type SecretResponse struct {
	Name  string
	Value string
}

type SecretsResponse struct {
	Secrets  Secrets
	Body     []byte
	Modified bool
	ETag     string
}

func NewOnboardbaseClient(onboardbaseAPIKey, onboardbasePasscode string) (*OnboardbaseClient, error) {
	client := &OnboardbaseClient{
		OnboardbaseAPIKey: onboardbaseAPIKey,
		OnboardbasePassCode: onboardbasePasscode,
		VerifyTLS:    true,
		UserAgent:    "onboardbase-external-secrets",
	}

	if err := client.SetBaseURL("https://public.Onboardbase.com/api/v1/"); err != nil {
		return nil, &APIError{Err: err, Message: "setting base URL failed"}
	}

	return client, nil
}

func (c *OnboardbaseClient) BaseURL() *url.URL {
	u := *c.baseURL
	return &u
}

func (c *OnboardbaseClient) SetBaseURL(urlStr string) error {
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

func (c *OnboardbaseClient) Authenticate() error {

	if _, err := c.performRequest("/team/members", "GET", headers{}, queryParams{	}, httpRequestBody{}); err != nil {
		return err
	}

	return nil
}

func (c *OnboardbaseClient) getSecretsFromPayload(data secretResponseBodyData) (map[string]interface{}, error) {
	kv := make(map[string]interface{})
	for _, secret := range data.Secrets {
		passphrase := c.OnboardbasePassCode
		decrypted := aesdecrypt.Run(secret, passphrase)
		fmt.Println(decrypted)

		// var decryptedJSON RawSecret
		// if err := json.Unmarshal([]byte(decrypted), &decryptedJSON); err != nil {
		//     return nil, &APIError{Err: err, Message: "unable to unmarshal secret payload", Data: decrypted}
	    // }
		// fmt.Println(decryptedJSON)
	}
	return kv, nil
}

func (c *OnboardbaseClient) GetSecret(request SecretsRequest) (*SecretResponse, error) {
	fmt.Println("Getting Secret")
	params := request.buildQueryParams()
	response, err := c.performRequest("/secrets", "GET", headers{}, params, httpRequestBody{})
	if err != nil {
		return nil, err
	}

	var data secretResponseBody
	if err := json.Unmarshal(response.Body, &data); err != nil {
		return nil, &APIError{Err: err, Message: "unable to unmarshal secret payload", Data: string(response.Body)}
	}

	secrets, _ := c.getSecretsFromPayload(data.Data)
	fmt.Println(secrets)

	// if data.Value.Computed == nil {
	// 	return nil, &APIError{Message: fmt.Sprintf("secret for project '%s' and environment '%s' not found", request.Project, request.Environment)}
	// }

	return &SecretResponse{Name: "", Value: ""}, nil
}

// GetSecrets should only have an ETag supplied if Secrets are cached as SecretsResponse.Secrets will be nil if 304 (not modified) returned.
func (c *OnboardbaseClient) GetSecrets(request SecretsRequest) (*SecretsResponse, error) {
	headers := headers{}

	params := request.buildQueryParams()
	response, apiErr := c.performRequest("/v3/configs/config/secrets/download", "GET", headers, params, httpRequestBody{})
	if apiErr != nil {
		return nil, apiErr
	}


	eTag := response.HTTPResponse.Header.Get("etag")

	var secrets Secrets
	if err := json.Unmarshal(response.Body, &secrets); err != nil {
		return nil, &APIError{Err: err, Message: "unable to unmarshal secrets payload"}
	}
	return &SecretsResponse{Modified: true, Secrets: secrets, Body: response.Body, ETag: eTag}, nil
}

func (c *OnboardbaseClient) UpdateSecrets(request UpdateSecretsRequest) error {
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


	if r.Project != "" {
		params["environment"] = r.Environment
	}

	return params
}

func (c *OnboardbaseClient) performRequest(path, method string, headers headers, params queryParams, body httpRequestBody) (*apiResponse, error) {
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
	req.Header.Set("api_key", c.OnboardbaseAPIKey)

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
		fmt.Println(r.StatusCode)

	if err != nil {
		return nil, &APIError{Err: err, Message: "unable to load response"}
	}
	defer r.Body.Close()

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
	message := fmt.Sprintf("Onboardbase API Client Error: %s", e.Message)
	if underlyingError := e.Err; underlyingError != nil {
		message = fmt.Sprintf("%s\n%s", message, underlyingError.Error())
	}
	if e.Data != "" {
		message = fmt.Sprintf("%s\nData: %s", message, e.Data)
	}
	return message
}
