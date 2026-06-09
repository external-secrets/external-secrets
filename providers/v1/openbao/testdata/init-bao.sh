#!/usr/bin/env bash

export BAO_TOKEN='root'
export BAO_ADDR='http://localhost:8200'

bao kv put -mount=secret foo bar=old_bazz lorem=old_ipsum
bao kv put -mount=secret foo bar=bazz lorem=ipsum

bao secrets enable -version=1 -path=secret_v1 kv
bao kv put -mount=secret_v1 foo bar=bazz_v1 lorem=ipsum_v1
