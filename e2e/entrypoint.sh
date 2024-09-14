#!/bin/bash

# Copyright External Secrets Inc. All Rights Reserved

set -euo pipefail

NC='\e[0m'
BGREEN='\e[32m'

E2E_NODES=${E2E_NODES:-5}

if [ ! -f "${HOME}/.kube/config" ]; then
  kubectl config set-cluster dev --certificate-authority=/var/run/secrets/kubernetes.io/serviceaccount/ca.crt --embed-certs=true --server="https://kubernetes.default/"
  kubectl config set-credentials user --token="$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)"
  kubectl config set-context default --cluster=dev --user=user
  kubectl config use-context default
fi

ginkgo_args=(
  "--randomize-suites"
  "--randomize-all"
  "--flake-attempts=2"
  "-p"
  "-trace"
  "-r"
  "-v"
  "-timeout=45m"
)

for SUITE in ${TEST_SUITES}; do
  echo -e "${BGREEN}Running suite ${SUITE} (LABELS=${GINKGO_LABELS})...${NC}"
  ACK_GINKGO_RC=true ginkgo "${ginkgo_args[@]}" \
    -label-filter="${GINKGO_LABELS}"            \
    -nodes="${E2E_NODES}"                       \
    /${SUITE}.test
done

