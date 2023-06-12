#!/usr/bin/sh
set -euo pipefail
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

rm -rf $DIR/generated/*

for providerdir in $(find $DIR/../../pkg/provider -maxdepth 1 -mindepth 1 | grep -vE "(util|testing|register)"); do
    provider=$(basename $providerdir)

    cd $providerdir
    if [ ! -f "go.mod" ]; then
        go mod init "github.com/external-secrets/external-secrets-provider-${provider}"
        go mod tidy
    fi
    cd $DIR

    # TODO:
    if [ "${provider}" == "yandex" ]; then
        continue;
    fi

    # grpc should not be generated
    if [ "${provider}" == "grpc" ]; then
        continue;
    fi

    pkgname=$provider
    # override import path, because provider directory structure is not standardised
    if [ "${provider}" == "gcp" ]; then
        pkgname="gcp/secretmanager"
    fi
    if [ "${provider}" == "azure" ]; then
        pkgname="azure/keyvault"
    fi

    echo "generating provider $provider from $providerdir"
    mkdir -p "${DIR}/generated/${provider}"
    sed "s\PROVIDER_NAME\\$pkgname\g" $DIR/provider.go.tmpl > "$DIR/generated/${provider}/main.go"
    cd $DIR/generated/${provider}/
    go mod init github.com/external-secrets/external-secrets-provider-$provider >/dev/null 2>&1
    sed -i "2i replace github.com/external-secrets/external-secrets/pkg/provider/${provider} => ../../../../pkg/provider/${provider}" go.mod
    cd $DIR
    go mod download
done
