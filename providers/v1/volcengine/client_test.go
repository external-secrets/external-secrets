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

package volcengine

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/volcengine/volcengine-go-sdk/service/kms"
	"github.com/volcengine/volcengine-go-sdk/volcengine/request"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

type fakePushSecretData struct {
	Metadata  *apiextensionsv1.JSON
	SecretKey string
	RemoteKey string
	Property  string
}

func (f fakePushSecretData) GetMetadata() *apiextensionsv1.JSON {
	return f.Metadata
}

func (f fakePushSecretData) GetSecretKey() string {
	return f.SecretKey
}

func (f fakePushSecretData) GetRemoteKey() string {
	return f.RemoteKey
}

func (f fakePushSecretData) GetProperty() string {
	return f.Property
}

type fakePushScretRemoteRef struct {
	RemoteKey string
	Property  string
}

func (f fakePushScretRemoteRef) GetRemoteKey() string {
	return f.RemoteKey
}

func (f fakePushScretRemoteRef) GetProperty() string {
	return f.Property
}

// MockKMSClient is a mock of KMSAPI interface.
type MockKMSClient struct {
	kms.KMSAPI
	DescribeRegionsFunc           func(*kms.DescribeRegionsInput) (*kms.DescribeRegionsOutput, error)
	DescribeSecretWithContextFunc func(context.Context, *kms.DescribeSecretInput, ...request.Option) (*kms.DescribeSecretOutput, error)
	GetSecretValueWithContextFunc func(context.Context, *kms.GetSecretValueInput, ...request.Option) (*kms.GetSecretValueOutput, error)
}

// DescribeRegions mocks the DescribeRegions method.
func (m *MockKMSClient) DescribeRegions(input *kms.DescribeRegionsInput) (*kms.DescribeRegionsOutput, error) {
	if m.DescribeRegionsFunc != nil {
		return m.DescribeRegionsFunc(input)
	}
	return nil, errors.New("DescribeRegions is not implemented")
}

// DescribeSecretWithContext mocks the DescribeSecretWithContext method.
func (m *MockKMSClient) DescribeSecretWithContext(ctx context.Context, input *kms.DescribeSecretInput, opts ...request.Option) (*kms.DescribeSecretOutput, error) {
	if m.DescribeSecretWithContextFunc != nil {
		return m.DescribeSecretWithContextFunc(ctx, input, opts...)
	}
	return nil, errors.New("DescribeSecretWithContext is not implemented")
}

// GetSecretValueWithContext mocks the GetSecretValueWithContext method.
func (m *MockKMSClient) GetSecretValueWithContext(ctx context.Context, input *kms.GetSecretValueInput, opts ...request.Option) (*kms.GetSecretValueOutput, error) {
	if m.GetSecretValueWithContextFunc != nil {
		return m.GetSecretValueWithContextFunc(ctx, input, opts...)
	}
	return nil, errors.New("GetSecretValueWithContext is not implemented")
}

func TestNew_should_return_a_new_client(t *testing.T) {
	mockKMS := &MockKMSClient{}
	client := NewClient(mockKMS)
	assert.NotNil(t, client)
	assert.Equal(t, mockKMS, client.kms)
}

func TestClient_PushSecret_should_return_not_implemented_error(t *testing.T) {
	client := &Client{}
	err := client.PushSecret(context.Background(), &corev1.Secret{}, &fakePushSecretData{})
	assert.Error(t, err)
	assert.Equal(t, notImplemented, err.Error())
}

func TestClient_DeleteSecret_should_return_not_implemented_error(t *testing.T) {
	client := &Client{}
	err := client.DeleteSecret(context.Background(), &fakePushSecretData{})
	assert.Error(t, err)
	assert.Equal(t, notImplemented, err.Error())
}

func TestClient_GetAllSecrets_should_return_not_implemented_error(t *testing.T) {
	client := &Client{}
	_, err := client.GetAllSecrets(context.Background(), esapi.ExternalSecretFind{})
	assert.Error(t, err)
	assert.Equal(t, notImplemented, err.Error())
}

func TestClient_Close_should_return_nil(t *testing.T) {
	client := &Client{}
	err := client.Close(context.Background())
	assert.NoError(t, err)
}

func TestClient_Validate_should_return_ready_when_kms_client_is_initialized(t *testing.T) {
	mockKMS := &MockKMSClient{
		DescribeRegionsFunc: func(*kms.DescribeRegionsInput) (*kms.DescribeRegionsOutput, error) {
			return &kms.DescribeRegionsOutput{}, nil
		},
	}
	client := NewClient(mockKMS)
	result, err := client.Validate()
	assert.NoError(t, err)
	assert.Equal(t, esapi.ValidationResultReady, result)
}

func TestClient_Validate_should_return_error_when_kms_client_is_not_initialized(t *testing.T) {
	client := NewClient(nil)
	result, err := client.Validate()
	assert.Error(t, err)
	assert.Equal(t, "kms client is not initialized", err.Error())
	assert.Equal(t, esapi.ValidationResultError, result)
}

func TestClient_GetSecret_should_return_secret_value_when_property_is_empty(t *testing.T) {
	secretValue := "my-secret-value"
	mockKMS := &MockKMSClient{
		GetSecretValueWithContextFunc: func(ctx context.Context, input *kms.GetSecretValueInput, opts ...request.Option) (*kms.GetSecretValueOutput, error) {
			return &kms.GetSecretValueOutput{
				SecretValue: &secretValue,
			}, nil
		},
	}
	client := NewClient(mockKMS)
	value, err := client.GetSecret(context.Background(), esapi.ExternalSecretDataRemoteRef{
		Key: "my-secret",
	})
	assert.NoError(t, err)
	assert.Equal(t, []byte(secretValue), value)
}

func TestClient_GetSecret_should_return_property_value_when_secret_is_json_and_property_exists(t *testing.T) {
	secretValue := `{"user":"admin","pass":"1234"}`
	mockKMS := &MockKMSClient{
		GetSecretValueWithContextFunc: func(ctx context.Context, input *kms.GetSecretValueInput, opts ...request.Option) (*kms.GetSecretValueOutput, error) {
			return &kms.GetSecretValueOutput{
				SecretValue: &secretValue,
			}, nil
		},
	}
	client := NewClient(mockKMS)
	value, err := client.GetSecret(context.Background(), esapi.ExternalSecretDataRemoteRef{
		Key:      "my-secret",
		Property: "pass",
	})
	assert.NoError(t, err)
	assert.Equal(t, []byte("1234"), value)
}

func TestClient_GetSecret_should_return_raw_json_value_when_property_is_json_object(t *testing.T) {
	secretValue := `{"config":{"foo":"bar"},"pass":"1234"}`
	mockKMS := &MockKMSClient{
		GetSecretValueWithContextFunc: func(ctx context.Context, input *kms.GetSecretValueInput, opts ...request.Option) (*kms.GetSecretValueOutput, error) {
			return &kms.GetSecretValueOutput{
				SecretValue: &secretValue,
			}, nil
		},
	}
	client := NewClient(mockKMS)
	value, err := client.GetSecret(context.Background(), esapi.ExternalSecretDataRemoteRef{
		Key:      "my-secret",
		Property: "config",
	})
	assert.NoError(t, err)
	assert.Equal(t, []byte(`{"foo":"bar"}`), value)
}

func TestClient_GetSecret_should_return_error_when_property_does_not_exist(t *testing.T) {
	secretValue := `{"user":"admin","pass":"1234"}`
	mockKMS := &MockKMSClient{
		GetSecretValueWithContextFunc: func(ctx context.Context, input *kms.GetSecretValueInput, opts ...request.Option) (*kms.GetSecretValueOutput, error) {
			return &kms.GetSecretValueOutput{
				SecretValue: &secretValue,
			}, nil
		},
	}
	client := NewClient(mockKMS)
	_, err := client.GetSecret(context.Background(), esapi.ExternalSecretDataRemoteRef{
		Key:      "my-secret",
		Property: "non-existent",
	})
	assert.Error(t, err)
	assert.Equal(t, `property "non-existent" not found in secret`, err.Error())
}

func TestClient_GetSecret_should_return_error_when_secret_is_not_valid_json(t *testing.T) {
	secretValue := `not-a-json`
	mockKMS := &MockKMSClient{
		GetSecretValueWithContextFunc: func(ctx context.Context, input *kms.GetSecretValueInput, opts ...request.Option) (*kms.GetSecretValueOutput, error) {
			return &kms.GetSecretValueOutput{
				SecretValue: &secretValue,
			}, nil
		},
	}
	client := NewClient(mockKMS)
	_, err := client.GetSecret(context.Background(), esapi.ExternalSecretDataRemoteRef{
		Key:      "my-secret",
		Property: "prop",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal secret")
}

func TestClient_GetSecret_should_return_error_when_api_call_fails(t *testing.T) {
	mockKMS := &MockKMSClient{
		GetSecretValueWithContextFunc: func(ctx context.Context, input *kms.GetSecretValueInput, opts ...request.Option) (*kms.GetSecretValueOutput, error) {
			return nil, errors.New("api error")
		},
	}
	client := NewClient(mockKMS)
	_, err := client.GetSecret(context.Background(), esapi.ExternalSecretDataRemoteRef{
		Key: "my-secret",
	})
	assert.Error(t, err)
	assert.Equal(t, "api error", err.Error())
}

func TestClient_GetSecret_should_return_error_when_secret_value_is_nil(t *testing.T) {
	mockKMS := &MockKMSClient{
		GetSecretValueWithContextFunc: func(ctx context.Context, input *kms.GetSecretValueInput, opts ...request.Option) (*kms.GetSecretValueOutput, error) {
			return &kms.GetSecretValueOutput{
				SecretValue: nil,
			}, nil
		},
	}
	client := NewClient(mockKMS)
	_, err := client.GetSecret(context.Background(), esapi.ExternalSecretDataRemoteRef{
		Key: "my-secret",
	})
	assert.Error(t, err)
	assert.Equal(t, "secret my-secret has no value", err.Error())
}

func TestClient_GetSecretMap_should_return_map_when_secret_is_valid_json(t *testing.T) {
	secretValue := `{"user":"admin"}`
	mockKMS := &MockKMSClient{
		GetSecretValueWithContextFunc: func(ctx context.Context, input *kms.GetSecretValueInput, opts ...request.Option) (*kms.GetSecretValueOutput, error) {
			return &kms.GetSecretValueOutput{
				SecretValue: &secretValue,
			}, nil
		},
	}
	client := NewClient(mockKMS)
	secretMap, err := client.GetSecretMap(context.Background(), esapi.ExternalSecretDataRemoteRef{
		Key: "my-secret",
	})
	assert.NoError(t, err)
	expectedMap := map[string][]byte{
		"user": []byte(`"admin"`),
	}
	assert.Equal(t, expectedMap, secretMap)
}

func TestClient_GetSecretMap_should_return_error_when_secret_is_not_valid_json(t *testing.T) {
	secretValue := `not-a-json`
	mockKMS := &MockKMSClient{
		GetSecretValueWithContextFunc: func(ctx context.Context, input *kms.GetSecretValueInput, opts ...request.Option) (*kms.GetSecretValueOutput, error) {
			return &kms.GetSecretValueOutput{
				SecretValue: &secretValue,
			}, nil
		},
	}
	client := NewClient(mockKMS)
	_, err := client.GetSecretMap(context.Background(), esapi.ExternalSecretDataRemoteRef{
		Key: "my-secret",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal secret")
}

func TestClient_GetSecretMap_should_return_error_when_api_call_fails(t *testing.T) {
	mockKMS := &MockKMSClient{
		GetSecretValueWithContextFunc: func(ctx context.Context, input *kms.GetSecretValueInput, opts ...request.Option) (*kms.GetSecretValueOutput, error) {
			return nil, errors.New("api error")
		},
	}
	client := NewClient(mockKMS)
	_, err := client.GetSecretMap(context.Background(), esapi.ExternalSecretDataRemoteRef{
		Key: "my-secret",
	})
	assert.Error(t, err)
	assert.Equal(t, "api error", err.Error())
}

func TestClient_SecretExists_should_return_error_when_secret_name_is_empty(t *testing.T) {
	mockKMS := &MockKMSClient{}
	c := NewClient(mockKMS)

	exists, err := c.SecretExists(context.Background(), fakePushScretRemoteRef{
		RemoteKey: "",
	})

	assert.False(t, exists)
	assert.Error(t, err)
	assert.Equal(t, "secret name is empty", err.Error())
}

func TestClient_SecretExists_should_return_error_when_describe_secret_fails(t *testing.T) {
	expectedErr := errors.New("failed to describe secret")
	mockKMS := &MockKMSClient{
		DescribeSecretWithContextFunc: func(ctx context.Context, input *kms.DescribeSecretInput, opts ...request.Option) (*kms.DescribeSecretOutput, error) {
			return nil, expectedErr
		},
	}
	c := NewClient(mockKMS)

	exists, err := c.SecretExists(context.Background(), fakePushScretRemoteRef{
		RemoteKey: "test-secret",
	})

	assert.False(t, exists)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestClient_SecretExists_should_return_true_when_secret_exists(t *testing.T) {
	mockKMS := &MockKMSClient{
		DescribeSecretWithContextFunc: func(ctx context.Context, input *kms.DescribeSecretInput, opts ...request.Option) (*kms.DescribeSecretOutput, error) {
			return &kms.DescribeSecretOutput{}, nil
		},
	}
	c := NewClient(mockKMS)

	exists, err := c.SecretExists(context.Background(), fakePushScretRemoteRef{
		RemoteKey: "test-secret",
	})

	assert.True(t, exists)
	assert.NoError(t, err)
}
