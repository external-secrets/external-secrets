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

func (obbc *OnboardbaseClient) GetSecrets(request client.SecretsRequest) (*client.SecretsResponse, error) {
	// Not implemented
	return &client.SecretsResponse{}, nil
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
