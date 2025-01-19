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
const errJSONSecretUnmarshal = "unable to unmarshal secret: %w"
const errNoAccessToken = "no access token was set"
const errAccessTokenAlreadySet = "access token already set"

// checkError checks for an error on the http response and generates an appropriate error if one is
// found.
func checkError(resp *http.Response) error {
	if resp.StatusCode == 200 {
		return nil
	}

	var errRes InfisicalAPIErrorResponse
	err := ReadAndUnmarshal(resp, &errRes)
	if err != nil {
		return errors.New("unexpected error: " + resp.Status)
	}

	if resp.StatusCode == 404 {
		return esv1beta1.NoSecretError{}
	} else if errRes.Message != "" {
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, errRes.Message)
	} else {
		return errors.New("API error (%d): could not unmarshal error response" + resp.Status)
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
		return errors.New(errAccessTokenAlreadySet)
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
		return errors.New(errNoAccessToken)
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
func (a *InfisicalClient) do(endpoint string, method string, params map[string]string, body any, response any) error {
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

	if err = checkError(resp); err != nil {
		return err
	}

	err = ReadAndUnmarshal(resp, response)
	if err != nil {
		return fmt.Errorf(errJSONSecretUnmarshal, err)
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

func ReadAndUnmarshal(resp *http.Response, target any) error {
	var buf bytes.Buffer
	defer resp.Body.Close()
	_, err := buf.ReadFrom(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(buf.Bytes(), target)
}
