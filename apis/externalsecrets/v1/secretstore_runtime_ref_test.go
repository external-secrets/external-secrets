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

func TestSecretStoreSpecDeepCopyPreservesProviderClassRuntimeRef(t *testing.T) {
	spec := &SecretStoreSpec{
		RuntimeRef: &StoreRuntimeRef{
			Kind: "ProviderClass",
			Name: "aws",
		},
	}

	out := spec.DeepCopy()
	if out.RuntimeRef == nil {
		t.Fatalf("expected RuntimeRef to be copied")
	}
	if out.RuntimeRef.Name != "aws" || out.RuntimeRef.Kind != "ProviderClass" {
		t.Fatalf("unexpected RuntimeRef copy: %#v", out.RuntimeRef)
	}
	if out.RuntimeRef == spec.RuntimeRef {
		t.Fatalf("expected RuntimeRef to be deep-copied")
	}
}
