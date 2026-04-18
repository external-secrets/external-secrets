package v2alpha1

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
)

func TestAddToSchemeRegistersProviderStores(t *testing.T) {
	s := runtime.NewScheme()
	if err := AddToScheme(s); err != nil {
		t.Fatalf("AddToScheme() error = %v", err)
	}

	tests := []struct {
		kind string
	}{
		{
			kind: "ProviderStore",
		},
		{
			kind: "ClusterProviderStore",
		},
	}

	for _, tt := range tests {
		obj, err := s.New(SchemeGroupVersion.WithKind(tt.kind))
		if err != nil {
			t.Fatalf("scheme.New(%q) error = %v", tt.kind, err)
		}
		switch tt.kind {
		case "ProviderStore":
			if _, ok := obj.(*ProviderStore); !ok {
				t.Fatalf("expected *ProviderStore, got %T", obj)
			}
		case "ClusterProviderStore":
			if _, ok := obj.(*ClusterProviderStore); !ok {
				t.Fatalf("expected *ClusterProviderStore, got %T", obj)
			}
		}
	}
}
