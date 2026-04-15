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

package apidocs

import (
	"os"
	"strings"
	"testing"

	"github.com/external-secrets/external-secrets/pkg/executil"
)

func TestImageTargetForwardsDockerBuildArgs(t *testing.T) {
	t.Parallel()

	cmd, err := executil.Command("make", "-n", "image", `DOCKER_BUILD_ARGS=--network host`)
	if err != nil {
		t.Fatalf("resolve make: %v", err)
	}
	cmd.Dir = "."
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make dry-run failed: %v\n%s", err, string(output))
	}

	if !strings.Contains(string(output), "docker build --network host -t github.com/external-secrets-mkdocs:latest -f Dockerfile .") {
		t.Fatalf("expected api docs image dry-run to include DOCKER_BUILD_ARGS, output:\n%s", string(output))
	}
}
