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

package clusterproviderclass

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	grpccommon "github.com/external-secrets/external-secrets/providers/v2/common/grpc"
)

func TestClusterProviderClassReconcileMarksReadyWhenHealthCheckSucceeds(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(esv1alpha1.AddToScheme(scheme))

	obj := &esv1alpha1.ClusterProviderClass{
		ObjectMeta: metav1.ObjectMeta{Name: "aws"},
		Spec: esv1alpha1.ClusterProviderClassSpec{
			Address: "provider-aws.external-secrets-system.svc:8080",
		},
	}

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(obj).
		WithStatusSubresource(obj).
		Build()

	r := &Reconciler{
		Client: kubeClient,
		Log:    logr.Discard(),
		CheckHealth: func(context.Context, string, *grpccommon.TLSConfig) error {
			return nil
		},
	}

	_, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: "aws"}})
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	updated := &esv1alpha1.ClusterProviderClass{}
	if err := kubeClient.Get(context.Background(), types.NamespacedName{Name: "aws"}, updated); err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if meta.FindStatusCondition(updated.Status.Conditions, "Ready") == nil {
		t.Fatalf("expected Ready condition to be set")
	}
}
