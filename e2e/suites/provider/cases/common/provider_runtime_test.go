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

import "testing"

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
