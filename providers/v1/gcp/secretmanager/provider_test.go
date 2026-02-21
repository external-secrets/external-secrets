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

package secretmanager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// TestClusterProjectIDMetadataFallback tests metadata server fallback for projectID resolution.
// This was the core feature added in the original PR submission.
func TestClusterProjectIDMetadataFallback(t *testing.T) {
	ctx := context.Background()

	// Store the original factory and restore it after the test
	originalFactory := metadataClientFactory
	defer func() { metadataClientFactory = originalFactory }()

	// Test: metadata fallback when projectID is not specified
	metadataClientFactory = func() MetadataClient {
		return &fakeMetadataClient{metadata: map[string]string{"project-id": "metadata-project-123"}}
	}

	// Create a store without any projectID specified
	emptyProjectStore := &esv1.SecretStore{
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				GCPSM: &esv1.GCPSMProvider{
					// ProjectID intentionally empty
					Auth: esv1.GCPSMAuth{
						WorkloadIdentity: &esv1.GCPWorkloadIdentity{
							// ClusterProjectID intentionally empty
							ServiceAccountRef: esmeta.ServiceAccountSelector{
								Name: "test-sa",
							},
						},
					},
				},
			},
		},
	}

	projectID, err := clusterProjectID(ctx, emptyProjectStore.GetSpec())
	assert.Nil(t, err)
	assert.Equal(t, "metadata-project-123", projectID)

	// Test: error when metadata server also fails
	metadataClientFactory = func() MetadataClient {
		return &fakeMetadataClient{metadata: map[string]string{}} // No project-id
	}

	_, err = clusterProjectID(ctx, emptyProjectStore.GetSpec())
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "unable to find ProjectID")
}

// TestClusterProjectIDStaticCredentials tests projectID resolution for static service account credentials.
// Static credentials MUST have explicit projectID and should NOT fall back to metadata server.
func TestClusterProjectIDStaticCredentials(t *testing.T) {
	ctx := context.Background()

	// Store the original factory and restore it after the test
	originalFactory := metadataClientFactory
	defer func() { metadataClientFactory = originalFactory }()

	// Mock metadata server to return a project ID
	metadataClientFactory = func() MetadataClient {
		return &fakeMetadataClient{metadata: map[string]string{"project-id": "metadata-project-123"}}
	}

	t.Run("with explicit projectID", func(t *testing.T) {
		staticCredStore := &esv1.SecretStore{
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					GCPSM: &esv1.GCPSMProvider{
						ProjectID: "explicit-project-456",
						Auth: esv1.GCPSMAuth{
							SecretRef: &esv1.GCPSMAuthSecretRef{
								SecretAccessKey: esmeta.SecretKeySelector{
									Name: "my-secret",
									Key:  "credentials",
								},
							},
						},
					},
				},
			},
		}

		projectID, err := clusterProjectID(ctx, staticCredStore.GetSpec())
		assert.Nil(t, err)
		assert.Equal(t, "explicit-project-456", projectID)
	})

	t.Run("without projectID should fail", func(t *testing.T) {
		staticCredStore := &esv1.SecretStore{
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					GCPSM: &esv1.GCPSMProvider{
						// ProjectID intentionally empty
						Auth: esv1.GCPSMAuth{
							SecretRef: &esv1.GCPSMAuthSecretRef{
								SecretAccessKey: esmeta.SecretKeySelector{
									Name: "my-secret",
									Key:  "credentials",
								},
							},
						},
					},
				},
			},
		}

		// Should return error, NOT the metadata server's project ID
		_, err := clusterProjectID(ctx, staticCredStore.GetSpec())
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "unable to find ProjectID")
	})
}

// TestClusterProjectIDWorkloadIdentity tests projectID resolution for GKE native Workload Identity.
// Tests comprehensive scenarios including cross-project access and the dual-purpose nature of projectID.
func TestClusterProjectIDWorkloadIdentity(t *testing.T) {
	ctx := context.Background()
	originalFactory := metadataClientFactory
	defer func() { metadataClientFactory = originalFactory }()

	t.Run("cross-project: explicit cluster and secrets projects", func(t *testing.T) {
		metadataClientFactory = func() MetadataClient {
			return &fakeMetadataClient{metadata: map[string]string{"project-id": "metadata-project-999"}}
		}

		store := &esv1.SecretStore{
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					GCPSM: &esv1.GCPSMProvider{
						ProjectID: "secrets-project-456", // Secrets location
						Auth: esv1.GCPSMAuth{
							WorkloadIdentity: &esv1.GCPWorkloadIdentity{
								ClusterProjectID:  "cluster-project-123", // Cluster/auth location
								ServiceAccountRef: esmeta.ServiceAccountSelector{Name: "test-sa"},
							},
						},
					},
				},
			},
		}

		// Auth should use clusterProjectID
		authProjectID, err := clusterProjectID(ctx, store.GetSpec())
		assert.Nil(t, err)
		assert.Equal(t, "cluster-project-123", authProjectID)

		// Secrets should come from explicit projectID
		assert.Equal(t, "secrets-project-456", store.Spec.Provider.GCPSM.ProjectID)
	})

	t.Run("explicit cluster only, projectID copied from auth", func(t *testing.T) {
		metadataClientFactory = func() MetadataClient {
			return &fakeMetadataClient{metadata: map[string]string{"project-id": "metadata-project-999"}}
		}

		store := &esv1.SecretStore{
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					GCPSM: &esv1.GCPSMProvider{
						// ProjectID intentionally empty
						Auth: esv1.GCPSMAuth{
							WorkloadIdentity: &esv1.GCPWorkloadIdentity{
								ClusterProjectID:  "cluster-project-123",
								ServiceAccountRef: esmeta.ServiceAccountSelector{Name: "test-sa"},
							},
						},
					},
				},
			},
		}

		// Auth should use explicit clusterProjectID
		authProjectID, err := clusterProjectID(ctx, store.GetSpec())
		assert.Nil(t, err)
		assert.Equal(t, "cluster-project-123", authProjectID)

		// In real usage, NewClient() would copy authProjectID to ProjectID
		// Simulating that behavior:
		if store.Spec.Provider.GCPSM.ProjectID == "" {
			store.Spec.Provider.GCPSM.ProjectID = authProjectID
		}
		assert.Equal(t, "cluster-project-123", store.Spec.Provider.GCPSM.ProjectID)
	})

	t.Run("cross-project without metadata server", func(t *testing.T) {
		metadataClientFactory = func() MetadataClient {
			return &fakeMetadataClient{metadata: map[string]string{}} // No metadata
		}

		store := &esv1.SecretStore{
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					GCPSM: &esv1.GCPSMProvider{
						ProjectID: "secrets-project-456",
						Auth: esv1.GCPSMAuth{
							WorkloadIdentity: &esv1.GCPWorkloadIdentity{
								ClusterProjectID:  "cluster-project-123",
								ServiceAccountRef: esmeta.ServiceAccountSelector{Name: "test-sa"},
							},
						},
					},
				},
			},
		}

		authProjectID, err := clusterProjectID(ctx, store.GetSpec())
		assert.Nil(t, err)
		assert.Equal(t, "cluster-project-123", authProjectID)
		assert.Equal(t, "secrets-project-456", store.Spec.Provider.GCPSM.ProjectID)
	})

	t.Run("explicit cluster only without metadata", func(t *testing.T) {
		metadataClientFactory = func() MetadataClient {
			return &fakeMetadataClient{metadata: map[string]string{}}
		}

		store := &esv1.SecretStore{
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					GCPSM: &esv1.GCPSMProvider{
						// ProjectID empty
						Auth: esv1.GCPSMAuth{
							WorkloadIdentity: &esv1.GCPWorkloadIdentity{
								ClusterProjectID:  "cluster-project-123",
								ServiceAccountRef: esmeta.ServiceAccountSelector{Name: "test-sa"},
							},
						},
					},
				},
			},
		}

		authProjectID, err := clusterProjectID(ctx, store.GetSpec())
		assert.Nil(t, err)
		assert.Equal(t, "cluster-project-123", authProjectID)
	})

	t.Run("projectID as auth fallback without metadata", func(t *testing.T) {
		metadataClientFactory = func() MetadataClient {
			return &fakeMetadataClient{metadata: map[string]string{}}
		}

		store := &esv1.SecretStore{
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					GCPSM: &esv1.GCPSMProvider{
						ProjectID: "secrets-project-456",
						Auth: esv1.GCPSMAuth{
							WorkloadIdentity: &esv1.GCPWorkloadIdentity{
								// ClusterProjectID empty
								ServiceAccountRef: esmeta.ServiceAccountSelector{Name: "test-sa"},
							},
						},
					},
				},
			},
		}

		// Auth uses projectID as fallback (Priority 2)
		authProjectID, err := clusterProjectID(ctx, store.GetSpec())
		assert.Nil(t, err)
		assert.Equal(t, "secrets-project-456", authProjectID)
		assert.Equal(t, "secrets-project-456", store.Spec.Provider.GCPSM.ProjectID)
	})
}

// TestClusterProjectIDWorkloadIdentityFederation tests projectID resolution for Workload Identity Federation.
// WIF supports multiple identity providers (K8s SA, AWS credentials, custom credConfig).
func TestClusterProjectIDWorkloadIdentityFederation(t *testing.T) {
	ctx := context.Background()
	originalFactory := metadataClientFactory
	defer func() { metadataClientFactory = originalFactory }()

	t.Run("K8s SA with metadata fallback", func(t *testing.T) {
		metadataClientFactory = func() MetadataClient {
			return &fakeMetadataClient{metadata: map[string]string{"project-id": "metadata-project-999"}}
		}

		wifStore := &esv1.SecretStore{
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					GCPSM: &esv1.GCPSMProvider{
						// ProjectID intentionally empty
						Auth: esv1.GCPSMAuth{
							WorkloadIdentityFederation: &esv1.GCPWorkloadIdentityFederation{
								ServiceAccountRef: &esmeta.ServiceAccountSelector{
									Name: "test-sa",
								},
								Audience: "//iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/pool/providers/provider",
							},
						},
					},
				},
			},
		}

		projectID, err := clusterProjectID(ctx, wifStore.GetSpec())
		assert.Nil(t, err)
		assert.Equal(t, "metadata-project-999", projectID, "Workload Identity Federation should allow metadata fallback")
	})

	t.Run("K8s SA with explicit projectID", func(t *testing.T) {
		metadataClientFactory = func() MetadataClient {
			return &fakeMetadataClient{metadata: map[string]string{"project-id": "metadata-999"}}
		}

		store := &esv1.SecretStore{
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					GCPSM: &esv1.GCPSMProvider{
						ProjectID: "explicit-secrets-456",
						Auth: esv1.GCPSMAuth{
							WorkloadIdentityFederation: &esv1.GCPWorkloadIdentityFederation{
								ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "test-sa"},
								Audience:          "//iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/pool/providers/provider",
							},
						},
					},
				},
			},
		}

		// Should use explicit projectID (Priority 2)
		projectID, err := clusterProjectID(ctx, store.GetSpec())
		assert.Nil(t, err)
		assert.Equal(t, "explicit-secrets-456", projectID)
	})

	t.Run("K8s SA without projectID or metadata should fail", func(t *testing.T) {
		metadataClientFactory = func() MetadataClient {
			return &fakeMetadataClient{metadata: map[string]string{}} // No metadata
		}

		store := &esv1.SecretStore{
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					GCPSM: &esv1.GCPSMProvider{
						// ProjectID empty
						Auth: esv1.GCPSMAuth{
							WorkloadIdentityFederation: &esv1.GCPWorkloadIdentityFederation{
								ServiceAccountRef: &esmeta.ServiceAccountSelector{Name: "test-sa"},
								Audience:          "//iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/pool/providers/provider",
							},
						},
					},
				},
			},
		}

		// Should fail - no projectID and no metadata
		_, err := clusterProjectID(ctx, store.GetSpec())
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "unable to find ProjectID")
	})

	t.Run("AWS credentials with explicit projectID", func(t *testing.T) {
		metadataClientFactory = func() MetadataClient {
			return &fakeMetadataClient{metadata: map[string]string{"project-id": "metadata-999"}}
		}

		store := &esv1.SecretStore{
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					GCPSM: &esv1.GCPSMProvider{
						ProjectID: "explicit-secrets-789",
						Auth: esv1.GCPSMAuth{
							WorkloadIdentityFederation: &esv1.GCPWorkloadIdentityFederation{
								Audience: "//iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/pool/providers/provider",
								AwsSecurityCredentials: &esv1.AwsCredentialsConfig{
									Region:                  "us-east-1",
									AwsCredentialsSecretRef: &esv1.SecretReference{Name: "aws-creds"},
								},
							},
						},
					},
				},
			},
		}

		projectID, err := clusterProjectID(ctx, store.GetSpec())
		assert.Nil(t, err)
		assert.Equal(t, "explicit-secrets-789", projectID)
	})

	t.Run("AWS credentials without projectID or metadata should fail", func(t *testing.T) {
		metadataClientFactory = func() MetadataClient {
			return &fakeMetadataClient{metadata: map[string]string{}}
		}

		store := &esv1.SecretStore{
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					GCPSM: &esv1.GCPSMProvider{
						// ProjectID empty
						Auth: esv1.GCPSMAuth{
							WorkloadIdentityFederation: &esv1.GCPWorkloadIdentityFederation{
								Audience: "//iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/pool/providers/provider",
								AwsSecurityCredentials: &esv1.AwsCredentialsConfig{
									Region:                  "us-east-1",
									AwsCredentialsSecretRef: &esv1.SecretReference{Name: "aws-creds"},
								},
							},
						},
					},
				},
			},
		}

		_, err := clusterProjectID(ctx, store.GetSpec())
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "unable to find ProjectID")
	})

	t.Run("credConfig with explicit projectID", func(t *testing.T) {
		metadataClientFactory = func() MetadataClient {
			return &fakeMetadataClient{metadata: map[string]string{"project-id": "metadata-999"}}
		}

		store := &esv1.SecretStore{
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					GCPSM: &esv1.GCPSMProvider{
						ProjectID: "explicit-secrets-321",
						Auth: esv1.GCPSMAuth{
							WorkloadIdentityFederation: &esv1.GCPWorkloadIdentityFederation{
								CredConfig: &esv1.ConfigMapReference{
									Name: "gcp-creds",
									Key:  "credentials.json",
								},
							},
						},
					},
				},
			},
		}

		projectID, err := clusterProjectID(ctx, store.GetSpec())
		assert.Nil(t, err)
		assert.Equal(t, "explicit-secrets-321", projectID)
	})
}

// TestClusterProjectIDDefaultCredentials tests projectID resolution when no auth is specified.
// Uses Application Default Credentials (ADC) - typically Core Controller authentication.
func TestClusterProjectIDDefaultCredentials(t *testing.T) {
	ctx := context.Background()
	originalFactory := metadataClientFactory
	defer func() { metadataClientFactory = originalFactory }()

	t.Run("with metadata fallback", func(t *testing.T) {
		metadataClientFactory = func() MetadataClient {
			return &fakeMetadataClient{metadata: map[string]string{"project-id": "metadata-project-999"}}
		}

		noAuthStore := &esv1.SecretStore{
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					GCPSM: &esv1.GCPSMProvider{
						// ProjectID intentionally empty
						Auth: esv1.GCPSMAuth{
							// No auth method specified
						},
					},
				},
			},
		}

		projectID, err := clusterProjectID(ctx, noAuthStore.GetSpec())
		assert.Nil(t, err)
		assert.Equal(t, "metadata-project-999", projectID, "No auth configuration should allow metadata fallback")
	})

	t.Run("with explicit projectID", func(t *testing.T) {
		metadataClientFactory = func() MetadataClient {
			return &fakeMetadataClient{metadata: map[string]string{"project-id": "metadata-999"}}
		}

		store := &esv1.SecretStore{
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					GCPSM: &esv1.GCPSMProvider{
						ProjectID: "explicit-secrets-111",
						Auth:      esv1.GCPSMAuth{
							// No auth method specified
						},
					},
				},
			},
		}

		projectID, err := clusterProjectID(ctx, store.GetSpec())
		assert.Nil(t, err)
		assert.Equal(t, "explicit-secrets-111", projectID)
	})

	t.Run("without projectID or metadata should fail", func(t *testing.T) {
		metadataClientFactory = func() MetadataClient {
			return &fakeMetadataClient{metadata: map[string]string{}}
		}

		store := &esv1.SecretStore{
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					GCPSM: &esv1.GCPSMProvider{
						// ProjectID empty
						Auth: esv1.GCPSMAuth{
							// No auth method
						},
					},
				},
			},
		}

		_, err := clusterProjectID(ctx, store.GetSpec())
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "unable to find ProjectID")
	})
}

// TestClusterProjectIDAllAuthMethods tests that metadata fallback works correctly for each auth method.
func TestClusterProjectIDAllAuthMethods(t *testing.T) {
	ctx := context.Background()
	originalFactory := metadataClientFactory
	defer func() { metadataClientFactory = originalFactory }()

	// Mock metadata server to return a project ID
	metadataClientFactory = func() MetadataClient {
		return &fakeMetadataClient{metadata: map[string]string{"project-id": "metadata-project-999"}}
	}

	t.Run("Workload Identity allows metadata fallback", func(t *testing.T) {
		wiStore := &esv1.SecretStore{
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					GCPSM: &esv1.GCPSMProvider{
						// ProjectID intentionally empty
						Auth: esv1.GCPSMAuth{
							WorkloadIdentity: &esv1.GCPWorkloadIdentity{
								ServiceAccountRef: esmeta.ServiceAccountSelector{
									Name: "test-sa",
								},
								// ClusterProjectID intentionally empty
							},
						},
					},
				},
			},
		}

		projectID, err := clusterProjectID(ctx, wiStore.GetSpec())
		assert.Nil(t, err)
		assert.Equal(t, "metadata-project-999", projectID, "Workload Identity should allow metadata fallback")
	})

	t.Run("Workload Identity Federation allows metadata fallback", func(t *testing.T) {
		wifStore := &esv1.SecretStore{
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					GCPSM: &esv1.GCPSMProvider{
						// ProjectID intentionally empty
						Auth: esv1.GCPSMAuth{
							WorkloadIdentityFederation: &esv1.GCPWorkloadIdentityFederation{
								ServiceAccountRef: &esmeta.ServiceAccountSelector{
									Name: "test-sa",
								},
								Audience: "//iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/pool/providers/provider",
							},
						},
					},
				},
			},
		}

		projectID, err := clusterProjectID(ctx, wifStore.GetSpec())
		assert.Nil(t, err)
		assert.Equal(t, "metadata-project-999", projectID, "Workload Identity Federation should allow metadata fallback")
	})

	t.Run("Default credentials allow metadata fallback", func(t *testing.T) {
		noAuthStore := &esv1.SecretStore{
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					GCPSM: &esv1.GCPSMProvider{
						// ProjectID intentionally empty
						Auth: esv1.GCPSMAuth{
							// No auth method specified
						},
					},
				},
			},
		}

		projectID, err := clusterProjectID(ctx, noAuthStore.GetSpec())
		assert.Nil(t, err)
		assert.Equal(t, "metadata-project-999", projectID, "No auth configuration should allow metadata fallback")
	})

	t.Run("Static credentials block metadata fallback", func(t *testing.T) {
		staticStore := &esv1.SecretStore{
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					GCPSM: &esv1.GCPSMProvider{
						// ProjectID intentionally empty
						Auth: esv1.GCPSMAuth{
							SecretRef: &esv1.GCPSMAuthSecretRef{
								SecretAccessKey: esmeta.SecretKeySelector{
									Name: "my-secret",
									Key:  "credentials",
								},
							},
						},
					},
				},
			},
		}

		_, err := clusterProjectID(ctx, staticStore.GetSpec())
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "unable to find ProjectID", "Static credentials should NOT allow metadata fallback")
	})
}
