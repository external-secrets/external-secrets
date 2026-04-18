package v1

import "testing"

func TestSecretStoreSpecDeepCopyPreservesRuntimeRef(t *testing.T) {
	spec := &SecretStoreSpec{
		RuntimeRef: &StoreRuntimeRef{
			Kind: "ClusterProviderClass",
			Name: "aws",
		},
	}

	out := spec.DeepCopy()
	if out.RuntimeRef == nil {
		t.Fatalf("expected RuntimeRef to be copied")
	}
	if out.RuntimeRef.Name != "aws" || out.RuntimeRef.Kind != "ClusterProviderClass" {
		t.Fatalf("unexpected RuntimeRef copy: %#v", out.RuntimeRef)
	}
	if out.RuntimeRef == spec.RuntimeRef {
		t.Fatalf("expected RuntimeRef to be deep-copied")
	}
}
