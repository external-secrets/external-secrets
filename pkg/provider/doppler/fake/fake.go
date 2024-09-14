//Copyright External Secrets Inc. All Rights Reserved

package fake

import (
	"errors"
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
