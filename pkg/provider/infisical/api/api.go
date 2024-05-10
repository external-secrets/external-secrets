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

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/metrics"
	"github.com/external-secrets/external-secrets/pkg/provider/infisical/constants"
)

type InfisicalClient struct {
	BaseURL      *url.URL
	client       *http.Client
	tokenManager TokenManager
}

type InfisicalApis interface {
	RefreshMachineIdentityAccessToken(data MachineIdentityUniversalAuthRefreshRequest) (*MachineIdentityDetailsResponse, error)
	MachineIdentityLoginViaUniversalAuth(data MachineIdentityUniversalAuthLoginRequest) (*MachineIdentityDetailsResponse, error)
	GetSecretsV3(data GetSecretsV3Request) (map[string]string, error)
	GetSecretByKeyV3(data GetSecretByKeyV3Request) (string, error)
}

type TokenManager interface {
	GetAccessToken() (string, error)
}

const UserAgentName = "k8-external-secrets-operator"
const errJSONSecretUnmarshal = "unable to unmarshal secret: %w"

func NewAPIClient(baseURL string) (*InfisicalClient, error) {
	baseParsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	api := &InfisicalClient{
		BaseURL: baseParsedURL,
		client:  &http.Client{},
	}

	return api, nil
}

func (a *InfisicalClient) SetTokenManager(tk TokenManager) {
	a.tokenManager = tk
}

func (a *InfisicalClient) resolveEndpoint(path string) string {
	return a.BaseURL.ResolveReference(&url.URL{Path: path}).String()
}

func (a *InfisicalClient) do(r *http.Request) (*http.Response, error) {
	if accessToken, err := a.tokenManager.GetAccessToken(); err == nil {
		r.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	}
	r.Header.Add("User-Agent", UserAgentName)
	r.Header.Add("Content-Type", "application/json")

	return a.client.Do(r)
}

func (a *InfisicalClient) RefreshMachineIdentityAccessToken(data MachineIdentityUniversalAuthRefreshRequest) (*MachineIdentityDetailsResponse, error) {
	endpointURL := a.resolveEndpoint("api/v1/auth/token/renew")
	body, err := MarhalReqBody(data)
	if err != nil {
		return nil, err
	}

	refreshTokenReq, err := http.NewRequest(http.MethodPost, endpointURL, body)
	metrics.ObserveAPICall(constants.ProviderName, "RefreshMachineIdentityAccessToken", err)
	if err != nil {
		return nil, err
	}

	rawRes, err := a.do(refreshTokenReq) //nolint:bodyclose // linters bug
	if err != nil {
		return nil, err
	}

	var res MachineIdentityDetailsResponse
	err = ReadAndUnmarshal(rawRes, &res)
	if err != nil {
		return nil, fmt.Errorf(errJSONSecretUnmarshal, err)
	}
	return &res, nil
}

func (a *InfisicalClient) MachineIdentityLoginViaUniversalAuth(data MachineIdentityUniversalAuthLoginRequest) (*MachineIdentityDetailsResponse, error) {
	endpointURL := a.resolveEndpoint("api/v1/auth/universal-auth/login")
	body, err := MarhalReqBody(data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, endpointURL, body)
	metrics.ObserveAPICall(constants.ProviderName, "MachineIdentityLoginViaUniversalAuth", err)
	if err != nil {
		return nil, err
	}

	rawRes, err := a.do(req) //nolint:bodyclose // linters bug
	if err != nil {
		return nil, err
	}

	var res MachineIdentityDetailsResponse
	err = ReadAndUnmarshal(rawRes, &res)
	if err != nil {
		return nil, fmt.Errorf(errJSONSecretUnmarshal, err)
	}
	return &res, nil
}

func (a *InfisicalClient) GetSecretsV3(data GetSecretsV3Request) (map[string]string, error) {
	endpointURL := a.resolveEndpoint("api/v3/secrets/raw")

	req, err := http.NewRequest(http.MethodGet, endpointURL, http.NoBody)
	q := req.URL.Query()
	q.Add("workspaceSlug", data.ProjectSlug)
	q.Add("environment", data.EnvironmentSlug)
	q.Add("secretPath", data.SecretPath)
	q.Add("include_imports", "true")
	q.Add("expandSecretReferences", "true")
	req.URL.RawQuery = q.Encode()

	metrics.ObserveAPICall(constants.ProviderName, "GetSecretsV3", err)
	if err != nil {
		return nil, err
	}

	rawRes, err := a.do(req) //nolint:bodyclose // linters bug
	if err != nil {
		return nil, err
	}

	var res GetSecretsV3Response
	err = ReadAndUnmarshal(rawRes, &res)
	if err != nil {
		return nil, fmt.Errorf(errJSONSecretUnmarshal, err)
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
	endpointURL := a.resolveEndpoint(fmt.Sprintf("api/v3/secrets/raw/%s", data.SecretKey))

	req, err := http.NewRequest(http.MethodGet, endpointURL, http.NoBody)
	q := req.URL.Query()
	q.Add("workspaceSlug", data.ProjectSlug)
	q.Add("environment", data.EnvironmentSlug)
	q.Add("secretPath", data.SecretPath)
	q.Add("include_imports", "true")
	req.URL.RawQuery = q.Encode()

	metrics.ObserveAPICall(constants.ProviderName, "GetSecretByKeyV3", err)
	if err != nil {
		return "", err
	}

	rawRes, err := a.do(req) //nolint:bodyclose // linters bug
	if err != nil {
		return "", err
	}
	if rawRes.StatusCode == 400 {
		var errRes InfisicalAPIErrorResponse
		err = ReadAndUnmarshal(rawRes, &errRes)
		if err != nil {
			return "", fmt.Errorf(errJSONSecretUnmarshal, err)
		}

		if errRes.Message == "Secret not found" {
			return "", esv1beta1.NoSecretError{}
		}
		return "", errors.New(errRes.Message)
	}

	var res GetSecretByKeyV3Response
	err = ReadAndUnmarshal(rawRes, &res)
	if err != nil {
		return "", fmt.Errorf(errJSONSecretUnmarshal, err)
	}

	return res.Secret.SecretValue, nil
}

func MarhalReqBody(data any) (*bytes.Reader, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(body), nil
}

func ReadAndUnmarshal(resp *http.Response, target any) error {
	var buf bytes.Buffer
	defer func() {
		if resp.Body != nil {
			resp.Body.Close()
		}
	}()
	_, err := buf.ReadFrom(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(buf.Bytes(), target)
}
