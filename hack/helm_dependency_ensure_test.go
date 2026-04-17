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
	"path/filepath"
	"testing"

	"github.com/external-secrets/external-secrets/pkg/executil"
)

const (
	buildMarkerFile  = "build-called"
	helperScriptPath = "helm.dependency.ensure.sh"
	chartPath        = "deploy/charts/external-secrets"
	helmBinEnvPrefix = "HELM_BIN="
	helperFailedMsg  = "helper failed: %v\n%s"
	scriptPreamble   = "#!/usr/bin/env bash\nset -euo pipefail\n\n"
	scriptFailure    = "echo \"unexpected helm invocation: $*\" >&2\nexit 1\n"
)

func TestHelmDependencyEnsureSkipsBuildWhenDependenciesAreReady(t *testing.T) {
	t.Parallel()

	assertHelperBuildBehavior(t, "v0.5.2", "ok", false, "expected helper to skip helm dependency build when dependencies are already available")
}

func TestHelmDependencyEnsureSkipsBuildWhenDependenciesAreUnpacked(t *testing.T) {
	t.Parallel()

	assertHelperBuildBehavior(t, "v0.6.0", "unpacked", false, "expected helper to skip helm dependency build when dependencies are unpacked locally")
}

func TestHelmDependencyEnsureBuildsWhenDependenciesAreMissing(t *testing.T) {
	t.Parallel()

	assertHelperBuildBehavior(t, "v0.5.2", "missing", true, "expected helper to invoke helm dependency build when dependencies are missing")
}

func assertHelperBuildBehavior(t *testing.T, version, status string, expectBuild bool, failureMsg string) {
	t.Helper()

	workdir := t.TempDir()
	marker := filepath.Join(workdir, buildMarkerFile)
	fakeHelm := writeFakeHelm(t, workdir, fakeHelmScript(version, status, marker))
	output, err := runHelper(t, fakeHelm)
	if err != nil {
		t.Fatalf(helperFailedMsg, err, string(output))
	}

	if expectBuild {
		if _, err := os.Stat(marker); err != nil {
			t.Fatalf("%s: %v", failureMsg, err)
		}
		return
	}

	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal(failureMsg)
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

func runHelper(t *testing.T, fakeHelm string) ([]byte, error) {
	t.Helper()

	cmd, err := executil.Command("bash", helperScriptPath, chartPath)
	if err != nil {
		return nil, err
	}
	cmd.Dir = "."
	cmd.Env = append(os.Environ(), helmBinEnvPrefix+fakeHelm)

	return cmd.CombinedOutput()
}

func fakeHelmScript(version, status, marker string) string {
	return fmt.Sprintf(
		"%s%s%s",
		scriptPreamble,
		fmt.Sprintf(
			`if [[ "$1" == "dependency" && "$2" == "list" ]]; then
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

`,
			version,
			status,
			marker,
		),
		scriptFailure,
	)
}
