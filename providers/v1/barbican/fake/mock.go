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

package fake

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/keymanager/v1/secrets"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

// MockKeyManagerClient is a mock implementation of the Gophercloud ServiceClient
// for testing purposes without requiring actual OpenStack services.
type MockKeyManagerClient struct {
	// Mock data storage
	secrets      map[string][]byte
	secretsInfo  map[string]secrets.Secret
	shouldError  bool
	errorMessage string
}

// NewMockKeyManagerClient creates a new mock client.
func NewMockKeyManagerClient() *MockKeyManagerClient {
	return &MockKeyManagerClient{
		secrets:     make(map[string][]byte),
		secretsInfo: make(map[string]secrets.Secret),
		shouldError: false,
	}
}

// WithSecret adds a secret to the mock client.
func (m *MockKeyManagerClient) WithSecret(uuid, name string, payload []byte) *MockKeyManagerClient {
	m.secrets[uuid] = payload
	m.secretsInfo[uuid] = secrets.Secret{
		SecretRef: fmt.Sprintf("https://barbican.example.com/v1/secrets/%s", uuid),
		Name:      name,
	}
	return m
}

// WithError configures the mock to return an error.
func (m *MockKeyManagerClient) WithError(errorMessage string) *MockKeyManagerClient {
	m.shouldError = true
	m.errorMessage = errorMessage
	return m
}

// Reset clears all mock data and error states.
func (m *MockKeyManagerClient) Reset() {
	m.secrets = make(map[string][]byte)
	m.secretsInfo = make(map[string]secrets.Secret)
	m.shouldError = false
	m.errorMessage = ""
}

// GetPayload mocks the secrets.GetPayload function.
func (m *MockKeyManagerClient) GetPayload(_ context.Context, _ *gophercloud.ServiceClient, uuid string, _ secrets.GetPayloadOptsBuilder) ([]byte, error) {
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}

	payload, exists := m.secrets[uuid]
	if !exists {
		return nil, fmt.Errorf("secret with UUID %s not found", uuid)
	}

	return payload, nil
}

// ListSecrets mocks the secrets.List function.
func (m *MockKeyManagerClient) ListSecrets(_ context.Context, _ *gophercloud.ServiceClient, opts secrets.ListOptsBuilder) ([]secrets.Secret, error) {
	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}

	var result = make([]secrets.Secret, 10)
	for _, secret := range m.secretsInfo {
		// Apply name filter if provided
		if opts != nil {
			if listOpts, ok := opts.(secrets.ListOpts); ok && listOpts.Name != "" {
				if secret.Name != listOpts.Name {
					continue
				}
			}
		}
		result = append(result, secret)
	}

	return result, nil
}

// MockPagination implements pagination.Page for testing.
type MockPagination struct {
	secrets []secrets.Secret
}

func (p MockPagination) NextPageURL() (string, error) {
	return "", nil
}

func (p MockPagination) IsEmpty() (bool, error) {
	return len(p.secrets) == 0, nil
}

func (p MockPagination) LastMarker() (string, error) {
	return "", nil
}

func (p MockPagination) GetBody() interface{} {
	return map[string]interface{}{
		"secrets": p.secrets,
	}
}

// MockPager implements pagination.Pager for testing.
type MockPager struct {
	page MockPagination
}

func (p MockPager) AllPages(_ context.Context) (pagination.Page, error) {
	return p.page, nil
}

func (p MockPager) EachPage(_ context.Context, fn func(pagination.Page) (bool, error)) error {
	cont, err := fn(p.page)
	if err != nil {
		return err
	}
	_ = cont
	return nil
}

// NewMockPager creates a new mock pager with the provided secrets.
func NewMockPager(secrets []secrets.Secret) MockPager {
	return MockPager{
		page: MockPagination{secrets: secrets},
	}
}
