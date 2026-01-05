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

package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

func TestNewTokenSource(t *testing.T) {
	tests := []struct {
		name        string
		auth        esv1.GCPSMAuth
		projectID   string
		storeKind   string
		namespace   string
		setupKube   func() *clientfake.ClientBuilder
		expectToken bool
		expectError bool
	}{
		// Note: Workload identity tests are skipped because they require GCP metadata server
		// or complex mocks. The functionality is tested in integration tests.
		{
			name: "service account key configured",
			auth: esv1.GCPSMAuth{
				SecretRef: &esv1.GCPSMAuthSecretRef{
					SecretAccessKey: esmeta.SecretKeySelector{
						Name: "gcp-secret",
						Key:  "credentials",
					},
				},
			},
			projectID: "test-project",
			storeKind: esv1.SecretStoreKind,
			namespace: "default",
			setupKube: func() *clientfake.ClientBuilder {
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "gcp-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"credentials": []byte(`{"type":"service_account","project_id":"test-project"}`),
					},
				}
				return clientfake.NewClientBuilder().WithObjects(secret)
			},
			expectToken: true,
			expectError: false,
		},
		{
			name:        "no auth configured - default credentials",
			auth:        esv1.GCPSMAuth{},
			projectID:   "test-project",
			storeKind:   esv1.SecretStoreKind,
			namespace:   "default",
			setupKube:   clientfake.NewClientBuilder,
			expectToken: true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kube := tt.setupKube().Build()
			ts, err := NewTokenSource(context.Background(), tt.auth, tt.projectID, tt.storeKind, kube, tt.namespace)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, ts)
			} else {
				assert.NoError(t, err)
				if tt.expectToken {
					assert.NotNil(t, ts)
				}
			}
		})
	}
}

func TestGenerateSignedJWTForVault(t *testing.T) {
	tests := []struct {
		name        string
		wi          *esv1.GCPWorkloadIdentity
		role        string
		projectID   string
		storeKind   string
		namespace   string
		setupKube   func() *clientfake.ClientBuilder
		expectError bool
		errorMsg    string
	}{
		// Note: Successful JWT generation test is skipped because it requires GCP IAM API
		// or complex mocks. The functionality is tested in integration tests.
		{
			name:        "no workload identity configured",
			wi:          nil,
			role:        "vault-role",
			projectID:   "test-project",
			storeKind:   esv1.SecretStoreKind,
			namespace:   "default",
			setupKube:   clientfake.NewClientBuilder,
			expectError: true,
			errorMsg:    "workload identity configuration is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kube := tt.setupKube().Build()
			jwt, err := GenerateSignedJWTForVault(context.Background(), tt.wi, tt.role, tt.projectID, tt.storeKind, kube, tt.namespace)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, jwt)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				// We can't actually test JWT generation without real GCP credentials,
				// but we can verify it doesn't error with the mock setup
				assert.NoError(t, err)
				assert.NotEmpty(t, jwt)
			}
		})
	}
}

func TestServiceAccountTokenSource(t *testing.T) {
	validSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gcp-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"credentials": []byte(`{
				"type": "service_account",
				"project_id": "test-project",
				"private_key_id": "key-id",
				"private_key": "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC\n-----END PRIVATE KEY-----\n",
				"client_email": "test@test-project.iam.gserviceaccount.com"
			}`),
		},
	}

	tests := []struct {
		name        string
		auth        esv1.GCPSMAuth
		storeKind   string
		namespace   string
		setupKube   func() *clientfake.ClientBuilder
		expectToken bool
		expectError bool
	}{
		{
			name: "valid service account key",
			auth: esv1.GCPSMAuth{
				SecretRef: &esv1.GCPSMAuthSecretRef{
					SecretAccessKey: esmeta.SecretKeySelector{
						Name: "gcp-secret",
						Key:  "credentials",
					},
				},
			},
			storeKind: esv1.SecretStoreKind,
			namespace: "default",
			setupKube: func() *clientfake.ClientBuilder {
				return clientfake.NewClientBuilder().WithObjects(validSecret)
			},
			expectToken: true,
			expectError: false,
		},
		{
			name: "secret not found",
			auth: esv1.GCPSMAuth{
				SecretRef: &esv1.GCPSMAuthSecretRef{
					SecretAccessKey: esmeta.SecretKeySelector{
						Name: "missing-secret",
						Key:  "credentials",
					},
				},
			},
			storeKind:   esv1.SecretStoreKind,
			namespace:   "default",
			setupKube:   clientfake.NewClientBuilder,
			expectToken: false,
			expectError: true,
		},
		{
			name:        "no secret ref configured",
			auth:        esv1.GCPSMAuth{},
			storeKind:   esv1.SecretStoreKind,
			namespace:   "default",
			setupKube:   clientfake.NewClientBuilder,
			expectToken: false,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kube := tt.setupKube().Build()
			ts, err := serviceAccountTokenSource(context.Background(), tt.auth, tt.storeKind, kube, tt.namespace)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, ts)
			} else {
				if tt.expectToken {
					assert.NoError(t, err)
					assert.NotNil(t, ts)
				} else {
					assert.NoError(t, err)
					assert.Nil(t, ts)
				}
			}
		})
	}
}

func TestTokenSourceFallback(t *testing.T) {
	// Test the fallback behavior: service account key -> workload identity -> default credentials
	tests := []struct {
		name        string
		auth        esv1.GCPSMAuth
		expectError bool
	}{
		{
			name:        "fallback to default credentials when nothing configured",
			auth:        esv1.GCPSMAuth{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kube := clientfake.NewClientBuilder().Build()
			ts, err := NewTokenSource(context.Background(), tt.auth, "test-project", esv1.SecretStoreKind, kube, "default")

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, ts)
			} else {
				assert.NoError(t, err)
				// Default credentials will return a token source
				assert.NotNil(t, ts)
			}
		})
	}
}
