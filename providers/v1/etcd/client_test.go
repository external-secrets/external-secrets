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

package etcd

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/providers/v1/etcd/fake"
)

func newTestClient(mockKV *fake.MockKV) *Client {
	return &Client{
		kv:     mockKV,
		prefix: "/external-secrets/",
	}
}

func TestGetSecret(t *testing.T) {
	testCases := []struct {
		name        string
		ref         esv1.ExternalSecretDataRemoteRef
		mockKV      *fake.MockKV
		expected    []byte
		expectedErr string
	}{
		{
			name: "get existing secret",
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "my-secret",
			},
			mockKV: &fake.MockKV{
				GetFunc: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
					assert.Equal(t, "/external-secrets/my-secret", key)
					sd := secretData{
						Data: map[string]string{
							"username": "admin",
							"password": "secret123",
						},
					}
					data, _ := json.Marshal(sd)
					return &clientv3.GetResponse{
						Kvs: []*mvccpb.KeyValue{
							{Key: []byte(key), Value: data},
						},
					}, nil
				},
			},
			expected:    []byte(`{"password":"secret123","username":"admin"}`),
			expectedErr: "",
		},
		{
			name: "get secret property",
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      "my-secret",
				Property: "password",
			},
			mockKV: &fake.MockKV{
				GetFunc: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
					sd := secretData{
						Data: map[string]string{
							"username": "admin",
							"password": "secret123",
						},
					}
					data, _ := json.Marshal(sd)
					return &clientv3.GetResponse{
						Kvs: []*mvccpb.KeyValue{
							{Key: []byte(key), Value: data},
						},
					}, nil
				},
			},
			expected:    []byte("secret123"),
			expectedErr: "",
		},
		{
			name: "secret not found",
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "nonexistent",
			},
			mockKV: &fake.MockKV{
				GetFunc: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
					return &clientv3.GetResponse{
						Kvs: []*mvccpb.KeyValue{},
					}, nil
				},
			},
			expected:    nil,
			expectedErr: "Secret does not exist",
		},
		{
			name: "property not found",
			ref: esv1.ExternalSecretDataRemoteRef{
				Key:      "my-secret",
				Property: "nonexistent",
			},
			mockKV: &fake.MockKV{
				GetFunc: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
					sd := secretData{
						Data: map[string]string{
							"username": "admin",
						},
					}
					data, _ := json.Marshal(sd)
					return &clientv3.GetResponse{
						Kvs: []*mvccpb.KeyValue{
							{Key: []byte(key), Value: data},
						},
					}, nil
				},
			},
			expected:    nil,
			expectedErr: "property nonexistent not found",
		},
		{
			name: "etcd error",
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "my-secret",
			},
			mockKV: &fake.MockKV{
				GetFunc: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
					return nil, errors.New("connection refused")
				},
			},
			expected:    nil,
			expectedErr: "failed to get secret",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := newTestClient(tc.mockKV)
			result, err := client.GetSecret(context.Background(), tc.ref)
			if tc.expectedErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestGetSecretMap(t *testing.T) {
	testCases := []struct {
		name        string
		ref         esv1.ExternalSecretDataRemoteRef
		mockKV      *fake.MockKV
		expected    map[string][]byte
		expectedErr string
	}{
		{
			name: "get secret map",
			ref: esv1.ExternalSecretDataRemoteRef{
				Key: "my-secret",
			},
			mockKV: &fake.MockKV{
				GetFunc: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
					sd := secretData{
						Data: map[string]string{
							"username": "admin",
							"password": "secret123",
						},
					}
					data, _ := json.Marshal(sd)
					return &clientv3.GetResponse{
						Kvs: []*mvccpb.KeyValue{
							{Key: []byte(key), Value: data},
						},
					}, nil
				},
			},
			expected: map[string][]byte{
				"username": []byte("admin"),
				"password": []byte("secret123"),
			},
			expectedErr: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := newTestClient(tc.mockKV)
			result, err := client.GetSecretMap(context.Background(), tc.ref)
			if tc.expectedErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestPushSecret(t *testing.T) {
	testCases := []struct {
		name        string
		secret      *corev1.Secret
		data        esv1.PushSecretData
		mockKV      *fake.MockKV
		expectedErr string
	}{
		{
			name: "push entire secret",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"username": []byte("admin"),
					"password": []byte("secret123"),
				},
			},
			data: &fakeRemoteRef{
				remoteKey: "my-secret",
			},
			mockKV: &fake.MockKV{
				GetFunc: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
					return &clientv3.GetResponse{Kvs: []*mvccpb.KeyValue{}}, nil
				},
				PutFunc: func(ctx context.Context, key, val string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error) {
					assert.Equal(t, "/external-secrets/my-secret", key)
					var sd secretData
					err := json.Unmarshal([]byte(val), &sd)
					require.NoError(t, err)
					assert.Equal(t, "admin", sd.Data["username"])
					assert.Equal(t, "secret123", sd.Data["password"])
					assert.Equal(t, "external-secrets", sd.Metadata["managed-by"])
					return &clientv3.PutResponse{}, nil
				},
			},
			expectedErr: "",
		},
		{
			name: "push specific key with property",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"api-key": []byte("my-api-key-value"),
				},
			},
			data: &fakeRemoteRef{
				remoteKey: "remote-secret",
				secretKey: "api-key",
				property:  "apiKey",
			},
			mockKV: &fake.MockKV{
				GetFunc: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
					return &clientv3.GetResponse{Kvs: []*mvccpb.KeyValue{}}, nil
				},
				PutFunc: func(ctx context.Context, key, val string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error) {
					var sd secretData
					err := json.Unmarshal([]byte(val), &sd)
					require.NoError(t, err)
					assert.Equal(t, "my-api-key-value", sd.Data["apiKey"])
					return &clientv3.PutResponse{}, nil
				},
			},
			expectedErr: "",
		},
		{
			name: "update existing secret",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"new-key": []byte("new-value"),
				},
			},
			data: &fakeRemoteRef{
				remoteKey: "existing-secret",
			},
			mockKV: &fake.MockKV{
				GetFunc: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
					existing := secretData{
						Data: map[string]string{
							"old-key": "old-value",
						},
						Metadata: map[string]string{
							"managed-by": "external-secrets",
						},
					}
					data, _ := json.Marshal(existing)
					return &clientv3.GetResponse{
						Kvs: []*mvccpb.KeyValue{
							{Key: []byte(key), Value: data},
						},
					}, nil
				},
				PutFunc: func(ctx context.Context, key, val string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error) {
					var sd secretData
					err := json.Unmarshal([]byte(val), &sd)
					require.NoError(t, err)
					// Should contain both old and new keys
					assert.Equal(t, "old-value", sd.Data["old-key"])
					assert.Equal(t, "new-value", sd.Data["new-key"])
					return &clientv3.PutResponse{}, nil
				},
			},
			expectedErr: "",
		},
		{
			name: "refuse to update secret not managed by external-secrets",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"key": []byte("value"),
				},
			},
			data: &fakeRemoteRef{
				remoteKey: "foreign-secret",
			},
			mockKV: &fake.MockKV{
				GetFunc: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
					existing := secretData{
						Data: map[string]string{
							"key": "foreign-value",
						},
						Metadata: map[string]string{
							"managed-by": "other-system",
						},
					}
					data, _ := json.Marshal(existing)
					return &clientv3.GetResponse{
						Kvs: []*mvccpb.KeyValue{
							{Key: []byte(key), Value: data},
						},
					}, nil
				},
			},
			expectedErr: "secret not managed by external-secrets",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := newTestClient(tc.mockKV)
			err := client.PushSecret(context.Background(), tc.secret, tc.data)
			if tc.expectedErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDeleteSecret(t *testing.T) {
	testCases := []struct {
		name        string
		remoteRef   esv1.PushSecretRemoteRef
		mockKV      *fake.MockKV
		expectedErr string
	}{
		{
			name: "delete entire secret",
			remoteRef: &fakeRemoteRef{
				remoteKey: "my-secret",
			},
			mockKV: &fake.MockKV{
				DeleteFunc: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.DeleteResponse, error) {
					assert.Equal(t, "/external-secrets/my-secret", key)
					return &clientv3.DeleteResponse{}, nil
				},
			},
			expectedErr: "",
		},
		{
			name: "delete single property",
			remoteRef: &fakeRemoteRef{
				remoteKey: "my-secret",
				property:  "password",
			},
			mockKV: &fake.MockKV{
				GetFunc: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
					sd := secretData{
						Data: map[string]string{
							"username": "admin",
							"password": "secret123",
						},
						Metadata: map[string]string{
							"managed-by": "external-secrets",
						},
					}
					data, _ := json.Marshal(sd)
					return &clientv3.GetResponse{
						Kvs: []*mvccpb.KeyValue{
							{Key: []byte(key), Value: data},
						},
					}, nil
				},
				PutFunc: func(ctx context.Context, key, val string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error) {
					var sd secretData
					err := json.Unmarshal([]byte(val), &sd)
					require.NoError(t, err)
					// Should only have username, not password
					assert.Equal(t, "admin", sd.Data["username"])
					_, hasPassword := sd.Data["password"]
					assert.False(t, hasPassword)
					return &clientv3.PutResponse{}, nil
				},
			},
			expectedErr: "",
		},
		{
			name: "delete last property removes entire secret",
			remoteRef: &fakeRemoteRef{
				remoteKey: "my-secret",
				property:  "only-key",
			},
			mockKV: &fake.MockKV{
				GetFunc: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
					sd := secretData{
						Data: map[string]string{
							"only-key": "value",
						},
						Metadata: map[string]string{
							"managed-by": "external-secrets",
						},
					}
					data, _ := json.Marshal(sd)
					return &clientv3.GetResponse{
						Kvs: []*mvccpb.KeyValue{
							{Key: []byte(key), Value: data},
						},
					}, nil
				},
				DeleteFunc: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.DeleteResponse, error) {
					return &clientv3.DeleteResponse{}, nil
				},
			},
			expectedErr: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := newTestClient(tc.mockKV)
			err := client.DeleteSecret(context.Background(), tc.remoteRef)
			if tc.expectedErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSecretExists(t *testing.T) {
	testCases := []struct {
		name        string
		remoteRef   esv1.PushSecretRemoteRef
		mockKV      *fake.MockKV
		expected    bool
		expectedErr string
	}{
		{
			name: "secret exists",
			remoteRef: &fakeRemoteRef{
				remoteKey: "my-secret",
			},
			mockKV: &fake.MockKV{
				GetFunc: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
					return &clientv3.GetResponse{
						Kvs: []*mvccpb.KeyValue{
							{Key: []byte(key), Value: []byte(`{"data": {"key": "value"}}`)},
						},
					}, nil
				},
			},
			expected:    true,
			expectedErr: "",
		},
		{
			name: "secret does not exist",
			remoteRef: &fakeRemoteRef{
				remoteKey: "nonexistent",
			},
			mockKV: &fake.MockKV{
				GetFunc: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
					return &clientv3.GetResponse{Kvs: []*mvccpb.KeyValue{}}, nil
				},
			},
			expected:    false,
			expectedErr: "",
		},
		{
			name: "property exists",
			remoteRef: &fakeRemoteRef{
				remoteKey: "my-secret",
				property:  "username",
			},
			mockKV: &fake.MockKV{
				GetFunc: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
					sd := secretData{
						Data: map[string]string{
							"username": "admin",
						},
					}
					data, _ := json.Marshal(sd)
					return &clientv3.GetResponse{
						Kvs: []*mvccpb.KeyValue{
							{Key: []byte(key), Value: data},
						},
					}, nil
				},
			},
			expected:    true,
			expectedErr: "",
		},
		{
			name: "property does not exist",
			remoteRef: &fakeRemoteRef{
				remoteKey: "my-secret",
				property:  "nonexistent",
			},
			mockKV: &fake.MockKV{
				GetFunc: func(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
					sd := secretData{
						Data: map[string]string{
							"username": "admin",
						},
					}
					data, _ := json.Marshal(sd)
					return &clientv3.GetResponse{
						Kvs: []*mvccpb.KeyValue{
							{Key: []byte(key), Value: data},
						},
					}, nil
				},
			},
			expected:    false,
			expectedErr: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := newTestClient(tc.mockKV)
			exists, err := client.SecretExists(context.Background(), tc.remoteRef)
			if tc.expectedErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, exists)
			}
		})
	}
}

func TestBuildKey(t *testing.T) {
	testCases := []struct {
		name     string
		prefix   string
		key      string
		expected string
	}{
		{
			name:     "normal key",
			prefix:   "/external-secrets/",
			key:      "my-secret",
			expected: "/external-secrets/my-secret",
		},
		{
			name:     "prefix without trailing slash",
			prefix:   "/external-secrets",
			key:      "my-secret",
			expected: "/external-secrets/my-secret",
		},
		{
			name:     "key with leading slash",
			prefix:   "/external-secrets/",
			key:      "/my-secret",
			expected: "/external-secrets/my-secret",
		},
		{
			name:     "nested key",
			prefix:   "/external-secrets/",
			key:      "path/to/secret",
			expected: "/external-secrets/path/to/secret",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := &Client{prefix: tc.prefix}
			result := client.buildKey(tc.key)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// fakeRemoteRef implements esv1.PushSecretData interface for testing.
type fakeRemoteRef struct {
	remoteKey string
	secretKey string
	property  string
}

func (f *fakeRemoteRef) GetRemoteKey() string {
	return f.remoteKey
}

func (f *fakeRemoteRef) GetSecretKey() string {
	return f.secretKey
}

func (f *fakeRemoteRef) GetProperty() string {
	return f.property
}

func (f *fakeRemoteRef) GetMetadata() *apiextensionsv1.JSON {
	return nil
}
