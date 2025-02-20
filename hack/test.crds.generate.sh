#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
BUNDLE_DIR="${1}"
OUTPUT_DIR="${2}"

cd "${SCRIPT_DIR}"/../

# Split the generated bundle yaml file
yq e -Ns "\"${OUTPUT_DIR}/\" + .spec.names.singular" "${BUNDLE_DIR}"/bundle.yaml
