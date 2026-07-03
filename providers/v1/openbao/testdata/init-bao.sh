#!/usr/bin/env bash

set -euo pipefail

export BAO_TOKEN='root'
export BAO_ADDR='http://localhost:8200'

# create a few v2 secrets
bao kv put -mount=secret foo bar=old_bazz lorem=old_ipsum
bao kv put -mount=secret foo bar=bazz lorem=ipsum
bao kv metadata put -mount=secret -custom-metadata=bar=meta foo

bao kv put -mount=secret foo2 bar2=bazz2
bao kv metadata put -mount=secret -custom-metadata=bar=meta2 foo2

bao kv put -mount=secret lorem ipsum=dolor
bao kv metadata put -mount=secret -custom-metadata=bar=meta lorem

# create v1 secret
bao secrets enable -version=1 -path=secret_v1 kv
bao kv put -mount=secret_v1 foo bar=bazz_v1 lorem=ipsum_v1

# create userpass auth
bao policy write read-kv testdata/policy-read-kv.hcl
bao auth enable --path=customuserpasspath userpass
bao write auth/customuserpasspath/users/alice password=bob4ever token_policies=read-kv

# create a namespace with a secret
bao namespace create my-namespace
bao secrets enable -version=2 --namespace=my-namespace kv
bao kv put -mount=kv --namespace=my-namespace foo namespaced-bar=namespaced-bazz
