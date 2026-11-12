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

Before disabling cluster stores, migrate or remove affected `PushSecret`s while access is still available, respecting their `deletionPolicy`.

If a `PushSecret` is already stuck deleting with `deletionPolicy: Delete`, **clean up its remote secrets before removing status entries or spec references**. An administrator must do this manually, or temporarily re-enable cluster-store processing to let the operator perform cleanup. With `deletionPolicy: None`, provider secrets are retained and manual cleanup is left to the deployer. Follow the finalizer instructions below after any required cleanup.

To resume a `PushSecret` that is **not deleting** after disabling cluster stores, **first clean up or migrate remote secrets in the affected cluster stores**, then remove **only the status entries for those stores** from `status.syncedPushSecrets`. Removing these entries makes the operator forget those remote secrets, preventing automatic cleanup on later deletion.

For example, to remove the status entry for `ClusterSecretStore/shared`:
```bash
kubectl patch pushsecret <name> -n <namespace> --subresource=status --type=json \
  -p='[{"op":"remove","path":"/status/syncedPushSecrets/ClusterSecretStore~1shared"}]'
```

These entries actively prevent reconciliation of the PushSecret if ClusterSecretStore is disabled.
When this is done, remove outdated cluster-store refs from `spec.secretStoreRefs`. For each affected `spec.dataTo` item, remove the item or replace its `storeRef` with a valid namespaced `SecretStore` reference. If the replacement names a store, it must also appear in `spec.secretStoreRefs`.

If removing cluster-store references leaves no target, configure a namespaced `SecretStore` target in `spec.secretStoreRefs`, or remove the `PushSecret` if it is no longer needed. An empty target list does not resume processing.

If the spec already references only namespaced stores and no spec edit is needed, change an annotation after the status patch to trigger reconciliation; status-only updates are ignored:

```bash
kubectl annotate pushsecret <name> -n <namespace> force-sync=$(date +%s) --overwrite
```

For a `PushSecret` that is stuck deleting, after completing any cleanup required by its `deletionPolicy`, inspect the finalizers and remove **only** the PushSecret finalizer to allow Kubernetes deletion:

```bash
kubectl get pushsecret <name> -n <namespace> -o jsonpath='{.metadata.finalizers}' # Show all finalizers
```

If `pushsecret.externalsecrets.io/finalizer` is absent, there is no PushSecret finalizer to remove. Otherwise, use its zero-based index in both `path` values below (the example assumes index `0`); do not remove other controllers' finalizers.

```bash
kubectl patch pushsecret <name> -n <namespace> --type=json \
  -p='[{"op":"test","path":"/metadata/finalizers/0","value":"pushsecret.externalsecrets.io/finalizer"},{"op":"remove","path":"/metadata/finalizers/0"}]'
kubectl get pushsecret <name> -n <namespace> -o jsonpath='{.metadata.finalizers}' # Show remaining finalizers
```

If `get` returns `NotFound`, the PushSecret has been deleted successfully. If the PushSecret finalizer remains, inspect the patch error (for example, permissions or admission webhook failures) before retrying. If its position changed, find its current index with `get` and update both `path` values.
