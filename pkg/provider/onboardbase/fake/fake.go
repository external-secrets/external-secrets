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

package fake

import (
	"fmt"
	"net/url"

	"github.com/google/go-cmp/cmp"

	"github.com/external-secrets/external-secrets/pkg/provider/onboardbase/client"
)

type OnboardbaseClient struct {
	getSecret func(request client.SecretRequest) (*client.SecretResponse, error)
}

func (obbc *OnboardbaseClient) BaseURL() *url.URL {
	return &url.URL{Scheme: "https", Host: "public.onboardbase.com"}
}

func (obbc *OnboardbaseClient) Authenticate() error {
	return nil
}

func (obbc *OnboardbaseClient) GetSecret(request client.SecretRequest) (*client.SecretResponse, error) {
	return obbc.getSecret(request)
}

func (obbc *OnboardbaseClient) GetSecrets(_ client.SecretsRequest) (*client.SecretsResponse, error) {
	return &client.SecretsResponse{}, nil
}

func (obbc *OnboardbaseClient) DeleteSecret(_ client.SecretRequest) error {
	return nil
}

func (obbc *OnboardbaseClient) WithValue(request client.SecretRequest, response *client.SecretResponse, err error) {
	if obbc != nil {
		obbc.getSecret = func(requestIn client.SecretRequest) (*client.SecretResponse, error) {
			if !cmp.Equal(requestIn, request) {
				return nil, fmt.Errorf("unexpected test argument")
			}
			return response, err
		}
	}
}
