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

package fake

import (
	"context"

	"github.com/sacloud/secretmanager-api-go"
	v1 "github.com/sacloud/secretmanager-api-go/apis/v1"
)

// MockSecretAPIClient is a mock implementation of secretmanager.SecretAPI for testing.
type MockSecretAPIClient struct {
	unveilFn func(ctx context.Context, params v1.Unveil) (*v1.Unveil, error)
	listFn   func(ctx context.Context) ([]v1.Secret, error)
	createFn func(ctx context.Context, params v1.CreateSecret) (*v1.Secret, error)
	updateFn func(ctx context.Context, params v1.CreateSecret) (*v1.Secret, error)
	deleteFn func(ctx context.Context, params v1.DeleteSecret) error
}

// Check if MockSecretAPIClient satisfies the secretmanager.SecretAPI interface.
var _ secretmanager.SecretAPI = &MockSecretAPIClient{}

// Unveil implements secretmanager.SecretAPI.
func (mc *MockSecretAPIClient) Unveil(ctx context.Context, params v1.Unveil) (*v1.Unveil, error) {
	return mc.unveilFn(ctx, params)
}

// List implements secretmanager.SecretAPI.
func (mc *MockSecretAPIClient) List(ctx context.Context) ([]v1.Secret, error) {
	return mc.listFn(ctx)
}

// Create implements secretmanager.SecretAPI.
func (mc *MockSecretAPIClient) Create(ctx context.Context, params v1.CreateSecret) (*v1.Secret, error) {
	return mc.createFn(ctx, params)
}

// Update implements secretmanager.SecretAPI.
func (mc *MockSecretAPIClient) Update(ctx context.Context, params v1.CreateSecret) (*v1.Secret, error) {
	return mc.updateFn(ctx, params)
}

// Delete implements secretmanager.SecretAPI.
func (mc *MockSecretAPIClient) Delete(ctx context.Context, params v1.DeleteSecret) error {
	return mc.deleteFn(ctx, params)
}

// WithUnveil configures the mock Unveil method.
func (mc *MockSecretAPIClient) WithUnveil(response *v1.Unveil, err error) {
	if mc != nil {
		mc.unveilFn = func(_ context.Context, _ v1.Unveil) (*v1.Unveil, error) {
			return response, err
		}
	}
}

// WithList configures the mock List method.
func (mc *MockSecretAPIClient) WithList(secrets []v1.Secret, err error) {
	if mc != nil {
		mc.listFn = func(_ context.Context) ([]v1.Secret, error) {
			return secrets, err
		}
	}
}

// WithCreate configures the mock Create method.
func (mc *MockSecretAPIClient) WithCreate(secret *v1.Secret, err error) {
	if mc != nil {
		mc.createFn = func(_ context.Context, _ v1.CreateSecret) (*v1.Secret, error) {
			return secret, err
		}
	}
}

// WithUpdate configures the mock Update method.
func (mc *MockSecretAPIClient) WithUpdate(secret *v1.Secret, err error) {
	if mc != nil {
		mc.updateFn = func(_ context.Context, _ v1.CreateSecret) (*v1.Secret, error) {
			return secret, err
		}
	}
}

// WithDelete configures the mock Delete method.
func (mc *MockSecretAPIClient) WithDelete(err error) {
	if mc != nil {
		mc.deleteFn = func(_ context.Context, _ v1.DeleteSecret) error {
			return err
		}
	}
}

// NewMockSecretAPIClient creates a new mock SecretAPI client with default no-op implementations.
func NewMockSecretAPIClient() *MockSecretAPIClient {
	return &MockSecretAPIClient{
		unveilFn: func(_ context.Context, _ v1.Unveil) (*v1.Unveil, error) {
			return &v1.Unveil{}, nil
		},
		listFn: func(_ context.Context) ([]v1.Secret, error) {
			return []v1.Secret{}, nil
		},
		createFn: func(_ context.Context, _ v1.CreateSecret) (*v1.Secret, error) {
			return &v1.Secret{}, nil
		},
		updateFn: func(_ context.Context, _ v1.CreateSecret) (*v1.Secret, error) {
			return &v1.Secret{}, nil
		},
		deleteFn: func(_ context.Context, _ v1.DeleteSecret) error {
			return nil
		},
	}
}
