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

package bitwarden

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

// Defined Header Keys.
const (
	WardenHeaderAccessToken = "Warden-Access-Token"
	WardenHeaderAPIURL      = "Warden-Api-Url"
	WardenHeaderIdentityURL = "Warden-Identity-Url"
)

type SecretResponse struct {
	CreationDate   string  `json:"creationDate"`
	ID             string  `json:"id"`
	Key            string  `json:"key"`
	Note           string  `json:"note"`
	OrganizationID string  `json:"organizationId"`
	ProjectID      *string `json:"projectId,omitempty"`
	RevisionDate   string  `json:"revisionDate"`
	Value          string  `json:"value"`
}

type SecretsDeleteResponse struct {
	Data []SecretDeleteResponse `json:"data"`
}

type SecretDeleteResponse struct {
	Error *string `json:"error,omitempty"`
	ID    string  `json:"id"`
}

type SecretIdentifiersResponse struct {
	Data []SecretIdentifierResponse `json:"data"`
}

type SecretIdentifierResponse struct {
	ID             string `json:"id"`
	Key            string `json:"key"`
	OrganizationID string `json:"organizationId"`
}

type SecretCreateRequest struct {
	Key  string `json:"key"`
	Note string `json:"note"`
	// Organization where the secret will be created
	OrganizationID string `json:"organizationId"`
	// IDs of the projects that this secret will belong to
	ProjectIDS []string `json:"projectIds,omitempty"`
	Value      string   `json:"value"`
}

type SecretPutRequest struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Note string `json:"note"`
	// Organization where the secret will be created
	OrganizationID string `json:"organizationId"`
	// IDs of the projects that this secret will belong to
	ProjectIDS []string `json:"projectIds,omitempty"`
	Value      string   `json:"value"`
}

// Client for the bitwarden SDK.
type Client interface {
	GetSecret(ctx context.Context, id string) (*SecretResponse, error)
	DeleteSecret(ctx context.Context, ids []string) (*SecretsDeleteResponse, error)
	CreateSecret(ctx context.Context, secret SecretCreateRequest) (*SecretResponse, error)
	UpdateSecret(ctx context.Context, secret SecretPutRequest) (*SecretResponse, error)
	ListSecrets(ctx context.Context, organizationID string) (*SecretIdentifiersResponse, error)
}

// SdkClient creates a client to talk to the bitwarden SDK server.
type SdkClient struct {
	apiURL                string
	identityURL           string
	token                 string
	bitwardenSdkServerURL string

	client *http.Client
}

func NewSdkClient(ctx context.Context, c client.Client, storeKind, namespace string, provider *v1beta1.BitwardenSecretsManagerProvider, token string) (*SdkClient, error) {
	httpsClient, err := newHTTPSClient(ctx, c, storeKind, namespace, provider)
	if err != nil {
		return nil, fmt.Errorf("error creating https client: %w", err)
	}

	return &SdkClient{
		apiURL:                provider.APIURL,
		identityURL:           provider.IdentityURL,
		bitwardenSdkServerURL: provider.BitwardenServerSDKURL,
		token:                 token,
		client:                httpsClient,
	}, nil
}

func (s *SdkClient) GetSecret(ctx context.Context, id string) (*SecretResponse, error) {
	body := struct {
		ID string `json:"id"`
	}{
		ID: id,
	}
	secretResp := &SecretResponse{}

	if err := s.performHTTPRequestOperation(ctx, params{
		method: http.MethodGet,
		url:    s.bitwardenSdkServerURL + "/rest/api/1/secret",
		body:   body,
		result: &secretResp,
	}); err != nil {
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}

	return secretResp, nil
}

func (s *SdkClient) DeleteSecret(ctx context.Context, ids []string) (*SecretsDeleteResponse, error) {
	body := struct {
		IDs []string `json:"ids"`
	}{
		IDs: ids,
	}

	secretResp := &SecretsDeleteResponse{}
	if err := s.performHTTPRequestOperation(ctx, params{
		method: http.MethodDelete,
		url:    s.bitwardenSdkServerURL + "/rest/api/1/secret",
		body:   body,
		result: &secretResp,
	}); err != nil {
		return nil, fmt.Errorf("failed to delete secret: %w", err)
	}

	return secretResp, nil
}

func (s *SdkClient) CreateSecret(ctx context.Context, createReq SecretCreateRequest) (*SecretResponse, error) {
	secretResp := &SecretResponse{}
	if err := s.performHTTPRequestOperation(ctx, params{
		method: http.MethodPost,
		url:    s.bitwardenSdkServerURL + "/rest/api/1/secret",
		body:   createReq,
		result: &secretResp,
	}); err != nil {
		return nil, fmt.Errorf("failed to create secret: %w", err)
	}

	return secretResp, nil
}

func (s *SdkClient) UpdateSecret(ctx context.Context, putReq SecretPutRequest) (*SecretResponse, error) {
	secretResp := &SecretResponse{}
	if err := s.performHTTPRequestOperation(ctx, params{
		method: http.MethodPut,
		url:    s.bitwardenSdkServerURL + "/rest/api/1/secret",
		body:   putReq,
		result: &secretResp,
	}); err != nil {
		return nil, fmt.Errorf("failed to update secret: %w", err)
	}

	return secretResp, nil
}

func (s *SdkClient) ListSecrets(ctx context.Context, organizationID string) (*SecretIdentifiersResponse, error) {
	body := struct {
		ID string `json:"organizationID"`
	}{
		ID: organizationID,
	}
	secretResp := &SecretIdentifiersResponse{}
	if err := s.performHTTPRequestOperation(ctx, params{
		method: http.MethodGet,
		url:    s.bitwardenSdkServerURL + "/rest/api/1/secrets",
		body:   body,
		result: &secretResp,
	}); err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	return secretResp, nil
}

func (s *SdkClient) constructSdkRequest(ctx context.Context, method, url string, body []byte) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to construct request: %w", err)
	}

	req.Header.Set(WardenHeaderAccessToken, s.token)
	req.Header.Set(WardenHeaderAPIURL, s.apiURL)
	req.Header.Set(WardenHeaderIdentityURL, s.identityURL)

	return req, nil
}

type params struct {
	method string
	url    string
	body   any
	result any
}

func (s *SdkClient) performHTTPRequestOperation(ctx context.Context, params params) error {
	data, err := json.Marshal(params.body)
	if err != nil {
		return fmt.Errorf("failed to marshal body: %w", err)
	}

	req, err := s.constructSdkRequest(ctx, params.method, params.url, data)
	if err != nil {
		return fmt.Errorf("failed to construct request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		content, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("failed to perform http request, got response: %s with status code %d", string(content), resp.StatusCode)
	}

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&params.result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}
