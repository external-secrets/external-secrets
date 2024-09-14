#!/bin/bash

# Copyright External Secrets Inc. All Rights Reserved

set -o errexit
set -o nounset
set -o pipefail

GOPATH=${GOPATH:-$(go env GOPATH)}

# "go env" doesn't print anything if GOBIN is the default, so we
# have to manually default it.
GOBIN=${GOBIN:-$(go env GOBIN)}
GOBIN=${GOBIN:-${GOPATH}/bin}

readonly HERE=$(cd $(dirname $0) && pwd)
readonly REPO=$(cd ${HERE}/../.. && pwd)

gendoc::build() {
    go install github.com/ahmetb/gen-crd-api-reference-docs
}

# Exec the doc generator.
gendoc::exec() {
    local readonly confdir="${REPO}/hack/api-docs"

    ${GOBIN}/gen-crd-api-reference-docs \
        -template-dir ${confdir} \
        -config ${confdir}/config.json \
        "$@"
}

if [ "$#" != "1" ]; then
    echo "usage: generate.sh OUTFILE"
    exit 2
fi

gendoc::build
gendoc::exec \
    -api-dir github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1 \
    -out-file "$1"
