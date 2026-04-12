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


package addon

import "testing"

func TestHelmDependencyUpdateEnabledByDefault(t *testing.T) {
	t.Setenv("E2E_SKIP_HELM_DEPENDENCY_UPDATE", "")

	if !helmDependencyUpdateEnabled() {
		t.Fatalf("expected helm dependency update to be enabled by default")
	}
}

func TestHelmDependencyUpdateCanBeSkipped(t *testing.T) {
	t.Setenv("E2E_SKIP_HELM_DEPENDENCY_UPDATE", "true")

	if helmDependencyUpdateEnabled() {
		t.Fatalf("expected helm dependency update to be disabled when E2E_SKIP_HELM_DEPENDENCY_UPDATE=true")
	}
}

func TestInstallArgsIncludeDependencyUpdateByDefault(t *testing.T) {
	t.Setenv("E2E_SKIP_HELM_DEPENDENCY_UPDATE", "")

	args := (&HelmChart{
		ReleaseName: "eso",
		Chart:       "/tmp/chart",
		Namespace:   "default",
	}).installArgs()

	if !contains(args, "--dependency-update") {
		t.Fatalf("expected install args to include --dependency-update, got %v", args)
	}
}

func TestInstallArgsOmitDependencyUpdateWhenSkipped(t *testing.T) {
	t.Setenv("E2E_SKIP_HELM_DEPENDENCY_UPDATE", "true")

	args := (&HelmChart{
		ReleaseName: "eso",
		Chart:       "/tmp/chart",
		Namespace:   "default",
	}).installArgs()

	if contains(args, "--dependency-update") {
		t.Fatalf("expected install args to omit --dependency-update when E2E_SKIP_HELM_DEPENDENCY_UPDATE=true, got %v", args)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
