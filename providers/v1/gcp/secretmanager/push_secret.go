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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"strings"

	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/tidwall/sjson"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils/metadata"
)

// PushSecretMetadataMergePolicy defines how metadata should be merged when pushing secrets.
type PushSecretMetadataMergePolicy string

const (
	// PushSecretMetadataMergePolicyReplace indicates that metadata should be replaced entirely.
	PushSecretMetadataMergePolicyReplace PushSecretMetadataMergePolicy = "Replace"
	// PushSecretMetadataMergePolicyMerge indicates that metadata should be merged.
	PushSecretMetadataMergePolicyMerge PushSecretMetadataMergePolicy = "Merge"
)

// PushSecretMetadataSpec defines the metadata specification for pushed secrets.
type PushSecretMetadataSpec struct {
	Annotations map[string]string             `json:"annotations,omitempty"`
	Labels      map[string]string             `json:"labels,omitempty"`
	Topics      []string                      `json:"topics,omitempty"`
	MergePolicy PushSecretMetadataMergePolicy `json:"mergePolicy,omitempty"`
	CMEKKeyName string                        `json:"cmekKeyName,omitempty"`
	// ReplicationLocation defines a single user-managed replication location
	// for the secret.
	//
	// Deprecated: use ReplicationLocations instead. When both fields are set,
	// ReplicationLocations takes precedence and ReplicationLocation is ignored.
	ReplicationLocation string `json:"replicationLocation,omitempty"`
	// ReplicationLocations defines one or more user-managed replication
	// locations for the secret. Each entry may carry its own CMEK key; see
	// PushSecretReplica. Plain location strings are also accepted as entries.
	ReplicationLocations []PushSecretReplica `json:"replicationLocations,omitempty"`
}

// PushSecretReplica pairs a user-managed replication location with the
// customer-managed encryption key used for that replica. CMEKKeyName is the
// full KMS key resource name, e.g.
// "projects/my-project/locations/us-east1/keyRings/my-keyring/cryptoKeys/my-key";
// leave it empty to use Google-managed encryption for that replica.
type PushSecretReplica struct {
	Location    string `json:"location"`
	CMEKKeyName string `json:"cmekKeyName,omitempty"`
}

// UnmarshalJSON accepts both a plain location string and an object with
// "location" and "cmekKeyName" fields, so existing metadata entries written
// as strings keep parsing after per-replica CMEK support was added.
func (r *PushSecretReplica) UnmarshalJSON(data []byte) error {
	var location string
	if err := json.Unmarshal(data, &location); err == nil {
		if location == "" {
			return errors.New("replicationLocations string entry must not be empty")
		}
		r.Location = location
		return nil
	}

	// Alias to strip the UnmarshalJSON method and avoid recursion.
	type replicaAlias PushSecretReplica
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var replica replicaAlias
	if err := decoder.Decode(&replica); err != nil {
		return fmt.Errorf("replicationLocations entry must be a location string or an object with \"location\" and optional \"cmekKeyName\": %w", err)
	}
	if replica.Location == "" {
		return errors.New("replicationLocations object entry requires a \"location\" field")
	}
	*r = PushSecretReplica(replica)
	return nil
}

// buildReplication converts a PushSecretMetadataSpec into the
// secretmanagerpb.Replication to use when creating a new Secret. It returns
// a nil replication when the spec supplies no replication configuration at
// all, so the caller keeps the default (Google-managed) automatic replication.
//
// The deprecated ReplicationLocation is normalized into a single
// ReplicationLocations entry, and entries without their own CMEKKeyName
// inherit the top-level CMEKKeyName. A CMEKKeyName with no locations falls
// back to Automatic replication carrying the key so the user's choice is not
// silently dropped; callers that do not use automatic replication (regional
// secrets) must skip the returned replication, and the global-location
// validation only makes sense in that context.
func buildReplication(spec PushSecretMetadataSpec, regionalStore bool) (*secretmanagerpb.Replication, error) {
	if len(spec.ReplicationLocations) == 0 && spec.ReplicationLocation != "" {
		spec.ReplicationLocations = []PushSecretReplica{{Location: spec.ReplicationLocation, CMEKKeyName: spec.CMEKKeyName}}
	}

	// Entries without their own key inherit the top-level CMEKKeyName. Copy
	// first so the caller's parsed spec is never mutated.
	replicasSpec := make([]PushSecretReplica, len(spec.ReplicationLocations))
	copy(replicasSpec, spec.ReplicationLocations)
	for i := range replicasSpec {
		if replicasSpec[i].CMEKKeyName == "" {
			replicasSpec[i].CMEKKeyName = spec.CMEKKeyName
		}
	}
	spec.ReplicationLocations = replicasSpec

	// Keep the previous behavior of not silently dropping a CMEK key when no
	// locations are given.
	if len(spec.ReplicationLocations) == 0 {
		if spec.CMEKKeyName == "" || regionalStore {
			// Regional secrets cannot carry replication at all; the key is
			// applied through the regional endpoint instead.
			return nil, nil
		}
		if err := validateGlobalCMEKKeyName(spec.CMEKKeyName); err != nil {
			return nil, err
		}
		return &secretmanagerpb.Replication{
			Replication: &secretmanagerpb.Replication_Automatic_{
				Automatic: &secretmanagerpb.Replication_Automatic{
					CustomerManagedEncryption: &secretmanagerpb.CustomerManagedEncryption{
						KmsKeyName: spec.CMEKKeyName,
					},
				},
			},
		}, nil
	}

	if regionalStore {
		// Regional secrets cannot carry replication at all; the replication
		// metadata is not used and its validity is not enforced.
		return nil, nil
	}

	replicas := make([]*secretmanagerpb.Replication_UserManaged_Replica, 0, len(spec.ReplicationLocations))
	for _, r := range spec.ReplicationLocations {
		replica := &secretmanagerpb.Replication_UserManaged_Replica{
			Location: r.Location,
		}
		if r.CMEKKeyName != "" {
			if err := validateReplicaCMEKKeyName(r.Location, r.CMEKKeyName); err != nil {
				return nil, err
			}
			replica.CustomerManagedEncryption = &secretmanagerpb.CustomerManagedEncryption{
				KmsKeyName: r.CMEKKeyName,
			}
		}
		replicas = append(replicas, replica)
	}
	return &secretmanagerpb.Replication{
		Replication: &secretmanagerpb.Replication_UserManaged_{
			UserManaged: &secretmanagerpb.Replication_UserManaged{
				Replicas: replicas,
			},
		},
	}, nil
}

// cmekKeyLocation returns the location segment of a KMS key resource name
// (e.g. "global" or "us-east1"). Keys without a location segment are left to
// the Secret Manager API to reject.
func cmekKeyLocation(key string) (string, bool) {
	const marker = "/locations/"
	_, rest, found := strings.Cut(key, marker)
	if !found {
		return "", false
	}
	location, _, _ := strings.Cut(rest, "/")
	return location, true
}

// validateReplicaCMEKKeyName reports an error when the given KMS key resource
// name is pinned to a location other than the replica's. Each user-managed
// replica must use a CMEK key from the replica's own location.
func validateReplicaCMEKKeyName(location, key string) error {
	keyLocation, found := cmekKeyLocation(key)
	if !found {
		return nil
	}
	if keyLocation != location {
		return fmt.Errorf("each user-managed replica must use a CMEK key from its own location: "+
			"replica %q got key in location %q", location, keyLocation)
	}
	return nil
}

// validateGlobalCMEKKeyName reports an error when the given KMS key resource
// name is pinned to a non-global location. Automatic replication only accepts
// CMEK keys from the "global" multi-region; regional keys require
// user-managed replication with a matching replica.
func validateGlobalCMEKKeyName(key string) error {
	location, found := cmekKeyLocation(key)
	if !found {
		return nil
	}
	if location != "global" {
		return fmt.Errorf("a CMEK key used without replicationLocations must come from the \"global\" location, "+
			"got key in location %q: use replicationLocations entries with per-replica keys for regional CMEK", location)
	}
	return nil
}

func newPushSecretBuilder(payload []byte, data esv1.PushSecretData) (pushSecretBuilder, error) {
	if data.GetProperty() == "" {
		return &psBuilder{
			payload:        payload,
			pushSecretData: data,
		}, nil
	}

	if data.GetMetadata() != nil {
		return nil, errors.New("cannot specify metadata and property at the same time")
	}

	return &propertyPSBuilder{
		payload:        payload,
		pushSecretData: data,
	}, nil
}

type pushSecretBuilder interface {
	buildMetadata(annotations, labels map[string]string, topics []*secretmanagerpb.Topic) (map[string]string, map[string]string, []string, error)
	needUpdate(original []byte) bool
	buildData(original []byte) ([]byte, error)
}

type psBuilder struct {
	payload        []byte
	pushSecretData esv1.PushSecretData
}

func (b *psBuilder) buildMetadata(_, labels map[string]string, _ []*secretmanagerpb.Topic) (map[string]string, map[string]string, []string, error) {
	if manager, ok := labels[managedByKey]; !ok || manager != managedByValue {
		return nil, nil, nil, fmt.Errorf("secret %v is not managed by external secrets", b.pushSecretData.GetRemoteKey())
	}

	var meta *metadata.PushSecretMetadata[PushSecretMetadataSpec]
	if b.pushSecretData.GetMetadata() != nil {
		var err error
		meta, err = metadata.ParseMetadataParameters[PushSecretMetadataSpec](b.pushSecretData.GetMetadata())
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to parse PushSecret metadata: %w", err)
		}
	}

	var spec PushSecretMetadataSpec
	if meta != nil {
		spec = meta.Spec
	}

	newLabels := map[string]string{}
	maps.Copy(newLabels, spec.Labels)
	if spec.MergePolicy == PushSecretMetadataMergePolicyMerge {
		// Keep labels from the existing GCP Secret Manager Secret
		maps.Copy(newLabels, labels)
	}
	newLabels[managedByKey] = managedByValue

	return spec.Annotations, newLabels, spec.Topics, nil
}

func (b *psBuilder) needUpdate(original []byte) bool {
	if original == nil {
		return true
	}

	return !bytes.Equal(b.payload, original)
}

func (b *psBuilder) buildData(_ []byte) ([]byte, error) {
	return b.payload, nil
}

type propertyPSBuilder struct {
	payload        []byte
	pushSecretData esv1.PushSecretData
}

func (b *propertyPSBuilder) buildMetadata(annotations, labels map[string]string, topics []*secretmanagerpb.Topic) (map[string]string, map[string]string, []string, error) {
	newAnnotations := map[string]string{}
	newLabels := map[string]string{}
	if annotations != nil {
		newAnnotations = annotations
	}
	if labels != nil {
		newLabels = labels
	}

	newLabels[managedByKey] = managedByValue

	result := make([]string, 0, len(topics))
	for _, t := range topics {
		result = append(result, t.Name)
	}

	return newAnnotations, newLabels, result, nil
}

func (b *propertyPSBuilder) needUpdate(original []byte) bool {
	if original == nil {
		return true
	}

	val, _ := getDataByProperty(original, b.pushSecretData.GetProperty())
	return !val.Exists() || val.String() != string(b.payload)
}

func (b *propertyPSBuilder) buildData(original []byte) ([]byte, error) {
	var base []byte
	if original != nil {
		base = original
	}
	return sjson.SetBytes(base, b.pushSecretData.GetProperty(), b.payload)
}
