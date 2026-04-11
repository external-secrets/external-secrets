package e2e

import (
	"os"
	"strings"
	"testing"
)

func TestRunScriptPassesHelmDependencyUpdateSkipEnv(t *testing.T) {
	content, err := os.ReadFile("run.sh")
	if err != nil {
		t.Fatalf("read run.sh: %v", err)
	}

	if !strings.Contains(string(content), `--env="E2E_SKIP_HELM_DEPENDENCY_UPDATE=${E2E_SKIP_HELM_DEPENDENCY_UPDATE:-}"`) {
		t.Fatalf("expected run.sh to pass E2E_SKIP_HELM_DEPENDENCY_UPDATE into the e2e pod")
	}
}
