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

package externalsecret

import (
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
)

func TestHandleGenerateSecretsChecksCurrentControllerClass(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := genv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		name       string
		class      string
		wantReject bool
	}{
		{name: "foreign generator", class: "other", wantReject: true},
		{name: "managed generator", class: "mine"},
		{name: "shared generator"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			gen := &genv1alpha1.Fake{Spec: genv1alpha1.FakeSpec{
				Controller: tt.class, Data: map[string]string{"key": "value"},
			}}
			gen.SetName("generator")
			gen.SetNamespace("team-a")
			kube := fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(gen).Build()
			r := &Reconciler{Client: kube, Scheme: scheme, ControllerClass: "mine", Log: logr.Discard()}
			ref := esv1.ExternalSecretDataFromRemoteRef{SourceRef: &esv1.StoreGeneratorSourceRef{
				GeneratorRef: &esv1.GeneratorRef{
					APIVersion: genv1alpha1.SchemeGroupVersion.String(), Kind: "Fake", Name: gen.Name,
				},
			}}

			data, err := r.handleGenerateSecrets(context.Background(), "team-a", ref, 0, nil)
			if tt.wantReject {
				if err == nil || !strings.Contains(err.Error(), "unmanaged generator") {
					t.Fatalf("error = %v, want unmanaged generator", err)
				}
				return
			}
			if err != nil || string(data["key"]) != "value" {
				t.Errorf("data = %v, error = %v, want key=value", data, err)
			}
		})
	}
}
