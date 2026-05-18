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

package secretmanager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mws.cloud/go-sdk/mws/errors"
	common "go.mws.cloud/go-sdk/service/common/model"
	"go.mws.cloud/go-sdk/service/secretmanager/client"
	"go.mws.cloud/go-sdk/service/secretmanager/model"
	"go.mws.cloud/go-sdk/service/secretmanager/sdk"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	v1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

type fakeSecretVersion struct {
	client.SecretVersion

	secretVersions map[string]map[string]model.SecretVersionDataSpec2
}

func (fake *fakeSecretVersion) GetData(ctx context.Context, req client.GetDataRequest) (*client.GetDataResponse, error) {
	secret, ok := fake.secretVersions[req.Name]
	if !ok {
		return &client.GetDataResponse{
			Response404: &common.ApiError{},
		}, nil
	}

	data, ok := secret[req.Version]
	if !ok {
		return &client.GetDataResponse{
			Response404: &common.ApiError{},
		}, nil
	}

	return &client.GetDataResponse{
		Response200: data,
	}, nil
}

func TestGetSecret(t *testing.T) {
	ctx := context.Background()

	fake := &fakeSecretVersion{
		secretVersions: map[string]map[string]model.SecretVersionDataSpec2{
			"secret-1": {
				"version-1": {
					"property-1": "data-1",
					"property-2": "data-2",
				},
				"current": {
					"property-3": "data-3",
				},
			},
		},
	}

	client := &Client{
		secretVersion: &sdk.SecretVersion{
			SecretVersionSugared: client.NewSecretVersionSugared(fake),
		},
	}

	_, err := client.GetSecret(ctx, v1.ExternalSecretDataRemoteRef{
		Key:     "non-existent",
		Version: "non-existent",
	})
	assert.ErrorIs(t, err, &errors.APIError{})

	data, err := client.GetSecret(ctx, v1.ExternalSecretDataRemoteRef{
		Key:     "secret-1",
		Version: "version-1",
	})
	assert.NoError(t, err)
	assert.JSONEq(t, `{"property-1":"data-1","property-2":"data-2"}`, string(data))

	data, err = client.GetSecret(ctx, v1.ExternalSecretDataRemoteRef{
		Key: "secret-1",
	})
	assert.NoError(t, err)
	assert.JSONEq(t, `{"property-3":"data-3"}`, string(data))

	data, err = client.GetSecret(ctx, v1.ExternalSecretDataRemoteRef{
		Key:      "secret-1",
		Version:  "version-1",
		Property: "property-1",
	})
	assert.NoError(t, err)
	assert.Equal(t, "data-1", string(data))

	data, err = client.GetSecret(ctx, v1.ExternalSecretDataRemoteRef{
		Key:      "secret-1",
		Property: "property-3",
	})
	assert.NoError(t, err)
	assert.Equal(t, "data-3", string(data))

	_, err = client.GetSecret(ctx, v1.ExternalSecretDataRemoteRef{
		Key:      "secret-1",
		Version:  "version-1",
		Property: "non-existent",
	})
	assert.ErrorContains(t, err, "does not contain property")
}

func TestGetSecretMap(t *testing.T) {
	ctx := context.Background()

	fake := &fakeSecretVersion{
		secretVersions: map[string]map[string]model.SecretVersionDataSpec2{
			"secret-1": {
				"version-1": {
					"property-1": "data-1",
					"property-2": "data-2",
				},
				"current": {
					"property-3": "data-3",
				},
			},
		},
	}

	client := &Client{
		secretVersion: &sdk.SecretVersion{
			SecretVersionSugared: client.NewSecretVersionSugared(fake),
		},
	}

	_, err := client.GetSecretMap(ctx, v1.ExternalSecretDataRemoteRef{
		Key:     "non-existent",
		Version: "non-existent",
	})
	assert.ErrorIs(t, err, &errors.APIError{})

	data, err := client.GetSecretMap(ctx, v1.ExternalSecretDataRemoteRef{
		Key:     "secret-1",
		Version: "version-1",
	})
	assert.NoError(t, err)
	assert.Equal(t, map[string][]byte{
		"property-1": []byte("data-1"),
		"property-2": []byte("data-2"),
	}, data)

	data, err = client.GetSecretMap(ctx, v1.ExternalSecretDataRemoteRef{
		Key: "secret-1",
	})
	assert.NoError(t, err)
	assert.Equal(t, map[string][]byte{
		"property-3": []byte("data-3"),
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
