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
	"errors"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

const (
	notFoundCESName = "test-ces"
	notFoundESName  = "test-es"
	notFoundNS      = "test-ns"
)

// ownedExternalSecret returns an ExternalSecret that passes isExternalSecretOwnedBy.
func ownedExternalSecret(uid types.UID) *esv1.ExternalSecret {
	yes := true
	return &esv1.ExternalSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      notFoundESName,
			Namespace: notFoundNS,
			UID:       uid,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: esv1.SchemeGroupVersion.String(),
				Kind:       esv1.ClusterExtSecretKind,
				Name:       notFoundCESName,
				Controller: &yes,
			}},
		},
	}
}

func notFoundScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := esv1.AddToScheme(s); err != nil {
		t.Fatalf("add esv1 to scheme: %v", err)
	}
	return s
}

// The Get in deleteExternalSecret may be served from a stale informer cache, so
// the object can already be gone when the Delete reaches the API server. That
// NotFound means the cleanup succeeded and must not be reported as a failure.
func TestDeleteExternalSecretIgnoresNotFoundOnDelete(t *testing.T) {
	scheme := notFoundScheme(t)
	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ownedExternalSecret("uid-1")).
		WithInterceptorFuncs(interceptor.Funcs{
			Delete: func(context.Context, client.WithWatch, client.Object, ...client.DeleteOption) error {
				return apierrors.NewNotFound(
					schema.GroupResource{Group: esv1.Group, Resource: "externalsecrets"},
					notFoundESName,
				)
			},
		}).
		Build()

	r := &Reconciler{Client: c, Scheme: scheme}
	if err := r.deleteExternalSecret(context.Background(), notFoundESName, notFoundCESName, notFoundNS); err != nil {
		t.Fatalf("a NotFound on delete must count as cleaned up, got: %v", err)
	}
}

// The UID precondition keeps the harmful half of the same race closed: if an
// unowned ExternalSecret of that name was recreated in the meantime, the delete
// must be rejected rather than remove somebody else's object.
func TestDeleteExternalSecretPinsTheObjectByUID(t *testing.T) {
	scheme := notFoundScheme(t)
	var seen *metav1.Preconditions
	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ownedExternalSecret("uid-1")).
		WithInterceptorFuncs(interceptor.Funcs{
			Delete: func(_ context.Context, _ client.WithWatch, _ client.Object, opts ...client.DeleteOption) error {
				do := &client.DeleteOptions{}
				do.ApplyOptions(opts)
				seen = do.Preconditions
				return nil
			},
		}).
		Build()

	r := &Reconciler{Client: c, Scheme: scheme}
	if err := r.deleteExternalSecret(context.Background(), notFoundESName, notFoundCESName, notFoundNS); err != nil {
		t.Fatalf("delete of an owned external secret: %v", err)
	}
	if seen == nil || seen.UID == nil {
		t.Fatal("delete carried no UID precondition, so it would also remove a recreated, unowned object")
	}
	if *seen.UID != "uid-1" {
		t.Fatalf("delete pinned the wrong object: %q", *seen.UID)
	}
}

// Any other failure must still surface.
func TestDeleteExternalSecretStillReportsOtherErrors(t *testing.T) {
	scheme := notFoundScheme(t)
	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ownedExternalSecret("uid-1")).
		WithInterceptorFuncs(interceptor.Funcs{
			Delete: func(context.Context, client.WithWatch, client.Object, ...client.DeleteOption) error {
				return apierrors.NewInternalError(errors.New("etcd unavailable"))
			},
		}).
		Build()

	r := &Reconciler{Client: c, Scheme: scheme}
	if err := r.deleteExternalSecret(context.Background(), notFoundESName, notFoundCESName, notFoundNS); err == nil {
		t.Fatal("a non-NotFound delete failure must be reported")
	}
}
