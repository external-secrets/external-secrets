/*
Copyright © The ESO Authors

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

package certificatemanager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mws.cloud/go-sdk/mws/errors"
	"go.mws.cloud/go-sdk/service/certmanager/client"
	"go.mws.cloud/go-sdk/service/certmanager/model"
	"go.mws.cloud/go-sdk/service/certmanager/sdk"
	common "go.mws.cloud/go-sdk/service/common/model"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	v1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

type fakeCertificate struct {
	client.Certificate

	certificates map[string]*model.CertificateContentResponse
}

func (fake *fakeCertificate) GetCertificateContent(ctx context.Context, req client.GetCertificateContentRequest) (*client.GetCertificateContentResponse, error) {
	certificate, ok := fake.certificates[req.Name]
	if !ok {
		return &client.GetCertificateContentResponse{
			Response404: &common.ApiError{},
		}, nil
	}

	return &client.GetCertificateContentResponse{
		Response200: certificate,
	}, nil
}

func TestGetSecret(t *testing.T) {
	ctx := context.Background()

	fake := &fakeCertificate{
		certificates: map[string]*model.CertificateContentResponse{
			"test-cert-1": {
				Certificate: "certificate-1",
				PrivateKey:  "private-key-1",
				ChainedCert: new("chained-cert-1"),
			},
			"test-cert-2": {
				Certificate: "certificate-2",
				PrivateKey:  "private-key-2",
			},
		},
	}

	client := &Client{
		certificate: &sdk.Certificate{
			CertificateSugared: client.NewCertificateSugared(fake),
		},
	}

	_, err := client.GetSecret(ctx, v1.ExternalSecretDataRemoteRef{
		Key: "non-existent",
	})
	assert.ErrorIs(t, err, &errors.APIError{})

	data, err := client.GetSecret(ctx, v1.ExternalSecretDataRemoteRef{
		Key: "test-cert-1",
	})
	assert.NoError(t, err)
	assert.JSONEq(t, `{"certificate":"certificate-1","privateKey":"private-key-1","chainedCert":"chained-cert-1"}`, string(data))

	data, err = client.GetSecret(ctx, v1.ExternalSecretDataRemoteRef{
		Key: "test-cert-2",
	})
	assert.NoError(t, err)
	assert.JSONEq(t, `{"certificate":"certificate-2","privateKey":"private-key-2"}`, string(data))

	data, err = client.GetSecret(ctx, v1.ExternalSecretDataRemoteRef{
		Key:      "test-cert-1",
		Property: "certificate",
	})
	assert.NoError(t, err)
	assert.Equal(t, "certificate-1", string(data))

	data, err = client.GetSecret(ctx, v1.ExternalSecretDataRemoteRef{
		Key:      "test-cert-1",
		Property: "privateKey",
	})
	assert.NoError(t, err)
	assert.Equal(t, "private-key-1", string(data))

	data, err = client.GetSecret(ctx, v1.ExternalSecretDataRemoteRef{
		Key:      "test-cert-1",
		Property: "chainedCert",
	})
	assert.NoError(t, err)
	assert.Equal(t, "chained-cert-1", string(data))

	_, err = client.GetSecret(ctx, v1.ExternalSecretDataRemoteRef{
		Key:      "test-cert-2",
		Property: "chainedCert",
	})
	assert.ErrorIs(t, err, errUnsetChainedCert)

	_, err = client.GetSecret(ctx, v1.ExternalSecretDataRemoteRef{
		Key:      "test-cert-2",
		Property: "invalid-property",
	})
	assert.ErrorContains(t, err, "invalid certificate property")
}

func TestGetSecretMap(t *testing.T) {
	ctx := context.Background()

	fake := &fakeCertificate{
		certificates: map[string]*model.CertificateContentResponse{
			"test-cert-1": {
				Certificate: "certificate-1",
				PrivateKey:  "private-key-1",
				ChainedCert: new("chained-cert-1"),
			},
			"test-cert-2": {
				Certificate: "certificate-2",
				PrivateKey:  "private-key-2",
			},
		},
	}

	client := &Client{
		certificate: &sdk.Certificate{
			CertificateSugared: client.NewCertificateSugared(fake),
		},
	}

	_, err := client.GetSecretMap(ctx, v1.ExternalSecretDataRemoteRef{
		Key: "non-existent",
	})
	assert.ErrorIs(t, err, &errors.APIError{})

	data, err := client.GetSecretMap(ctx, v1.ExternalSecretDataRemoteRef{
		Key: "test-cert-1",
	})
	assert.NoError(t, err)
	assert.Equal(t, map[string][]byte{
		"certificate": []byte("certificate-1"),
		"privateKey":  []byte("private-key-1"),
		"chainedCert": []byte("chained-cert-1"),
	}, data)

	data, err = client.GetSecretMap(ctx, v1.ExternalSecretDataRemoteRef{
		Key: "test-cert-2",
	})
	assert.NoError(t, err)
	assert.Equal(t, map[string][]byte{
		"certificate": []byte("certificate-2"),
		"privateKey":  []byte("private-key-2"),
	}, data)
}

func TestGetAllSecrets(t *testing.T) {
	ctx := context.Background()
	client := &Client{}

	_, err := client.GetAllSecrets(ctx, v1.ExternalSecretFind{})
	assert.ErrorIs(t, err, errNotImplemented)
}

func TestPushSecret(t *testing.T) {
	ctx := context.Background()
	client := &Client{}

	err := client.PushSecret(ctx, nil, nil)
	assert.ErrorIs(t, err, errNotImplemented)
}

func TestDeleteSecret(t *testing.T) {
	ctx := context.Background()
	client := &Client{}

	err := client.DeleteSecret(ctx, nil)
	assert.ErrorIs(t, err, errNotImplemented)
}

func TestSecretExists(t *testing.T) {
	ctx := context.Background()
	client := &Client{}

	_, err := client.SecretExists(ctx, nil)
	assert.ErrorIs(t, err, errNotImplemented)
}

func TestValidate(t *testing.T) {
	client := &Client{}

	result, err := client.Validate()
	assert.NoError(t, err)
	assert.Equal(t, esv1.ValidationResultUnknown, result)
}
