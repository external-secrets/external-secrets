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
	"testing"

	"github.com/stretchr/testify/assert"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

func TestClusterProjectIDMetadataFallback(t *testing.T) {
	originalFactory := metadataClientFactory
	defer func() { metadataClientFactory = originalFactory }()

	metadataClientFactory = func() MetadataClient {
		return &fakeMetadataClient{metadata: map[string]string{"project-id": "metadata-project-123"}}
	}

	store := &esv1.SecretStore{
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				GCPSM: &esv1.GCPSMProvider{
					Auth: esv1.GCPSMAuth{
						WorkloadIdentity: &esv1.GCPWorkloadIdentity{
							ServiceAccountRef: esmeta.ServiceAccountSelector{Name: "test-sa"},
						},
					},
				},
			},
		},
	}

	projectID, err := clusterProjectID(t.Context(), store.GetSpec())
	assert.Nil(t, err)
	assert.Equal(t, "metadata-project-123", projectID)

	// metadata server returns nothing -> error
	metadataClientFactory = func() MetadataClient {
		return &fakeMetadataClient{metadata: map[string]string{}}
	}

	_, err = clusterProjectID(t.Context(), store.GetSpec())
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "unable to find ProjectID")
}

func TestClusterProjectIDStaticCredentials(t *testing.T) {
	originalFactory := metadataClientFactory
	defer func() { metadataClientFactory = originalFactory }()

	metadataClientFactory = func() MetadataClient {
		return &fakeMetadataClient{metadata: map[string]string{"project-id": "metadata-project-123"}}
	}

	t.Run("with explicit projectID", func(t *testing.T) {
		store := &esv1.SecretStore{
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					GCPSM: &esv1.GCPSMProvider{
						ProjectID: "explicit-project-456",
						Auth: esv1.GCPSMAuth{
							SecretRef: &esv1.GCPSMAuthSecretRef{
								SecretAccessKey: esmeta.SecretKeySelector{Name: "my-secret", Key: "credentials"},
							},
						},
					},
				},
			},
		}

		projectID, err := clusterProjectID(t.Context(), store.GetSpec())
		assert.Nil(t, err)
		assert.Equal(t, "explicit-project-456", projectID)
	})

	t.Run("without projectID should fail, not fall back to metadata", func(t *testing.T) {
		store := &esv1.SecretStore{
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					GCPSM: &esv1.GCPSMProvider{
						Auth: esv1.GCPSMAuth{
							SecretRef: &esv1.GCPSMAuthSecretRef{
								SecretAccessKey: esmeta.SecretKeySelector{Name: "my-secret", Key: "credentials"},
							},
						},
					},
				},
			},
		}

		_, err := clusterProjectID(t.Context(), store.GetSpec())
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "unable to find ProjectID")
	})
}

func TestClusterProjectIDWorkloadIdentityFederation(t *testing.T) {
	originalFactory := metadataClientFactory
	defer func() { metadataClientFactory = originalFactory }()

	t.Run("metadata fallback", func(t *testing.T) {
		metadataClientFactory = func() MetadataClient {
			return &fakeMetadataClient{metadata: map[string]string{"project-id": "metadata-project-999"}}
		}

		store := &esv1.SecretStore{
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					GCPSM: &esv1.GCPSMProvider{
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

		projectID, err := clusterProjectID(t.Context(), store.GetSpec())
		assert.Nil(t, err)
		assert.Equal(t, "metadata-project-999", projectID)
	})

	t.Run("explicit projectID takes precedence", func(t *testing.T) {
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

		projectID, err := clusterProjectID(t.Context(), store.GetSpec())
		assert.Nil(t, err)
		assert.Equal(t, "explicit-secrets-456", projectID)
	})

	t.Run("no projectID and no metadata should fail", func(t *testing.T) {
		metadataClientFactory = func() MetadataClient {
			return &fakeMetadataClient{metadata: map[string]string{}}
		}

		store := &esv1.SecretStore{
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					GCPSM: &esv1.GCPSMProvider{
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

		_, err := clusterProjectID(t.Context(), store.GetSpec())
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "unable to find ProjectID")
	})
}

func TestClusterProjectIDDefaultCredentials(t *testing.T) {
	originalFactory := metadataClientFactory
	defer func() { metadataClientFactory = originalFactory }()

	t.Run("metadata fallback", func(t *testing.T) {
		metadataClientFactory = func() MetadataClient {
			return &fakeMetadataClient{metadata: map[string]string{"project-id": "metadata-project-999"}}
		}

		store := &esv1.SecretStore{
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					GCPSM: &esv1.GCPSMProvider{},
				},
			},
		}

		projectID, err := clusterProjectID(t.Context(), store.GetSpec())
		assert.Nil(t, err)
		assert.Equal(t, "metadata-project-999", projectID)
	})

	t.Run("explicit projectID takes precedence", func(t *testing.T) {
		metadataClientFactory = func() MetadataClient {
			return &fakeMetadataClient{metadata: map[string]string{"project-id": "metadata-999"}}
		}

		store := &esv1.SecretStore{
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					GCPSM: &esv1.GCPSMProvider{
						ProjectID: "explicit-secrets-111",
					},
				},
			},
		}

		projectID, err := clusterProjectID(t.Context(), store.GetSpec())
		assert.Nil(t, err)
		assert.Equal(t, "explicit-secrets-111", projectID)
	})

	t.Run("no projectID and no metadata should fail", func(t *testing.T) {
		metadataClientFactory = func() MetadataClient {
			return &fakeMetadataClient{metadata: map[string]string{}}
		}

		store := &esv1.SecretStore{
			Spec: esv1.SecretStoreSpec{
				Provider: &esv1.SecretStoreProvider{
					GCPSM: &esv1.GCPSMProvider{},
				},
			},
		}

		_, err := clusterProjectID(t.Context(), store.GetSpec())
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "unable to find ProjectID")
	})
}

func TestValidateStoreNilGCPSM(t *testing.T) {
	p := &Provider{}

	_, err := p.ValidateStore(&esv1.SecretStore{
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				GCPSM: nil,
			},
		},
	})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), errInvalidGCPProv)
}
