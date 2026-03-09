#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
BUNDLE_DIR="${1}"
OUTPUT_DIR="${2}"

cd "${SCRIPT_DIR}"/../

# Split the generated bundle yaml file
yq e -Ns "\"${OUTPUT_DIR}/\" + .spec.names.singular" "${BUNDLE_DIR}"/bundle.yaml

# Handle singular-name collisions explicitly.
# The generator and provider APIs both define a Fake CRD. Keep the historical
# generator sample at fake.yml, and write the provider sample separately so both
# can be tested.
yq e 'select(.spec.group == "generators.external-secrets.io" and .spec.names.singular == "fake")' \
  "${BUNDLE_DIR}"/bundle.yaml > "${OUTPUT_DIR}/fake.yml"
yq e 'select(.spec.group == "provider.external-secrets.io" and .spec.names.singular == "fake")' \
  "${BUNDLE_DIR}"/bundle.yaml > "${OUTPUT_DIR}/provider-fake.yml"
