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

package clusterexternalsecret

import (
	"context"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// TestDeleteExternalSecretIgnoresNotFound verifies that a Delete that returns
// NotFound (e.g. the child ExternalSecret was already removed between the
// cached Get and the Delete) is treated as a successful cleanup rather than a
// failure.
func TestDeleteExternalSecretIgnoresNotFound(t *testing.T) {
	es := &esv1.ExternalSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-es",
			Namespace: "test-ns",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: esv1.SchemeGroupVersion.String(),
				Kind:       esv1.ClusterExtSecretKind,
				Name:       "test-ces",
				Controller: ptr.To(true),
			}},
		},
	}

	scheme := runtime.NewScheme()
	if err := esv1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add scheme: %v", err)
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(es).
		WithInterceptorFuncs(interceptor.Funcs{
			Delete: func(ctx context.Context, c crclient.WithWatch, obj crclient.Object, opts ...crclient.DeleteOption) error {
				return apierrors.NewNotFound(schema.GroupResource{Group: esv1.Group, Resource: "externalsecrets"}, obj.GetName())
			},
		}).
		Build()

	r := &Reconciler{Client: cl}

	if err := r.deleteExternalSecret(context.Background(), "test-es", "test-ces", "test-ns"); err != nil {
		t.Fatalf("expected NotFound on delete to be ignored, got: %v", err)
	}
}
