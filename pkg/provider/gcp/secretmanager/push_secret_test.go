package secretmanager

import (
	"testing"

	testingfake "github.com/external-secrets/external-secrets/pkg/provider/testing/fake"
	"github.com/stretchr/testify/assert"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
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
				Raw: []byte(`{"annotations":{"key1":"value1"},"labels":{"key2":"value2"}}`),
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
				Raw: []byte(`{"annotations":{"key1":"value1"},"labels":{"key2":"value2"},"mergePolicy":"Merge"}`),
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
