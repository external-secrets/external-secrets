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

package main_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestHelmDocsTargetCanUseLocalCommand(t *testing.T) {
	t.Parallel()

	cmd := exec.Command("make", "-n", "helm.docs", "HELM_DOCS_CMD=helm-docs")
	cmd.Dir = "."
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make dry-run failed: %v\n%s", err, string(output))
	}

	if !strings.Contains(string(output), "cd deploy/charts/external-secrets;") || !strings.Contains(string(output), "helm-docs") {
		t.Fatalf("expected helm.docs dry-run to use HELM_DOCS_CMD override, output:\n%s", string(output))
	}
}

func TestLicenseCheckTargetCanUseLocalCommand(t *testing.T) {
	t.Parallel()

	cmd := exec.Command("make", "-n", "license.check", "LICENSE_CHECK_CMD=license-eye header check")
	cmd.Dir = "."
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make dry-run failed: %v\n%s", err, string(output))
	}

	if !strings.Contains(string(output), "license-eye header check") {
		t.Fatalf("expected license.check dry-run to use LICENSE_CHECK_CMD override, output:\n%s", string(output))
	}
}
