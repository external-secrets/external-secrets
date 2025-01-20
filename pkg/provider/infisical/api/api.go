/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or impliec.
See the License for the specific language governing permissions and
limitations under the License.
*/

package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/metrics"
	"github.com/external-secrets/external-secrets/pkg/provider/infisical/constants"
)

type InfisicalClient struct {
	BaseURL *url.URL
	client  *http.Client
	token   string
}

type InfisicalApis interface {
	MachineIdentityLoginViaUniversalAuth(data MachineIdentityUniversalAuthLoginRequest) (*MachineIdentityDetailsResponse, error)
	GetSecretsV3(data GetSecretsV3Request) (map[string]string, error)
	GetSecretByKeyV3(data GetSecretByKeyV3Request) (string, error)
	RevokeAccessToken() error
}

const (
	machineIdentityLoginViaUniversalAuth = "MachineIdentityLoginViaUniversalAuth"
	getSecretsV3                         = "GetSecretsV3"
	getSecretByKeyV3                     = "GetSecretByKeyV3"
	revokeAccessToken                    = "RevokeAccessToken"
)

const UserAgentName = "k8-external-secrets-operator"

var errJSONUnmarshal = errors.New("unable to unmarshal API response")
var errNoAccessToken = errors.New("unexpected error: no access token available to revoke")
var errAccessTokenAlreadyRetrieved = errors.New("unexpected error: access token was already retrieved")

type InfisicalAPIError struct {
	StatusCode int
	Err        any
	Message    any
	Details    any
}

func (e *InfisicalAPIError) Error() string {
	if e.Details != nil {
		detailsJSON, _ := json.Marshal(e.Details)
		return fmt.Sprintf("API error (%d): error=%v message=%v, details=%s", e.StatusCode, e.Err, e.Message, string(detailsJSON))
	} else {
		return fmt.Sprintf("API error (%d): error=%v message=%v", e.StatusCode, e.Err, e.Message)
	}
}

// checkError checks for an error on the http response and generates an appropriate error if one is
// found.
func checkError(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return nil
	}

	var buf bytes.Buffer
	_, err := buf.ReadFrom(resp.Body)
	if err != nil {
		return fmt.Errorf("API error (%d) and failed to read response body: %w", resp.StatusCode, err)
	}

	// Attempt to unmarshal the response body into an InfisicalAPIErrorResponse.
	var errRes InfisicalAPIErrorResponse
	err = json.Unmarshal(buf.Bytes(), &errRes)
	// Non-200 errors that cannot be unmarshaled must be handled, as errors could come from outside of
	// Infisical.
	if err != nil {
		return fmt.Errorf("API error (%d), could not unmarshal error response: %w", resp.StatusCode, err)
	} else if errRes.StatusCode == 0 {
		// When the InfisicalResponse has a zero-value status code, then the
		// response was either malformed or not from Infisical. Instead, just return
		// the error string from the response.
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, buf.String())
	} else {
		return &InfisicalAPIError{
			StatusCode: resp.StatusCode,
			Message:    errRes.Message,
			Err:        errRes.Error,
			Details:    errRes.Details,
		}
	}
}

func NewAPIClient(baseURL string, client *http.Client) (*InfisicalClient, error) {
	baseParsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	api := &InfisicalClient{
		BaseURL: baseParsedURL,
		client:  client,
	}

	return api, nil
}

func (a *InfisicalClient) SetTokenViaMachineIdentity(clientID, clientSecret string) error {
	if a.token != "" {
		return errAccessTokenAlreadyRetrieved
	}

	var loginResponse MachineIdentityDetailsResponse
	err := a.do(
		"api/v1/auth/universal-auth/login",
		http.MethodPost,
		map[string]string{},
		MachineIdentityUniversalAuthLoginRequest{
			ClientID:     clientID,
			ClientSecret: clientSecret,
		},
		&loginResponse,
	)
	metrics.ObserveAPICall(constants.ProviderName, machineIdentityLoginViaUniversalAuth, err)

	if err != nil {
		return err
	}

	a.token = loginResponse.AccessToken
	return nil
}

func (a *InfisicalClient) RevokeAccessToken() error {
	if a.token == "" {
		return errNoAccessToken
	}

	var revokeResponse RevokeMachineIdentityAccessTokenResponse
	err := a.do(
		"api/v1/auth/token/revoke",
		http.MethodPost,
		map[string]string{},
		RevokeMachineIdentityAccessTokenRequest{AccessToken: a.token},
		&revokeResponse,
	)
	metrics.ObserveAPICall(constants.ProviderName, revokeAccessToken, err)

	if err != nil {
		return err
	}

	a.token = ""
	return nil
}

func (a *InfisicalClient) resolveEndpoint(path string) string {
	return a.BaseURL.ResolveReference(&url.URL{Path: path}).String()
}

func (a *InfisicalClient) addHeaders(r *http.Request) {
	if a.token != "" {
		r.Header.Add("Authorization", "Bearer "+a.token)
	}
	r.Header.Add("User-Agent", UserAgentName)
	r.Header.Add("Content-Type", "application/json")
}

// do is a generic function that makes an API call to the Infisical API, and handle the response
// (including if an API error is returned).
func (a *InfisicalClient) do(endpoint, method string, params map[string]string, body, response any) error {
	endpointURL := a.resolveEndpoint(endpoint)

	bodyReader, err := MarshalReqBody(body)
	if err != nil {
		return err
	}

	r, err := http.NewRequest(method, endpointURL, bodyReader)
	if err != nil {
		return err
	}

	a.addHeaders(r)

	q := r.URL.Query()
	for key, value := range params {
		q.Add(key, value)
	}
	r.URL.RawQuery = q.Encode()

	resp, err := a.client.Do(r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := checkError(resp); err != nil {
		return err
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(bodyBytes, response)
	if err != nil {
		// Importantly, we do not include the response in the actual error to avoid
		// leaking anything sensitive.
		return errJSONUnmarshal
	}

	return nil
}

func (a *InfisicalClient) GetSecretsV3(data GetSecretsV3Request) (map[string]string, error) {
	params := map[string]string{
		"workspaceSlug":          data.ProjectSlug,
		"environment":            data.EnvironmentSlug,
		"secretPath":             data.SecretPath,
		"include_imports":        "true",
		"expandSecretReferences": "true",
		"recursive":              strconv.FormatBool(data.Recursive),
	}

	res := GetSecretsV3Response{}
	err := a.do(
		"api/v3/secrets/raw",
		http.MethodGet,
		params,
		http.NoBody,
		&res,
	)
	metrics.ObserveAPICall(constants.ProviderName, getSecretsV3, err)
	if err != nil {
		return nil, err
	}

	secrets := make(map[string]string)
	for _, s := range res.ImportedSecrets {
		for _, el := range s.Secrets {
			secrets[el.SecretKey] = el.SecretValue
		}
	}
	for _, el := range res.Secrets {
		secrets[el.SecretKey] = el.SecretValue
	}

	return secrets, nil
}

func (a *InfisicalClient) GetSecretByKeyV3(data GetSecretByKeyV3Request) (string, error) {
	params := map[string]string{
		"workspaceSlug":   data.ProjectSlug,
		"environment":     data.EnvironmentSlug,
		"secretPath":      data.SecretPath,
		"include_imports": "true",
	}

	endpointURL := fmt.Sprintf("api/v3/secrets/raw/%s", data.SecretKey)

	res := GetSecretByKeyV3Response{}
	err := a.do(
		endpointURL,
		http.MethodGet,
		params,
		http.NoBody,
		&res,
	)
	metrics.ObserveAPICall(constants.ProviderName, getSecretByKeyV3, err)
	if err != nil {
		var apiErr *InfisicalAPIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			return "", esv1beta1.NoSecretError{}
		}
		return "", err
	}

	return res.Secret.SecretValue, nil
}

func MarshalReqBody(data any) (*bytes.Reader, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(body), nil
}
