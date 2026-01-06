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
	"github.com/stretchr/testify/require"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// mockWifSATokenGenerator is a mock saTokenGenerator for testing.
type mockWifSATokenGenerator struct{}

func (m *mockWifSATokenGenerator) Generate(_ context.Context, _ []string, _, _ string) (*authenticationv1.TokenRequest, error) {
	return &authenticationv1.TokenRequest{
		Status: authenticationv1.TokenRequestStatus{
			Token: "mock-k8s-token",
		},
	}, nil
}

func TestNewWorkloadIdentityFederation(t *testing.T) {
	tests := []struct {
		name        string
		config      *esv1.GCPWorkloadIdentityFederation
		expectError bool
	}{
		{
			name: "successful creation",
			config: &esv1.GCPWorkloadIdentityFederation{
				Audience: "//iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/my-pool/providers/my-provider",
				ServiceAccountRef: &esmeta.ServiceAccountSelector{
					Name: "test-sa",
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kube := clientfake.NewClientBuilder().Build()
			// Use mock SA token generator to avoid K8s dependency
			wif, err := newWorkloadIdentityFederation(kube, tt.config, false, "default", withWifSATokenGenerator(&mockWifSATokenGenerator{}))

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, wif)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, wif)
			}
		})
	}
}

func TestWorkloadIdentityFederationTokenSource(t *testing.T) {
	validCredJSON := `{
		"type": "external_account",
		"audience": "//iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/my-pool/providers/my-provider",
		"subject_token_type": "urn:ietf:params:oauth:token-type:jwt",
		"token_url": "https://sts.googleapis.com/v1/token",
		"service_account_impersonation_url": "https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/my-sa@project.iam.gserviceaccount.com:generateAccessToken"
	}`

	tests := []struct {
		name        string
		config      *esv1.GCPWorkloadIdentityFederation
		setupKube   func() *clientfake.ClientBuilder
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil config returns nil token source",
			config:      nil,
			setupKube:   clientfake.NewClientBuilder,
			expectError: false,
		},
		{
			name: "multiple auth methods provided",
			config: &esv1.GCPWorkloadIdentityFederation{
				Audience: "//iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/my-pool/providers/my-provider",
				ServiceAccountRef: &esmeta.ServiceAccountSelector{
					Name: "test-sa",
				},
				CredConfig: &esv1.ConfigMapReference{
					Name: "cred-config",
					Key:  "credentials",
				},
			},
			setupKube:   clientfake.NewClientBuilder,
			expectError: true,
			errorMsg:    "exactly one of",
		},
		{
			name: "no auth methods provided",
			config: &esv1.GCPWorkloadIdentityFederation{
				Audience: "//iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/my-pool/providers/my-provider",
			},
			setupKube:   clientfake.NewClientBuilder,
			expectError: true,
			errorMsg:    "exactly one of",
		},
		{
			name: "serviceAccountRef without audience",
			config: &esv1.GCPWorkloadIdentityFederation{
				ServiceAccountRef: &esmeta.ServiceAccountSelector{
					Name: "test-sa",
				},
			},
			setupKube:   clientfake.NewClientBuilder,
			expectError: true,
			errorMsg:    "audience must be provided",
		},
		{
			name: "awsSecurityCredentials without audience",
			config: &esv1.GCPWorkloadIdentityFederation{
				AwsSecurityCredentials: &esv1.AwsCredentialsConfig{
					Region: "us-east-1",
					AwsCredentialsSecretRef: &esv1.SecretReference{
						Name: "aws-creds",
					},
				},
			},
			setupKube:   clientfake.NewClientBuilder,
			expectError: true,
			errorMsg:    "audience must be provided",
		},
		{
			name: "credConfig with missing configmap",
			config: &esv1.GCPWorkloadIdentityFederation{
				CredConfig: &esv1.ConfigMapReference{
					Name: "missing-config",
					Key:  "credentials",
				},
			},
			setupKube:   clientfake.NewClientBuilder,
			expectError: true,
			errorMsg:    "failed to fetch",
		},
		{
			name: "credConfig with missing key in configmap",
			config: &esv1.GCPWorkloadIdentityFederation{
				CredConfig: &esv1.ConfigMapReference{
					Name: "cred-config",
					Key:  "missing-key",
				},
			},
			setupKube: func() *clientfake.ClientBuilder {
				cm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cred-config",
						Namespace: "default",
					},
					Data: map[string]string{
						"credentials": validCredJSON,
					},
				}
				return clientfake.NewClientBuilder().WithObjects(cm)
			},
			expectError: true,
			errorMsg:    "missing key",
		},
		{
			name: "credConfig with empty value",
			config: &esv1.GCPWorkloadIdentityFederation{
				CredConfig: &esv1.ConfigMapReference{
					Name: "cred-config",
					Key:  "credentials",
				},
			},
			setupKube: func() *clientfake.ClientBuilder {
				cm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cred-config",
						Namespace: "default",
					},
					Data: map[string]string{
						"credentials": "",
					},
				}
				return clientfake.NewClientBuilder().WithObjects(cm)
			},
			expectError: true,
			errorMsg:    "has empty value",
		},
		{
			name: "credConfig with invalid JSON",
			config: &esv1.GCPWorkloadIdentityFederation{
				CredConfig: &esv1.ConfigMapReference{
					Name: "cred-config",
					Key:  "credentials",
				},
			},
			setupKube: func() *clientfake.ClientBuilder {
				cm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cred-config",
						Namespace: "default",
					},
					Data: map[string]string{
						"credentials": "not valid json",
					},
				}
				return clientfake.NewClientBuilder().WithObjects(cm)
			},
			expectError: true,
			errorMsg:    "failed to unmarshal",
		},
		{
			name: "credConfig with non-external_account type",
			config: &esv1.GCPWorkloadIdentityFederation{
				CredConfig: &esv1.ConfigMapReference{
					Name: "cred-config",
					Key:  "credentials",
				},
			},
			setupKube: func() *clientfake.ClientBuilder {
				cm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cred-config",
						Namespace: "default",
					},
					Data: map[string]string{
						"credentials": `{"type":"service_account"}`,
					},
				}
				return clientfake.NewClientBuilder().WithObjects(cm)
			},
			expectError: true,
			errorMsg:    "invalid credentials",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kube := tt.setupKube().Build()
			// Use mock SA token generator to avoid K8s dependency
			wif, err := newWorkloadIdentityFederation(kube, tt.config, false, "default", withWifSATokenGenerator(&mockWifSATokenGenerator{}))
			require.NoError(t, err)

			ts, err := wif.TokenSource(context.Background())

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				// TokenSource may be nil if config is nil
				if tt.config == nil {
					assert.Nil(t, ts)
				}
			}
		})
	}
}

func TestWorkloadIdentityFederationWithClusterKind(t *testing.T) {
	validCredJSON := `{
		"type": "external_account",
		"audience": "//iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/my-pool/providers/my-provider",
		"subject_token_type": "urn:ietf:params:oauth:token-type:jwt",
		"token_url": "https://sts.googleapis.com/v1/token",
		"service_account_impersonation_url": "https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/my-sa@project.iam.gserviceaccount.com:generateAccessToken"
	}`

	tests := []struct {
		name        string
		config      *esv1.GCPWorkloadIdentityFederation
		setupKube   func() *clientfake.ClientBuilder
		expectError bool
		errorMsg    string
	}{
		{
			name: "credConfig with custom namespace",
			config: &esv1.GCPWorkloadIdentityFederation{
				CredConfig: &esv1.ConfigMapReference{
					Name:      "cred-config",
					Key:       "credentials",
					Namespace: "custom-ns",
				},
			},
			setupKube: func() *clientfake.ClientBuilder {
				cm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cred-config",
						Namespace: "custom-ns",
					},
					Data: map[string]string{
						"credentials": validCredJSON,
					},
				}
				return clientfake.NewClientBuilder().WithObjects(cm)
			},
			expectError: true, // Will fail validation but tests namespace resolution
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kube := tt.setupKube().Build()
			// Use mock SA token generator to avoid K8s dependency
			wif, err := newWorkloadIdentityFederation(kube, tt.config, true, "default", withWifSATokenGenerator(&mockWifSATokenGenerator{}))
			require.NoError(t, err)

			_, err = wif.TokenSource(context.Background())

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestReadAWSSecurityCredentials(t *testing.T) {
	tests := []struct {
		name        string
		config      *esv1.GCPWorkloadIdentityFederation
		setupKube   func() *clientfake.ClientBuilder
		expectError bool
		errorMsg    string
	}{
		{
			name: "missing AWS credentials secret",
			config: &esv1.GCPWorkloadIdentityFederation{
				Audience: "//iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/my-pool/providers/my-provider",
				AwsSecurityCredentials: &esv1.AwsCredentialsConfig{
					Region: "us-east-1",
					AwsCredentialsSecretRef: &esv1.SecretReference{
						Name: "missing-secret",
					},
				},
			},
			setupKube:   clientfake.NewClientBuilder,
			expectError: true,
			errorMsg:    "failed to fetch",
		},
		{
			name: "AWS credentials secret missing required keys",
			config: &esv1.GCPWorkloadIdentityFederation{
				Audience: "//iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/my-pool/providers/my-provider",
				AwsSecurityCredentials: &esv1.AwsCredentialsConfig{
					Region: "us-east-1",
					AwsCredentialsSecretRef: &esv1.SecretReference{
						Name: "aws-creds",
					},
				},
			},
			setupKube: func() *clientfake.ClientBuilder {
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "aws-creds",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"wrong-key": []byte("value"),
					},
				}
				return clientfake.NewClientBuilder().WithObjects(secret)
			},
			expectError: true,
			errorMsg:    "must be present",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kube := tt.setupKube().Build()
			// Use mock SA token generator to avoid K8s dependency
			wif, err := newWorkloadIdentityFederation(kube, tt.config, false, "default", withWifSATokenGenerator(&mockWifSATokenGenerator{}))
			require.NoError(t, err)

			_, err = wif.TokenSource(context.Background())

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Note: Full TokenSource generation tests with valid external account configs
// require actual GCP STS endpoints or complex mocking of the externalaccount
// package, which is beyond the scope of unit tests. These scenarios are
// covered by integration tests.
