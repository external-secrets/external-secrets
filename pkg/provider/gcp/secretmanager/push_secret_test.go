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
	"testing"

	"github.com/stretchr/testify/assert"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	testingfake "github.com/external-secrets/external-secrets/pkg/provider/testing/fake"
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
