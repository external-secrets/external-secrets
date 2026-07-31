#!/usr/bin/env bash

# Copyright 2020 The Kubernetes Authors.
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

readonly HERE=$(cd $(dirname $0) && pwd)
readonly REPO=$(cd ${HERE}/../.. && pwd)
readonly CRD_REF_DOCS=${REPO}/bin/crd-ref-docs

# Exec the doc generator.
gendoc::exec() {
    local readonly confdir="${REPO}/hack/api-docs"

    ${CRD_REF_DOCS} \
        --source-path="${REPO}/apis" \
        --config="${confdir}/config.yaml" \
        --renderer=markdown \
        --output-mode=single \
        --output-path="$1"
    sed -z -i 's/\n*$/\n/' "$1"
}

if [ "$#" != "1" ]; then
    echo "usage: generate.sh OUTFILE"
    exit 2
fi

gendoc::exec "$1"
