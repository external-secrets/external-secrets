#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
BUNDLE_DIR="${1}"
HELM_DIR="${2}"

if [[ "$OSTYPE" == "darwin"* ]]; then
  SEDPRG="gsed"
else
  SEDPRG="sed"
fi

cd "${SCRIPT_DIR}"/../

# Split the generated bundle yaml file to inject control flags
yq e -Ns "\"${HELM_DIR}/templates/crds/\" + .spec.names.singular" ${BUNDLE_DIR}/bundle.yaml

# Add helm if statement for controlling the install of CRDs
for i in "${HELM_DIR}"/templates/crds/*.yml; do
  export CRDS_FLAG_NAME="create$(yq e '.spec.names.kind' $i)"
  cp "$i" "$i.bkp"
  if [[ "$CRDS_FLAG_NAME" == *"Cluster"* ]]; then
    echo "{{- if and (.Values.installCRDs) (.Values.crds.$CRDS_FLAG_NAME) }}" > "$i"
  elif [[ "$CRDS_FLAG_NAME" == *"PushSecret"* ]]; then
			echo "{{- if and (.Values.installCRDs) (.Values.crds.$CRDS_FLAG_NAME) }}" > "$i"
  else
    echo "{{- if .Values.installCRDs }}" > "$i"
  fi
  cat "$i.bkp" >> "$i"
  echo "{{- end }}" >> "$i"
  rm "$i.bkp"
  $SEDPRG -i 's/name: kubernetes/name: {{ include "external-secrets.fullname" . }}-webhook/g' "$i"
  $SEDPRG -i 's/namespace: default/namespace: {{ .Release.Namespace | quote }}/g' "$i"
  $SEDPRG -i '0,/annotations/!b;//a\    {{- with .Values.crds.annotations }}\n    {{- toYaml . | nindent 4}}\n    {{- end }}\n    {{- if and .Values.crds.conversion.enabled .Values.webhook.certManager.enabled .Values.webhook.certManager.addInjectorAnnotations }}\n    cert-manager.io/inject-ca-from: {{ .Release.Namespace }}/{{ include "external-secrets.fullname" . }}-webhook\n    {{- end }}' "$i"

  $SEDPRG -i '/  conversion:/i{{- if .Values.crds.conversion.enabled }}' "$i"
  echo "{{- end }}" >> "$i"
  mv "$i" "${i%.yml}.yaml"
done