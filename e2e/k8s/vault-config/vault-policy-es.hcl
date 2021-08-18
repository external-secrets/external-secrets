path "secret/+/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}

path "secret_v1/+/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}
