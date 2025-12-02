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

package doppler

import (
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/external-secrets/external-secrets/providers/v1/doppler/client"
)

const testSecretValue = "value"

// mockClient implements SecretsClientInterface for testing retry logic.
type mockClient struct {
	authenticateCalls  int
	getSecretCalls     int
	getSecretsCalls    int
	updateSecretsCalls int
	failUntilCall      int
	returnError        error
	secretResponse     *client.SecretResponse
	secretsResponse    *client.SecretsResponse
}

func (m *mockClient) BaseURL() *url.URL {
	return &url.URL{Scheme: "https", Host: "api.doppler.com"}
}

func (m *mockClient) Authenticate() error {
	m.authenticateCalls++
	if m.authenticateCalls < m.failUntilCall {
		return m.returnError
	}
	return nil
}

func (m *mockClient) GetSecret(_ client.SecretRequest) (*client.SecretResponse, error) {
	m.getSecretCalls++
	if m.getSecretCalls < m.failUntilCall {
		return nil, m.returnError
	}
	return m.secretResponse, nil
}

func (m *mockClient) GetSecrets(_ client.SecretsRequest) (*client.SecretsResponse, error) {
	m.getSecretsCalls++
	if m.getSecretsCalls < m.failUntilCall {
		return nil, m.returnError
	}
	return m.secretsResponse, nil
}

func (m *mockClient) UpdateSecrets(_ client.UpdateSecretsRequest) error {
	m.updateSecretsCalls++
	if m.updateSecretsCalls < m.failUntilCall {
		return m.returnError
	}
	return nil
}

func TestRetryClientSuccessOnFirstAttempt(t *testing.T) {
	mock := &mockClient{
		failUntilCall:   1, // succeed on first call
		secretResponse:  &client.SecretResponse{Name: "test", Value: testSecretValue},
		secretsResponse: &client.SecretsResponse{Secrets: client.Secrets{"test": testSecretValue}},
	}

	retryClient := newRetryableClient(mock, 3, 10*time.Millisecond)

	// Test Authenticate
	if err := retryClient.Authenticate(); err != nil {
		t.Errorf("Authenticate should succeed on first attempt, got error: %v", err)
	}
	if mock.authenticateCalls != 1 {
		t.Errorf("Expected 1 authenticate call, got %d", mock.authenticateCalls)
	}

	// Test GetSecret
	resp, err := retryClient.GetSecret(client.SecretRequest{Name: "test"})
	if err != nil {
		t.Errorf("GetSecret should succeed on first attempt, got error: %v", err)
	}
	if resp == nil || resp.Value != testSecretValue {
		t.Errorf("GetSecret returned unexpected response: %v", resp)
	}
	if mock.getSecretCalls != 1 {
		t.Errorf("Expected 1 getSecret call, got %d", mock.getSecretCalls)
	}

	// Test GetSecrets
	respSecrets, err := retryClient.GetSecrets(client.SecretsRequest{})
	if err != nil {
		t.Errorf("GetSecrets should succeed on first attempt, got error: %v", err)
	}
	if respSecrets == nil || respSecrets.Secrets["test"] != testSecretValue {
		t.Errorf("GetSecrets returned unexpected response: %v", respSecrets)
	}
	if mock.getSecretsCalls != 1 {
		t.Errorf("Expected 1 getSecrets call, got %d", mock.getSecretsCalls)
	}

	// Test UpdateSecrets
	if err := retryClient.UpdateSecrets(client.UpdateSecretsRequest{}); err != nil {
		t.Errorf("UpdateSecrets should succeed on first attempt, got error: %v", err)
	}
	if mock.updateSecretsCalls != 1 {
		t.Errorf("Expected 1 updateSecrets call, got %d", mock.updateSecretsCalls)
	}
}

func TestRetryClientSuccessAfterRetries(t *testing.T) {
	testError := errors.New("temporary error")
	mock := &mockClient{
		failUntilCall:   3, // fail twice, succeed on third attempt
		returnError:     testError,
		secretResponse:  &client.SecretResponse{Name: "test", Value: testSecretValue},
		secretsResponse: &client.SecretsResponse{Secrets: client.Secrets{"test": testSecretValue}},
	}

	retryClient := newRetryableClient(mock, 5, 1*time.Millisecond)

	// Test Authenticate - should retry and eventually succeed
	if err := retryClient.Authenticate(); err != nil {
		t.Errorf("Authenticate should succeed after retries, got error: %v", err)
	}
	if mock.authenticateCalls != 3 {
		t.Errorf("Expected 3 authenticate calls (2 failures + 1 success), got %d", mock.authenticateCalls)
	}

	// Reset for GetSecret test
	mock.getSecretCalls = 0
	resp, err := retryClient.GetSecret(client.SecretRequest{Name: "test"})
	if err != nil {
		t.Errorf("GetSecret should succeed after retries, got error: %v", err)
	}
	if resp == nil || resp.Value != testSecretValue {
		t.Errorf("GetSecret returned unexpected response: %v", resp)
	}
	if mock.getSecretCalls != 3 {
		t.Errorf("Expected 3 getSecret calls (2 failures + 1 success), got %d", mock.getSecretCalls)
	}
}

func TestRetryClientFailureAfterMaxRetries(t *testing.T) {
	testError := errors.New("persistent error")
	mock := &mockClient{
		failUntilCall: 100, // always fail
		returnError:   testError,
	}

	retryClient := newRetryableClient(mock, 3, 1*time.Millisecond)

	// Test Authenticate - should fail after max retries
	err := retryClient.Authenticate()
	if err == nil {
		t.Error("Authenticate should fail after max retries")
	}
	if !errors.Is(err, testError) {
		t.Errorf("Expected error %v, got %v", testError, err)
	}
	// With maxRetries=3, backoff.Steps=3 means 3 total attempts
	if mock.authenticateCalls != 3 {
		t.Errorf("Expected 3 authenticate calls, got %d", mock.authenticateCalls)
	}

	// Reset for GetSecret test
	mock.getSecretCalls = 0
	_, err = retryClient.GetSecret(client.SecretRequest{Name: "test"})
	if err == nil {
		t.Error("GetSecret should fail after max retries")
	}
	if !errors.Is(err, testError) {
		t.Errorf("Expected error %v, got %v", testError, err)
	}
	if mock.getSecretCalls != 3 {
		t.Errorf("Expected 3 getSecret calls, got %d", mock.getSecretCalls)
	}
}

func TestRetryClientRetryInterval(t *testing.T) {
	testError := errors.New("temporary error")
	mock := &mockClient{
		failUntilCall: 3, // fail twice
		returnError:   testError,
	}

	retryInterval := 20 * time.Millisecond
	retryClient := newRetryableClient(mock, 5, retryInterval)

	start := time.Now()
	_ = retryClient.Authenticate()
	elapsed := time.Since(start)

	// Should have waited at least retryInterval (for the first retry)
	// Note: DefaultBackoff has Factor=5.0 and Jitter=0.1, so delays increase exponentially
	// We just verify it took some reasonable time with retries
	minExpected := retryInterval
	if elapsed < minExpected {
		t.Errorf("Expected at least %v elapsed time for retries, got %v", minExpected, elapsed)
	}

	// Sanity check - with exponential backoff (Factor=5.0), shouldn't take too excessively long
	// First retry: ~20ms, Second retry: ~100ms (20ms * 5), Total: ~120ms + execution time
	maxExpected := 1 * time.Second
	if elapsed > maxExpected {
		t.Errorf("Expected less than %v elapsed time, got %v (may indicate exponential backoff issue)", maxExpected, elapsed)
	}
}

func TestRetryClientBaseURL(t *testing.T) {
	mock := &mockClient{}
	retryClient := newRetryableClient(mock, 3, 10*time.Millisecond)

	baseURL := retryClient.BaseURL()
	if baseURL == nil {
		t.Error("BaseURL should not be nil")
	}
	if baseURL.Host != "api.doppler.com" {
		t.Errorf("Expected host 'api.doppler.com', got '%s'", baseURL.Host)
	}
}

func TestRetryClientZeroRetries(t *testing.T) {
	testError := errors.New("error")
	mock := &mockClient{
		failUntilCall: 5, // fail multiple times
		returnError:   testError,
	}

	// maxRetries = 0 means we don't override Steps, so it uses DefaultBackoff.Steps = 4
	retryClient := newRetryableClient(mock, 0, 1*time.Millisecond)

	err := retryClient.Authenticate()
	if err == nil {
		t.Error("Expected error with failing calls")
	}
	// With maxRetries=0, Steps is not overridden, so it uses DefaultBackoff.Steps=4
	if mock.authenticateCalls != 4 {
		t.Errorf("Expected 4 authenticate calls (DefaultBackoff.Steps), got %d", mock.authenticateCalls)
	}
}
