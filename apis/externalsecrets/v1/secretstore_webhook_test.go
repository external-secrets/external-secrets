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

package v1

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSecretStoreDefaulterDefaultsRuntimeRefKindToProviderClass(t *testing.T) {
	store := &SecretStore{
		Spec: SecretStoreSpec{
			RuntimeRef: &StoreRuntimeRef{
				Name: "aws",
			},
		},
	}

	if err := (&secretStoreDefaulter{}).Default(context.Background(), store); err != nil {
		t.Fatalf("Default() error = %v", err)
	}
	if store.Spec.RuntimeRef.Kind != "ProviderClass" {
		t.Fatalf("expected runtimeRef.kind to default to ProviderClass, got %q", store.Spec.RuntimeRef.Kind)
	}
}

func TestClusterSecretStoreDefaulterDefaultsRuntimeRefKindToClusterProviderClass(t *testing.T) {
	store := &ClusterSecretStore{
		Spec: SecretStoreSpec{
			RuntimeRef: &StoreRuntimeRef{
				Name: "aws",
			},
		},
	}

	if err := (&clusterSecretStoreDefaulter{}).Default(context.Background(), store); err != nil {
		t.Fatalf("Default() error = %v", err)
	}
	if store.Spec.RuntimeRef.Kind != "ClusterProviderClass" {
		t.Fatalf("expected runtimeRef.kind to default to ClusterProviderClass, got %q", store.Spec.RuntimeRef.Kind)
	}
}

func TestSecretStoreDefaulterDoesNotDefaultProviderRefNamespace(t *testing.T) {
	store := &SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
		},
		Spec: SecretStoreSpec{
			ProviderRef: &StoreProviderRef{
				Name:      "aws",
				Namespace: "other",
			},
			RuntimeRef: &StoreRuntimeRef{
				Name: "aws",
			},
		},
	}

	if err := (&secretStoreDefaulter{}).Default(context.Background(), store); err != nil {
		t.Fatalf("Default() error = %v", err)
	}
	if store.Spec.ProviderRef.Namespace != "other" {
		t.Fatalf("expected providerRef.namespace to remain %q, got %q", "other", store.Spec.ProviderRef.Namespace)
	}
}
