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


package e2e

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestClassicMakeTargetBuildsOnlyControllerImageOnce(t *testing.T) {
	t.Parallel()

	dryRun := runMakeDryRun(t, "test", "VERSION=test-version", `TEST_SUITES=provider`, `GINKGO_LABELS=kubernetes && !v2`)

	if !strings.Contains(dryRun, "docker.build.controller.e2e") {
		t.Fatalf("expected classic test dry-run to build the controller image via docker.build.controller.e2e, output:\n%s", dryRun)
	}
	if strings.Contains(dryRun, "docker.build.provider.kubernetes") {
		t.Fatalf("expected classic test dry-run to omit kubernetes provider image builds, output:\n%s", dryRun)
	}
	if strings.Contains(dryRun, "docker.build.provider.aws") {
		t.Fatalf("expected classic test dry-run to omit aws provider image builds, output:\n%s", dryRun)
	}
	if strings.Contains(dryRun, "docker.build.provider.fake") {
		t.Fatalf("expected classic test dry-run to omit fake provider image builds, output:\n%s", dryRun)
	}
	if count := strings.Count(dryRun, `kind load docker-image --name="external-secrets" ghcr.io/external-secrets/external-secrets:test-version`); count != 1 {
		t.Fatalf("expected classic test dry-run to load the controller image once, got %d occurrences, output:\n%s", count, dryRun)
	}
	if strings.Contains(dryRun, "ghcr.io/external-secrets/provider-kubernetes:test-version") {
		t.Fatalf("expected classic test dry-run to avoid loading the kubernetes provider image, output:\n%s", dryRun)
	}
	if !strings.Contains(dryRun, "../hack/helm.dependency.ensure.sh ../deploy/charts/external-secrets") {
		t.Fatalf("expected classic test dry-run to ensure helm dependencies before copying the chart, output:\n%s", dryRun)
	}
}

func TestV2MakeTargetCanSkipKubernetesProviderBuild(t *testing.T) {
	t.Parallel()

	defaultDryRun := runMakeDryRun(t, "test.v2", "VERSION=test-version")
	if !strings.Contains(defaultDryRun, "docker.build.provider.kubernetes") {
		t.Fatalf("expected default test.v2 dry-run to build the kubernetes provider image, output:\n%s", defaultDryRun)
	}
	if !strings.Contains(defaultDryRun, "ghcr.io/external-secrets/provider-kubernetes:test-version") {
		t.Fatalf("expected default test.v2 dry-run to still load the kubernetes provider image, output:\n%s", defaultDryRun)
	}
	if !strings.Contains(defaultDryRun, "../hack/helm.dependency.ensure.sh ../deploy/charts/external-secrets") {
		t.Fatalf("expected default test.v2 dry-run to ensure helm dependencies before copying the chart, output:\n%s", defaultDryRun)
	}

	skippedDryRun := runMakeDryRun(t, "test.v2", "VERSION=test-version", "SKIP_PROVIDER_KUBERNETES_BUILD=true")
	if strings.Contains(skippedDryRun, "docker.build.provider.kubernetes") {
		t.Fatalf("expected skipped test.v2 dry-run to omit the kubernetes provider build, output:\n%s", skippedDryRun)
	}
	if !strings.Contains(skippedDryRun, "ghcr.io/external-secrets/provider-kubernetes:test-version") {
		t.Fatalf("expected skipped test.v2 dry-run to still load the kubernetes provider image, output:\n%s", skippedDryRun)
	}
	if !strings.Contains(skippedDryRun, "../hack/helm.dependency.ensure.sh ../deploy/charts/external-secrets") {
		t.Fatalf("expected skipped test.v2 dry-run to ensure helm dependencies before copying the chart, output:\n%s", skippedDryRun)
	}
}

func runMakeDryRun(t *testing.T, target string, extraArgs ...string) string {
	t.Helper()

	args := append([]string{"-n", target}, extraArgs...)
	cmd := exec.Command("make", args...)
	cmd.Dir = "."
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make dry-run failed: %v\n%s", err, string(output))
	}

	return string(output)
}
