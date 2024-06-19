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

// SdkClient creates a client to talk to the bitwarden SDK server.
type SdkClient struct {
	apiURL                string
	identityURL           string
	token                 string
	bitwardenSdkServerURL string

	client *http.Client
}

func NewSdkClient(apiURL, identityURL, bitwardenURL, token string, caBundle []byte) (*SdkClient, error) {
	client, err := newHTTPSClient(caBundle)
	if err != nil {
		return nil, fmt.Errorf("error creating https client: %w", err)
	}

	return &SdkClient{
		apiURL:                apiURL,
		identityURL:           identityURL,
		token:                 token,
		client:                client,
		bitwardenSdkServerURL: bitwardenURL,
	}, nil
}

// TODO: Clean all of this up and refactor the duplications.

func (s *SdkClient) GetSecret(ctx context.Context, id string) (*SecretResponse, error) {
	body := struct {
		ID string `json:"id"`
	}{
		ID: id,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal body: %w", err)
	}

	req, err := s.constructSdkRequest(ctx, http.MethodGet, s.bitwardenSdkServerURL+"/rest/api/1/secret", data)
	if err != nil {
		return nil, fmt.Errorf("failed to construct request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		content, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("failed to get secret by id %s, got response: %s", id, string(content))
	}

	decoder := json.NewDecoder(resp.Body)

	var secretResp *SecretResponse
	if err := decoder.Decode(&secretResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return secretResp, nil
}

func (s *SdkClient) DeleteSecret(ctx context.Context, ids []string) (*SecretsDeleteResponse, error) {
	body := struct {
		IDs []string `json:"ids"`
	}{
		IDs: ids,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal body: %w", err)
	}

	req, err := s.constructSdkRequest(ctx, http.MethodDelete, s.bitwardenSdkServerURL+"/rest/api/1/secret", data)
	if err != nil {
		return nil, fmt.Errorf("failed to construct request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		content, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("failed to delete secrets, got response: %s", string(content))
	}

	decoder := json.NewDecoder(resp.Body)

	var secretResp *SecretsDeleteResponse
	if err := decoder.Decode(&secretResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return secretResp, nil
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

func (s *SdkClient) CreateSecret(ctx context.Context, createReq SecretCreateRequest) (*SecretResponse, error) {
	data, err := json.Marshal(createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal body: %w", err)
	}

	req, err := s.constructSdkRequest(ctx, http.MethodPost, s.bitwardenSdkServerURL+"/rest/api/1/secret", data)
	if err != nil {
		return nil, fmt.Errorf("failed to construct request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		content, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("failed to get secrets, got response: %s", string(content))
	}

	decoder := json.NewDecoder(resp.Body)

	var secretResp *SecretResponse
	if err := decoder.Decode(&secretResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return secretResp, nil
}

func (s *SdkClient) ListSecrets(ctx context.Context, organizationID string) (*SecretIdentifiersResponse, error) {
	body := struct {
		ID string `json:"organizationID"`
	}{
		ID: organizationID,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal body: %w", err)
	}

	req, err := s.constructSdkRequest(ctx, http.MethodGet, s.bitwardenSdkServerURL+"/rest/api/1/secrets", data)
	if err != nil {
		return nil, fmt.Errorf("failed to construct request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		content, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("failed to get secrets, got response: %s", string(content))
	}

	decoder := json.NewDecoder(resp.Body)

	var secretResp *SecretIdentifiersResponse
	if err := decoder.Decode(&secretResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
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
