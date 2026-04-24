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

package kubernetes

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"

	. "github.com/onsi/gomega"
)

func TestExternalSecretConditionHasStatus(t *testing.T) {
	t.Helper()
	RegisterTestingT(t)

	condition := &esv1.ExternalSecretStatusCondition{Status: corev1.ConditionTrue}

	if !externalSecretConditionHasStatus(condition, corev1.ConditionTrue) {
		t.Fatalf("expected helper to match ExternalSecret corev1 condition status")
	}

	if externalSecretConditionHasStatus(condition, corev1.ConditionFalse) {
		t.Fatalf("expected helper not to match a different ExternalSecret corev1 condition status")
	}
}

func TestDeleteExternalSecretAndWaitDeletesExternalSecret(t *testing.T) {
	t.Helper()
	RegisterTestingT(t)

	cl := newClusterProviderScenarioTestClient(t, &esv1.ExternalSecret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example",
			Namespace: "test",
		},
	})

	err := deleteExternalSecretAndWait(context.Background(), cl, types.NamespacedName{
		Name:      "example",
		Namespace: "test",
	})
	Expect(err).NotTo(HaveOccurred())

	var externalSecret esv1.ExternalSecret
	err = cl.Get(context.Background(), client.ObjectKey{Name: "example", Namespace: "test"}, &externalSecret)
	Expect(apierrors.IsNotFound(err)).To(BeTrue())
}

func TestDeleteExternalSecretAndWaitIgnoresMissingExternalSecret(t *testing.T) {
	t.Helper()
	RegisterTestingT(t)

	cl := newClusterProviderScenarioTestClient(t)

	err := deleteExternalSecretAndWait(context.Background(), cl, types.NamespacedName{
		Name:      "missing",
		Namespace: "test",
	})
	Expect(err).NotTo(HaveOccurred())
}

func TestClusterProviderV2NamespacesForManifestScope(t *testing.T) {
	t.Helper()

	layout := newClusterProviderV2Layout("workload-ns", "case", esv1.AuthenticationScopeManifestNamespace, func(string) string {
		t.Fatal("did not expect provider namespace factory to run for manifest scope")
		return ""
	})

	if layout.backendNamespace != "workload-ns" {
		t.Fatalf("expected backend namespace to use workload namespace, got %q", layout.backendNamespace)
	}
	if layout.providerRefNamespace != "" {
		t.Fatalf("expected provider ref namespace to be omitted for manifest scope, got %q", layout.providerRefNamespace)
	}
	if layout.providerNamespace != "workload-ns" {
		t.Fatalf("expected provider namespace to use workload namespace, got %q", layout.providerNamespace)
	}
}

func TestClusterProviderV2NamespacesForProviderScope(t *testing.T) {
	t.Helper()

	calledWith := ""
	layout := newClusterProviderV2Layout("workload-ns", "case", esv1.AuthenticationScopeProviderNamespace, func(prefix string) string {
		calledWith = prefix
		return "provider-ns"
	})

	if calledWith != "case-provider" {
		t.Fatalf("expected provider namespace factory to use case-provider prefix, got %q", calledWith)
	}
	if layout.backendNamespace != "provider-ns" {
		t.Fatalf("expected backend namespace to use provider namespace, got %q", layout.backendNamespace)
	}
	if layout.providerRefNamespace != "provider-ns" {
		t.Fatalf("expected provider ref namespace to use provider namespace, got %q", layout.providerRefNamespace)
	}
	if layout.providerNamespace != "provider-ns" {
		t.Fatalf("expected provider namespace to use provider namespace, got %q", layout.providerNamespace)
	}
}

func TestClusterProviderV2ProviderConfigNamespaceUsesBackendNamespaceForManifestScope(t *testing.T) {
	t.Helper()

	scenario := &clusterProviderV2Scenario{
		backendNamespace:     "workload-ns",
		providerRefNamespace: "",
	}

	if got := scenario.providerConfigNamespace(); got != "workload-ns" {
		t.Fatalf("expected provider config namespace to use backend namespace, got %q", got)
	}
}

func TestClusterProviderV2ProviderConfigNamespaceUsesExplicitProviderRefNamespace(t *testing.T) {
	t.Helper()

	scenario := &clusterProviderV2Scenario{
		backendNamespace:     "workload-ns",
		providerRefNamespace: "provider-ns",
	}

	if got := scenario.providerConfigNamespace(); got != "provider-ns" {
		t.Fatalf("expected provider config namespace to use providerRef namespace, got %q", got)
	}
}

func newClusterProviderScenarioTestClient(t *testing.T, objs ...client.Object) client.Client {
	t.Helper()

	scheme := runtime.NewScheme()
	Expect(corev1.AddToScheme(scheme)).To(Succeed())
	Expect(esv1.AddToScheme(scheme)).To(Succeed())

	builder := fake.NewClientBuilder().WithScheme(scheme)
	if len(objs) > 0 {
		builder = builder.WithObjects(objs...)
	}
	return builder.Build()
}
