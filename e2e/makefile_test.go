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

const (
	testVersionArg           = "VERSION=test-version"
	kubernetesBuildTarget    = "docker.build.provider.kubernetes"
	gcpBuildTarget           = "docker.build.provider.gcp"
	kubernetesProviderImage  = "ghcr.io/external-secrets/provider-kubernetes:test-version"
	gcpProviderImage         = "ghcr.io/external-secrets/provider-gcp:test-version"
	gcpProviderImageLoad     = `kind load docker-image --name="external-secrets" ghcr.io/external-secrets/provider-gcp:test-version`
	helmDependencyEnsureCmd  = "../hack/helm.dependency.ensure.sh ../deploy/charts/external-secrets"
	controllerImageLoadCount = `kind load docker-image --name="external-secrets" ghcr.io/external-secrets/external-secrets:test-version`
	controllerImageBuildCmd  = "docker.build.controller.e2e"
	dockerCleanupCmd         = "docker system prune --all --force --volumes"
)

func TestClassicMakeTargetBuildsOnlyControllerImageOnce(t *testing.T) {
	t.Parallel()

	dryRun := runMakeDryRun(t, "test", testVersionArg, `TEST_SUITES=provider`, `GINKGO_LABELS=kubernetes && !v2`)

	if !strings.Contains(dryRun, controllerImageBuildCmd) {
		t.Fatalf("expected classic test dry-run to build the controller image via docker.build.controller.e2e, output:\n%s", dryRun)
	}
	if strings.Contains(dryRun, kubernetesBuildTarget) {
		t.Fatalf("expected classic test dry-run to omit kubernetes provider image builds, output:\n%s", dryRun)
	}
	if strings.Contains(dryRun, "docker.build.provider.aws") {
		t.Fatalf("expected classic test dry-run to omit aws provider image builds, output:\n%s", dryRun)
	}
	if strings.Contains(dryRun, "docker.build.provider.fake") {
		t.Fatalf("expected classic test dry-run to omit fake provider image builds, output:\n%s", dryRun)
	}
	if strings.Contains(dryRun, gcpBuildTarget) {
		t.Fatalf("expected classic test dry-run to omit gcp provider image builds, output:\n%s", dryRun)
	}
	if count := strings.Count(dryRun, controllerImageLoadCount); count != 1 {
		t.Fatalf("expected classic test dry-run to load the controller image once, got %d occurrences, output:\n%s", count, dryRun)
	}
	if strings.Contains(dryRun, kubernetesProviderImage) {
		t.Fatalf("expected classic test dry-run to avoid loading the kubernetes provider image, output:\n%s", dryRun)
	}
	if strings.Contains(dryRun, gcpProviderImage) {
		t.Fatalf("expected classic test dry-run to avoid loading the gcp provider image, output:\n%s", dryRun)
	}
	if !strings.Contains(dryRun, helmDependencyEnsureCmd) {
		t.Fatalf("expected classic test dry-run to ensure helm dependencies before copying the chart, output:\n%s", dryRun)
	}
}

func TestV2MakeTargetCanSkipKubernetesProviderBuild(t *testing.T) {
	t.Parallel()

	defaultDryRun := runMakeDryRun(t, "test.v2", testVersionArg)
	if !strings.Contains(defaultDryRun, kubernetesBuildTarget) {
		t.Fatalf("expected default test.v2 dry-run to build the kubernetes provider image, output:\n%s", defaultDryRun)
	}
	if count := strings.Count(defaultDryRun, controllerImageBuildCmd); count != 1 {
		t.Fatalf("expected default test.v2 dry-run to build the controller image once, got %d occurrences, output:\n%s", count, defaultDryRun)
	}
	if !strings.Contains(defaultDryRun, "docker.build.provider.aws") {
		t.Fatalf("expected default test.v2 dry-run to build the aws provider image, output:\n%s", defaultDryRun)
	}
	if !strings.Contains(defaultDryRun, "docker.build.provider.fake") {
		t.Fatalf("expected default test.v2 dry-run to build the fake provider image, output:\n%s", defaultDryRun)
	}
	if count := strings.Count(defaultDryRun, gcpBuildTarget); count != 1 {
		t.Fatalf("expected default test.v2 dry-run to build the gcp provider image once, got %d occurrences, output:\n%s", count, defaultDryRun)
	}
	if count := strings.Count(defaultDryRun, controllerImageLoadCount); count != 1 {
		t.Fatalf("expected default test.v2 dry-run to load the controller image once, got %d occurrences, output:\n%s", count, defaultDryRun)
	}
	if !strings.Contains(defaultDryRun, kubernetesProviderImage) {
		t.Fatalf("expected default test.v2 dry-run to still load the kubernetes provider image, output:\n%s", defaultDryRun)
	}
	if !strings.Contains(defaultDryRun, "ghcr.io/external-secrets/provider-aws:test-version") {
		t.Fatalf("expected default test.v2 dry-run to load the aws provider image, output:\n%s", defaultDryRun)
	}
	if !strings.Contains(defaultDryRun, "ghcr.io/external-secrets/provider-fake:test-version") {
		t.Fatalf("expected default test.v2 dry-run to load the fake provider image, output:\n%s", defaultDryRun)
	}
	if !strings.Contains(defaultDryRun, gcpProviderImage) {
		t.Fatalf("expected default test.v2 dry-run to load the gcp provider image, output:\n%s", defaultDryRun)
	}
	if count := strings.Count(defaultDryRun, gcpProviderImageLoad); count != 1 {
		t.Fatalf("expected default test.v2 dry-run to load the gcp provider image once, got %d occurrences, output:\n%s", count, defaultDryRun)
	}
	if !strings.Contains(defaultDryRun, helmDependencyEnsureCmd) {
		t.Fatalf("expected default test.v2 dry-run to ensure helm dependencies before copying the chart, output:\n%s", defaultDryRun)
	}
	if !strings.Contains(defaultDryRun, `TEST_SUITES="provider"`) {
		t.Fatalf("expected default test.v2 dry-run to run the provider suite, output:\n%s", defaultDryRun)
	}
	if strings.Contains(defaultDryRun, dockerCleanupCmd) {
		t.Fatalf("expected default test.v2 dry-run to avoid CI-only docker cleanup, output:\n%s", defaultDryRun)
	}

	skippedDryRun := runMakeDryRun(t, "test.v2", testVersionArg, "SKIP_PROVIDER_KUBERNETES_BUILD=true")
	if strings.Contains(skippedDryRun, kubernetesBuildTarget) {
		t.Fatalf("expected skipped test.v2 dry-run to omit the kubernetes provider build, output:\n%s", skippedDryRun)
	}
	if count := strings.Count(skippedDryRun, controllerImageBuildCmd); count != 1 {
		t.Fatalf("expected skipped test.v2 dry-run to build the controller image once, got %d occurrences, output:\n%s", count, skippedDryRun)
	}
	if !strings.Contains(skippedDryRun, "docker.build.provider.fake") {
		t.Fatalf("expected skipped test.v2 dry-run to still build the fake provider image, output:\n%s", skippedDryRun)
	}
	if count := strings.Count(skippedDryRun, gcpBuildTarget); count != 1 {
		t.Fatalf("expected skipped test.v2 dry-run to still build the gcp provider image once, got %d occurrences, output:\n%s", count, skippedDryRun)
	}
	if count := strings.Count(skippedDryRun, controllerImageLoadCount); count != 1 {
		t.Fatalf("expected skipped test.v2 dry-run to load the controller image once, got %d occurrences, output:\n%s", count, skippedDryRun)
	}
	if !strings.Contains(skippedDryRun, kubernetesProviderImage) {
		t.Fatalf("expected skipped test.v2 dry-run to still load the kubernetes provider image, output:\n%s", skippedDryRun)
	}
	if !strings.Contains(skippedDryRun, "ghcr.io/external-secrets/provider-aws:test-version") {
		t.Fatalf("expected skipped test.v2 dry-run to still load the aws provider image, output:\n%s", skippedDryRun)
	}
	if !strings.Contains(skippedDryRun, "ghcr.io/external-secrets/provider-fake:test-version") {
		t.Fatalf("expected skipped test.v2 dry-run to still load the fake provider image, output:\n%s", skippedDryRun)
	}
	if !strings.Contains(skippedDryRun, gcpProviderImage) {
		t.Fatalf("expected skipped test.v2 dry-run to still load the gcp provider image, output:\n%s", skippedDryRun)
	}
	if count := strings.Count(skippedDryRun, gcpProviderImageLoad); count != 1 {
		t.Fatalf("expected skipped test.v2 dry-run to still load the gcp provider image once, got %d occurrences, output:\n%s", count, skippedDryRun)
	}
	if !strings.Contains(skippedDryRun, helmDependencyEnsureCmd) {
		t.Fatalf("expected skipped test.v2 dry-run to ensure helm dependencies before copying the chart, output:\n%s", skippedDryRun)
	}
	if !strings.Contains(skippedDryRun, `TEST_SUITES="provider"`) {
		t.Fatalf("expected skipped test.v2 dry-run to still run the provider suite, output:\n%s", skippedDryRun)
	}
}

func TestV2MakeTargetPrunesDockerImagesInCI(t *testing.T) {
	t.Parallel()

	dryRun := runMakeDryRunWithEnv(t, []string{"CI=true"}, "test.v2", testVersionArg)
	if count := strings.Count(dryRun, dockerCleanupCmd); count != 1 {
		t.Fatalf("expected CI test.v2 dry-run to prune docker state once, got %d occurrences, output:\n%s", count, dryRun)
	}
}

func TestV2OperationalMakeTarget(t *testing.T) {
	t.Parallel()

	dryRun := runMakeDryRun(t, "test.v2.operational", testVersionArg)
	if !strings.Contains(dryRun, "V2_GINKGO_LABELS=") || !strings.Contains(dryRun, "v2 && operational && fake") {
		t.Fatalf("expected operational labels in target, got:\n%s", dryRun)
	}
	if !strings.Contains(dryRun, "V2_TEST_SUITES=") || !strings.Contains(dryRun, "provider") {
		t.Fatalf("expected provider suite in target, got:\n%s", dryRun)
	}
	if !strings.Contains(dryRun, `TEST_SUITES="provider"`) {
		t.Fatalf("expected operational target to render provider-only v2 suites, got:\n%s", dryRun)
	}
}

func runMakeDryRun(t *testing.T, target string, extraArgs ...string) string {
	t.Helper()
	return runMakeDryRunWithEnv(t, nil, target, extraArgs...)
}

func runMakeDryRunWithEnv(t *testing.T, extraEnv []string, target string, extraArgs ...string) string {
	t.Helper()

	args := append([]string{"-n", target}, extraArgs...)
	cmd := exec.Command("make", args...)
	cmd.Dir = "."
	cmd.Env = append(os.Environ(), extraEnv...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make dry-run failed: %v\n%s", err, string(output))
	}

	return string(output)
}
