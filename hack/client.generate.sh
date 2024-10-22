#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

GO_CMD=${1:-go}
PKG_ROOT=$(realpath "$(dirname ${BASH_SOURCE[0]})/..")
CODEGEN_PKG=$($GO_CMD list -m -f "{{.Dir}}" k8s.io/code-generator)

cd $PKG_ROOT

source "${CODEGEN_PKG}/kube_codegen.sh"

kube::codegen::gen_client \
  --output-dir pkg/client \
  --output-pkg github.com/external-secrets/external-secrets/pkg/client \
  --boilerplate hack/boilerplate.go.txt \
  apis

# clean up temporary libraries added in go.mod by code-generator
"${GO_CMD}" mod tidy
