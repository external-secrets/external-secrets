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
	"net/url"
	"time"

	"k8s.io/client-go/util/retry"

	dclient "github.com/external-secrets/external-secrets/providers/v1/doppler/client"
)

// retryableClient wraps a Doppler client with retry logic.
type retryableClient struct {
	client     SecretsClientInterface
	maxRetries int
	retryDelay time.Duration
}

// newRetryableClient creates a new retryable Doppler client wrapper.
func newRetryableClient(client SecretsClientInterface, maxRetries int, retryInterval time.Duration) *retryableClient {
	return &retryableClient{
		client:     client,
		maxRetries: maxRetries,
		retryDelay: retryInterval,
	}
}

// BaseURL returns the base URL of the wrapped client.
func (r *retryableClient) BaseURL() *url.URL {
	return r.client.BaseURL()
}

// Authenticate authenticates with retry logic.
func (r *retryableClient) Authenticate() error {
	backoff := retry.DefaultBackoff
	if r.retryDelay > 0 {
		backoff.Duration = r.retryDelay
	}
	if r.maxRetries > 0 {
		backoff.Steps = r.maxRetries
	}
	return retry.OnError(backoff, func(error) bool { return true }, func() error {
		return r.client.Authenticate()
	})
}

// GetSecret retrieves a secret with retry logic.
func (r *retryableClient) GetSecret(request dclient.SecretRequest) (*dclient.SecretResponse, error) {
	var result *dclient.SecretResponse
	backoff := retry.DefaultBackoff
	if r.retryDelay > 0 {
		backoff.Duration = r.retryDelay
	}
	if r.maxRetries > 0 {
		backoff.Steps = r.maxRetries
	}
	err := retry.OnError(backoff, func(error) bool { return true }, func() error {
		var err error
		result, err = r.client.GetSecret(request)
		return err
	})
	return result, err
}

// GetSecrets retrieves secrets with retry logic.
func (r *retryableClient) GetSecrets(request dclient.SecretsRequest) (*dclient.SecretsResponse, error) {
	var result *dclient.SecretsResponse
	backoff := retry.DefaultBackoff
	if r.retryDelay > 0 {
		backoff.Duration = r.retryDelay
	}
	if r.maxRetries > 0 {
		backoff.Steps = r.maxRetries
	}
	err := retry.OnError(backoff, func(error) bool { return true }, func() error {
		var err error
		result, err = r.client.GetSecrets(request)
		return err
	})
	return result, err
}

// UpdateSecrets updates secrets with retry logic.
func (r *retryableClient) UpdateSecrets(request dclient.UpdateSecretsRequest) error {
	backoff := retry.DefaultBackoff
	if r.retryDelay > 0 {
		backoff.Duration = r.retryDelay
	}
	if r.maxRetries > 0 {
		backoff.Steps = r.maxRetries
	}
	return retry.OnError(backoff, func(error) bool { return true }, func() error {
		return r.client.UpdateSecrets(request)
	})
}
