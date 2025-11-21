// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

// generator_test.go
package sendgrid

import (
	"context"
	"errors"
	"testing"

	"github.com/sendgrid/rest"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// MockClient é uma implementação mock da interface Client.
type MockClient struct {
	GetAPIResponse    *rest.Response
	GetAPIError       error
	PostAPIResponse   *rest.Response
	PostAPIError      error
	DeleteAPIResponse *rest.Response
	DeleteAPIError    error
	PutAPIResponse    *rest.Response
	PutAPIError       error
	PatchAPIResponse  *rest.Response
	PatchAPIError     error
	GetRequestErr     error
}

func (m *MockClient) API(request rest.Request) (*rest.Response, error) {
	switch request.Method {
	case rest.Get:
		return m.GetAPIResponse, m.GetAPIError
	case rest.Post:
		return m.PostAPIResponse, m.PostAPIError
	case rest.Delete:
		return m.DeleteAPIResponse, m.DeleteAPIError
	case rest.Patch:
		return m.PatchAPIResponse, m.PatchAPIError
	case rest.Put:
		return m.PutAPIResponse, m.PutAPIError
	default:
		return m.GetAPIResponse, m.GetAPIError
	}
}

func (m *MockClient) GetRequest(apiKey, endpoint, host string) rest.Request {
	return rest.Request{}
}

func (m *MockClient) SetDataResidency(request rest.Request, dataResidency string) (rest.Request, error) {
	return request, nil
}

func TestGenerator_Generate(t *testing.T) {
	kube := fake.NewClientBuilder().WithObjects(&v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sendgrid-api-key",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"apiKey": []byte("foo"),
		},
	}).Build()
	namespace := "default"

	tests := []struct {
		name       string
		jsonSpec   *apiextensions.JSON
		setupMock  func() *MockClient
		expectErr  bool
		expectData map[string][]byte
	}{
		{
			name: "Success",
			jsonSpec: &apiextensions.JSON{
				Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: SendgridAuthorizationToken
metadata:
  name: my-sendgrid-generator
spec:
  scopes:
    - alerts.create
    - alerts.read
  auth: 
    secretRef:
      apiKeySecretRef:
        name: sendgrid-api-key
        key: apiKey
`),
			},
			setupMock: func() *MockClient {
				return &MockClient{
					PostAPIResponse: &rest.Response{
						StatusCode: 200,
						Body:       `{"api_key": "newly-created-api-key"}`,
					},
				}
			},
			expectErr: false,
			expectData: map[string][]byte{
				"apiKey": []byte("newly-created-api-key"),
			},
		},
		{
			name:     "No spec error",
			jsonSpec: nil,
			setupMock: func() *MockClient {
				return &MockClient{}
			},
			expectErr: true,
		}, {
			name: "Creation Failed",
			jsonSpec: &apiextensions.JSON{
				Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: SendgridAuthorizationToken
metadata:
  name: my-sendgrid-generator
spec:
  scopes:
    - alerts.create
    - alerts.read
  auth: 
    secretRef:
      apiKeySecretRef:
        name: sendgrid-api-key
        key: apiKey
`),
			},
			setupMock: func() *MockClient {
				return &MockClient{
					PostAPIResponse: &rest.Response{
						StatusCode: 400,
						Body:       `{"error": "bad request"}`,
					},
					PostAPIError: errors.New("an error occurred"),
				}
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := &Generator{}

			mockClient := tt.setupMock()
			data, _, err := generator.generate(context.Background(), tt.jsonSpec, kube, namespace, mockClient)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectData, data)
			}
		})
	}
}

func TestGenerator_Cleanup(t *testing.T) {
	kube := fake.NewClientBuilder().WithObjects(&v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sendgrid-api-key",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"apiKey": []byte("foo"),
		},
	}).Build()
	namespace := "default"

	tests := []struct {
		name       string
		jsonSpec   *apiextensions.JSON
		state      *apiextensions.JSON
		setupMock  func() *MockClient
		expectErr  bool
		expectData map[string][]byte
	}{
		{
			name: "Success",
			jsonSpec: &apiextensions.JSON{
				Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: SendgridAuthorizationToken
metadata:
  name: my-sendgrid-generator
spec:
  scopes:
    - alerts.create
    - alerts.read
  auth: 
    secretRef:
      apiKeySecretRef:
        name: sendgrid-api-key
        key: apiKey
`),
			},
			state: &apiextensions.JSON{
				Raw: []byte(`{"apiKeyID": "foo", "apiKeyName": "bar"}`),
			},
			setupMock: func() *MockClient {
				return &MockClient{
					DeleteAPIResponse: &rest.Response{
						StatusCode: 200,
					},
				}
			},
			expectErr: false,
			expectData: map[string][]byte{
				"apiKey": []byte("newly-created-api-key"),
			},
		},
		{
			name:     "No spec error",
			jsonSpec: nil,
			state: &apiextensions.JSON{
				Raw: []byte(`{"apiKeyID": "foo", "apiKeyName": "bar"}`),
			},
			setupMock: func() *MockClient {
				return &MockClient{}
			},
			expectErr: true,
		},

		{
			name: "No previous state error",
			jsonSpec: &apiextensions.JSON{
				Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: SendgridAuthorizationToken
metadata:
  name: my-sendgrid-generator
spec:
  scopes:
    - alerts.create
    - alerts.read
  auth: 
    secretRef:
      apiKeySecretRef:
        name: sendgrid-api-key
        key: apiKey
`)},
			state: nil,
			setupMock: func() *MockClient {
				return &MockClient{}
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := &Generator{}

			mockClient := tt.setupMock()
			err := generator.cleanup(context.Background(), tt.jsonSpec, tt.state, kube, namespace, mockClient)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
