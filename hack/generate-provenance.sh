#!/usr/bin/env bash
# generate-provenance.sh - Generate SLSA Provenance v0.2 in-toto statement for container images.
# Drop-in replacement for philips-labs/slsa-provenance-action (container subcommand).
#
# Required environment variables (set automatically in GitHub Actions):
#   GITHUB_REPOSITORY    - owner/repo
#   GITHUB_SHA           - commit SHA
#   GITHUB_RUN_ID        - workflow run ID
#   GITHUB_WORKFLOW      - workflow name
#
# Usage:
#   ./hack/generate-provenance.sh \
#     --repository <image-repo> \
#     --digest <sha256:...> \
#     --tags <tag> \
#     --output-path <output.intoto.jsonl>

set -euo pipefail

REPOSITORY=""
DIGEST=""
TAGS=""
OUTPUT_PATH=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --repository) REPOSITORY="$2"; shift 2 ;;
    --digest)     DIGEST="$2"; shift 2 ;;
    --tags)       TAGS="$2"; shift 2 ;;
    --output-path) OUTPUT_PATH="$2"; shift 2 ;;
    *) echo "Unknown argument: $1" >&2; exit 1 ;;
  esac
done

if [[ -z "$REPOSITORY" || -z "$DIGEST" || -z "$OUTPUT_PATH" ]]; then
  echo "Error: --repository, --digest, and --output-path are required" >&2
  exit 1
fi

# Strip the sha256: prefix for the digest value
DIGEST_VALUE="${DIGEST#sha256:}"

REPO_URL="https://github.com/${GITHUB_REPOSITORY}"
BUILD_INVOCATION_ID="${REPO_URL}/actions/runs/${GITHUB_RUN_ID}"
BUILD_FINISHED_ON="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

# Build subject name: repo:tag if tags provided, otherwise repo
if [[ -n "$TAGS" ]]; then
  SUBJECT_NAME="${REPOSITORY}:${TAGS}"
else
  SUBJECT_NAME="${REPOSITORY}"
fi

jq -n \
  --arg type "https://in-toto.io/Statement/v0.1" \
  --arg predicateType "https://slsa.dev/provenance/v0.2" \
  --arg subjectName "$SUBJECT_NAME" \
  --arg digestValue "$DIGEST_VALUE" \
  --arg builderId "${REPO_URL}/Attestations/GitHubHostedActions@v1" \
  --arg buildType "https://github.com/Attestations/GitHubActionsWorkflow@v1" \
  --arg entryPoint "${GITHUB_WORKFLOW:-}" \
  --arg buildInvocationId "$BUILD_INVOCATION_ID" \
  --arg buildFinishedOn "$BUILD_FINISHED_ON" \
  --arg materialUri "git+${REPO_URL}" \
  --arg materialSha "${GITHUB_SHA}" \
  '{
    "_type": $type,
    "subject": [
      {
        "name": $subjectName,
        "digest": {
          "sha256": $digestValue
        }
      }
    ],
    "predicateType": $predicateType,
    "predicate": {
      "builder": {
        "id": $builderId
      },
      "buildType": $buildType,
      "invocation": {
        "configSource": {
          "entryPoint": $entryPoint
        },
        "parameters": null,
        "environment": null
      },
      "metadata": {
        "buildInvocationId": $buildInvocationId,
        "buildFinishedOn": $buildFinishedOn,
        "completeness": {
          "parameters": false,
          "environment": false,
          "materials": false
        },
        "reproducible": false
      },
      "materials": [
        {
          "uri": $materialUri,
          "digest": {
            "sha1": $materialSha
          }
        }
      ]
    }
  }' > "$OUTPUT_PATH"

echo "Provenance saved to ${OUTPUT_PATH}"
