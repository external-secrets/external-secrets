package e2e

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestV2MakeTargetCanSkipKubernetesProviderBuild(t *testing.T) {
	t.Parallel()

	defaultDryRun := runMakeDryRun(t, "VERSION=test-version")
	if !strings.Contains(defaultDryRun, "docker.build.provider.kubernetes") {
		t.Fatalf("expected default test.v2 dry-run to build the kubernetes provider image, output:\n%s", defaultDryRun)
	}
	if !strings.Contains(defaultDryRun, "ghcr.io/external-secrets/provider-kubernetes:test-version") {
		t.Fatalf("expected default test.v2 dry-run to still load the kubernetes provider image, output:\n%s", defaultDryRun)
	}

	skippedDryRun := runMakeDryRun(t, "VERSION=test-version", "SKIP_PROVIDER_KUBERNETES_BUILD=true")
	if strings.Contains(skippedDryRun, "docker.build.provider.kubernetes") {
		t.Fatalf("expected skipped test.v2 dry-run to omit the kubernetes provider build, output:\n%s", skippedDryRun)
	}
	if !strings.Contains(skippedDryRun, "ghcr.io/external-secrets/provider-kubernetes:test-version") {
		t.Fatalf("expected skipped test.v2 dry-run to still load the kubernetes provider image, output:\n%s", skippedDryRun)
	}
}

func runMakeDryRun(t *testing.T, extraArgs ...string) string {
	t.Helper()

	args := append([]string{"-n", "test.v2"}, extraArgs...)
	cmd := exec.Command("make", args...)
	cmd.Dir = "."
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make dry-run failed: %v\n%s", err, string(output))
	}

	return string(output)
}
