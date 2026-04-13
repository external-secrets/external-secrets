package hack

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestHelmDependencyEnsureSkipsBuildWhenDependenciesAreReady(t *testing.T) {
	t.Parallel()

	workdir := t.TempDir()
	marker := filepath.Join(workdir, "build-called")
	fakeHelm := writeFakeHelm(t, workdir, "#!/usr/bin/env bash\nset -euo pipefail\n\nif [[ \"$1\" == \"dependency\" && \"$2\" == \"list\" ]]; then\n  cat <<'EOF'\nNAME                \tVERSION\tREPOSITORY                           \tSTATUS\nbitwarden-sdk-server\tv0.5.2 \toci://ghcr.io/external-secrets/charts\tok\nEOF\n  exit 0\nfi\n\nif [[ \"$1\" == \"dependency\" && \"$2\" == \"build\" ]]; then\n  touch \""+marker+"\"\n  exit 0\nfi\n\necho \"unexpected helm invocation: $*\" >&2\nexit 1\n")

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
	fakeHelm := writeFakeHelm(t, workdir, "#!/usr/bin/env bash\nset -euo pipefail\n\nif [[ \"$1\" == \"dependency\" && \"$2\" == \"list\" ]]; then\n  cat <<'EOF'\nNAME                \tVERSION\tREPOSITORY                           \tSTATUS\nbitwarden-sdk-server\tv0.6.0 \toci://ghcr.io/external-secrets/charts\tunpacked\nEOF\n  exit 0\nfi\n\nif [[ \"$1\" == \"dependency\" && \"$2\" == \"build\" ]]; then\n  touch \""+marker+"\"\n  exit 0\nfi\n\necho \"unexpected helm invocation: $*\" >&2\nexit 1\n")

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
	fakeHelm := writeFakeHelm(t, workdir, "#!/usr/bin/env bash\nset -euo pipefail\n\nif [[ \"$1\" == \"dependency\" && \"$2\" == \"list\" ]]; then\n  cat <<'EOF'\nNAME                \tVERSION\tREPOSITORY                           \tSTATUS\nbitwarden-sdk-server\tv0.5.2 \toci://ghcr.io/external-secrets/charts\tmissing\nEOF\n  exit 0\nfi\n\nif [[ \"$1\" == \"dependency\" && \"$2\" == \"build\" ]]; then\n  touch \""+marker+"\"\n  exit 0\nfi\n\necho \"unexpected helm invocation: $*\" >&2\nexit 1\n")

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
