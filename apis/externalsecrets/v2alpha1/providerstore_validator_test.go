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
