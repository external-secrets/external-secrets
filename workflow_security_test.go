/*
Copyright © The ESO Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

const managedE2EWorkflowPath = ".github/workflows/e2e-managed.yml"

var nextWorkflowJob = regexp.MustCompile(`(?m)^  [A-Za-z0-9_-]+:\s*$`)

func TestManagedE2EWorkflowPinsApprovedHead(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(managedE2EWorkflowPath)
	if err != nil {
		t.Fatalf("read %s: %v", managedE2EWorkflowPath, err)
	}
	workflow := string(data)

	for _, forbidden := range []string{"refs/pull/", "${{ github.sha }}"} {
		if strings.Contains(workflow, forbidden) {
			t.Errorf("managed E2E workflow contains mutable identity %q", forbidden)
		}
	}

	setup := workflowJobBlock(t, workflow, "setup")
	for _, required := range []string{
		"approved_repository: ${{ steps.approval.outputs.approved_repository }}",
		"approved_sha: ${{ steps.approval.outputs.approved_sha }}",
		"github.rest.pulls.get",
		"if (!/^[0-9a-f]{40}$/i.test(approvedSha))",
		"liveHeadRepository.toLowerCase() !== approvedRepository.toLowerCase()",
		"pull.head.sha.toLowerCase() !== approvedSha.toLowerCase()",
		"const ref = approvedSha",
	} {
		if !strings.Contains(setup, required) {
			t.Errorf("setup job is missing approved-head contract %q", required)
		}
	}

	for _, job := range []string{"test-aws", "test-gcp", "test-azure"} {
		block := workflowJobBlock(t, workflow, job)
		for _, required := range []string{
			"needs: [setup]",
			"repository: ${{ needs.setup.outputs.approved_repository }}",
			"ref: ${{ needs.setup.outputs.approved_sha }}",
			"persist-credentials: false",
		} {
			if !strings.Contains(block, required) {
				t.Errorf("%s job is missing approved-head contract %q", job, required)
			}
		}
		if got := strings.Count(block, "uses: actions/checkout@"); got != 1 {
			t.Errorf("%s job has %d checkout steps, want 1", job, got)
		}
		if got := strings.Count(block, "${{ needs.setup.outputs.approved_sha }}"); got < 5 {
			t.Errorf("%s job uses approved SHA %d times, want at least 5 for checkout and caches", job, got)
		}
	}
}

func workflowJobBlock(t *testing.T, workflow, job string) string {
	t.Helper()

	marker := "  " + job + ":\n"
	start := strings.Index(workflow, marker)
	if start < 0 {
		t.Fatalf("workflow job %q not found", job)
	}
	bodyStart := start + len(marker)
	if next := nextWorkflowJob.FindStringIndex(workflow[bodyStart:]); next != nil {
		return workflow[start : bodyStart+next[0]]
	}
	return workflow[start:]
}
