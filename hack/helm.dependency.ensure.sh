#!/usr/bin/env bash

set -euo pipefail

chart_dir="${1:?usage: helm.dependency.ensure.sh <chart-dir>}"
helm_bin="${HELM_BIN:-helm}"

if "${helm_bin}" dependency list "${chart_dir}" 2>/dev/null | awk 'NR == 1 { next } NF && $NF != "ok" && $NF != "unpacked" { status = 1 } END { exit status }'; then
  echo "Helm dependencies already present for ${chart_dir}; skipping dependency build"
  exit 0
fi

"${helm_bin}" dependency build "${chart_dir}"
