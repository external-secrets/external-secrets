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
	"encoding/json"
	"testing"

	secretmanagerpb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"github.com/external-secrets/external-secrets/runtime/esutils/metadata"
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

	userManaged := func(replicas ...*secretmanagerpb.Replication_UserManaged_Replica) *secretmanagerpb.Replication {
		return &secretmanagerpb.Replication{
			Replication: &secretmanagerpb.Replication_UserManaged_{
				UserManaged: &secretmanagerpb.Replication_UserManaged{Replicas: replicas},
			},
		}
	}
	automatic := func(key string) *secretmanagerpb.Replication {
		return &secretmanagerpb.Replication{
			Replication: &secretmanagerpb.Replication_Automatic_{
				Automatic: &secretmanagerpb.Replication_Automatic{
					CustomerManagedEncryption: &secretmanagerpb.CustomerManagedEncryption{KmsKeyName: key},
				},
			},
		}
	}
	replica := func(loc, key string) *secretmanagerpb.Replication_UserManaged_Replica {
		r := &secretmanagerpb.Replication_UserManaged_Replica{Location: loc}
		if key != "" {
			r.CustomerManagedEncryption = &secretmanagerpb.CustomerManagedEncryption{KmsKeyName: key}
		}
		return r
	}

	tests := []struct {
		name          string
		spec          PushSecretMetadataSpec
		regionalStore bool
		want          *secretmanagerpb.Replication
		expectedErr   string
	}{
		{
			name: "no replication configured",
			spec: PushSecretMetadataSpec{},
			want: nil,
		},
		{
			name: "no replication configured for a regional store",
			spec: PushSecretMetadataSpec{},
			// Regional secrets cannot carry replication at all.
			regionalStore: true,
			want:          nil,
		},
		{
			name: "single location via deprecated field",
			spec: PushSecretMetadataSpec{ReplicationLocation: "us-east1"},
			want: userManaged(replica("us-east1", "")),
		},
		{
			name: "single location via new field",
			spec: PushSecretMetadataSpec{ReplicationLocations: []PushSecretReplica{{Location: "us-east1"}}},
			want: userManaged(replica("us-east1", "")),
		},
		{
			name: "multiple locations",
			spec: PushSecretMetadataSpec{ReplicationLocations: []PushSecretReplica{
				{Location: "us-east1"}, {Location: "europe-west1"}, {Location: "asia-southeast1"},
			}},
			want: userManaged(replica("us-east1", ""), replica("europe-west1", ""), replica("asia-southeast1", "")),
		},
		{
			name: "new field takes precedence over deprecated when both set",
			spec: PushSecretMetadataSpec{
				ReplicationLocation:  "us-east1",
				ReplicationLocations: []PushSecretReplica{{Location: "europe-west1"}, {Location: "asia-southeast1"}},
			},
			want: userManaged(replica("europe-west1", ""), replica("asia-southeast1", "")),
		},
		{
			name: "top-level regional CMEK applied to its matching location",
			spec: PushSecretMetadataSpec{
				ReplicationLocations: []PushSecretReplica{{Location: "us-east1"}},
				CMEKKeyName:          "projects/p/locations/us-east1/keyRings/r/cryptoKeys/k",
			},
			want: userManaged(replica("us-east1", "projects/p/locations/us-east1/keyRings/r/cryptoKeys/k")),
		},
		{
			name: "inherited CMEK key location mismatch rejected",
			spec: PushSecretMetadataSpec{
				ReplicationLocations: []PushSecretReplica{{Location: "europe-west1"}},
				CMEKKeyName:          "projects/p/locations/us-east1/keyRings/r/cryptoKeys/k",
			},
			expectedErr: "must come from the \"europe-west1\" location",
		},
		{
			name: "global top-level CMEK cannot be inherited by user-managed replicas",
			spec: PushSecretMetadataSpec{
				ReplicationLocations: []PushSecretReplica{{Location: "us-east1"}, {Location: "europe-west1"}},
				CMEKKeyName:          cmek,
			},
			expectedErr: "must come from the \"us-east1\" location",
		},
		{
			name: "CMEK without locations falls back to automatic replication carrying CMEK",
			spec: PushSecretMetadataSpec{CMEKKeyName: cmek},
			want: automatic(cmek),
		},
		{
			name:        "regional CMEK without locations rejected on the automatic path",
			spec:        PushSecretMetadataSpec{CMEKKeyName: "projects/p/locations/us-east1/keyRings/r/cryptoKeys/k"},
			expectedErr: "cannot be used: the CMEK key must come from the \"global\" location",
		},
		{
			name: "regional CMEK without locations accepted for a regional store",
			spec: PushSecretMetadataSpec{CMEKKeyName: "projects/p/locations/us-east1/keyRings/r/cryptoKeys/k"},
			// Documented regional-secrets pattern: the key is applied through
			// the regional endpoint, so no replication is emitted and no
			// global-location validation applies.
			regionalStore: true,
			want:          nil,
		},
		{
			name: "replication metadata is not validated for a regional store",
			spec: PushSecretMetadataSpec{
				ReplicationLocations: []PushSecretReplica{{Location: "europe-west1", CMEKKeyName: "projects/p/locations/us-east1/keyRings/r/cryptoKeys/k"}},
			},
			// The regional path never emits replication, so its metadata is
			// not validated either.
			regionalStore: true,
			want:          nil,
		},
		{
			name: "per-replica CMEK keys",
			spec: PushSecretMetadataSpec{
				ReplicationLocations: []PushSecretReplica{
					{Location: "us-east4", CMEKKeyName: "projects/p/locations/us-east4/keyRings/r/cryptoKeys/k1"},
					{Location: "us-west3", CMEKKeyName: "projects/p/locations/us-west3/keyRings/r/cryptoKeys/k2"},
				},
			},
			want: userManaged(replica("us-east4", "projects/p/locations/us-east4/keyRings/r/cryptoKeys/k1"), replica("us-west3", "projects/p/locations/us-west3/keyRings/r/cryptoKeys/k2")),
		},
		{
			name: "per-replica keys override the top-level CMEK",
			spec: PushSecretMetadataSpec{
				ReplicationLocations: []PushSecretReplica{
					{Location: "us-east1", CMEKKeyName: "projects/p/locations/us-east1/keyRings/r/cryptoKeys/own"},
				},
				CMEKKeyName: "projects/p/locations/global/keyRings/r/cryptoKeys/top",
			},
			want: userManaged(replica("us-east1", "projects/p/locations/us-east1/keyRings/r/cryptoKeys/own")),
		},
		{
			name: "mixed entries with and without per-replica CMEK",
			spec: PushSecretMetadataSpec{
				ReplicationLocations: []PushSecretReplica{
					{Location: "us-east1", CMEKKeyName: "projects/p/locations/us-east1/keyRings/r/cryptoKeys/k"},
					{Location: "europe-west1", CMEKKeyName: "projects/p/locations/europe-west1/keyRings/r/cryptoKeys/k2"},
				},
			},
			want: userManaged(replica("us-east1", "projects/p/locations/us-east1/keyRings/r/cryptoKeys/k"), replica("europe-west1", "projects/p/locations/europe-west1/keyRings/r/cryptoKeys/k2")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildReplication(tt.spec, tt.regionalStore)
			if tt.expectedErr != "" {
				assert.ErrorContains(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.True(t, proto.Equal(tt.want, got), "want %v, got %v", tt.want, got)
			}
		})
	}
}

func TestReplicationLocationsUnmarshal(t *testing.T) {
	tests := []struct {
		name        string
		raw         string
		want        []PushSecretReplica
		expectedErr string
	}{
		{
			name: "plain location strings",
			raw:  `["us-east1", "europe-west1"]`,
			want: []PushSecretReplica{{Location: "us-east1"}, {Location: "europe-west1"}},
		},
		{
			name: "objects with per-replica CMEK",
			raw:  `[{"location":"us-east4","cmekKeyName":"projects/p/locations/us-east4/keyRings/r/cryptoKeys/k1"}]`,
			want: []PushSecretReplica{{Location: "us-east4", CMEKKeyName: "projects/p/locations/us-east4/keyRings/r/cryptoKeys/k1"}},
		},
		{
			name: "mixed strings and objects",
			raw:  `["us-east1", {"location":"us-west3"}]`,
			want: []PushSecretReplica{{Location: "us-east1"}, {Location: "us-west3"}},
		},
		{
			name:        "unknown fields rejected",
			raw:         `[{"location":"us-east1","bogus":"x"}]`,
			expectedErr: "unknown field",
		},
		{
			name:        "object without location rejected",
			raw:         `[{"cmekKeyName":"projects/p/locations/us-east1/keyRings/r/cryptoKeys/k"}]`,
			expectedErr: "requires a \"location\" field",
		},
		{
			name:        "non-string non-object rejected",
			raw:         `[42]`,
			expectedErr: "must be a location string or an object",
		},
		{
			name:        "empty string entry rejected",
			raw:         `[""]`,
			expectedErr: "must not be empty",
		},
		{
			name:        "null entry rejected",
			raw:         `[null]`,
			expectedErr: "must not be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []PushSecretReplica
			err := json.Unmarshal([]byte(tt.raw), &got)
			if tt.expectedErr != "" {
				assert.ErrorContains(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestParseMetadataWithStructuredReplicationLocations(t *testing.T) {
	// Exercises the full metadata parse path (YAML -> JSON -> PushSecretReplica
	// unmarshalling) with both string and object entries.
	raw := []byte(`{
		"apiVersion": "kubernetes.external-secrets.io/v1alpha1",
		"kind": "PushSecretMetadata",
		"spec": {
			"cmekKeyName": "projects/p/locations/us-east1/keyRings/r/cryptoKeys/top",
			"replicationLocations": [
				"us-east1",
				{"location": "us-east4", "cmekKeyName": "projects/p/locations/us-east4/keyRings/r/cryptoKeys/k1"}
			]
		}
	}`)
	meta, err := metadata.ParseMetadataParameters[PushSecretMetadataSpec](&apiextensionsv1.JSON{Raw: raw})
	assert.NoError(t, err)
	replication, err := buildReplication(meta.Spec, false)
	assert.NoError(t, err)
	replicas := replication.GetUserManaged().GetReplicas()
	want := []PushSecretReplica{
		{Location: "us-east1", CMEKKeyName: "projects/p/locations/us-east1/keyRings/r/cryptoKeys/top"},
		{Location: "us-east4", CMEKKeyName: "projects/p/locations/us-east4/keyRings/r/cryptoKeys/k1"},
	}
	assert.Len(t, replicas, len(want))
	for i := range want {
		assert.True(t, proto.Equal(&secretmanagerpb.Replication_UserManaged_Replica{
			Location: want[i].Location,
			CustomerManagedEncryption: func() *secretmanagerpb.CustomerManagedEncryption {
				if want[i].CMEKKeyName == "" {
					return nil
				}
				return &secretmanagerpb.CustomerManagedEncryption{KmsKeyName: want[i].CMEKKeyName}
			}(),
		}, replicas[i]), "replica %d mismatch", i)
	}
}

func TestValidateCMEKKeyLocation(t *testing.T) {
	tests := []struct {
		name        string
		location    string
		key         string
		expectedErr bool
	}{
		{
			name:     "global location expected and matched",
			location: "global",
			key:      "projects/p/locations/global/keyRings/r/cryptoKeys/k",
		},
		{
			name:     "matching regional location accepted",
			location: "us-east1",
			key:      "projects/p/locations/us-east1/keyRings/r/cryptoKeys/k",
		},
		{
			name:     "no location segment left to the API",
			location: "us-east1",
			key:      "projects/p/keyRings/r/cryptoKeys/k",
		},
		{
			name:        "regional keyring location rejected for global",
			location:    "global",
			key:         "projects/p/locations/us-east1/keyRings/r/cryptoKeys/k",
			expectedErr: true,
		},
		{
			name:        "multi-region keyring location rejected for a regional replica",
			location:    "us-east1",
			key:         "projects/p/locations/europe/keyRings/r/cryptoKeys/k",
			expectedErr: true,
		},
		{
			name:        "different region rejected",
			location:    "us-east1",
			key:         "projects/p/locations/us-west3/keyRings/r/cryptoKeys/k",
			expectedErr: true,
		},
		{
			name:        "global key rejected for a regional replica",
			location:    "us-east1",
			key:         "projects/p/locations/global/keyRings/r/cryptoKeys/k",
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCMEKKeyLocation(tt.location, tt.key)
			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
