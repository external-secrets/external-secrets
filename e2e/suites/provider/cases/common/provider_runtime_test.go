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

package common

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/external-secrets/external-secrets-e2e/framework"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
)

func TestClusterProviderExternalSecretRuntimeSupportsAuthLifecycle(t *testing.T) {
	runtimeWithoutHooks := &ClusterProviderExternalSecretRuntime{}
	if runtimeWithoutHooks.SupportsAuthLifecycle() {
		t.Fatalf("expected SupportsAuthLifecycle to return false when both hooks are nil")
	}

	runtimeWithBreakOnly := &ClusterProviderExternalSecretRuntime{
		BreakAuth: func() {},
	}
	if runtimeWithBreakOnly.SupportsAuthLifecycle() {
		t.Fatalf("expected SupportsAuthLifecycle to return false when RepairAuth is nil")
	}

	runtimeWithRepairOnly := &ClusterProviderExternalSecretRuntime{
		RepairAuth: func() {},
	}
	if runtimeWithRepairOnly.SupportsAuthLifecycle() {
		t.Fatalf("expected SupportsAuthLifecycle to return false when BreakAuth is nil")
	}

	runtimeWithBothHooks := &ClusterProviderExternalSecretRuntime{
		BreakAuth:  func() {},
		RepairAuth: func() {},
	}
	if !runtimeWithBothHooks.SupportsAuthLifecycle() {
		t.Fatalf("expected SupportsAuthLifecycle to return true when both hooks are present")
	}
}

func TestClusterProviderPushRuntimeSupportsAuthLifecycle(t *testing.T) {
	runtimeWithoutHooks := &ClusterProviderPushRuntime{}
	if runtimeWithoutHooks.SupportsAuthLifecycle() {
		t.Fatalf("expected SupportsAuthLifecycle to return false when both hooks are nil")
	}

	runtimeWithBreakOnly := &ClusterProviderPushRuntime{
		BreakAuth: func() {},
	}
	if runtimeWithBreakOnly.SupportsAuthLifecycle() {
		t.Fatalf("expected SupportsAuthLifecycle to return false when RepairAuth is nil")
	}

	runtimeWithRepairOnly := &ClusterProviderPushRuntime{
		RepairAuth: func() {},
	}
	if runtimeWithRepairOnly.SupportsAuthLifecycle() {
		t.Fatalf("expected SupportsAuthLifecycle to return false when BreakAuth is nil")
	}

	runtimeWithBothHooks := &ClusterProviderPushRuntime{
		BreakAuth:  func() {},
		RepairAuth: func() {},
	}
	if !runtimeWithBothHooks.SupportsAuthLifecycle() {
		t.Fatalf("expected SupportsAuthLifecycle to return true when both hooks are present")
	}
}

func TestClusterProviderPushRuntimeSupportsRemoteAbsenceAssertions(t *testing.T) {
	runtimeWithoutExpectation := &ClusterProviderPushRuntime{}
	if runtimeWithoutExpectation.SupportsRemoteAbsenceAssertions() {
		t.Fatalf("expected SupportsRemoteAbsenceAssertions to return false when ExpectNoRemoteSecret is nil")
	}

	runtimeWithExpectation := &ClusterProviderPushRuntime{
		ExpectNoRemoteSecret: func(_, _ string) {},
	}
	if !runtimeWithExpectation.SupportsRemoteAbsenceAssertions() {
		t.Fatalf("expected SupportsRemoteAbsenceAssertions to return true when ExpectNoRemoteSecret is present")
	}
}

func TestClusterProviderPushRuntimeSupportsRemoteNamespaceOverrides(t *testing.T) {
	runtimeWithoutFactory := &ClusterProviderPushRuntime{}
	if runtimeWithoutFactory.SupportsRemoteNamespaceOverrides() {
		t.Fatalf("expected SupportsRemoteNamespaceOverrides to return false when CreateWritableRemoteScope is nil")
	}

	runtimeWithFactory := &ClusterProviderPushRuntime{
		CreateWritableRemoteScope: func(_ string) string { return "override-namespace" },
	}
	if !runtimeWithFactory.SupportsRemoteNamespaceOverrides() {
		t.Fatalf("expected SupportsRemoteNamespaceOverrides to return true when CreateWritableRemoteScope is present")
	}
}

func TestApplyClusterProviderPushSecretPanicsWithClearMessageWhenRuntimeNil(t *testing.T) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatalf("expected panic when runtime is nil")
		}
		message, ok := recovered.(string)
		if !ok {
			t.Fatalf("expected panic message to be string, got %T", recovered)
		}
		if !strings.Contains(message, "cluster provider push harness returned nil runtime") {
			t.Fatalf("expected panic message to mention nil runtime guard, got %q", message)
		}
	}()

	applyClusterProviderPushSecret(nil, nil, "remote-secret")
}

func TestApplyClusterProviderPushSecretUsesSafeObjectNameIndependentOfRemoteKey(t *testing.T) {
	tc := &framework.TestCase{
		PushSecret: &esv1alpha1.PushSecret{},
		PushSecretSource: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: "push-provider-source",
			},
		},
	}
	runtime := &ClusterProviderPushRuntime{
		ClusterProviderName: "push-provider-cluster-provider",
	}

	applyClusterProviderPushSecret(tc, runtime, "/e2e/test-ns/push-provider-remote")

	if got, want := tc.PushSecret.ObjectMeta.Name, "push-provider-source-push-secret"; got != want {
		t.Fatalf("expected PushSecret name %q, got %q", want, got)
	}
	if got, want := tc.PushSecret.Spec.Data[0].Match.RemoteRef.RemoteKey, "/e2e/test-ns/push-provider-remote"; got != want {
		t.Fatalf("expected remote key %q, got %q", want, got)
	}
}

func TestClusterProviderManifestNamespaceUsesMakeRemoteRefKey(t *testing.T) {
	f := &framework.Framework{
		Namespace: &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "test-ns"},
		},
		MakeRemoteRefKey: func(base string) string { return "scoped-" + base },
	}
	tc := &framework.TestCase{
		Framework:        f,
		ExternalSecret:   &esv1.ExternalSecret{},
		ExpectedSecret:   &corev1.Secret{},
		PushSecret:       &esv1alpha1.PushSecret{},
		PushSecretSource: &corev1.Secret{},
	}

	_, apply := ClusterProviderManifestNamespace(f, ClusterProviderExternalSecretHarness{
		Prepare: func(_ *framework.TestCase, _ ClusterProviderConfig) *ClusterProviderExternalSecretRuntime {
			return &ClusterProviderExternalSecretRuntime{ClusterProviderName: "cluster-provider"}
		},
	})
	apply(tc)

	if _, ok := tc.Secrets["scoped-manifest-source"]; !ok {
		t.Fatalf("expected cluster provider sync case to use MakeRemoteRefKey, got %v", tc.Secrets)
	}
	if got := tc.ExternalSecret.Spec.Data[0].RemoteRef.Key; got != "scoped-manifest-source" {
		t.Fatalf("expected remote ref key %q, got %q", "scoped-manifest-source", got)
	}
}

func TestClusterProviderDeniedByConditionsUsesMakeRemoteRefKey(t *testing.T) {
	f := &framework.Framework{
		Namespace: &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "test-ns"},
		},
		MakeRemoteRefKey: func(base string) string { return "scoped-" + base },
	}
	tc := &framework.TestCase{
		Framework:      f,
		ExternalSecret: &esv1.ExternalSecret{},
	}

	_, apply := ClusterProviderDeniedByConditions(f, ClusterProviderExternalSecretHarness{})
	apply(tc)

	if _, ok := tc.Secrets["scoped-denied-source"]; !ok {
		t.Fatalf("expected cluster provider deny case to use MakeRemoteRefKey, got %v", tc.Secrets)
	}
	if got := tc.ExternalSecret.Spec.Data[0].RemoteRef.Key; got != "scoped-denied-source" {
		t.Fatalf("expected remote ref key %q, got %q", "scoped-denied-source", got)
	}
}
