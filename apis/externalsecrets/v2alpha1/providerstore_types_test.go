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
