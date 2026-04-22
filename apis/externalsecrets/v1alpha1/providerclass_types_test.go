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

package v1alpha1

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestAddToSchemeRegistersProviderClass(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme: %v", err)
	}

	gvks, _, err := scheme.ObjectKinds(&ProviderClass{})
	if err != nil {
		t.Fatalf("ObjectKinds: %v", err)
	}
	expected := SchemeGroupVersion.WithKind("ProviderClass")
	found := false
	for _, gvk := range gvks {
		if gvk == expected {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected scheme to register %s, got %v", expected, gvks)
	}
}

func TestProviderClassDeepCopyCopiesStatus(t *testing.T) {
	original := &ProviderClass{
		Status: ProviderClassStatus{
			Conditions: []metav1.Condition{
				{
					Type:   "Ready",
					Status: metav1.ConditionTrue,
					Reason: "Configured",
				},
			},
		},
	}

	copied := original.DeepCopy()
	if copied == nil {
		t.Fatalf("expected DeepCopy to return a value")
	}
	if len(copied.Status.Conditions) != 1 {
		t.Fatalf("expected copied conditions, got %v", copied.Status.Conditions)
	}
	if copied.Status.Conditions[0].Reason != "Configured" {
		t.Fatalf("unexpected copied condition: %#v", copied.Status.Conditions[0])
	}

	original.Status.Conditions[0].Reason = "Updated"
	if copied.Status.Conditions[0].Reason == "Updated" {
		t.Fatalf("expected conditions to be deep-copied")
	}
}
