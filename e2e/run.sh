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
set -o errexit
set -o nounset
set -o pipefail

if ! command -v kind --version &> /dev/null; then
  echo "kind is not installed. Use the package manager or visit the official site https://kind.sigs.k8s.io/"
  exit 1
fi

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd $DIR

echo "Kubernetes cluster:"
kubectl get nodes -o wide

echo -e "Granting permissions to e2e service account..."
kubectl create serviceaccount external-secrets-e2e || true
kubectl create clusterrolebinding permissive-binding \
  --clusterrole=cluster-admin \
  --user=admin \
  --user=kubelet \
  --serviceaccount=default:external-secrets-e2e || true

echo -e "Waiting service account..."; \
until kubectl get secret | grep -q -e ^external-secrets-e2e-token; do \
  echo -e "waiting for api token"; \
  sleep 3; \
done

kubectl apply -f ${DIR}/k8s/deploy/crds

echo -e "Starting the e2e test pod"
FOCUS=${FOCUS:-.*}
export FOCUS

kubectl run --rm \
  --attach \
  --restart=Never \
  --env="FOCUS=${FOCUS}" \
  --env="GCP_SM_SA_JSON=${GCP_SM_SA_JSON}" \
  --env="AZURE_CLIENT_ID=${AZURE_CLIENT_ID}" \
  --env="AZURE_CLIENT_SECRET=${AZURE_CLIENT_SECRET}" \
  --env="TENANT_ID=${TENANT_ID}" \
  --env="VAULT_URL=${VAULT_URL}" \
  --overrides='{ "apiVersion": "v1", "spec":{"serviceAccountName": "external-secrets-e2e"}}' \
  e2e --image=local/external-secrets-e2e:test
