path "secret/*" {
  capabilities = ["read"]
}

path "my-namespace/kv/*" {
  capabilities = ["read"]
}
