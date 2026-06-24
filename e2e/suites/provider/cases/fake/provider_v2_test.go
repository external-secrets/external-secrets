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

package fake

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	metav1api "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/external-secrets/external-secrets-e2e/framework"
	frameworkv2 "github.com/external-secrets/external-secrets-e2e/framework/v2"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

func TestFakeBackendTargetUsesProviderNamespaceAndSelector(t *testing.T) {
	target := fakeBackendTarget()
	if target.Namespace != frameworkv2.ProviderNamespace {
		t.Fatalf("expected provider namespace %q, got %q", frameworkv2.ProviderNamespace, target.Namespace)
	}
	if target.PodLabelSelector != "app.kubernetes.io/name=external-secrets-provider-fake" {
		t.Fatalf("unexpected selector %q", target.PodLabelSelector)
	}
}

func TestUpsertFakeProviderDataReplacesMatchingEntry(t *testing.T) {
	input := []esv1.FakeProviderData{
		{Key: "other", Value: "untouched"},
		{Key: "remote", Value: "old", Version: "v1"},
	}

	got := upsertFakeProviderData(input, esv1.FakeProviderData{
		Key:     "remote",
		Value:   "new",
		Version: "v1",
	})

	want := []esv1.FakeProviderData{
		{Key: "other", Value: "untouched"},
		{Key: "remote", Value: "new", Version: "v1"},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("upsertFakeProviderData mismatch (-want +got):\n%s", diff)
	}
}

func TestRemoveFakeProviderDataRemovesOnlyExactMatch(t *testing.T) {
	input := []esv1.FakeProviderData{
		{Key: "remote", Value: "keep-version", Version: "v2"},
		{Key: "remote", Value: "drop-version", Version: "v1"},
		{Key: "other", Value: "keep"},
	}

	got := removeFakeProviderData(input, "remote", "v1")

	want := []esv1.FakeProviderData{
		{Key: "remote", Value: "keep-version", Version: "v2"},
		{Key: "other", Value: "keep"},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("removeFakeProviderData mismatch (-want +got):\n%s", diff)
	}
}

func TestProviderReferenceNamespace(t *testing.T) {
	if got := providerReferenceNamespace(esv1.AuthenticationScopeManifestNamespace, "provider-ns"); got != "" {
		t.Fatalf("expected empty providerRef namespace for manifest scope, got %q", got)
	}
	if got := providerReferenceNamespace(esv1.AuthenticationScopeProviderNamespace, "provider-ns"); got != "provider-ns" {
		t.Fatalf("expected providerRef namespace for provider scope, got %q", got)
	}
}

func TestFakeConfigNamespaceForAuthScope(t *testing.T) {
	if got := fakeConfigNamespaceForAuthScope(esv1.AuthenticationScopeManifestNamespace, "manifest-ns", "provider-ns"); got != "manifest-ns" {
		t.Fatalf("expected manifest namespace for manifest scope, got %q", got)
	}
	if got := fakeConfigNamespaceForAuthScope(esv1.AuthenticationScopeProviderNamespace, "manifest-ns", "provider-ns"); got != "provider-ns" {
		t.Fatalf("expected provider namespace for provider scope, got %q", got)
	}
}

func TestCreateOrUpdateReadbackExternalSecretUpdatesExistingObject(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := esv1.AddToScheme(scheme); err != nil {
		t.Fatalf("add external secrets scheme: %v", err)
	}

	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&esv1.ExternalSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "readback",
			Namespace: "default",
		},
		Spec: esv1.ExternalSecretSpec{
			Target: esv1.ExternalSecretTarget{Name: "old-target"},
		},
	}).Build()

	updated := &esv1.ExternalSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "readback",
			Namespace: "default",
		},
		Spec: esv1.ExternalSecretSpec{
			Target: esv1.ExternalSecretTarget{Name: "new-target"},
		},
	}
	f := &framework.Framework{CRClient: cl}
	if err := createOrUpdateReadbackExternalSecret(context.Background(), f, updated); err != nil {
		t.Fatalf("createOrUpdateReadbackExternalSecret returned error: %v", err)
	}

	var got esv1.ExternalSecret
	if err := cl.Get(context.Background(), client.ObjectKeyFromObject(updated), &got); err != nil {
		t.Fatalf("get external secret: %v", err)
	}
	if got.Spec.Target.Name != "new-target" {
		t.Fatalf("expected updated target name, got %q", got.Spec.Target.Name)
	}
}

type flakyReadbackCreateClient struct {
	client.Client
	createErrs  []error
	createCalls int
}

func (c *flakyReadbackCreateClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	callIndex := c.createCalls
	c.createCalls++
	if callIndex < len(c.createErrs) && c.createErrs[callIndex] != nil {
		return c.createErrs[callIndex]
	}
	return c.Client.Create(ctx, obj, opts...)
}

func TestCreateOrUpdateReadbackExternalSecretRetriesMissingAPIResourceErrors(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := esv1.AddToScheme(scheme); err != nil {
		t.Fatalf("add external secrets scheme: %v", err)
	}

	baseClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	cl := &flakyReadbackCreateClient{
		Client: baseClient,
		createErrs: []error{
			&metav1api.NoResourceMatchError{},
		},
	}
	f := &framework.Framework{CRClient: cl}

	externalSecret := &esv1.ExternalSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "readback",
			Namespace: "default",
		},
	}

	if err := createOrUpdateReadbackExternalSecret(context.Background(), f, externalSecret); err != nil {
		t.Fatalf("createOrUpdateReadbackExternalSecret returned error: %v", err)
	}
	if cl.createCalls != 2 {
		t.Fatalf("expected 2 create calls, got %d", cl.createCalls)
	}

	var got esv1.ExternalSecret
	if err := baseClient.Get(context.Background(), client.ObjectKeyFromObject(externalSecret), &got); err != nil {
		t.Fatalf("get external secret: %v", err)
	}
}
