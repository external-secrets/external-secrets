//Copyright External Secrets Inc. All Rights Reserved

package fake

import (
	"errors"
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
				return nil, errors.New("unexpected test argument")
			}
			return response, err
		}
	}
}
