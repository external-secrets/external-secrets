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
	"net/http"
	"testing"

	"cloud.google.com/go/iam/credentials/apiv1/credentialspb"
	"github.com/googleapis/gax-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// Mock implementations for testing.
type mockIamClient struct {
	generateAccessTokenFunc func(ctx context.Context, req *credentialspb.GenerateAccessTokenRequest, opts ...gax.CallOption) (*credentialspb.GenerateAccessTokenResponse, error)
	signJwtFunc             func(ctx context.Context, req *credentialspb.SignJwtRequest, opts ...gax.CallOption) (*credentialspb.SignJwtResponse, error)
	closeFunc               func() error
}

func (m *mockIamClient) GenerateAccessToken(ctx context.Context, req *credentialspb.GenerateAccessTokenRequest, opts ...gax.CallOption) (*credentialspb.GenerateAccessTokenResponse, error) {
	if m.generateAccessTokenFunc != nil {
		return m.generateAccessTokenFunc(ctx, req, opts...)
	}
	return &credentialspb.GenerateAccessTokenResponse{
		AccessToken: "mock-gcp-access-token",
	}, nil
}

func (m *mockIamClient) SignJwt(ctx context.Context, req *credentialspb.SignJwtRequest, opts ...gax.CallOption) (*credentialspb.SignJwtResponse, error) {
	if m.signJwtFunc != nil {
		return m.signJwtFunc(ctx, req, opts...)
	}
	return &credentialspb.SignJwtResponse{
		SignedJwt: "mock-signed-jwt",
	}, nil
}

func (m *mockIamClient) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

type mockMetadataClient struct {
	instanceAttributeFunc func(ctx context.Context, attr string) (string, error)
	projectIDFunc         func(ctx context.Context) (string, error)
}

func (m *mockMetadataClient) InstanceAttributeValueWithContext(ctx context.Context, attr string) (string, error) {
	if m.instanceAttributeFunc != nil {
		return m.instanceAttributeFunc(ctx, attr)
	}
	if attr == "cluster-location" {
		return "us-central1", nil
	}
	if attr == "cluster-name" {
		return "test-cluster", nil
	}
	return "", nil
}

func (m *mockMetadataClient) ProjectIDWithContext(ctx context.Context) (string, error) {
	if m.projectIDFunc != nil {
		return m.projectIDFunc(ctx)
	}
	return "test-project", nil
}

type mockSATokenGenerator struct {
	generateFunc func(ctx context.Context, audiences []string, name, namespace string) (*authenticationv1.TokenRequest, error)
}

func (m *mockSATokenGenerator) Generate(ctx context.Context, audiences []string, name, namespace string) (*authenticationv1.TokenRequest, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, audiences, name, namespace)
	}
	return &authenticationv1.TokenRequest{
		Status: authenticationv1.TokenRequestStatus{
			Token: "mock-k8s-token",
		},
	}, nil
}

type mockIDBindTokenGenerator struct {
	generateFunc func(ctx context.Context, client *http.Client, k8sToken, idPool, idProvider string) (*oauth2.Token, error)
}

func (m *mockIDBindTokenGenerator) Generate(ctx context.Context, client *http.Client, k8sToken, idPool, idProvider string) (*oauth2.Token, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, client, k8sToken, idPool, idProvider)
	}
	return &oauth2.Token{
		AccessToken: "mock-identity-binding-token",
	}, nil
}

func TestTokenSourceWithWorkloadIdentity(t *testing.T) {
	tests := []struct {
		name        string
		auth        esv1.GCPSMAuth
		setupKube   func() *clientfake.ClientBuilder
		setupMock   func(*workloadIdentity)
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful token source without GCP SA annotation",
			auth: esv1.GCPSMAuth{
				WorkloadIdentity: &esv1.GCPWorkloadIdentity{
					ClusterLocation: "us-central1",
					ClusterName:     "test-cluster",
					ServiceAccountRef: esmeta.ServiceAccountSelector{
						Name: "sa-no-annotation",
					},
				},
			},
			setupKube: func() *clientfake.ClientBuilder {
				saNoAnnotation := &corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sa-no-annotation",
						Namespace: "default",
					},
				}
				return clientfake.NewClientBuilder().WithObjects(saNoAnnotation)
			},
			setupMock: func(wi *workloadIdentity) {
				wi.metadataClient = &mockMetadataClient{}
				wi.saTokenGenerator = &mockSATokenGenerator{}
				wi.idBindTokenGenerator = &mockIDBindTokenGenerator{}
			},
			expectError: false, // Succeeds because it returns identitybindingtoken directly
		},
		{
			name: "service account not found",
			auth: esv1.GCPSMAuth{
				WorkloadIdentity: &esv1.GCPWorkloadIdentity{
					ClusterLocation: "us-central1",
					ClusterName:     "test-cluster",
					ServiceAccountRef: esmeta.ServiceAccountSelector{
						Name: "missing-sa",
					},
				},
			},
			setupKube: clientfake.NewClientBuilder,
			setupMock: func(wi *workloadIdentity) {
				wi.metadataClient = &mockMetadataClient{}
				wi.saTokenGenerator = &mockSATokenGenerator{}
				wi.idBindTokenGenerator = &mockIDBindTokenGenerator{}
			},
			expectError: true,
			errorMsg:    "not found",
		},
		{
			name: "ClusterSecretStore with custom namespace",
			auth: esv1.GCPSMAuth{
				WorkloadIdentity: &esv1.GCPWorkloadIdentity{
					ClusterLocation: "us-central1",
					ClusterName:     "test-cluster",
					ServiceAccountRef: esmeta.ServiceAccountSelector{
						Name:      "test-sa",
						Namespace: ptr.To("custom-ns"),
					},
				},
			},
			setupKube: func() *clientfake.ClientBuilder {
				// Create SA without GCP annotation to avoid IAM API calls
				saCustomNs := &corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-sa",
						Namespace: "custom-ns",
					},
				}
				return clientfake.NewClientBuilder().WithObjects(saCustomNs)
			},
			setupMock: func(wi *workloadIdentity) {
				wi.metadataClient = &mockMetadataClient{}
				wi.saTokenGenerator = &mockSATokenGenerator{}
				wi.idBindTokenGenerator = &mockIDBindTokenGenerator{}
			},
			expectError: false,
		},
		{
			name: "successful token source with GCP SA annotation via GenerateAccessToken",
			auth: esv1.GCPSMAuth{
				WorkloadIdentity: &esv1.GCPWorkloadIdentity{
					ClusterLocation: "us-central1",
					ClusterName:     "test-cluster",
					ServiceAccountRef: esmeta.ServiceAccountSelector{
						Name: "sa-with-annotation",
					},
				},
			},
			setupKube: func() *clientfake.ClientBuilder {
				// Create SA with the GCP service account annotation to exercise the IAM path
				saWithAnnotation := &corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sa-with-annotation",
						Namespace: "default",
						Annotations: map[string]string{
							"iam.gke.io/gcp-service-account": "test-sa@test-project.iam.gserviceaccount.com",
						},
					},
				}
				return clientfake.NewClientBuilder().WithObjects(saWithAnnotation)
			},
			setupMock: func(wi *workloadIdentity) {
				wi.metadataClient = &mockMetadataClient{}
				wi.saTokenGenerator = &mockSATokenGenerator{}
				wi.idBindTokenGenerator = &mockIDBindTokenGenerator{}
				// Inject mock IAM client creator to exercise the GenerateAccessToken flow
				wi.iamClientCreator = func(_ context.Context, _ oauth2.TokenSource) (IamClient, error) {
					return &mockIamClient{
						generateAccessTokenFunc: func(_ context.Context, req *credentialspb.GenerateAccessTokenRequest, _ ...gax.CallOption) (*credentialspb.GenerateAccessTokenResponse, error) {
							// Verify the request contains the expected service account
							assert.Contains(t, req.Name, "test-sa@test-project.iam.gserviceaccount.com")
							return &credentialspb.GenerateAccessTokenResponse{
								AccessToken: "mock-gcp-access-token-from-iam",
							}, nil
						},
					}, nil
				}
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kube := tt.setupKube().Build()

			// Create workloadIdentity with mock SA token generator to avoid K8s dependency
			wi, err := newWorkloadIdentity(withSATokenGenerator(&mockSATokenGenerator{}))
			require.NoError(t, err)

			if tt.setupMock != nil {
				tt.setupMock(wi)
			}

			isClusterKind := tt.auth.WorkloadIdentity.ServiceAccountRef.Namespace != nil
			ts, err := wi.TokenSource(context.Background(), tt.auth, isClusterKind, kube, "default")

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, ts)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, ts)
			}
		})
	}
}

func TestSignedJWTForVault(t *testing.T) {
	tests := []struct {
		name           string
		wi             *esv1.GCPWorkloadIdentity
		role           string
		setupKube      func() *clientfake.ClientBuilder
		setupMock      func(*workloadIdentity)
		expectError    bool
		errorMsg       string
		expectedJWT    string
		validateCalled bool
	}{
		{
			name: "successful JWT generation with GCP SA annotation",
			wi: &esv1.GCPWorkloadIdentity{
				ClusterLocation: "us-central1",
				ClusterName:     "test-cluster",
				ServiceAccountRef: esmeta.ServiceAccountSelector{
					Name: "sa-with-annotation",
				},
			},
			role: "my-vault-role",
			setupKube: func() *clientfake.ClientBuilder {
				saWithAnnotation := &corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sa-with-annotation",
						Namespace: "default",
						Annotations: map[string]string{
							"iam.gke.io/gcp-service-account": "test-sa@test-project.iam.gserviceaccount.com",
						},
					},
				}
				return clientfake.NewClientBuilder().WithObjects(saWithAnnotation)
			},
			setupMock: func(wi *workloadIdentity) {
				wi.metadataClient = &mockMetadataClient{}
				wi.idBindTokenGenerator = &mockIDBindTokenGenerator{}
				// Inject mock IAM client creator that returns a mock SignJwt response
				wi.iamClientCreator = func(_ context.Context, _ oauth2.TokenSource) (IamClient, error) {
					return &mockIamClient{
						signJwtFunc: func(_ context.Context, req *credentialspb.SignJwtRequest, _ ...gax.CallOption) (*credentialspb.SignJwtResponse, error) {
							// Verify the request contains the expected service account
							assert.Contains(t, req.Name, "test-sa@test-project.iam.gserviceaccount.com")
							// Verify the payload contains the expected audience format
							assert.Contains(t, req.Payload, "vault/my-vault-role")
							return &credentialspb.SignJwtResponse{
								SignedJwt: "mock-signed-jwt-for-vault",
							}, nil
						},
					}, nil
				}
			},
			expectError:    false,
			expectedJWT:    "mock-signed-jwt-for-vault",
			validateCalled: true,
		},
		{
			name: "service account without GCP annotation",
			wi: &esv1.GCPWorkloadIdentity{
				ClusterLocation: "us-central1",
				ClusterName:     "test-cluster",
				ServiceAccountRef: esmeta.ServiceAccountSelector{
					Name: "sa-no-annotation",
				},
			},
			role: "my-vault-role",
			setupKube: func() *clientfake.ClientBuilder {
				saNoAnnotation := &corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "sa-no-annotation",
						Namespace: "default",
					},
				}
				return clientfake.NewClientBuilder().WithObjects(saNoAnnotation)
			},
			expectError: true,
			errorMsg:    "missing required annotation",
		},
		{
			name: "service account not found",
			wi: &esv1.GCPWorkloadIdentity{
				ClusterLocation: "us-central1",
				ClusterName:     "test-cluster",
				ServiceAccountRef: esmeta.ServiceAccountSelector{
					Name: "missing-sa",
				},
			},
			role:        "my-vault-role",
			setupKube:   clientfake.NewClientBuilder,
			expectError: true,
			errorMsg:    "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kube := tt.setupKube().Build()

			// Create workloadIdentity with mock SA token generator to avoid K8s dependency
			wi, err := newWorkloadIdentity(withSATokenGenerator(&mockSATokenGenerator{}))
			require.NoError(t, err)

			// Inject additional mocks - apply defaults first
			wi.metadataClient = &mockMetadataClient{}
			wi.idBindTokenGenerator = &mockIDBindTokenGenerator{}

			// Apply test-specific mock setup if provided
			if tt.setupMock != nil {
				tt.setupMock(wi)
			}

			jwt, err := wi.SignedJWTForVault(context.Background(), tt.wi, tt.role, false, kube, "default")

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, jwt)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, jwt)
				if tt.expectedJWT != "" {
					assert.Equal(t, tt.expectedJWT, jwt)
				}
			}
		})
	}
}

// Note: TestJWTExpirationTime and detailed JWT signing validation tests require
// actual GCP IAM API access, which cannot be easily mocked without refactoring
// the production code. These aspects are covered by integration tests.

func TestWorkloadIdentityClose(t *testing.T) {
	// Create workloadIdentity with mock SA token generator to avoid K8s dependency
	wi, err := newWorkloadIdentity(withSATokenGenerator(&mockSATokenGenerator{}))
	require.NoError(t, err)

	err = wi.Close()
	assert.NoError(t, err, "Close should not return an error")
}
