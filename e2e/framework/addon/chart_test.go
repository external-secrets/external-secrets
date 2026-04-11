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
