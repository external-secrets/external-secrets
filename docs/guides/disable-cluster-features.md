# Deploying without ClusterSecretStore and ClusterExternalSecret

When deploying External Secrets Operator via Helm chart, the default configuration will install `ClusterSecretStore` and `ClusterExternalSecret` CRDs and these objects will be processed by the operator.

In order to disable both or one of these features, it is necessary to configure the `crds.*` Helm value, as well as the `process*` Helm value, as these 2 values are connected.

If you would like to install the operator without `ClusterSecretStore` and `ClusterExternalSecret` management, you will have to :

* set `crds.createClusterExternalSecret` to false
* set `crds.createClusterSecretStore` to false
* set `processClusterExternalSecret` to false
* set `processClusterStore` to false

Example:

```bash
helm install external-secrets external-secrets/external-secrets --set crds.createClusterExternalSecret=false \
--set crds.createClusterSecretStore=false \
--set processClusterExternalSecret=false \
--set processClusterStore=false
```
