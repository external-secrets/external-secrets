# Lifecycle
The External Secrets Operator manages the lifecycle of secrets in Kubernetes. With `creationPolicy` and `deletionPolicy` you get fine-grained control of its lifecycle.

!!! note "Creation/Deletion Policy Combinations"
    Some combinations of creationPolicy/deletionPolicy are not allowed as they would delete existing secrets:
    <br/>- `deletionPolicy=Delete` & `creationPolicy=Merge`
    <br/>- `deletionPolicy=Delete` & `creationPolicy=None`
    <br/>- `deletionPolicy=Merge` & `creationPolicy=None`

## Creation Policy
The field `spec.target.creationPolicy` defines how the operator creates the a secret.

### Owner (default)
The External Secret Operator creates secret and sets the `ownerReference` field on the Secret. This secret is subject to [garbage collection](https://kubernetes.io/docs/concepts/architecture/garbage-collection/) if the initial `ExternalSecret` is absent. If a secret with the same name already exists that is not owned by the controller it will result in a conflict. The operator will just error out, not claiming the ownership.

!!! note "Secrets with `ownerReference` field not found"
    If the secret exists and the ownerReference field is not found, the controller treats this secret as orphaned. It will take ownership of this secret by adding an `ownerReference` field and updating it.

### Orphan
The operator creates the secret but does not set the `ownerReference` on the Secret. That means the Secret will not be subject to garbage collection. If a secret with the same name already exists it will be updated.

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


