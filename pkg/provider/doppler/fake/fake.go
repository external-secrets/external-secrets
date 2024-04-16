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

package fake

import (
	"fmt"
	"net/url"

	"github.com/google/go-cmp/cmp"

	"github.com/external-secrets/external-secrets/pkg/provider/doppler/client"
)

type DopplerClient struct {
	getSecret     func(request client.SecretRequest) (*client.SecretResponse, error)
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

func (dc *DopplerClient) GetSecrets(_ client.SecretsRequest) (*client.SecretsResponse, error) {
	// Not implemented
	return &client.SecretsResponse{}, nil
}

func (dc *DopplerClient) UpdateSecrets(request client.UpdateSecretsRequest) error {
	return dc.updateSecrets(request)
}

func (dc *DopplerClient) WithValue(request client.SecretRequest, response *client.SecretResponse, err error) {
	if dc != nil {
		dc.getSecret = func(requestIn client.SecretRequest) (*client.SecretResponse, error) {
			if !cmp.Equal(requestIn, request) {
				return nil, fmt.Errorf("unexpected test argument")
			}
			return response, err
		}
	}
}

func (dc *DopplerClient) WithUpdateValue(request client.UpdateSecretsRequest, err error) {
	if dc != nil {
		dc.updateSecrets = func(requestIn client.UpdateSecretsRequest) error {
			if !cmp.Equal(requestIn, request) {
				return fmt.Errorf("unexpected test argument")
			}
			return err
		}
	}
}
