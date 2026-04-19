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

package addon

import (
	"context"
	"testing"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestIsExternalSecretsCRDGroup(t *testing.T) {
	t.Helper()

	if !isExternalSecretsCRDGroup("external-secrets.io") {
		t.Fatal("expected external-secrets.io group to match")
	}
	if !isExternalSecretsCRDGroup("provider.external-secrets.io") {
		t.Fatal("expected provider.external-secrets.io group to match")
	}
	if isExternalSecretsCRDGroup("example.com") {
		t.Fatal("did not expect unrelated group to match")
	}
}

func TestListExternalSecretsCRDsFiltersByGroup(t *testing.T) {
	t.Helper()

	scheme := runtime.NewScheme()
	if err := apiextensionsv1.AddToScheme(scheme); err != nil {
		t.Fatalf("add apiextensions scheme: %v", err)
	}

	cfg := &Config{
		CRClient: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&apiextensionsv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{Name: "externalsecrets.external-secrets.io"},
				Spec:       apiextensionsv1.CustomResourceDefinitionSpec{Group: "external-secrets.io"},
			},
			&apiextensionsv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{Name: "providers.provider.external-secrets.io"},
				Spec:       apiextensionsv1.CustomResourceDefinitionSpec{Group: "provider.external-secrets.io"},
			},
			&apiextensionsv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{Name: "widgets.example.com"},
				Spec:       apiextensionsv1.CustomResourceDefinitionSpec{Group: "example.com"},
			},
		).Build(),
	}

	crds, err := listExternalSecretsCRDs(context.Background(), cfg)
	if err != nil {
		t.Fatalf("listExternalSecretsCRDs returned error: %v", err)
	}
	if len(crds) != 2 {
		t.Fatalf("expected 2 external-secrets CRDs, got %d", len(crds))
	}
}
