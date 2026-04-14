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

package util

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
)

func TestClearKnownNamespaceFinalizers(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "target-secret",
				Namespace:  "e2e-tests-demo-12345",
				Finalizers: []string{"example.com/finalizer"},
			},
		},
		&esv1alpha1.PushSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "push-secret",
				Namespace:  "e2e-tests-demo-12345",
				Finalizers: []string{"pushsecret.externalsecrets.io/finalizer"},
			},
		},
	).Build()

	if err := ClearKnownNamespaceFinalizers(ctx, cl, "e2e-tests-demo-12345"); err != nil {
		t.Fatalf("ClearKnownNamespaceFinalizers() error = %v", err)
	}

	var secret corev1.Secret
	if err := cl.Get(ctx, client.ObjectKey{Name: "target-secret", Namespace: "e2e-tests-demo-12345"}, &secret); err != nil {
		t.Fatalf("Get(secret) error = %v", err)
	}
	if len(secret.Finalizers) != 0 {
		t.Fatalf("expected secret finalizers to be cleared, got %v", secret.Finalizers)
	}

	var pushSecret esv1alpha1.PushSecret
	if err := cl.Get(ctx, client.ObjectKey{Name: "push-secret", Namespace: "e2e-tests-demo-12345"}, &pushSecret); err != nil {
		t.Fatalf("Get(pushsecret) error = %v", err)
	}
	if len(pushSecret.Finalizers) != 0 {
		t.Fatalf("expected pushsecret finalizers to be cleared, got %v", pushSecret.Finalizers)
	}
}

func TestCleanupTerminatingE2ENamespaces(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	now := metav1.Now()
	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "e2e-tests-demo-12345",
				Finalizers: []string{"kubernetes"},
				DeletionTimestamp: &now,
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "plain-namespace",
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "target-secret",
				Namespace:  "e2e-tests-demo-12345",
				Finalizers: []string{"example.com/finalizer"},
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "untouched-secret",
				Namespace:  "plain-namespace",
				Finalizers: []string{"example.com/finalizer"},
			},
		},
	).Build()

	if err := CleanupTerminatingE2ENamespaces(ctx, cl); err != nil {
		t.Fatalf("CleanupTerminatingE2ENamespaces() error = %v", err)
	}

	var cleaned corev1.Secret
	if err := cl.Get(ctx, client.ObjectKey{Name: "target-secret", Namespace: "e2e-tests-demo-12345"}, &cleaned); err != nil {
		t.Fatalf("Get(cleaned) error = %v", err)
	}
	if len(cleaned.Finalizers) != 0 {
		t.Fatalf("expected terminating e2e namespace secret finalizers to be cleared, got %v", cleaned.Finalizers)
	}

	var untouched corev1.Secret
	if err := cl.Get(ctx, client.ObjectKey{Name: "untouched-secret", Namespace: "plain-namespace"}, &untouched); err != nil {
		t.Fatalf("Get(untouched) error = %v", err)
	}
	if len(untouched.Finalizers) != 1 {
		t.Fatalf("expected non-e2e namespace secret finalizers to remain, got %v", untouched.Finalizers)
	}
}
