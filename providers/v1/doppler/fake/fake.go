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
	"errors"
	"net/url"

	"github.com/google/go-cmp/cmp"

	"github.com/external-secrets/external-secrets/providers/v1/doppler/client"
)

type DopplerClient struct {
	getSecret     func(request client.SecretRequest) (*client.SecretResponse, error)
	getSecrets    func(request client.SecretsRequest) (*client.SecretsResponse, error)
	updateSecrets func(request client.UpdateSecretsRequest) error
}

func (dc *DopplerClient) BaseURL() *url.URL {
	return &url.URL{Scheme: "https", Host: "api.doppler.com"}
}

func (dc *DopplerClient) Authenticate() error {
	return nil
}

func (dc *DopplerClient) GetSecret(request client.SecretRequest) (*client.SecretResponse, error) {
	return dc.getSecret(request)
}

func (dc *DopplerClient) GetSecrets(request client.SecretsRequest) (*client.SecretsResponse, error) {
	if dc.getSecrets != nil {
		return dc.getSecrets(request)
	}
	// Default: return empty response with Modified=true (simulates fresh fetch)
	return &client.SecretsResponse{Modified: true}, nil
}

func (dc *DopplerClient) UpdateSecrets(request client.UpdateSecretsRequest) error {
	return dc.updateSecrets(request)
}

func (dc *DopplerClient) WithValue(request client.SecretRequest, response *client.SecretResponse, err error) {
	if dc != nil {
		dc.getSecret = func(requestIn client.SecretRequest) (*client.SecretResponse, error) {
			if !cmp.Equal(requestIn, request) {
				return nil, errors.New("unexpected test argument")
			}
			return response, err
		}
	}
}

func (dc *DopplerClient) WithUpdateValue(request client.UpdateSecretsRequest, err error) {
	if dc != nil {
		dc.updateSecrets = func(requestIn client.UpdateSecretsRequest) error {
			if !cmp.Equal(requestIn, request) {
				return errors.New("unexpected test argument")
			}
			return err
		}
	}
}

func (dc *DopplerClient) WithSecretsValue(response *client.SecretsResponse, err error) {
	if dc != nil {
		dc.getSecrets = func(_ client.SecretsRequest) (*client.SecretsResponse, error) {
			return response, err
		}
	}
}

func (dc *DopplerClient) WithSecretsFunc(fn func(request client.SecretsRequest) (*client.SecretsResponse, error)) {
	if dc != nil {
		dc.getSecrets = fn
	}
}

func (dc *DopplerClient) WithSecretFunc(fn func(request client.SecretRequest) (*client.SecretResponse, error)) {
	if dc != nil {
		dc.getSecret = fn
	}
}
