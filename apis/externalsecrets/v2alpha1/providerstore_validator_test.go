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

package v2alpha1

import (
	"context"
	"strings"
	"testing"
)

func TestProviderStoreValidateCreateRejectsCrossNamespaceBackendRef(t *testing.T) {
	store := &ProviderStore{
		Spec: ProviderStoreSpec{
			RuntimeRef: StoreRuntimeRef{Name: "runtime"},
			BackendRef: BackendObjectReference{
				APIVersion: "provider.aws.external-secrets.io/v2alpha1",
				Kind:       "SecretsManagerStore",
				Name:       "team-a-backend",
				Namespace:  "team-b",
			},
		},
	}
	store.Namespace = "team-a"

	if _, err := (&ProviderStoreValidator{}).ValidateCreate(context.Background(), store); err == nil {
		t.Fatal("expected cross-namespace backendRef to be rejected")
	}
}

func TestClusterProviderStoreValidateCreateAllowsOmittedBackendNamespace(t *testing.T) {
	store := &ClusterProviderStore{
		Spec: ClusterProviderStoreSpec{
			RuntimeRef: StoreRuntimeRef{Name: "runtime"},
			BackendRef: BackendObjectReference{
				APIVersion: "provider.aws.external-secrets.io/v2alpha1",
				Kind:       "SecretsManagerStore",
				Name:       "shared-backend",
			},
		},
	}

	if _, err := (&ClusterProviderStoreValidator{}).ValidateCreate(context.Background(), store); err != nil {
		t.Fatalf("expected omitted backendRef.namespace to be allowed: %v", err)
	}
}

func TestClusterProviderStoreValidateCreateRejectsInvalidNamespaceRegex(t *testing.T) {
	store := &ClusterProviderStore{
		Spec: ClusterProviderStoreSpec{
			RuntimeRef: StoreRuntimeRef{Name: "runtime"},
			BackendRef: BackendObjectReference{
				APIVersion: "provider.aws.external-secrets.io/v2alpha1",
				Kind:       "SecretsManagerStore",
				Name:       "shared-backend",
			},
			Conditions: []StoreNamespaceCondition{
				{
					NamespaceRegexes: []string{`\1`},
				},
			},
		},
	}

	_, err := (&ClusterProviderStoreValidator{}).ValidateCreate(context.Background(), store)
	if err == nil {
		t.Fatal("expected invalid namespace regex to be rejected")
	}
	if !strings.Contains(err.Error(), "failed to compile 0th namespace regex in 0th condition") {
		t.Fatalf("expected regex compilation failure, got: %v", err)
	}
}
