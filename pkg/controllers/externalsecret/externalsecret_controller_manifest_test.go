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

package externalsecret

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

func TestIsGenericTarget(t *testing.T) {
	tests := []struct {
		name     string
		es       *esv1.ExternalSecret
		expected bool
	}{
		{
			name: "nil manifest - Secret target",
			es: &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{
						Manifest: nil,
					},
				},
			},
			expected: false,
		},
		{
			name: "ConfigMap manifest target",
			es: &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{
						Manifest: &esv1.ManifestReference{
							APIVersion: "v1",
							Kind:       "ConfigMap",
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "Custom Resource manifest target",
			es: &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{
						Manifest: &esv1.ManifestReference{
							APIVersion: "argoproj.io/v1alpha1",
							Kind:       "Application",
						},
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isGenericTarget(tt.es)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateGenericTarget(t *testing.T) {
	tests := []struct {
		name                string
		es                  *esv1.ExternalSecret
		allowGenericTargets bool
		expectedError       bool
		errorContains       string
	}{
		{
			name: "ConfigMap target - flag enabled - valid",
			es: &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{
						Manifest: &esv1.ManifestReference{
							APIVersion: "v1",
							Kind:       "ConfigMap",
						},
					},
				},
			},
			allowGenericTargets: true,
			expectedError:       false,
		},
		{
			name: "ConfigMap target - flag disabled",
			es: &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{
						Manifest: &esv1.ManifestReference{
							APIVersion: "v1",
							Kind:       "ConfigMap",
						},
					},
				},
			},
			allowGenericTargets: false,
			expectedError:       true,
			errorContains:       "generic targets are disabled",
		},
		{
			name: "Missing APIVersion",
			es: &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{
						Manifest: &esv1.ManifestReference{
							APIVersion: "",
							Kind:       "ConfigMap",
						},
					},
				},
			},
			allowGenericTargets: true,
			expectedError:       true,
			errorContains:       "apiVersion is required",
		},
		{
			name: "Missing Kind",
			es: &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{
						Manifest: &esv1.ManifestReference{
							APIVersion: "v1",
							Kind:       "",
						},
					},
				},
			},
			allowGenericTargets: true,
			expectedError:       true,
			errorContains:       "kind is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				AllowGenericTargets: tt.allowGenericTargets,
			}
			log := ctrl.Log.WithName("test")

			err := r.validateGenericTarget(log, tt.es)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetTargetGVK(t *testing.T) {
	tests := []struct {
		name     string
		es       *esv1.ExternalSecret
		expected schema.GroupVersionKind
	}{
		{
			name: "ConfigMap target",
			es: &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{
						Manifest: &esv1.ManifestReference{
							APIVersion: "v1",
							Kind:       "ConfigMap",
						},
					},
				},
			},
			expected: schema.GroupVersionKind{
				Group:   "",
				Version: "v1",
				Kind:    "ConfigMap",
			},
		},
		{
			name: "ArgoCD Application target",
			es: &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{
						Manifest: &esv1.ManifestReference{
							APIVersion: "argoproj.io/v1alpha1",
							Kind:       "Application",
						},
					},
				},
			},
			expected: schema.GroupVersionKind{
				Group:   "argoproj.io",
				Version: "v1alpha1",
				Kind:    "Application",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getTargetGVK(tt.es)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetTargetName(t *testing.T) {
	tests := []struct {
		name     string
		es       *esv1.ExternalSecret
		expected string
	}{
		{
			name: "Use target name when specified",
			es: &esv1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-external-secret",
				},
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{
						Name: "custom-target-name",
					},
				},
			},
			expected: "custom-target-name",
		},
		{
			name: "Use ExternalSecret name when target name not specified",
			es: &esv1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-external-secret",
				},
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{
						Name: "",
					},
				},
			},
			expected: "my-external-secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getTargetName(tt.es)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCreateSimpleManifest(t *testing.T) {
	tests := []struct {
		name     string
		kind     string
		dataMap  map[string][]byte
		validate func(t *testing.T, obj *unstructured.Unstructured)
	}{
		{
			name: "ConfigMap with data",
			kind: "ConfigMap",
			dataMap: map[string][]byte{
				"key1": []byte("value1"),
				"key2": []byte("value2"),
			},
			validate: func(t *testing.T, obj *unstructured.Unstructured) {
				// Directly access the data field
				data, ok := obj.Object["data"].(map[string]string)
				require.True(t, ok, "data should be map[string]string")
				assert.Equal(t, "value1", data["key1"])
				assert.Equal(t, "value2", data["key2"])
			},
		},
		{
			name: "Custom resource with spec.data",
			kind: "CustomResource",
			dataMap: map[string][]byte{
				"config": []byte("my-config"),
			},
			validate: func(t *testing.T, obj *unstructured.Unstructured) {
				spec, ok := obj.Object["spec"].(map[string]interface{})
				require.True(t, ok, "spec should be map[string]interface{}")
				data, ok := spec["data"].(map[string]string)
				require.True(t, ok, "spec.data should be map[string]string")
				assert.Equal(t, "my-config", data["config"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{}
			obj := &unstructured.Unstructured{
				Object: make(map[string]interface{}),
			}
			obj.SetKind(tt.kind)

			result, err := r.createSimpleManifest(obj, tt.dataMap)

			require.NoError(t, err)
			assert.NotNil(t, result)
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestApplyTemplateToManifest_SimpleConfigMap(t *testing.T) {
	// Setup
	_ = esv1.AddToScheme(scheme.Scheme)
	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).Build()

	r := &Reconciler{
		Client: fakeClient,
	}

	es := &esv1.ExternalSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-es",
			Namespace: "default",
		},
		Spec: esv1.ExternalSecretSpec{
			Target: esv1.ExternalSecretTarget{
				Name: "test-configmap",
				Manifest: &esv1.ManifestReference{
					APIVersion: "v1",
					Kind:       "ConfigMap",
				},
			},
		},
	}

	dataMap := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
	}

	// Execute
	result, err := r.applyTemplateToManifest(context.Background(), es, dataMap)

	// Verify
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "ConfigMap", result.GetKind())
	assert.Equal(t, "test-configmap", result.GetName())
	assert.Equal(t, "default", result.GetNamespace())

	// Verify data
	data, ok := result.Object["data"].(map[string]string)
	require.True(t, ok, "data should be map[string]string")
	assert.Equal(t, "value1", data["key1"])
	assert.Equal(t, "value2", data["key2"])

	// Verify managed label
	labels := result.GetLabels()
	assert.Equal(t, esv1.LabelManagedValue, labels[esv1.LabelManaged])
}

func TestApplyTemplateToManifest_WithMetadata(t *testing.T) {
	// Setup
	_ = esv1.AddToScheme(scheme.Scheme)
	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).Build()

	r := &Reconciler{
		Client: fakeClient,
	}

	es := &esv1.ExternalSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-es",
			Namespace: "default",
		},
		Spec: esv1.ExternalSecretSpec{
			Target: esv1.ExternalSecretTarget{
				Name: "test-configmap",
				Manifest: &esv1.ManifestReference{
					APIVersion: "v1",
					Kind:       "ConfigMap",
				},
				Template: &esv1.ExternalSecretTemplate{
					EngineVersion: esv1.TemplateEngineV2, // Set engine version
					Metadata: esv1.ExternalSecretTemplateMetadata{
						Labels: map[string]string{
							"app":  "myapp",
							"tier": "backend",
						},
						Annotations: map[string]string{
							"description": "This is a test",
						},
					},
				},
			},
		},
	}

	dataMap := map[string][]byte{
		"config": []byte("test-config"),
	}

	// Execute
	result, err := r.applyTemplateToManifest(context.Background(), es, dataMap)

	// Verify
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify labels
	labels := result.GetLabels()
	assert.Equal(t, "myapp", labels["app"])
	assert.Equal(t, "backend", labels["tier"])
	assert.Equal(t, esv1.LabelManagedValue, labels[esv1.LabelManaged])

	// Verify annotations
	annotations := result.GetAnnotations()
	assert.Equal(t, "This is a test", annotations["description"])
}

func TestGetGenericResource(t *testing.T) {
	// Setup
	_ = esv1.AddToScheme(scheme.Scheme)

	// Create a ConfigMap to find
	existingConfigMap := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      "test-cm",
				"namespace": "default",
			},
			"data": map[string]interface{}{
				"key": "value",
			},
		},
	}

	_ = esv1.AddToScheme(scheme.Scheme)
	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(existingConfigMap).Build()

	r := &Reconciler{
		Client: fakeClient,
	}

	es := &esv1.ExternalSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-es",
			Namespace: "default",
		},
		Spec: esv1.ExternalSecretSpec{
			Target: esv1.ExternalSecretTarget{
				Name: "test-cm",
				Manifest: &esv1.ManifestReference{
					APIVersion: "v1",
					Kind:       "ConfigMap",
				},
			},
		},
	}

	// Execute
	result, err := r.getGenericResource(context.Background(), logr.Discard(), es)

	// Verify
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "ConfigMap", result.GetKind())
	assert.Equal(t, "test-cm", result.GetName())

	// Verify data
	data, found, err := unstructured.NestedStringMap(result.Object, "data")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "value", data["key"])
}

func TestGetGenericResource_NotFound(t *testing.T) {
	// Setup
	_ = esv1.AddToScheme(scheme.Scheme)
	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).Build()

	r := &Reconciler{
		Client: fakeClient,
	}

	es := &esv1.ExternalSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-es",
			Namespace: "default",
		},
		Spec: esv1.ExternalSecretSpec{
			Target: esv1.ExternalSecretTarget{
				Name: "nonexistent-cm",
				Manifest: &esv1.ManifestReference{
					APIVersion: "v1",
					Kind:       "ConfigMap",
				},
			},
		},
	}

	// Execute
	result, err := r.getGenericResource(context.Background(), logr.Discard(), es)

	// Verify - should return an error and nil result when resource doesn't exist
	assert.Error(t, err)
	assert.True(t, apierrors.IsNotFound(err))
	assert.Nil(t, result)
}

func init() {
	// Initialize scheme for tests
	_ = esv1.AddToScheme(scheme.Scheme)
	_ = v1.AddToScheme(scheme.Scheme)
}

func TestApplyTemplateToManifest_LiteralWithDeployment(t *testing.T) {
	// Test that literal templates work with complex objects like Deployments
	_ = esv1.AddToScheme(scheme.Scheme)
	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).Build()

	r := &Reconciler{
		Client: fakeClient,
	}

	es := &esv1.ExternalSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-es",
			Namespace: "default",
		},
		Spec: esv1.ExternalSecretSpec{
			Target: esv1.ExternalSecretTarget{
				Name: "test-deployment",
				Manifest: &esv1.ManifestReference{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
				},
				Template: &esv1.ExternalSecretTemplate{
					EngineVersion: esv1.TemplateEngineV2,
					TemplateFrom: []esv1.TemplateFrom{
						{
							Target: "spec",
							Literal: ptr.To(`
replicas: {{ .replicas }}
selector:
  matchLabels:
    app: myapp
template:
  metadata:
    labels:
      app: myapp
  spec:
    containers:
    - name: nginx
      image: nginx:{{ .version }}
      ports:
      - containerPort: 80
`),
						},
					},
				},
			},
		},
	}

	dataMap := map[string][]byte{
		"replicas": []byte("3"),
		"version":  []byte("1.21"),
	}

	result, err := r.applyTemplateToManifest(context.Background(), es, dataMap)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Deployment", result.GetKind())
	assert.Equal(t, "test-deployment", result.GetName())

	spec, found, err := unstructured.NestedMap(result.Object, "spec")
	require.NoError(t, err)
	require.True(t, found, "spec should exist")

	replicas, found, err := unstructured.NestedInt64(result.Object, "spec", "replicas")
	require.NoError(t, err)
	require.True(t, found, "spec.replicas should exist")
	assert.Equal(t, int64(3), replicas)

	containers, found, err := unstructured.NestedSlice(result.Object, "spec", "template", "spec", "containers")
	require.NoError(t, err)
	require.True(t, found, "containers should exist")
	require.Len(t, containers, 1, "should have 1 container")

	container, ok := containers[0].(map[string]any)
	require.True(t, ok, "container should be a map")
	assert.Equal(t, "nginx:1.21", container["image"])

	t.Logf("Result spec: %+v", spec)
}
