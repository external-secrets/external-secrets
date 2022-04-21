#!/bin/bash

# Copyright 2019 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

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
  "-progress"
  "-trace"
  "--slow-spec-threshold=5m"
  "-r"
  "-v"
  "-timeout=45m"
)

kubectl apply -f /k8s/deploy/crds

for SUITE in ${TEST_SUITES}; do
  echo -e "${BGREEN}Running suite ${SUITE} (LABELS=${GINKGO_LABELS})...${NC}"
  ACK_GINKGO_RC=true ginkgo "${ginkgo_args[@]}" \
    -label-filter="${GINKGO_LABELS}"            \
    -nodes="${E2E_NODES}"                       \
    /${SUITE}.test
done

