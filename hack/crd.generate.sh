#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
BUNDLE_DIR="${1}"
CRD_DIR="${2}"
BUNDLE_YAML="${BUNDLE_DIR}/bundle.yaml"

cd "${SCRIPT_DIR}"/../

go run sigs.k8s.io/controller-tools/cmd/controller-gen \
  object:headerFile="hack/boilerplate.go.txt" \
  paths="./apis/..."
go run sigs.k8s.io/controller-tools/cmd/controller-gen crd \
  paths="./apis/..." \
  output:crd:artifacts:config="${CRD_DIR}/bases"

## Update resources list from kustomization.yaml
ls "${CRD_DIR}"/bases | grep -v "kustomization.yaml" | jq -R -s -c 'split("\n")[:-1]' | yq -p=json - > kustomize-files.yaml
yq -i '.resources = (load("kustomize-files.yaml"))' "${CRD_DIR}"/bases/kustomization.yaml
rm kustomize-files.yaml

# Remove extra header lines in generated CRDs
# This is needed for building the helm chart
for f in "${CRD_DIR}"/bases/*.yaml; do
  if [[ $f == *kustomization.yaml ]];
  then
      continue;
  fi;
  tail -n +2 < "$f" > "$f.bkp"
  cp "$f.bkp" "$f"
  rm "$f.bkp"
done

shopt -s extglob
yq e \
    '.spec.conversion.strategy = "Webhook" | .spec.conversion.webhook.conversionReviewVersions = ["v1"] | .spec.conversion.webhook.clientConfig.service.name = "kubernetes" | .spec.conversion.webhook.clientConfig.service.namespace = "default" |	.spec.conversion.webhook.clientConfig.service.path = "/convert"' \
    "${CRD_DIR}"/bases/!(kustomization).yaml > "${BUNDLE_YAML}"
