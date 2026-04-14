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

func TestV2MakeTargetCanSkipKubernetesProviderBuild(t *testing.T) {
	t.Parallel()

	defaultDryRun := runMakeDryRun(t, "VERSION=test-version")
	if !strings.Contains(defaultDryRun, "docker.build.provider.kubernetes") {
		t.Fatalf("expected default test.v2 dry-run to build the kubernetes provider image, output:\n%s", defaultDryRun)
	}
	if !strings.Contains(defaultDryRun, "docker.build.provider.aws") {
		t.Fatalf("expected default test.v2 dry-run to build the aws provider image, output:\n%s", defaultDryRun)
	}
	if !strings.Contains(defaultDryRun, "docker.build.provider.fake") {
		t.Fatalf("expected default test.v2 dry-run to build the fake provider image, output:\n%s", defaultDryRun)
	}
	if !strings.Contains(defaultDryRun, "ghcr.io/external-secrets/provider-kubernetes:test-version") {
		t.Fatalf("expected default test.v2 dry-run to still load the kubernetes provider image, output:\n%s", defaultDryRun)
	}
	if !strings.Contains(defaultDryRun, "ghcr.io/external-secrets/provider-aws:test-version") {
		t.Fatalf("expected default test.v2 dry-run to load the aws provider image, output:\n%s", defaultDryRun)
	}
	if !strings.Contains(defaultDryRun, "ghcr.io/external-secrets/provider-fake:test-version") {
		t.Fatalf("expected default test.v2 dry-run to load the fake provider image, output:\n%s", defaultDryRun)
	}

	skippedDryRun := runMakeDryRun(t, "VERSION=test-version", "SKIP_PROVIDER_KUBERNETES_BUILD=true")
	if strings.Contains(skippedDryRun, "docker.build.provider.kubernetes") {
		t.Fatalf("expected skipped test.v2 dry-run to omit the kubernetes provider build, output:\n%s", skippedDryRun)
	}
	if !strings.Contains(skippedDryRun, "docker.build.provider.fake") {
		t.Fatalf("expected skipped test.v2 dry-run to still build the fake provider image, output:\n%s", skippedDryRun)
	}
	if !strings.Contains(skippedDryRun, "ghcr.io/external-secrets/provider-kubernetes:test-version") {
		t.Fatalf("expected skipped test.v2 dry-run to still load the kubernetes provider image, output:\n%s", skippedDryRun)
	}
	if !strings.Contains(skippedDryRun, "ghcr.io/external-secrets/provider-aws:test-version") {
		t.Fatalf("expected skipped test.v2 dry-run to still load the aws provider image, output:\n%s", skippedDryRun)
	}
	if !strings.Contains(skippedDryRun, "ghcr.io/external-secrets/provider-fake:test-version") {
		t.Fatalf("expected skipped test.v2 dry-run to still load the fake provider image, output:\n%s", skippedDryRun)
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
