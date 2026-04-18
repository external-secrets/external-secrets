package v1alpha1

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
)

func TestAddToSchemeRegistersClusterProviderClass(t *testing.T) {
	s := runtime.NewScheme()
	if err := AddToScheme(s); err != nil {
		t.Fatalf("AddToScheme() error = %v", err)
	}

	obj, err := s.New(SchemeGroupVersion.WithKind("ClusterProviderClass"))
	if err != nil {
		t.Fatalf("scheme.New() error = %v", err)
	}
	if _, ok := obj.(*ClusterProviderClass); !ok {
		t.Fatalf("expected *ClusterProviderClass, got %T", obj)
	}
}
