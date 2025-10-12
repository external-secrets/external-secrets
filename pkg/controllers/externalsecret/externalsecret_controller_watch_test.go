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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/log"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

func TestEnsureWatchForGVK(t *testing.T) {
	tests := []struct {
		name          string
		gvk           schema.GroupVersionKind
		expectError   bool
		errorContains string
	}{
		{
			name: "ConfigMap GVK",
			gvk: schema.GroupVersionKind{
				Group:   "",
				Version: "v1",
				Kind:    "ConfigMap",
			},
			expectError: false,
		},
		{
			name: "Custom Resource GVK",
			gvk: schema.GroupVersionKind{
				Group:   "argoproj.io",
				Version: "v1alpha1",
				Kind:    "Application",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: This test can't fully test watch registration without a real controller
			// We're mainly testing that the function doesn't panic and handles the GVK correctly
			r := &Reconciler{
				Log:                   log.Log.WithName("test"),
				AllowNonSecretTargets: true,
			}

			// error is expected as there is no controller setup.
			err := r.ensureWatchForGVK(tt.gvk)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "controller not initialized")
		})
	}
}

func TestGetTargetResourceIndex(t *testing.T) {
	tests := []struct {
		name           string
		es             *esv1.ExternalSecret
		expectedValues []string
	}{
		{
			name: "ConfigMap target",
			es: &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{
						Name: "my-configmap",
						Manifest: &esv1.ManifestTarget{
							APIVersion: "v1",
							Kind:       "ConfigMap",
						},
					},
				},
			},
			expectedValues: []string{"/v1/ConfigMap/my-configmap"},
		},
		{
			name: "ArgoCD Application target",
			es: &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{
						Name: "my-app",
						Manifest: &esv1.ManifestTarget{
							APIVersion: "argoproj.io/v1alpha1",
							Kind:       "Application",
						},
					},
				},
			},
			expectedValues: []string{"argoproj.io/v1alpha1/Application/my-app"},
		},
		{
			name: "Secret target (no manifest)",
			es: &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{
						Name:     "my-secret",
						Manifest: nil,
					},
				},
			},
			expectedValues: nil, // Secrets don't get indexed for non-Secret resources
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the indexer function used in SetupWithManager
			if !isNonSecretTarget(tt.es) {
				assert.Nil(t, tt.expectedValues)
				return
			}

			gvk := getTargetGVK(tt.es)
			targetName := getTargetName(tt.es)
			indexValue := gvk.Group + "/" + gvk.Version + "/" + gvk.Kind + "/" + targetName

			require.Len(t, tt.expectedValues, 1)
			assert.Equal(t, tt.expectedValues[0], indexValue)
		})
	}
}

func TestWatchedGVKTracking(t *testing.T) {
	r := &Reconciler{
		Log: log.Log.WithName("test"),
	}

	gvk1 := schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMap"}
	gvk2 := schema.GroupVersionKind{Group: "argoproj.io", Version: "v1alpha1", Kind: "Application"}

	_, watched1 := r.watchedGVKs.Load(gvk1.String())
	assert.False(t, watched1)
	r.watchedGVKs.Store(gvk1.String(), true)

	_, watched1 = r.watchedGVKs.Load(gvk1.String())
	assert.True(t, watched1)

	_, watched2 := r.watchedGVKs.Load(gvk2.String())
	assert.False(t, watched2)

	r.watchedGVKs.Store(gvk2.String(), true)
	_, watched2 = r.watchedGVKs.Load(gvk2.String())
	assert.True(t, watched2)
}

func TestGVKFromManifestTarget(t *testing.T) {
	tests := []struct {
		name        string
		manifest    *esv1.ManifestTarget
		expectedGVK schema.GroupVersionKind
	}{
		{
			name: "Core v1 ConfigMap",
			manifest: &esv1.ManifestTarget{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			},
			expectedGVK: schema.GroupVersionKind{
				Group:   "",
				Version: "v1",
				Kind:    "ConfigMap",
			},
		},
		{
			name: "ArgoCD Application",
			manifest: &esv1.ManifestTarget{
				APIVersion: "argoproj.io/v1alpha1",
				Kind:       "Application",
			},
			expectedGVK: schema.GroupVersionKind{
				Group:   "argoproj.io",
				Version: "v1alpha1",
				Kind:    "Application",
			},
		},
		{
			name: "Networking v1 Ingress",
			manifest: &esv1.ManifestTarget{
				APIVersion: "networking.k8s.io/v1",
				Kind:       "Ingress",
			},
			expectedGVK: schema.GroupVersionKind{
				Group:   "networking.k8s.io",
				Version: "v1",
				Kind:    "Ingress",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			es := &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{
						Manifest: tt.manifest,
					},
				},
			}

			gvk := getTargetGVK(es)
			assert.Equal(t, tt.expectedGVK.Group, gvk.Group)
			assert.Equal(t, tt.expectedGVK.Version, gvk.Version)
			assert.Equal(t, tt.expectedGVK.Kind, gvk.Kind)
		})
	}
}
