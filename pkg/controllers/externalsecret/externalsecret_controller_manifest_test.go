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
	"testing"
	"time"

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
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	runtimecache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/source"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/pkg/controllers/externalsecret/esmetrics"
	ctrlmetrics "github.com/external-secrets/external-secrets/pkg/controllers/metrics"
	"github.com/external-secrets/external-secrets/runtime/esutils"
)

func TestApplyTemplateToManifestKeepsTargetIdentity(t *testing.T) {
	_ = esv1.AddToScheme(scheme.Scheme)
	r := &Reconciler{Client: fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).Build(), Scheme: scheme.Scheme}

	tests := []struct {
		name, target, literal string
		existing              bool
		wantErr               string
	}{
		{"namespace via literal", "metadata.namespace", "victim", false, "rendered target changed protected metadata"},
		{"namespace on existing object", "metadata.namespace", "victim", true, "rendered target changed protected metadata"},
		{"namespace via metadata map", "metadata", "namespace: victim", true, "rendered target changed protected metadata"},
		{"name via literal", "metadata.name", "pwned-by-attacker", false, "rendered target changed protected metadata"},
		{"malformed metadata", "metadata", "null", false, "rendered target changed protected metadata"},
		{"same identity via metadata map", "metadata", "namespace: attacker\nname: harmless-local-name", false, ""},
		{"metadata map omits identity", "metadata", "labels:\n  app: test", false, ""},
		{"metadata labels remain templatable", "metadata.labels", "app: test", false, ""},
		{"kind via literal", "kind", "Secret", false, `rendered target GVK "/v1, Kind=Secret" differs from declared target GVK "/v1, Kind=ConfigMap"`},
		{"API version via literal", "apiVersion", "other.io/v1", false, `rendered target GVK "other.io/v1, Kind=ConfigMap" differs from declared target GVK "/v1, Kind=ConfigMap"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			es := &esv1.ExternalSecret{
				Name: "evil-es", Namespace: "attacker",
				Spec: esv1.ExternalSecretSpec{Target: esv1.ExternalSecretTarget{
					Name: "harmless-local-name", CreationPolicy: esv1.CreatePolicyOrphan,
					Manifest: &esv1.ManifestReference{APIVersion: "v1", Kind: "ConfigMap"},
					Template: &esv1.ExternalSecretTemplate{EngineVersion: esv1.TemplateEngineV2,
						TemplateFrom: []esv1.TemplateFrom{{Target: tt.target, Literal: &tt.literal}}},
				}},
			}
			var existing *unstructured.Unstructured
			if tt.existing {
				existing = &unstructured.Unstructured{Object: map[string]any{
					"apiVersion": "v1", "kind": "ConfigMap",
					"metadata": map[string]any{"name": "harmless-local-name", "namespace": "attacker"},
					"data":     map[string]any{"key": "old"},
				}}
			}

			result, err := r.applyTemplateToManifest(context.Background(), es, map[string][]byte{"key": []byte("value")}, existing)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				assert.Nil(t, result)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "attacker", result.GetNamespace())
			assert.Equal(t, "harmless-local-name", result.GetName())
		})
	}
}

func TestApplyTemplateToManifestProtectsMetadata(t *testing.T) {
	_ = esv1.AddToScheme(scheme.Scheme)
	r := &Reconciler{Client: fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).Build(), Scheme: scheme.Scheme}

	tests := []struct {
		name, target, literal string
		existing              bool
		wantErr               bool
	}{
		{"owner references", "metadata.ownerReferences", "- apiVersion: v1\n  kind: ConfigMap\n  name: victim\n  uid: victim-uid", false, true},
		{"finalizers", "metadata.finalizers", "- attacker.io/hold", false, true},
		{"uid on create", "metadata.uid", "victim-uid", false, true},
		{"creation timestamp on create", "metadata.creationTimestamp", "2025-01-01T00:00:00Z", false, true},
		{"uid", "metadata.uid", "victim-uid", true, true},
		{"resource version", "metadata.resourceVersion", "1234", true, true},
		{"replace retained metadata", "metadata", "name: target\nnamespace: ns", true, false},
		{"replace retained metadata with labels", "metadata", "labels:\n  app: test", true, false},
		{"change retained uid via metadata map", "metadata", "name: target\nnamespace: ns\nuid: victim-uid", true, true},
		{"ordinary annotations", "metadata.annotations", "app.example.com/key: value", true, false},
		{"replaced labels map", "metadata.labels", "null", true, false},
		{"replaced annotations map", "metadata.annotations", "null", true, false},
		{"controller labels and annotation", "metadata", `name: target
namespace: ns
labels:
  reconcile.external-secrets.io/managed: 'false'
  reconcile.external-secrets.io/created-by: spoofed
  app: test
annotations:
  reconcile.external-secrets.io/data-hash: spoofed
  app.example.com/key: value`, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			es := &esv1.ExternalSecret{
				Name: "es", Namespace: "ns",
				Spec: esv1.ExternalSecretSpec{Target: esv1.ExternalSecretTarget{
					Name: "target", CreationPolicy: esv1.CreatePolicyOrphan,
					Manifest: &esv1.ManifestReference{APIVersion: "v1", Kind: "ConfigMap"},
					Template: &esv1.ExternalSecretTemplate{EngineVersion: esv1.TemplateEngineV2,
						TemplateFrom: []esv1.TemplateFrom{{Target: tt.target, Literal: &tt.literal}}},
				}},
			}
			var existing *unstructured.Unstructured
			if tt.existing {
				existing = &unstructured.Unstructured{Object: map[string]any{
					"apiVersion": "v1", "kind": "ConfigMap",
					"metadata": map[string]any{
						"name": "target", "namespace": "ns", "uid": "original-uid", "resourceVersion": "1",
						"creationTimestamp": "2025-01-01T00:00:00Z",
						"managedFields":     []any{map[string]any{"manager": "kube-apiserver", "operation": "Update", "apiVersion": "v1"}},
						"ownerReferences":   []any{map[string]any{"apiVersion": "v1", "kind": "ConfigMap", "name": "unrelated", "uid": "unrelated-uid"}},
						"labels":            map[string]any{"other-controller": "keep"},
						"annotations":       map[string]any{"other-controller": "keep", "app.example.com/key": "old"},
					},
					"data": map[string]any{"key": "old"},
				}}
			}
			result, err := r.applyTemplateToManifest(context.Background(), es, map[string][]byte{"key": []byte("value")}, existing)
			if tt.wantErr {
				require.EqualError(t, err, "rendered target changed protected metadata")
				assert.Nil(t, result)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, esv1.LabelManagedValue, result.GetLabels()[esv1.LabelManaged])
			assert.Empty(t, result.GetLabels()[esv1.LabelOwner])
			hash, err := genericTargetContentHash(result)
			require.NoError(t, err)
			assert.Equal(t, hash, result.GetAnnotations()[esv1.AnnotationDataHash])
			if tt.existing {
				assert.Equal(t, "keep", result.GetLabels()["other-controller"])
				assert.Equal(t, "keep", result.GetAnnotations()["other-controller"])
				for _, key := range []string{"name", "namespace", "uid", "resourceVersion", "creationTimestamp", "managedFields", "ownerReferences"} {
					assert.Equal(t, existing.Object["metadata"].(map[string]any)[key], result.Object["metadata"].(map[string]any)[key], key)
				}
			}
			if tt.name == "ordinary annotations" || tt.name == "controller labels and annotation" {
				assert.Equal(t, "value", result.GetAnnotations()["app.example.com/key"])
			}
		})
	}
}

func TestApplyTemplateToManifestExistingMetadataAcrossPolicies(t *testing.T) {
	_ = esv1.AddToScheme(scheme.Scheme)
	r := &Reconciler{Client: fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).Build(), Scheme: scheme.Scheme}
	literal := "labels:\n  app: test\nannotations:\n  app.example.com/key: value"
	for _, policy := range []esv1.ExternalSecretCreationPolicy{
		esv1.CreatePolicyMerge, esv1.CreatePolicyCreateOrMerge, esv1.CreatePolicyOrphan, esv1.CreatePolicyOwner,
	} {
		t.Run(string(policy), func(t *testing.T) {
			es := &esv1.ExternalSecret{
				Name: "es", Namespace: "ns", UID: "es-uid",
				Spec: esv1.ExternalSecretSpec{Target: esv1.ExternalSecretTarget{
					Name: "target", CreationPolicy: policy,
					Manifest: &esv1.ManifestReference{APIVersion: "example.io/v1", Kind: "Widget"},
					Template: &esv1.ExternalSecretTemplate{EngineVersion: esv1.TemplateEngineV2,
						TemplateFrom: []esv1.TemplateFrom{
							{Target: "metadata", Literal: &literal},
							{Target: "spec.new", Literal: new("key: new")},
						}},
				}},
			}
			existing := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "example.io/v1", "kind": "Widget",
				"metadata": map[string]any{
					"name": "target", "namespace": "ns", "uid": "target-uid", "resourceVersion": "12",
					"creationTimestamp": "2025-01-01T00:00:00Z",
					"managedFields":     []any{map[string]any{"manager": "kube-apiserver", "operation": "Update", "apiVersion": "example.io/v1"}},
					"finalizers":        []any{"unrelated.example.io/cleanup"},
					"ownerReferences":   []any{map[string]any{"apiVersion": "v1", "kind": "ConfigMap", "name": "unrelated", "uid": "unrelated-uid"}},
					"labels":            map[string]any{"other-controller": "keep"},
					"annotations":       map[string]any{"other-controller": "keep"},
				},
				"spec": map[string]any{"old": "keep"},
			}}
			result, err := r.applyTemplateToManifest(context.Background(), es, nil, existing)
			require.NoError(t, err)
			for _, key := range []string{"name", "namespace", "uid", "resourceVersion", "creationTimestamp", "managedFields", "finalizers"} {
				assert.Equal(t, existing.Object["metadata"].(map[string]any)[key], result.Object["metadata"].(map[string]any)[key], key)
			}
			assert.Equal(t, "unrelated", result.GetOwnerReferences()[0].Name)
			assert.Equal(t, "keep", result.GetLabels()["other-controller"])
			assert.Equal(t, "test", result.GetLabels()["app"])
			assert.Equal(t, "keep", result.GetAnnotations()["other-controller"])
			assert.Equal(t, "value", result.GetAnnotations()["app.example.com/key"])
			assert.Equal(t, "new", result.Object["spec"].(map[string]any)["new"].(map[string]any)["key"])
			if policy == esv1.CreatePolicyMerge || policy == esv1.CreatePolicyCreateOrMerge {
				assert.Equal(t, "keep", result.Object["spec"].(map[string]any)["old"])
			} else {
				assert.NotContains(t, result.Object["spec"].(map[string]any), "old")
			}
			if policy == esv1.CreatePolicyOwner {
				require.Len(t, result.GetOwnerReferences(), 2)
				assert.Equal(t, "es", result.GetOwnerReferences()[1].Name)
			} else {
				require.Len(t, result.GetOwnerReferences(), 1)
			}
			// Rendering must not mutate the object fetched from the API.
			assert.Equal(t, "keep", existing.Object["spec"].(map[string]any)["old"])
		})
	}
}

// TestApplyTemplateToManifestOwnershipOfExistingTarget checks ownership against the live
// target's owner references. Retaining existing metadata is what makes these references
// visible to applyOwnership in the first place.
func TestApplyTemplateToManifestOwnershipOfExistingTarget(t *testing.T) {
	_ = esv1.AddToScheme(scheme.Scheme)

	esControllerRef := func(name string) metav1.OwnerReference {
		return metav1.OwnerReference{
			APIVersion: esv1.SchemeGroupVersion.String(), Kind: esv1.ExtSecretKind,
			Name: name, UID: types.UID(name + "-uid"), Controller: new(true),
		}
	}
	unrelatedRef := metav1.OwnerReference{
		APIVersion: "v1", Kind: "ConfigMap", Name: "unrelated", UID: "unrelated-uid",
	}

	tests := []struct {
		name     string
		policy   esv1.ExternalSecretCreationPolicy
		owners   []metav1.OwnerReference
		wantErr  error
		wantRefs []string
	}{
		{
			name:    "Owner refuses to replace another ExternalSecret controller",
			policy:  esv1.CreatePolicyOwner,
			owners:  []metav1.OwnerReference{esControllerRef("other-es")},
			wantErr: ErrSecretIsOwned,
		},
		{
			name:    "Orphan refuses to write a target owned by another ExternalSecret",
			policy:  esv1.CreatePolicyOrphan,
			owners:  []metav1.OwnerReference{esControllerRef("other-es")},
			wantErr: ErrSecretIsOwned,
		},
		{
			name:    "Merge refuses to write a target owned by another ExternalSecret",
			policy:  esv1.CreatePolicyMerge,
			owners:  []metav1.OwnerReference{esControllerRef("other-es")},
			wantErr: ErrSecretIsOwned,
		},
		{
			name:    "Owner refuses to replace a controller of another kind",
			policy:  esv1.CreatePolicyOwner,
			owners:  []metav1.OwnerReference{{APIVersion: "apps/v1", Kind: "Deployment", Name: "owner-dep", UID: "dep-uid", Controller: new(true)}},
			wantErr: ErrSecretSetCtrlRef,
		},
		{
			name:     "Owner adopts a target that only has non-controller references",
			policy:   esv1.CreatePolicyOwner,
			owners:   []metav1.OwnerReference{unrelatedRef},
			wantRefs: []string{"unrelated", "es"},
		},
		{
			name:     "Orphan removes only its own controller reference",
			policy:   esv1.CreatePolicyOrphan,
			owners:   []metav1.OwnerReference{unrelatedRef, esControllerRef("es")},
			wantRefs: []string{"unrelated"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{Client: fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).Build(), Scheme: scheme.Scheme}
			es := &esv1.ExternalSecret{
				Name: "es", Namespace: "ns", UID: "es-uid",
				Spec: esv1.ExternalSecretSpec{Target: esv1.ExternalSecretTarget{
					Name: "target", CreationPolicy: tt.policy,
					Manifest: &esv1.ManifestReference{APIVersion: "v1", Kind: "ConfigMap"},
				}},
			}
			existing := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "v1", "kind": "ConfigMap",
				"metadata": map[string]any{"name": "target", "namespace": "ns", "uid": "target-uid"},
			}}
			existing.SetOwnerReferences(tt.owners)

			result, err := r.applyTemplateToManifest(context.Background(), es, map[string][]byte{"key": []byte("value")}, existing)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)

			names := make([]string, 0, len(result.GetOwnerReferences()))
			for _, ref := range result.GetOwnerReferences() {
				names = append(names, ref.Name)
			}
			assert.Equal(t, tt.wantRefs, names)
		})
	}
}

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
		{
			name: "core Secret target - flag enabled - rejected",
			es: &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{
						Manifest: &esv1.ManifestReference{
							APIVersion: "v1",
							Kind:       "Secret",
						},
					},
				},
			},
			allowGenericTargets: true,
			expectedError:       true,
			errorContains:       "must not target the core v1 Secret kind",
		},
		{
			name: "core Secret target through an explicit empty group - rejected",
			es: &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{
						Manifest: &esv1.ManifestReference{
							APIVersion: "/v1",
							Kind:       "Secret",
						},
					},
				},
			},
			allowGenericTargets: true,
			expectedError:       true,
			errorContains:       "must not target the core v1 Secret kind",
		},
		{
			name: "Secret kind in a custom group - flag enabled - valid",
			es: &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{
						Manifest: &esv1.ManifestReference{
							APIVersion: "example.io/v1",
							Kind:       "Secret",
						},
					},
				},
			},
			allowGenericTargets: true,
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
				Name: "my-external-secret",
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
				Name: "my-external-secret",
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
		Name:      "test-es",
		Namespace: "default",
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
	result, err := r.applyTemplateToManifest(context.Background(), es, dataMap, nil)

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
		Name:      "test-es",
		Namespace: "default",
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
	result, err := r.applyTemplateToManifest(context.Background(), es, dataMap, nil)

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
		Name:      "test-es",
		Namespace: "default",
		UID:       "abc-123",
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

	result, err := r.applyTemplateToManifest(context.Background(), es, map[string][]byte{"key": []byte("val")}, nil)

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
				Name:      "test-es",
				Namespace: "default",
				UID:       "abc-123",
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
		Name:      "test-es",
		Namespace: "default",
		UID:       "abc-123",
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

	result, err := r.applyTemplateToManifest(context.Background(), es, map[string][]byte{"key": []byte("val")}, nil)

	require.NoError(t, err)
	assert.Empty(t, result.GetOwnerReferences())
}

func TestApplyTemplateToManifest_PropagatesESLabelsAndAnnotations(t *testing.T) {
	_ = esv1.AddToScheme(scheme.Scheme)
	fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	r := &Reconciler{Client: fakeClient, Scheme: scheme.Scheme}

	es := &esv1.ExternalSecret{
		Name:      "test-es",
		Namespace: "default",
		Labels: map[string]string{
			"app.kubernetes.io/instance": "my-argocd-app",
			"custom-label":               "custom-value",
		},
		Annotations: map[string]string{
			"argocd.argoproj.io/tracking-id": "my-argocd-app:external-secrets.io/ExternalSecret:default/test-es",
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

	result, err := r.applyTemplateToManifest(context.Background(), es, map[string][]byte{"key": []byte("val")}, nil)

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
		Name:      "test-es",
		Namespace: "default",
		Labels: map[string]string{
			"app.kubernetes.io/instance": "my-argocd-app",
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

	result, err := r.applyTemplateToManifest(context.Background(), es, map[string][]byte{"key": []byte("val")}, nil)

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
		Name:      "test-es",
		Namespace: "default",
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

	result, err := r.applyTemplateToManifest(context.Background(), es, map[string][]byte{"key": []byte("val")}, nil)

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
		Name:      "test-es",
		Namespace: "default",
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
		Name:      "test-es",
		Namespace: "default",
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
		Name:      "test-es",
		Namespace: "default",
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

	result, err := r.applyTemplateToManifest(context.Background(), es, dataMap, nil)

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
		Name:      "test-es",
		Namespace: "default",
		Spec: esv1.ExternalSecretSpec{
			Target: esv1.ExternalSecretTarget{
				Name:           "test-slack-config",
				CreationPolicy: esv1.CreatePolicyMerge,
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

	existingResource := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "notification.toolkit.fluxcd.io/v1beta1",
			"kind":       "Provider",
			"metadata": map[string]any{
				"name":            "test-slack-config",
				"namespace":       "default",
				"resourceVersion": "12345",
				"uid":             "test-uid-123",
			},
			"spec": map[string]any{
				"type": "slack",
				"slack": map[string]any{
					"channel":  "general",
					"username": "bot",
				},
			},
		},
	}

	dataMap := map[string][]byte{
		"url": []byte("https://hooks.slack.com/services/XXX"),
	}

	result, err := r.applyTemplateToManifest(context.Background(), es, dataMap, existingResource)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Provider", result.GetKind())
	assert.Equal(t, "test-slack-config", result.GetName())

	specType, found, err := unstructured.NestedString(result.Object, "spec", "type")
	require.NoError(t, err)
	require.True(t, found, "spec.type should be preserved")
	assert.Equal(t, "slack", specType, "spec.type should be preserved from existing resource")

	slackChannel, found, err := unstructured.NestedString(result.Object, "spec", "slack", "channel")
	require.NoError(t, err)
	require.True(t, found, "spec.slack.channel should be preserved")
	assert.Equal(t, "general", slackChannel, "spec.slack.channel should be preserved from existing resource")

	slackUsername, found, err := unstructured.NestedString(result.Object, "spec", "slack", "username")
	require.NoError(t, err)
	require.True(t, found, "spec.slack.username should be preserved")
	assert.Equal(t, "bot", slackUsername, "spec.slack.username should be preserved from existing resource")

	apiURL, found, err := unstructured.NestedString(result.Object, "spec", "slack", "api_url")
	require.NoError(t, err)
	require.True(t, found, "spec.slack.api_url should be added from template")
	assert.Equal(t, "https://hooks.slack.com/services/XXX", apiURL, "spec.slack.api_url should come from template")
	assert.Equal(t, "12345", result.GetResourceVersion(), "resourceVersion should be preserved")
	assert.Equal(t, "test-uid-123", string(result.GetUID()), "uid should be preserved")
	t.Logf("Result spec: %+v", result.Object["spec"])
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

// noopInformerManager satisfies InformerManager for reconcile-level tests, which do not
// exercise drift detection.
type noopInformerManager struct{}

func (noopInformerManager) EnsureInformer(context.Context, schema.GroupVersionKind, types.NamespacedName) (bool, error) {
	return false, nil
}
func (noopInformerManager) ReleaseInformer(context.Context, schema.GroupVersionKind, types.NamespacedName) error {
	return nil
}
func (noopInformerManager) IsManaged(schema.GroupVersionKind) bool { return false }
func (noopInformerManager) GetInformer(schema.GroupVersionKind) (runtimecache.Informer, bool) {
	return nil, false
}
func (noopInformerManager) Source() source.TypedSource[ctrl.Request] { return nil }
func (noopInformerManager) SetQueue(workqueue.TypedRateLimitingInterface[ctrl.Request]) error {
	return nil
}

// TestReconcileGenericTargetUsesLiveTarget drives reconcileGenericTarget against a fake
// API server. The renderer tests above can only prove that applyTemplateToManifest honors
// the metadata it is handed; this proves the caller actually hands it the live object.
func TestReconcileGenericTargetUsesLiveTarget(t *testing.T) {
	_ = esv1.AddToScheme(scheme.Scheme)

	const (
		namespace  = "ns"
		targetName = "shared-configmap"
		storeName  = "test-store"
	)

	newES := func(name string) *esv1.ExternalSecret {
		return &esv1.ExternalSecret{
			Name: name, Namespace: namespace, UID: types.UID(name + "-uid"),
			Spec: esv1.ExternalSecretSpec{
				RefreshInterval: &metav1.Duration{Duration: time.Hour},
				SecretStoreRef:  esv1.SecretStoreRef{Name: storeName, Kind: esv1.SecretStoreKind},
				Data: []esv1.ExternalSecretData{{
					SecretKey: "key",
					RemoteRef: esv1.ExternalSecretDataRemoteRef{Key: "remote-key"},
				}},
				Target: esv1.ExternalSecretTarget{
					Name:           targetName,
					CreationPolicy: esv1.CreatePolicyOwner,
					Manifest:       &esv1.ManifestReference{APIVersion: "v1", Kind: "ConfigMap"},
				},
			},
		}
	}

	store := &esv1.SecretStore{
		Name: storeName, Namespace: namespace,
		Spec: esv1.SecretStoreSpec{Provider: &esv1.SecretStoreProvider{
			AWS: &esv1.AWSProvider{Service: esv1.AWSServiceSecretsManager},
		}},
	}

	fakeProvider.Reset()
	defer fakeProvider.Reset()
	fakeProvider.WithGetSecret([]byte("value"), nil)

	r := &Reconciler{
		Client:              fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(store).Build(),
		Scheme:              scheme.Scheme,
		Log:                 logr.Discard(),
		recorder:            record.NewFakeRecorder(100),
		AllowGenericTargets: true,
		informerManager:     noopInformerManager{},
	}

	reconcile := func(t *testing.T, es *esv1.ExternalSecret) {
		t.Helper()
		resourceLabels := ctrlmetrics.RefineNonConditionMetricLabels(
			map[string]string{"name": es.Name, "namespace": es.Namespace})
		_, err := r.reconcileGenericTarget(context.Background(), es, logr.Discard(), time.Now(),
			resourceLabels, esmetrics.GetCounterVec(esmetrics.SyncCallsErrorKey))
		require.NoError(t, err)
	}
	liveTarget := func(t *testing.T) *unstructured.Unstructured {
		t.Helper()
		target := &unstructured.Unstructured{}
		target.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("ConfigMap"))
		require.NoError(t, r.Get(context.Background(),
			types.NamespacedName{Name: targetName, Namespace: namespace}, target))
		return target
	}

	esA := newES("es-a")
	reconcile(t, esA)

	target := liveTarget(t)
	require.Len(t, target.GetOwnerReferences(), 1)
	assert.Equal(t, "es-a", target.GetOwnerReferences()[0].Name)
	assert.Equal(t, "value", target.Object["data"].(map[string]any)["key"])

	// another controller adds metadata to the live target out of band
	target.SetFinalizers([]string{"unrelated.example.io/cleanup"})
	labels := target.GetLabels()
	labels["other-controller"] = "keep"
	target.SetLabels(labels)
	require.NoError(t, r.Update(context.Background(), target))

	// the next sync of the owning ExternalSecret must not drop that metadata
	fakeProvider.WithGetSecret([]byte("rotated"), nil)
	reconcile(t, newES("es-a"))

	target = liveTarget(t)
	assert.Equal(t, []string{"unrelated.example.io/cleanup"}, target.GetFinalizers())
	assert.Equal(t, "keep", target.GetLabels()["other-controller"])
	assert.Equal(t, "rotated", target.Object["data"].(map[string]any)["key"])
	require.Len(t, target.GetOwnerReferences(), 1)
	assert.Equal(t, "es-a", target.GetOwnerReferences()[0].Name)

	// a second ExternalSecret must not take over a target owned by the first
	esB := newES("es-b")
	reconcile(t, esB)

	require.NotEmpty(t, esB.Status.Conditions)
	assert.Equal(t, esv1.ConditionReasonSecretOwnedByOther, esB.Status.Conditions[0].Reason)
	target = liveTarget(t)
	require.Len(t, target.GetOwnerReferences(), 1)
	assert.Equal(t, "es-a", target.GetOwnerReferences()[0].Name)
	assert.Equal(t, "rotated", target.Object["data"].(map[string]any)["key"])
}

// TestApplyTemplateToManifestAcceptsRestatedMetadata pins the promise made in
// docs/guides/ownership-deletion-policy.md: a template that reproduces a protected
// metadata value unchanged is accepted, and only a changed value is rejected. Numbers
// are the interesting case, because the live object and the template reach the
// comparison through different decoders.
func TestApplyTemplateToManifestAcceptsRestatedMetadata(t *testing.T) {
	_ = esv1.AddToScheme(scheme.Scheme)
	r := &Reconciler{Client: fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).Build(), Scheme: scheme.Scheme}

	for _, tt := range []struct {
		name, literal string
		wantErr       bool
	}{
		{"number restated unchanged", "generation: 2", false},
		{"number changed", "generation: 3", true},
		{"string restated unchanged", "uid: target-uid", false},
		{"string changed", "uid: victim-uid", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			es := &esv1.ExternalSecret{
				Name: "es", Namespace: "ns", UID: "es-uid",
				Spec: esv1.ExternalSecretSpec{Target: esv1.ExternalSecretTarget{
					Name: "target", CreationPolicy: esv1.CreatePolicyOrphan,
					Manifest: &esv1.ManifestReference{APIVersion: "example.io/v1", Kind: "Widget"},
					Template: &esv1.ExternalSecretTemplate{EngineVersion: esv1.TemplateEngineV2,
						TemplateFrom: []esv1.TemplateFrom{{Target: "metadata", Literal: &tt.literal}}},
				}},
			}
			existing := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "example.io/v1", "kind": "Widget",
				"metadata": map[string]any{
					"name": "target", "namespace": "ns", "uid": "target-uid",
					"resourceVersion": "12", "generation": int64(2),
				},
			}}

			result, err := r.applyTemplateToManifest(context.Background(), es, map[string][]byte{"key": []byte("value")}, existing)
			if tt.wantErr {
				require.ErrorIs(t, err, errProtectedMetadata)
				return
			}
			require.NoError(t, err)
			metadata := result.Object["metadata"].(map[string]any)
			assert.Equal(t, int64(2), metadata["generation"])
			assert.Equal(t, "target-uid", metadata["uid"])
		})
	}
}

// TestReconcileRejectsStoredSecretManifest covers an ExternalSecret that already names a core
// Secret manifest, which the admission webhook now refuses but cannot retroactively remove.
// The controller has to refuse it too, and must leave whatever sits at the target name alone
// rather than adopting it through the generic write path.
func TestReconcileRejectsStoredSecretManifest(t *testing.T) {
	_ = esv1.AddToScheme(scheme.Scheme)

	const (
		namespace  = "ns"
		esName     = "stored-es"
		targetName = "target-secret"
		storeName  = "test-store"
	)

	for _, tt := range []struct {
		name         string
		targetExists bool
	}{
		{"missing target is not created", false},
		{"existing target is not mutated", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			es := &esv1.ExternalSecret{
				Name: esName, Namespace: namespace, UID: "es-uid",
				Spec: esv1.ExternalSecretSpec{
					RefreshInterval: &metav1.Duration{Duration: time.Hour},
					SecretStoreRef:  esv1.SecretStoreRef{Name: storeName, Kind: esv1.SecretStoreKind},
					Data: []esv1.ExternalSecretData{{
						SecretKey: "key",
						RemoteRef: esv1.ExternalSecretDataRemoteRef{Key: "remote-key"},
					}},
					Target: esv1.ExternalSecretTarget{
						Name:           targetName,
						CreationPolicy: esv1.CreatePolicyOwner,
						Manifest:       &esv1.ManifestReference{APIVersion: "v1", Kind: "Secret"},
					},
				},
			}
			store := &esv1.SecretStore{
				Name: storeName, Namespace: namespace,
				Spec: esv1.SecretStoreSpec{Provider: &esv1.SecretStoreProvider{
					AWS: &esv1.AWSProvider{Service: esv1.AWSServiceSecretsManager},
				}},
			}

			objects := []client.Object{es, store}
			if tt.targetExists {
				objects = append(objects, &v1.Secret{
					Name: targetName, Namespace: namespace,
					Data: map[string][]byte{"untouched": []byte("original")},
				})
			}

			fakeClient := fakeclient.NewClientBuilder().WithScheme(scheme.Scheme).
				WithObjects(objects...).
				WithStatusSubresource(&esv1.ExternalSecret{}).Build()
			r := &Reconciler{
				Client: fakeClient, SecretClient: fakeClient, Scheme: scheme.Scheme,
				Log: logr.Discard(), recorder: record.NewFakeRecorder(100),
				// the unsafe flag is on, so the ban is the only thing refusing this
				AllowGenericTargets: true,
				informerManager:     noopInformerManager{},
				RequeueInterval:     time.Hour,
			}

			// a configuration error is not retryable, so no error is returned for requeue
			_, err := r.Reconcile(context.Background(), ctrl.Request{
				Name: esName, Namespace: namespace,
			})
			require.NoError(t, err)

			synced := &esv1.ExternalSecret{}
			require.NoError(t, r.Get(context.Background(),
				types.NamespacedName{Name: esName, Namespace: namespace}, synced))
			cond := esv1.GetExternalSecretCondition(synced.Status, esv1.ExternalSecretReady)
			require.NotNil(t, cond)
			assert.Equal(t, v1.ConditionFalse, cond.Status)

			target := &v1.Secret{}
			getErr := r.Get(context.Background(),
				types.NamespacedName{Name: targetName, Namespace: namespace}, target)
			if !tt.targetExists {
				assert.True(t, apierrors.IsNotFound(getErr), "target must not be created")
				return
			}
			require.NoError(t, getErr)
			assert.Equal(t, map[string][]byte{"untouched": []byte("original")}, target.Data)
			assert.Empty(t, target.OwnerReferences)
			assert.Empty(t, target.Labels[esv1.LabelManaged])
		})
	}
}
