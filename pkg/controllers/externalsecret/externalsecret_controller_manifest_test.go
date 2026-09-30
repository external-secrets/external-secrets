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

package externalsecret

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
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
				spec, ok := obj.Object["spec"].(map[string]any)
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
				Object: make(map[string]any),
			}
			obj.SetKind(tt.kind)

			result := r.createSimpleManifest(obj, tt.dataMap)
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
		Scheme: scheme.Scheme,
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
		Scheme: scheme.Scheme,
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

func TestApplyTemplateToManifest_AppliesOwnershipWhenCreationPolicyOwner(t *testing.T) {
	_ = esv1.AddToScheme(scheme.Scheme)
	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	r := &Reconciler{Client: fakeClient, Scheme: scheme.Scheme}

	es := &esv1.ExternalSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-es",
			Namespace: "default",
			UID:       "abc-123",
		},
		Spec: esv1.ExternalSecretSpec{
			Target: esv1.ExternalSecretTarget{
				Name:           "test-configmap",
				CreationPolicy: esv1.CreatePolicyOwner,
				Manifest: &esv1.ManifestReference{
					APIVersion: "v1",
					Kind:       "ConfigMap",
				},
			},
		},
	}

	result, err := r.applyTemplateToManifest(context.Background(), es, map[string][]byte{"key": []byte("val")})

	require.NoError(t, err)
	owners := result.GetOwnerReferences()
	require.Len(t, owners, 1)
	assert.Equal(t, "test-es", owners[0].Name)
	assert.True(t, *owners[0].Controller)
	labels := result.GetLabels()
	assert.Equal(t, esutils.ObjectHash("default/test-es"), labels[esv1.LabelOwner])
}

func TestApplyOwnership(t *testing.T) {
	_ = esv1.AddToScheme(scheme.Scheme)
	isController := true

	tests := []struct {
		name           string
		creationPolicy esv1.ExternalSecretCreationPolicy
		existing       *unstructured.Unstructured
		expectedErr    error
		validate       func(t *testing.T, result *unstructured.Unstructured)
	}{
		{
			name:           "removes LabelOwner when policy is not Owner",
			creationPolicy: esv1.CreatePolicyOrphan,
			existing: func() *unstructured.Unstructured {
				u := &unstructured.Unstructured{}
				u.SetLabels(map[string]string{
					esv1.LabelOwner: esutils.ObjectHash("default/test-es"),
				})
				return u
			}(),
			validate: func(t *testing.T, result *unstructured.Unstructured) {
				assert.Empty(t, result.GetLabels()[esv1.LabelOwner])
			},
		},
		{
			name:           "removes owner reference when policy changes from Owner to Orphan",
			creationPolicy: esv1.CreatePolicyOrphan,
			existing: func() *unstructured.Unstructured {
				u := &unstructured.Unstructured{}
				u.SetGroupVersionKind(schema.GroupVersionKind{Version: "v1", Kind: "ConfigMap"})
				u.SetOwnerReferences([]metav1.OwnerReference{
					{
						APIVersion: esv1.SchemeGroupVersion.String(),
						Kind:       esv1.ExtSecretKind,
						Name:       "test-es",
						UID:        "abc-123",
						Controller: &isController,
					},
				})
				return u
			}(),
			validate: func(t *testing.T, result *unstructured.Unstructured) {
				assert.Empty(t, result.GetOwnerReferences())
			},
		},
		{
			name:           "returns ErrSecretIsOwned when owned by a different ExternalSecret",
			creationPolicy: esv1.CreatePolicyOwner,
			existing: func() *unstructured.Unstructured {
				u := &unstructured.Unstructured{}
				u.SetOwnerReferences([]metav1.OwnerReference{
					{
						APIVersion: esv1.SchemeGroupVersion.String(),
						Kind:       esv1.ExtSecretKind,
						Name:       "other-es",
						UID:        "xyz-999",
						Controller: &isController,
					},
				})
				return u
			}(),
			expectedErr: ErrSecretIsOwned,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).Build()
			r := &Reconciler{Client: fakeClient, Scheme: scheme.Scheme}

			es := &esv1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-es",
					Namespace: "default",
					UID:       "abc-123",
				},
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{
						Name:           "test-configmap",
						CreationPolicy: tt.creationPolicy,
						Manifest: &esv1.ManifestReference{
							APIVersion: "v1",
							Kind:       "ConfigMap",
						},
					},
				},
			}

			err := r.applyOwnership(es, tt.existing)

			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, tt.existing)
			}
		})
	}
}

func TestApplyTemplateToManifest_NoOwnerRefWhenCreationPolicyOrphan(t *testing.T) {
	_ = esv1.AddToScheme(scheme.Scheme)
	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	r := &Reconciler{Client: fakeClient, Scheme: scheme.Scheme}

	es := &esv1.ExternalSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-es",
			Namespace: "default",
			UID:       "abc-123",
		},
		Spec: esv1.ExternalSecretSpec{
			Target: esv1.ExternalSecretTarget{
				Name:           "test-configmap",
				CreationPolicy: esv1.CreatePolicyOrphan,
				Manifest: &esv1.ManifestReference{
					APIVersion: "v1",
					Kind:       "ConfigMap",
				},
			},
		},
	}

	result, err := r.applyTemplateToManifest(context.Background(), es, map[string][]byte{"key": []byte("val")})

	require.NoError(t, err)
	assert.Empty(t, result.GetOwnerReferences())
}

func TestApplyTemplateToManifest_PropagatesESLabelsAndAnnotations(t *testing.T) {
	_ = esv1.AddToScheme(scheme.Scheme)
	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	r := &Reconciler{Client: fakeClient, Scheme: scheme.Scheme}

	es := &esv1.ExternalSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-es",
			Namespace: "default",
			Labels: map[string]string{
				"app.kubernetes.io/instance": "my-argocd-app",
				"custom-label":               "custom-value",
			},
			Annotations: map[string]string{
				"argocd.argoproj.io/tracking-id": "my-argocd-app:external-secrets.io/ExternalSecret:default/test-es",
			},
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

	result, err := r.applyTemplateToManifest(context.Background(), es, map[string][]byte{"key": []byte("val")})

	require.NoError(t, err)
	labels := result.GetLabels()
	assert.Equal(t, "my-argocd-app", labels["app.kubernetes.io/instance"])
	assert.Equal(t, "custom-value", labels["custom-label"])
	assert.Equal(t, esv1.LabelManagedValue, labels[esv1.LabelManaged])

	annotations := result.GetAnnotations()
	assert.Equal(t, "my-argocd-app:external-secrets.io/ExternalSecret:default/test-es", annotations["argocd.argoproj.io/tracking-id"])
}

func TestApplyTemplateToManifest_TemplateMetadataWinsOverESLabels(t *testing.T) {
	_ = esv1.AddToScheme(scheme.Scheme)
	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	r := &Reconciler{Client: fakeClient, Scheme: scheme.Scheme}

	es := &esv1.ExternalSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-es",
			Namespace: "default",
			Labels: map[string]string{
				"app.kubernetes.io/instance": "my-argocd-app",
			},
		},
		Spec: esv1.ExternalSecretSpec{
			Target: esv1.ExternalSecretTarget{
				Name: "test-configmap",
				Manifest: &esv1.ManifestReference{
					APIVersion: "v1",
					Kind:       "ConfigMap",
				},
				Template: &esv1.ExternalSecretTemplate{
					EngineVersion: esv1.TemplateEngineV2,
					Metadata: esv1.ExternalSecretTemplateMetadata{
						Labels: map[string]string{
							"app": "explicit-template-label",
						},
					},
				},
			},
		},
	}

	result, err := r.applyTemplateToManifest(context.Background(), es, map[string][]byte{"key": []byte("val")})

	require.NoError(t, err)
	labels := result.GetLabels()
	assert.Equal(t, "explicit-template-label", labels["app"])
	assert.Empty(t, labels["app.kubernetes.io/instance"])
	assert.Equal(t, esv1.LabelManagedValue, labels[esv1.LabelManaged])
}

func TestApplyTemplateToManifest_NoESLabels(t *testing.T) {
	_ = esv1.AddToScheme(scheme.Scheme)
	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	r := &Reconciler{Client: fakeClient, Scheme: scheme.Scheme}

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

	result, err := r.applyTemplateToManifest(context.Background(), es, map[string][]byte{"key": []byte("val")})

	require.NoError(t, err)
	labels := result.GetLabels()
	assert.Equal(t, esv1.LabelManagedValue, labels[esv1.LabelManaged])
	assert.Len(t, labels, 1)
}

func TestGetGenericResource(t *testing.T) {
	// Setup
	_ = esv1.AddToScheme(scheme.Scheme)

	// Create a ConfigMap to find
	existingConfigMap := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]any{
				"name":      "test-cm",
				"namespace": "default",
			},
			"data": map[string]any{
				"key": "value",
			},
		},
	}

	_ = esv1.AddToScheme(scheme.Scheme)
	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(existingConfigMap).Build()

	r := &Reconciler{
		Client: fakeClient,
		Scheme: scheme.Scheme,
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
		Scheme: scheme.Scheme,
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
		Scheme: scheme.Scheme,
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
							Literal: new(`
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

func TestApplyTemplateToManifest_MergeBehavior(t *testing.T) {
	_ = esv1.AddToScheme(scheme.Scheme)
	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).Build()

	r := &Reconciler{
		Client: fakeClient,
		Scheme: scheme.Scheme,
	}

	es := &esv1.ExternalSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-es",
			Namespace: "default",
		},
		Spec: esv1.ExternalSecretSpec{
			Target: esv1.ExternalSecretTarget{
				Name: "test-slack-config",
				Manifest: &esv1.ManifestReference{
					APIVersion: "notification.toolkit.fluxcd.io/v1beta1",
					Kind:       "Provider",
				},
				Template: &esv1.ExternalSecretTemplate{
					EngineVersion: esv1.TemplateEngineV2,
					TemplateFrom: []esv1.TemplateFrom{
						{
							Target:  "spec.slack",
							Literal: new(`api_url: {{ .url }}`),
						},
					},
				},
			},
		},
	}

	dataMap := map[string][]byte{
		"url": []byte("https://hooks.slack.com/services/XXX"),
	}

	result, err := r.applyTemplateToManifest(context.Background(), es, dataMap)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Provider", result.GetKind())
	assert.Equal(t, "test-slack-config", result.GetName())

	specType, found, err := unstructured.NestedString(result.Object, "spec", "type")
	require.NoError(t, err)
	assert.False(t, found, "partial payload should not include unrelated spec fields")
	assert.Empty(t, specType)

	slackChannel, found, err := unstructured.NestedString(result.Object, "spec", "slack", "channel")
	require.NoError(t, err)
	assert.False(t, found, "partial payload should not include unrelated slack fields")
	assert.Empty(t, slackChannel)

	apiURL, found, err := unstructured.NestedString(result.Object, "spec", "slack", "api_url")
	require.NoError(t, err)
	require.True(t, found, "spec.slack.api_url should be present in partial payload")
	assert.Equal(t, "https://hooks.slack.com/services/XXX", apiURL)
	t.Logf("Result spec: %+v", result.Object["spec"])
}

func TestApplyTemplateToManifest_ManagedContentHash(t *testing.T) {
	_ = esv1.AddToScheme(scheme.Scheme)
	r := &Reconciler{
		Client: fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).Build(),
		Scheme: scheme.Scheme,
	}

	tests := []struct {
		name               string
		es                 *esv1.ExternalSecret
		dataMap            map[string][]byte
		wantDifferFromFull bool
	}{
		{
			name: "stamps managed hash for nested TemplateFrom path",
			es: &esv1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{Name: "test-es", Namespace: "default"},
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{
						Name: "test-provider",
						Manifest: &esv1.ManifestReference{
							APIVersion: "notification.toolkit.fluxcd.io/v1beta1",
							Kind:       "Provider",
						},
						Template: &esv1.ExternalSecretTemplate{
							EngineVersion: esv1.TemplateEngineV2,
							TemplateFrom: []esv1.TemplateFrom{
								{
									Target:  "spec.slack",
									Literal: new(`api_url: {{ .url }}`),
								},
							},
						},
					},
				},
			},
			dataMap: map[string][]byte{"url": []byte("https://example.com")},
		},
		{
			name: "stamps managed hash for Annotations shorthand, not full spec",
			es: &esv1.ExternalSecret{
				ObjectMeta: metav1.ObjectMeta{Name: "test-es", Namespace: "default"},
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{
						Name: "test-provider",
						Manifest: &esv1.ManifestReference{
							APIVersion: "notification.toolkit.fluxcd.io/v1beta1",
							Kind:       "Provider",
						},
						Template: &esv1.ExternalSecretTemplate{
							EngineVersion: esv1.TemplateEngineV2,
							TemplateFrom: []esv1.TemplateFrom{
								{
									Target:  "Annotations",
									Literal: new(`eso.injected: "{{ .value }}"`),
								},
							},
						},
					},
				},
			},
			dataMap:            map[string][]byte{"value": []byte("secret")},
			wantDifferFromFull: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := r.applyTemplateToManifest(context.Background(), tt.es, tt.dataMap)
			require.NoError(t, err)

			gotHash := result.GetAnnotations()[esv1.AnnotationDataHash]
			require.NotEmpty(t, gotHash)

			// Recompute from the stamped object without the hash annotation,
			// matching what applyTemplateToManifest hashed before writing it.
			rehashObj := result.DeepCopy()
			ann := rehashObj.GetAnnotations()
			delete(ann, esv1.AnnotationDataHash)
			rehashObj.SetAnnotations(ann)

			wantHash, err := genericTargetManagedContentHash(tt.es, rehashObj)
			require.NoError(t, err)
			assert.Equal(t, wantHash, gotHash)

			if tt.wantDifferFromFull {
				fullHash, err := genericTargetContentHash(rehashObj)
				require.NoError(t, err)
				assert.NotEqual(t, fullHash, gotHash, "AnnotationDataHash should use managed paths, not the full parent field")
			}
		})
	}
}

func TestStrategicMergePatchBody(t *testing.T) {
	tests := []struct {
		name        string
		patch       *unstructured.Unstructured
		wantKeys    []string
		wantMissing []string
		wantErr     bool
	}{
		{
			name: "includes metadata and spec, excludes apiVersion",
			patch: &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "apps/v1",
					"kind":       "Deployment",
					"metadata": map[string]any{
						"labels": map[string]any{
							esv1.LabelManaged: esv1.LabelManagedValue,
						},
						"annotations": map[string]any{
							esv1.AnnotationDataHash: "abc123",
						},
					},
					"spec": map[string]any{
						"replicas": int64(1),
					},
				},
			},
			wantKeys:    []string{"metadata", "spec"},
			wantMissing: []string{"apiVersion", "kind"},
		},
		{
			name: "includes data for ConfigMap-style payloads",
			patch: &unstructured.Unstructured{
				Object: map[string]any{
					"metadata": map[string]any{
						"labels": map[string]any{esv1.LabelManaged: esv1.LabelManagedValue},
					},
					"data": map[string]any{
						"key": "value",
					},
				},
			},
			wantKeys:    []string{"metadata", "data"},
			wantMissing: []string{"spec"},
		},
		{
			name: "skips empty metadata and spec maps",
			patch: &unstructured.Unstructured{
				Object: map[string]any{
					"metadata": map[string]any{},
					"spec":     map[string]any{},
					"data": map[string]any{
						"only": "this",
					},
				},
			},
			wantKeys:    []string{"data"},
			wantMissing: []string{"metadata", "spec"},
		},
		{
			name: "errors when patch payload is empty",
			patch: &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "ConfigMap",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := strategicMergePatchBody(tt.patch)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, body)
				return
			}

			require.NoError(t, err)
			var decoded map[string]any
			require.NoError(t, json.Unmarshal(body, &decoded))
			for _, key := range tt.wantKeys {
				assert.Contains(t, decoded, key)
			}
			for _, key := range tt.wantMissing {
				assert.NotContains(t, decoded, key)
			}
		})
	}
}

func TestNormalizeTemplateTargetPath(t *testing.T) {
	tests := []struct {
		name   string
		target string
		want   string
	}{
		{
			name:   "annotations shorthand",
			target: "annotations",
			want:   "metadata.annotations",
		},
		{
			name:   "Annotations case-insensitive",
			target: "Annotations",
			want:   "metadata.annotations",
		},
		{
			name:   "labels shorthand",
			target: "labels",
			want:   "metadata.labels",
		},
		{
			name:   "Labels case-insensitive",
			target: "Labels",
			want:   "metadata.labels",
		},
		{
			name:   "nested camelCase unchanged",
			target: "spec.controllerConfig.annotations",
			want:   "spec.controllerConfig.annotations",
		},
		{
			name:   "data unchanged",
			target: "data",
			want:   "data",
		},
		{
			name:   "empty unchanged",
			target: "",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizeTemplateTargetPath(tt.target))
		})
	}
}

func TestTemplateTargetPaths(t *testing.T) {
	tests := []struct {
		name string
		es   *esv1.ExternalSecret
		want []string
	}{
		{
			name: "nil template",
			es: &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{},
				},
			},
			want: nil,
		},
		{
			name: "TemplateFrom target path",
			es: &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{
						Template: &esv1.ExternalSecretTemplate{
							TemplateFrom: []esv1.TemplateFrom{
								{Target: "spec.slack"},
							},
						},
					},
				},
			},
			want: []string{"spec.slack"},
		},
		{
			name: "empty TemplateFrom target defaults to Data",
			es: &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{
						Template: &esv1.ExternalSecretTemplate{
							TemplateFrom: []esv1.TemplateFrom{
								{Literal: new("key: value")},
							},
						},
					},
				},
			},
			want: []string{string(esv1.TemplateTargetData)},
		},
		{
			name: "Annotations shorthand is normalized",
			es: &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{
						Template: &esv1.ExternalSecretTemplate{
							TemplateFrom: []esv1.TemplateFrom{
								{Target: "Annotations"},
							},
						},
					},
				},
			},
			want: []string{"metadata.annotations"},
		},
		{
			name: "Template.Data is included",
			es: &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{
						Template: &esv1.ExternalSecretTemplate{
							TemplateFrom: []esv1.TemplateFrom{
								{Target: "spec.slack"},
							},
							Data: map[string]string{
								"extra": "value",
							},
						},
					},
				},
			},
			want: []string{"spec.slack", string(esv1.TemplateTargetData)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, templateTargetPaths(tt.es))
		})
	}
}

func TestExtractTemplateTargetContent(t *testing.T) {
	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"metadata": map[string]any{
				"annotations": map[string]any{
					"eso": "injected",
				},
			},
			"spec": map[string]any{
				"type": "slack",
				"slack": map[string]any{
					"api_url": "https://example.com",
				},
			},
		},
	}

	tests := []struct {
		name  string
		paths []string
		want  map[string]any
	}{
		{
			name:  "extracts nested managed path",
			paths: []string{"spec.slack"},
			want: map[string]any{
				"spec": map[string]any{
					"slack": map[string]any{
						"api_url": "https://example.com",
					},
				},
			},
		},
		{
			name:  "skips missing paths",
			paths: []string{"spec.missing", "spec.slack"},
			want: map[string]any{
				"spec": map[string]any{
					"slack": map[string]any{
						"api_url": "https://example.com",
					},
				},
			},
		},
		{
			name:  "extracts metadata annotations",
			paths: []string{"metadata.annotations"},
			want: map[string]any{
				"metadata": map[string]any{
					"annotations": map[string]any{
						"eso": "injected",
					},
				},
			},
		},
		{
			name:  "all missing yields empty map",
			paths: []string{"spec.does.not.exist"},
			want:  map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractTemplateTargetContent(obj, tt.paths)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGenericTargetManagedContentHash(t *testing.T) {
	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"metadata": map[string]any{
				"annotations": map[string]any{
					"eso": "injected",
				},
			},
			"spec": map[string]any{
				"type": "slack",
				"slack": map[string]any{
					"channel": "general",
					"api_url": "https://example.com",
				},
			},
		},
	}

	tests := []struct {
		name string
		es   *esv1.ExternalSecret
		obj  *unstructured.Unstructured
		want func(t *testing.T, hash string, obj *unstructured.Unstructured)
	}{
		{
			name: "hashes only managed template path",
			es: &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{
						Template: &esv1.ExternalSecretTemplate{
							TemplateFrom: []esv1.TemplateFrom{
								{Target: "spec.slack"},
							},
						},
					},
				},
			},
			obj: obj,
			want: func(t *testing.T, hash string, obj *unstructured.Unstructured) {
				expected, err := extractTemplateTargetContent(obj, []string{"spec.slack"})
				require.NoError(t, err)
				assert.Equal(t, esutils.ObjectHash(expected), hash)
			},
		},
		{
			name: "unmanaged sibling fields do not change the hash",
			es: &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{
						Template: &esv1.ExternalSecretTemplate{
							TemplateFrom: []esv1.TemplateFrom{
								{Target: "spec.slack"},
							},
						},
					},
				},
			},
			obj: obj,
			want: func(t *testing.T, hash string, obj *unstructured.Unstructured) {
				changed := obj.DeepCopy()
				require.NoError(t, unstructured.SetNestedField(changed.Object, "other", "spec", "type"))
				changedHash, err := genericTargetManagedContentHash(&esv1.ExternalSecret{
					Spec: esv1.ExternalSecretSpec{
						Target: esv1.ExternalSecretTarget{
							Template: &esv1.ExternalSecretTemplate{
								TemplateFrom: []esv1.TemplateFrom{
									{Target: "spec.slack"},
								},
							},
						},
					},
				}, changed)
				require.NoError(t, err)
				assert.Equal(t, hash, changedHash)
			},
		},
		{
			name: "falls back to full content hash when template is nil",
			es: &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{},
				},
			},
			obj: obj,
			want: func(t *testing.T, hash string, obj *unstructured.Unstructured) {
				expected, err := genericTargetContentHash(obj)
				require.NoError(t, err)
				assert.Equal(t, expected, hash)
			},
		},
		{
			name: "falls back to full content hash when managed paths are missing",
			es: &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{
						Template: &esv1.ExternalSecretTemplate{
							TemplateFrom: []esv1.TemplateFrom{
								{Target: "spec.does.not.exist"},
							},
						},
					},
				},
			},
			obj: obj,
			want: func(t *testing.T, hash string, obj *unstructured.Unstructured) {
				expected, err := genericTargetContentHash(obj)
				require.NoError(t, err)
				assert.Equal(t, expected, hash)
			},
		},
		{
			name: "hashes normalized Annotations shorthand",
			es: &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{
						Template: &esv1.ExternalSecretTemplate{
							TemplateFrom: []esv1.TemplateFrom{
								{Target: "Annotations"},
							},
						},
					},
				},
			},
			obj: obj,
			want: func(t *testing.T, hash string, obj *unstructured.Unstructured) {
				expected, err := extractTemplateTargetContent(obj, []string{"metadata.annotations"})
				require.NoError(t, err)
				assert.Equal(t, esutils.ObjectHash(expected), hash)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := genericTargetManagedContentHash(tt.es, tt.obj)
			require.NoError(t, err)
			require.NotEmpty(t, hash)
			tt.want(t, hash, tt.obj)
		})
	}
}

func TestGenericTargetContentHash(t *testing.T) {
	tests := []struct {
		name    string
		obj     *unstructured.Unstructured
		wantErr bool
	}{
		{
			name: "hashes spec field",
			obj: &unstructured.Unstructured{
				Object: map[string]any{
					"spec": map[string]any{"key": "val"},
				},
			},
		},
		{
			name: "hashes data field when no spec",
			obj: &unstructured.Unstructured{
				Object: map[string]any{
					"data": map[string]any{"key": "val"},
				},
			},
		},
		{
			name: "prefers spec over data",
			obj: &unstructured.Unstructured{
				Object: map[string]any{
					"spec": map[string]any{"a": "1"},
					"data": map[string]any{"b": "2"},
				},
			},
		},
		{
			name: "errors when neither spec nor data",
			obj: &unstructured.Unstructured{
				Object: map[string]any{
					"status": map[string]any{"ready": true},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := genericTargetContentHash(tt.obj)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, hash)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, hash)
		})
	}

	t.Run("spec preferred over data produces spec hash", func(t *testing.T) {
		specData := map[string]any{"a": "1"}
		obj := &unstructured.Unstructured{
			Object: map[string]any{
				"spec": specData,
				"data": map[string]any{"b": "2"},
			},
		}
		hash, err := genericTargetContentHash(obj)
		require.NoError(t, err)
		assert.Equal(t, esutils.ObjectHash(specData), hash)
	})
}

func TestIsGenericTargetValid(t *testing.T) {
	makeES := func(policy esv1.ExternalSecretCreationPolicy) *esv1.ExternalSecret {
		return &esv1.ExternalSecret{
			Spec: esv1.ExternalSecretSpec{
				Target: esv1.ExternalSecretTarget{
					CreationPolicy: policy,
				},
			},
		}
	}

	makeTarget := func(uid string, labels map[string]string, annotations map[string]string, obj map[string]any) *unstructured.Unstructured {
		u := &unstructured.Unstructured{Object: obj}
		if uid != "" {
			u.SetUID(types.UID(uid))
		}
		u.SetLabels(labels)
		u.SetAnnotations(annotations)
		return u
	}

	t.Run("orphan policy always valid", func(t *testing.T) {
		valid, err := isGenericTargetValid(nil, makeES(esv1.CreatePolicyOrphan))
		require.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("nil target is invalid", func(t *testing.T) {
		valid, err := isGenericTargetValid(nil, makeES(esv1.CreatePolicyOwner))
		require.NoError(t, err)
		assert.False(t, valid)
	})

	t.Run("empty UID is invalid", func(t *testing.T) {
		obj := &unstructured.Unstructured{Object: map[string]any{}}
		valid, err := isGenericTargetValid(obj, makeES(esv1.CreatePolicyOwner))
		require.NoError(t, err)
		assert.False(t, valid)
	})

	t.Run("not managed is invalid", func(t *testing.T) {
		obj := makeTarget("some-uid", map[string]string{}, nil, map[string]any{
			"spec": map[string]any{"key": "val"},
		})
		valid, err := isGenericTargetValid(obj, makeES(esv1.CreatePolicyOwner))
		require.NoError(t, err)
		assert.False(t, valid)
	})

	t.Run("hash mismatch is invalid", func(t *testing.T) {
		obj := makeTarget(
			"some-uid",
			map[string]string{esv1.LabelManaged: esv1.LabelManagedValue},
			map[string]string{esv1.AnnotationDataHash: "wrong-hash"},
			map[string]any{"spec": map[string]any{"key": "val"}},
		)
		valid, err := isGenericTargetValid(obj, makeES(esv1.CreatePolicyOwner))
		require.NoError(t, err)
		assert.False(t, valid)
	})

	t.Run("matching hash is valid", func(t *testing.T) {
		specData := map[string]any{"key": "val"}
		hash := esutils.ObjectHash(specData)
		obj := makeTarget(
			"some-uid",
			map[string]string{esv1.LabelManaged: esv1.LabelManagedValue},
			map[string]string{esv1.AnnotationDataHash: hash},
			map[string]any{"spec": specData},
		)
		valid, err := isGenericTargetValid(obj, makeES(esv1.CreatePolicyOwner))
		require.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("errors when target has no spec or data", func(t *testing.T) {
		obj := makeTarget(
			"some-uid",
			map[string]string{esv1.LabelManaged: esv1.LabelManagedValue},
			nil,
			map[string]any{"status": map[string]any{}},
		)
		_, err := isGenericTargetValid(obj, makeES(esv1.CreatePolicyOwner))
		assert.Error(t, err)
	})
}
