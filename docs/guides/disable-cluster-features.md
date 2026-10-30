# Deploying without ClusterSecretStore and ClusterExternalSecret and ClusterPushSecret

When deploying External Secrets Operator via Helm chart, the default configuration will install `ClusterSecretStore` and `ClusterExternalSecret` and other CRDs and these objects will be processed by the operator.

In order to disable both or one of these features, it is necessary to configure the `crds.*` Helm value, as well as the `process*` Helm value, as these 2 values are connected.

If you would like to install the operator without `ClusterSecretStore` and `ClusterExternalSecret` and `ClusterPushSecret` management, you will have to :

* set `crds.createClusterExternalSecret` to false
* set `crds.createClusterSecretStore` to false
* set `crds.createClusterPushSecret` to false
* set `processClusterExternalSecret` to false
* set `processClusterStore` to false
* set `processClusterPushSecret` to false

Example:

```bash
helm install external-secrets external-secrets/external-secrets --set crds.createClusterExternalSecret=false \
--set crds.createClusterSecretStore=false \
--set crds.createClusterPushSecret=false \
--set processClusterExternalSecret=false \
--set processClusterStore=false \
--set processClusterPushSecret=false
```

When `processClusterStore=false` (or `--enable-cluster-store-reconciler=false`), a `PushSecret` referencing a `ClusterSecretStore` is skipped entirely, even if it also references namespaced `SecretStore`s. This includes `spec.dataTo[].storeRef`, references selected by labels, and cluster stores previously recorded in `status.syncedPushSecrets`. The operator will not push to or delete from those stores; remote secrets already pushed remain in the provider.

Before disabling cluster stores, migrate or remove affected `PushSecret`s and clean up their remote secrets while access is still available. To resume a `PushSecret` after disabling cluster stores, **start by removing the status entries** in `status.syncedPushSecrets`.

For example, to remove the status entry for `ClusterSecretStore/shared`:
```bash
kubectl patch pushsecret <name> -n <namespace> --subresource=status --type=json \
  -p='[{"op":"remove","path":"/status/syncedPushSecrets/ClusterSecretStore~1shared"}]'
```

These entries actively prevent reconciliation of the PushSecret if ClusterSecretStore is disabled.

If a `PushSecret` with `deletionPolicy: Delete` is already stuck deleting, an administrator must clean up the remote secrets manually (or temporarily re-enable cluster-store processing to let the operator clean them up). After confirming cleanup, inspect the finalizers and remove **only** the PushSecret finalizer to allow Kubernetes deletion:

```bash
kubectl get pushsecret <name> -n <namespace> -o jsonpath='{.metadata.finalizers}'
kubectl patch pushsecret <name> -n <namespace> --type=json \
  -p='[{"op":"remove","path":"/metadata/finalizers/0"}]'
```

The patch is appropriate only if `pushsecret.externalsecrets.io/finalizer` is the sole finalizer. If there are others, remove that finalizer by its actual array index instead; do not remove other controllers' finalizers.

When this is done, remove outdated cluster-store refs from `spec.secretStoreRefs` and `spec.dataTo[].storeRef`.

