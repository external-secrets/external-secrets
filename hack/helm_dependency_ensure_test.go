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

package hack

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestHelmDependencyEnsureSkipsBuildWhenDependenciesAreReady(t *testing.T) {
	t.Parallel()

	workdir := t.TempDir()
	marker := filepath.Join(workdir, "build-called")
	fakeHelm := writeFakeHelm(t, workdir, fakeHelmScript("v0.5.2", "ok", marker))

	cmd := exec.Command("bash", "helm.dependency.ensure.sh", "deploy/charts/external-secrets")
	cmd.Dir = "."
	cmd.Env = append(os.Environ(), "HELM_BIN="+fakeHelm)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("helper failed: %v\n%s", err, string(output))
	}

	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("expected helper to skip helm dependency build when dependencies are already available")
	}
}

func TestHelmDependencyEnsureSkipsBuildWhenDependenciesAreUnpacked(t *testing.T) {
	t.Parallel()

	workdir := t.TempDir()
	marker := filepath.Join(workdir, "build-called")
	fakeHelm := writeFakeHelm(t, workdir, fakeHelmScript("v0.6.0", "unpacked", marker))

	cmd := exec.Command("bash", "helm.dependency.ensure.sh", "deploy/charts/external-secrets")
	cmd.Dir = "."
	cmd.Env = append(os.Environ(), "HELM_BIN="+fakeHelm)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("helper failed: %v\n%s", err, string(output))
	}

	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("expected helper to skip helm dependency build when dependencies are unpacked locally")
	}
}

func TestHelmDependencyEnsureBuildsWhenDependenciesAreMissing(t *testing.T) {
	t.Parallel()

	workdir := t.TempDir()
	marker := filepath.Join(workdir, "build-called")
	fakeHelm := writeFakeHelm(t, workdir, fakeHelmScript("v0.5.2", "missing", marker))

	cmd := exec.Command("bash", "helm.dependency.ensure.sh", "deploy/charts/external-secrets")
	cmd.Dir = "."
	cmd.Env = append(os.Environ(), "HELM_BIN="+fakeHelm)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("helper failed: %v\n%s", err, string(output))
	}

	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("expected helper to invoke helm dependency build when dependencies are missing: %v", err)
	}
}

func writeFakeHelm(t *testing.T, dir, content string) string {
	t.Helper()

	path := filepath.Join(dir, "helm")
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write fake helm: %v", err)
	}

	return path
}

func fakeHelmScript(version, status, marker string) string {
	return fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail

if [[ "$1" == "dependency" && "$2" == "list" ]]; then
  cat <<'EOF'
NAME                	VERSION	REPOSITORY                           	STATUS
bitwarden-sdk-server	%s	oci://ghcr.io/external-secrets/charts	%s
EOF
  exit 0
fi

if [[ "$1" == "dependency" && "$2" == "build" ]]; then
  touch %q
  exit 0
fi

echo "unexpected helm invocation: $*" >&2
exit 1
`, version, status, marker)
}
