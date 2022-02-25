#!/bin/sh
set -euxo pipefail;

export VAULT_TOKEN=${1}

# ------------------
#   SECRET BACKENDS
# ------------------
vault secrets enable -path=secret -version=2 kv
vault secrets enable -path=secret_v1 -version=1 kv

# ------------------
#   CERT AUTH
#   https://www.vaultproject.io/docs/auth/cert
# ------------------
vault auth enable cert
vault policy write \
    external-secrets-operator \
    /etc/vault-config/vault-policy-es.hcl

vault write auth/cert/certs/external-secrets-operator \
    display_name=external-secrets-operator \
    policies=external-secrets-operator \
    certificate=@/etc/vault-config/es-client.pem \
    ttl=3600

# test certificate login
unset VAULT_TOKEN
vault login \
    -client-cert=/etc/vault-config/es-client.pem \
    -client-key=/etc/vault-config/es-client-key.pem \
    -method=cert \
    name=external-secrets-operator

vault kv put secret/foo/bar baz=bang
vault kv get secret/foo/bar

# ------------------
#   App Role AUTH
#   https://www.vaultproject.io/docs/auth/approle
# ------------------
export VAULT_TOKEN=${1}
vault auth enable -path=myapprole approle

vault write auth/myapprole/role/eso-e2e-role \
    secret_id_ttl=10m \
    token_num_uses=10 \
    token_policies=external-secrets-operator \
    token_ttl=1h \
    token_max_ttl=4h \
    secret_id_num_uses=40

# ------------------
#   JWT AUTH
#   https://www.vaultproject.io/docs/auth/jwt
# ------------------
vault auth enable -path=myjwt jwt

vault write auth/myjwt/config \
   jwt_validation_pubkeys=@/etc/vault-config/jwt-pubkey.pem \
   bound_issuer="example.iss" \
   default_role="external-secrets-operator"

vault write auth/myjwt/role/external-secrets-operator \
    role_type="jwt" \
    bound_subject="vault@example" \
    bound_audiences="vault.client" \
    user_claim="user" \
    policies=external-secrets-operator \
    ttl=1h

vault auth enable -path=myjwtk8s jwt

vault write auth/myjwtk8s/config \
   oidc_discovery_url=https://kubernetes.default.svc.cluster.local \
   oidc_discovery_ca_pem=@/var/run/secrets/kubernetes.io/serviceaccount/ca.crt \
   bound_issuer="https://kubernetes.default.svc.cluster.local" \
   default_role="external-secrets-operator"

vault write auth/myjwtk8s/role/external-secrets-operator \
    role_type="jwt" \
    bound_audiences="vault.client" \
    user_claim="sub" \
    policies=external-secrets-operator \
    ttl=1h

# ------------------
#   Kubernetes AUTH
#   https://www.vaultproject.io/docs/auth/kubernetes
# ------------------
vault auth enable -path=mykubernetes kubernetes
vault write auth/mykubernetes/config \
    kubernetes_host=https://kubernetes.default.svc.cluster.local \
    kubernetes_ca_cert=@/var/run/secrets/kubernetes.io/serviceaccount/ca.crt

vault write auth/mykubernetes/role/external-secrets-operator \
    bound_service_account_names=* \
    bound_service_account_namespaces=* \
    policies=external-secrets-operator \
    ttl=1h
