# Lifecycle
The External Secrets Operator manages the lifecycle of secrets in Kubernetes. With `refreshPolicy`,   `creationPolicy` and `deletionPolicy` you get fine-grained control of its lifecycle.

!!! note "Creation/Deletion Policy Combinations"
    Some combinations of creationPolicy/deletionPolicy are not allowed as they would delete existing secrets:
    <br/>- `deletionPolicy=Delete` & `creationPolicy=Merge`
    <br/>- `deletionPolicy=Delete` & `creationPolicy=None`
    <br/>- `deletionPolicy=Merge` & `creationPolicy=None`

## Refresh Policy
The field `spec.refreshPolicy` defines how the operator refreshes the a secret.

### Periodic (default) 
Refreshes the secret at a fixed interval via `spec.refreshInterval`. Due to backwards compatibility, setting a refresh interval of 0 will result in the same behavior as `CreatedOnce`.

### OnChange
Refreshes the secret only when the ExternalSecret is updated.  

### CreatedOnce
Refreshes the secret only once, when the ExternalSecret is created.

## Creation Policy
The field `spec.target.creationPolicy` defines how the operator creates the a secret.

### Owner (default)
The External Secret Operator creates secret and sets the `ownerReference` field on the Secret. This secret is subject to [garbage collection](https://kubernetes.io/docs/concepts/architecture/garbage-collection/) if the initial `ExternalSecret` is absent. If a secret with the same name already exists that is not owned by the controller it will result in a conflict. The operator will just error out, not claiming the ownership.

!!! note "Secrets with `ownerReference` field not found"
    If the secret exists and the ownerReference field is not found, the controller treats this secret as orphaned. It will take ownership of this secret by adding an `ownerReference` field and updating it.

### Orphan
Whenever triggered via `RefreshPolicy` conditions, the operator creates/updates 
the target Secret according to the provider available information. 
However, the operator will not watch on Secret Changes (delete/updates), nor trigger 
[garbage collection](https://kubernetes.io/docs/concepts/architecture/garbage-collection/) when the `ExternalSecret` object is deleted.

!!! warning "Unwanted reverts of manual changes"
    If you set `spec.refreshPolicy` to `Periodic` or `OnChange` and `spec.target.creationPolicy` to `Orphan`,
    any changes manually done to the Secret will eventually be replaced on the next sync interval
    or on the next update to `ExternalSecret` object. That manual change is then lost forever.
    Use `creationPolicy=Orphan` with caution.

### Merge
The operator does not create a secret. Instead, it expects the secret to already exist. Values from the secret provider will be merged into the existing secret. Note: the controller takes ownership of a field even if it is owned by a different entity. Multiple ExternalSecrets can use `creationPolicy=Merge` with a single secret as long as the fields don't collide - otherwise you end up in an oscillating state.

### None
The operator does not create or update the secret, this is basically a no-op.

## Deletion Policy
DeletionPolicy defines what should happen if a given secret gets deleted **from the provider**.

DeletionPolicy is only supported on the specific providers, please refer to our [stability/support table](../introduction/stability-support.md).

### Retain (default)
Retain will retain the secret if all provider secrets have been deleted.
If a provider secret does not exist the ExternalSecret gets into the
SecretSyncedError status.

### Delete
Delete deletes the secret if all provider secrets are deleted.
If a secret gets deleted on the provider side and is not accessible
anymore this is not considered an error and the ExternalSecret
does not go into SecretSyncedError status. This is also true for new
ExternalSecrets mapping to non-existing secrets in the provider.

### Merge
Merge removes keys in the secret, but not the secret itself.
If a secret gets deleted on the provider side and is not accessible
anymore this is not considered an error and the ExternalSecret
does not go into SecretSyncedError status.


