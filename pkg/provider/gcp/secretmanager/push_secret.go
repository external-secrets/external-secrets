/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

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

	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/tidwall/sjson"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

type PushSecretMetadataMergePolicy string

const (
	PushSecretMetadataMergePolicyReplace PushSecretMetadataMergePolicy = "Replace"
	PushSecretMetadataMergePolicyMerge   PushSecretMetadataMergePolicy = "Merge"
)

type Metadata struct {
	Annotations map[string]string             `json:"annotations"`
	Labels      map[string]string             `json:"labels"`
	Topics      []string                      `json:"topics,omitempty"`
	MergePolicy PushSecretMetadataMergePolicy `json:"mergePolicy,omitempty"`
}

func newPushSecretBuilder(payload []byte, data esv1beta1.PushSecretData) (pushSecretBuilder, error) {
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
	pushSecretData esv1beta1.PushSecretData
}

func (b *psBuilder) buildMetadata(_, labels map[string]string, _ []*secretmanagerpb.Topic) (map[string]string, map[string]string, []string, error) {
	if manager, ok := labels[managedByKey]; !ok || manager != managedByValue {
		return nil, nil, nil, fmt.Errorf("secret %v is not managed by external secrets", b.pushSecretData.GetRemoteKey())
	}

	var metadata Metadata
	if b.pushSecretData.GetMetadata() != nil {
		decoder := json.NewDecoder(bytes.NewReader(b.pushSecretData.GetMetadata().Raw))
		// Want to return an error if unknown fields exist
		decoder.DisallowUnknownFields()

		if err := decoder.Decode(&metadata); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to decode PushSecret metadata: %w", err)
		}

		if metadata.MergePolicy == "" {
			// Set default MergePolicy to be Replace
			metadata.MergePolicy = PushSecretMetadataMergePolicyReplace
		}
	}

	newLabels := map[string]string{}
	maps.Copy(newLabels, metadata.Labels)
	if metadata.MergePolicy == PushSecretMetadataMergePolicyMerge {
		// Keep labels from the existing GCP Secret Manager Secret
		maps.Copy(newLabels, labels)
	}
	newLabels[managedByKey] = managedByValue

	return metadata.Annotations, newLabels, metadata.Topics, nil
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
	pushSecretData esv1beta1.PushSecretData
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

	val := getDataByProperty(original, b.pushSecretData.GetProperty())
	return !val.Exists() || val.String() != string(b.payload)
}

func (b *propertyPSBuilder) buildData(original []byte) ([]byte, error) {
	var base []byte
	if original != nil {
		base = original
	}
	return sjson.SetBytes(base, b.pushSecretData.GetProperty(), b.payload)
}
