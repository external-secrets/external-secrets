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
	"testing"

	secretmanagerpb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/stretchr/testify/assert"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	testingfake "github.com/external-secrets/external-secrets/runtime/testing/fake"
)

func TestBuildMetadata(t *testing.T) {
	tests := []struct {
		name                string
		labels              map[string]string
		metadata            *apiextensionsv1.JSON
		expectedError       bool
		expectedLabels      map[string]string
		expectedAnnotations map[string]string
		expectedTopics      []string
	}{
		{
			name: "secret not managed by external secrets",
			labels: map[string]string{
				"someKey": "someValue",
			},
			expectedError: true,
		},
		{
			name: "metadata with default MergePolicy of Replace",
			labels: map[string]string{
				managedByKey:   managedByValue,
				"someOtherKey": "someOtherValue",
			},
			metadata: &apiextensionsv1.JSON{
				Raw: []byte(`{
					"apiVersion": "kubernetes.external-secrets.io/v1alpha1",
					"kind": "PushSecretMetadata",
					"spec": {
						"annotations": {"key1":"value1"},
						"labels": {"key2":"value2"}
					}
				}`),
			},
			expectedError: false,
			expectedLabels: map[string]string{
				managedByKey: managedByValue,
				"key2":       "value2",
			},
			expectedAnnotations: map[string]string{
				"key1": "value1",
			},
			expectedTopics: nil,
		},
		{
			name: "metadata with merge policy",
			labels: map[string]string{
				managedByKey:  managedByValue,
				"existingKey": "existingValue",
			},
			metadata: &apiextensionsv1.JSON{
				Raw: []byte(`{
					"apiVersion": "kubernetes.external-secrets.io/v1alpha1",
					"kind": "PushSecretMetadata",
					"spec": {
						"annotations": {"key1":"value1"},
						"labels": {"key2":"value2"},
						"mergePolicy": "Merge"
					}
				}`),
			},
			expectedError: false,
			expectedLabels: map[string]string{
				managedByKey:  managedByValue,
				"existingKey": "existingValue",
				"key2":        "value2",
			},
			expectedAnnotations: map[string]string{
				"key1": "value1",
			},
			expectedTopics: nil,
		},
		{
			name: "metadata with CMEK key name",
			labels: map[string]string{
				managedByKey: managedByValue,
			},
			metadata: &apiextensionsv1.JSON{
				Raw: []byte(`{
					"apiVersion": "kubernetes.external-secrets.io/v1alpha1",
					"kind": "PushSecretMetadata",
					"spec": {
						"annotations": {"key1":"value1"},
						"labels": {"key2":"value2"},
						"cmekKeyName": "projects/my-project/locations/us-east1/keyRings/my-keyring/cryptoKeys/my-key"
					}
				}`),
			},
			expectedError: false,
			expectedLabels: map[string]string{
				managedByKey: managedByValue,
				"key2":       "value2",
			},
			expectedAnnotations: map[string]string{
				"key1": "value1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			psData := testingfake.PushSecretData{
				Metadata: tt.metadata,
			}
			builder := &psBuilder{
				pushSecretData: psData,
			}

			annotations, labels, topics, err := builder.buildMetadata(nil, tt.labels, nil)
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedLabels, labels)
				assert.Equal(t, tt.expectedAnnotations, annotations)
				assert.Equal(t, tt.expectedTopics, topics)
			}
		})
	}
}

func TestBuildReplication(t *testing.T) {
	const cmek = "projects/p/locations/global/keyRings/r/cryptoKeys/k"

	tests := []struct {
		name              string
		spec              PushSecretMetadataSpec
		wantNil           bool
		wantUserManaged   bool
		expectedLocations []string
		expectedCMEK      string
	}{
		{
			name:    "no replication configured",
			spec:    PushSecretMetadataSpec{},
			wantNil: true,
		},
		{
			name: "single location via deprecated field",
			spec: PushSecretMetadataSpec{
				ReplicationLocation: "us-east1",
			},
			wantUserManaged:   true,
			expectedLocations: []string{"us-east1"},
		},
		{
			name: "single location via new field",
			spec: PushSecretMetadataSpec{
				ReplicationLocations: []string{"us-east1"},
			},
			wantUserManaged:   true,
			expectedLocations: []string{"us-east1"},
		},
		{
			name: "multiple locations",
			spec: PushSecretMetadataSpec{
				ReplicationLocations: []string{"us-east1", "europe-west1", "asia-southeast1"},
			},
			wantUserManaged:   true,
			expectedLocations: []string{"us-east1", "europe-west1", "asia-southeast1"},
		},
		{
			name: "new field takes precedence over deprecated when both set",
			spec: PushSecretMetadataSpec{
				ReplicationLocation:  "us-east1",
				ReplicationLocations: []string{"europe-west1", "asia-southeast1"},
			},
			wantUserManaged:   true,
			expectedLocations: []string{"europe-west1", "asia-southeast1"},
		},
		{
			name: "multiple locations with CMEK applied to all",
			spec: PushSecretMetadataSpec{
				ReplicationLocations: []string{"us-east1", "europe-west1"},
				CMEKKeyName:          cmek,
			},
			wantUserManaged:   true,
			expectedLocations: []string{"us-east1", "europe-west1"},
			expectedCMEK:      cmek,
		},
		{
			name: "CMEK without locations falls back to automatic replication carrying CMEK",
			spec: PushSecretMetadataSpec{
				CMEKKeyName: cmek,
			},
			expectedCMEK: cmek,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildReplication(tt.spec)
			if tt.wantNil {
				assert.Nil(t, got)
				return
			}
			assert.NotNil(t, got)

			if tt.wantUserManaged {
				um, ok := got.Replication.(*secretmanagerpb.Replication_UserManaged_)
				assert.True(t, ok, "expected UserManaged replication")
				assert.Len(t, um.UserManaged.Replicas, len(tt.expectedLocations))
				for i, loc := range tt.expectedLocations {
					assert.Equal(t, loc, um.UserManaged.Replicas[i].Location)
					if tt.expectedCMEK != "" {
						assert.NotNil(t, um.UserManaged.Replicas[i].CustomerManagedEncryption)
						assert.Equal(t, tt.expectedCMEK, um.UserManaged.Replicas[i].CustomerManagedEncryption.KmsKeyName)
					} else {
						assert.Nil(t, um.UserManaged.Replicas[i].CustomerManagedEncryption)
					}
				}
				return
			}

			// CMEK-only path: expect Automatic replication with CMEK attached.
			auto, ok := got.Replication.(*secretmanagerpb.Replication_Automatic_)
			assert.True(t, ok, "expected Automatic replication")
			assert.NotNil(t, auto.Automatic.CustomerManagedEncryption)
			assert.Equal(t, tt.expectedCMEK, auto.Automatic.CustomerManagedEncryption.KmsKeyName)
		})
	}
}
