![ClusterSecretStore](./pictures/diagrams-high-level-cluster-detail.png)

The `ClusterSecretStore` is a cluster scoped SecretStore that can be used by all
`ExternalSecrets` from all namespaces unless you pin down its usage by using
RBAC or Admission Control.
