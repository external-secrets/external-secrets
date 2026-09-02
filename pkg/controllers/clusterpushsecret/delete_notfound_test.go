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

package clusterpushsecret

import (
	"context"
	"errors"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
)

func ownedPushSecret(name, namespace, cpsName string) *v1alpha1.PushSecret {
	return &v1alpha1.PushSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: v1alpha1.SchemeGroupVersion.String(),
					Kind:       "ClusterPushSecret",
					Name:       cpsName,
					Controller: ptrTo(true),
				},
			},
		},
	}
}

func ptrTo[T any](v T) *T { return &v }

func newScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := v1alpha1.AddToScheme(s); err != nil {
		t.Fatalf("adding to scheme: %v", err)
	}
	return s
}

// The controller cache can still hold a PushSecret the API server has already
// dropped, so a NotFound from the delete is the state the cleanup wanted
// rather than a failure.
func TestDeletePushSecretIgnoresNotFound(t *testing.T) {
	ps := ownedPushSecret("ps", "ns", "cps")
	c := fake.NewClientBuilder().
		WithScheme(newScheme(t)).
		WithObjects(ps).
		WithInterceptorFuncs(interceptor.Funcs{
			Delete: func(_ context.Context, _ client.WithWatch, obj client.Object, _ ...client.DeleteOption) error {
				return apierrors.NewNotFound(schema.GroupResource{Group: v1alpha1.Group, Resource: "pushsecrets"}, obj.GetName())
			},
		}).
		Build()

	r := &Reconciler{Client: c, Scheme: c.Scheme()}

	if err := r.deletePushSecret(context.Background(), "ps", "cps", "ns"); err != nil {
		t.Fatalf("expected no error for an already deleted push secret, got %v", err)
	}
}

func TestDeletePushSecretReturnsOtherDeleteErrors(t *testing.T) {
	ps := ownedPushSecret("ps", "ns", "cps")
	wantErr := errors.New("boom")
	c := fake.NewClientBuilder().
		WithScheme(newScheme(t)).
		WithObjects(ps).
		WithInterceptorFuncs(interceptor.Funcs{
			Delete: func(_ context.Context, _ client.WithWatch, _ client.Object, _ ...client.DeleteOption) error {
				return wantErr
			},
		}).
		Build()

	r := &Reconciler{Client: c, Scheme: c.Scheme()}

	err := r.deletePushSecret(context.Background(), "ps", "cps", "ns")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected the delete error to be returned, got %v", err)
	}
}
