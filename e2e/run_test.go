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
