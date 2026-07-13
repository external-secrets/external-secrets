# Lifecycle
The External Secrets Operator manages the lifecycle of secrets in Kubernetes. With `refreshPolicy`, `creationPolicy` and `deletionPolicy` you get fine-grained control of its lifecycle.

!!! note "Creation/Deletion Policy Combinations"
    Some combinations of creationPolicy/deletionPolicy are not allowed as they would delete existing secrets:
    <br/>- `deletionPolicy=Delete` & `creationPolicy=Merge`
    <br/>- `deletionPolicy=Delete` & `creationPolicy=CreateOrMerge`
    <br/>- `deletionPolicy=Delete` & `creationPolicy=None`
    <br/>- `deletionPolicy=Merge` & `creationPolicy=None`

## Refresh Policy
The field `spec.refreshPolicy` defines how the operator refreshes the secret.

### Periodic (default)
Refreshes the secret at a fixed interval via `spec.refreshInterval`. Due to backwards compatibility, setting a refresh interval of 0 will result in the same behavior as `CreatedOnce`.

### OnChange
Refreshes the secret only when the ExternalSecret is updated.

### CreatedOnce
Refreshes the secret only once per `ExternalSecret` object, on its first reconcile.
The sync state lives on the `ExternalSecret`'s status, so deleting and recreating the
`ExternalSecret` resets it and triggers another one-time sync that overwrites the
existing target Secret.

!!! note "State is not persisted across ExternalSecret objects"
    ESO is stateless across `ExternalSecret` objects: it does not persist sync claims
    or reclaim orphaned Secrets when an `ExternalSecret` is recreated. `creationPolicy`
    (including `Orphan`) does not protect a pre-existing target Secret from being
    rewritten on that sync. To keep an already-populated Secret from being overwritten
    on recreation, set `spec.target.immutable: true` (optionally together with
    `refreshPolicy: CreatedOnce`).

## Creation Policy
The field `spec.target.creationPolicy` defines how the operator creates the secret.

### Owner (default)
The External Secret Operator creates secret and sets the `ownerReference` field on the Secret. This secret is subject to [garbage collection](https://kubernetes.io/docs/concepts/architecture/garbage-collection/) if the initial `ExternalSecret` is absent. If a secret with the same name already exists that is not owned by the controller it will result in a conflict. The operator will just error out, not claiming the ownership.

!!! note "Secrets with `ownerReference` field not found"
    If the secret exists and the ownerReference field is not found, the controller treats this secret as orphaned. It will take ownership of this secret by adding an `ownerReference` field and updating it.

### Orphan
Whenever triggered via `RefreshPolicy` conditions, the operator creates/updates
the target Secret according to the provider available information.
It does not set an `ownerReference`, so it does not trigger
[garbage collection](https://kubernetes.io/docs/concepts/architecture/garbage-collection/) of the Secret when the `ExternalSecret` object is deleted.
The operator still watches the Secret, but for `Orphan` it re-syncs only when a refresh
is due (see the warning below and the [behavior matrix](#behavior-matrix)), not on
every change to the Secret.

!!! warning "Unwanted reverts of manual changes"
    If you set `spec.refreshPolicy` to `Periodic` or `OnChange` and `spec.target.creationPolicy` to `Orphan`,
    any changes manually done to the Secret will eventually be replaced on the next sync interval
    or on the next update to `ExternalSecret` object. That manual change is then lost forever.
    Use `creationPolicy=Orphan` with caution.

### Merge
The operator does not create a secret. Instead, it expects the secret to already exist. Values from the secret provider will be merged into the existing secret. Note: the controller takes ownership of a field even if it is owned by a different entity. Multiple ExternalSecrets can use `creationPolicy=Merge` with a single secret as long as the fields don't collide - otherwise you end up in an oscillating state.

### CreateOrMerge
Creates the Secret if it does not exist and, if it does, merges the provider values
into it while preserving keys owned by others (like `Merge`), without setting an
`ownerReference` (like `Orphan`). Unlike `Merge` it also creates a missing Secret,
so a deleted target is recreated immediately (driven by the Secret watch) while the
`ExternalSecret` exists; unlike `Owner` the Secret is retained when the
`ExternalSecret` is deleted. Combined with `spec.target.immutable: true` it gives
"create once, freeze, recreate if deleted, keep on ExternalSecret deletion". Note:
`deletionPolicy=Delete` is not allowed, the controller does not own the Secret.

### None
The operator does not create or update the secret, this is basically a no-op.

## Immutable target
Setting `spec.target.immutable: true` marks the generated `Kind=Secret` as immutable,
so its data is never rewritten once the Secret exists (Kubernetes itself also blocks
edits to an immutable Secret's data). It does not affect whether a missing Secret is
created or recreated, that is governed by `creationPolicy` and `refreshPolicy`. See the
[behavior matrix](#behavior-matrix) for the full interaction.

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
`deletionPolicy=Delete` only takes effect with `creationPolicy=Owner`; the operator
refuses to delete a Secret it does not own and reports a `SecretSyncedError` otherwise.

### Merge
Merge removes keys in the secret, but not the secret itself.
If a secret gets deleted on the provider side and is not accessible
anymore this is not considered an error and the ExternalSecret
does not go into SecretSyncedError status.

## Behavior matrix

How `creationPolicy` and `refreshPolicy` combine to drive the Secret operations.
`deletionPolicy` is a separate axis (covered below the table) that only fires when
the provider returns no data.

| creationPolicy    | refreshPolicy | Create if missing | Reflect source change | Overwrite on ES re-create | Recreate if Secret deleted | Retain on ES delete   |
|-------------------|---------------|-------------------|-----------------------|---------------------------|----------------------------|-----------------------|
| **Owner**         | Periodic      | Yes               | Yes (at interval)     | Yes                       | Yes (immediate)            | No (GC via ownerRef)  |
| **Owner**         | OnChange      | Yes               | Only on ES change     | Yes                       | Yes (immediate)            | No (GC via ownerRef)  |
| **Owner**         | CreatedOnce   | Yes               | No                    | Yes                       | Yes (immediate)            | No (GC via ownerRef)  |
| **Orphan**        | Periodic      | Yes               | Yes (at interval)     | Yes                       | Yes (at interval)          | Yes                   |
| **Orphan**        | OnChange      | Yes               | Only on ES change     | Yes                       | No (until ES change)       | Yes                   |
| **Orphan**        | CreatedOnce   | Yes               | No                    | Yes                       | No                         | Yes                   |
| **Merge**         | Periodic      | No (waits)        | Yes (at interval)     | Yes (if target exists)    | No (never creates)         | Yes                   |
| **Merge**         | OnChange      | No (waits)        | Only on ES change     | Yes (if target exists)    | No (never creates)         | Yes                   |
| **Merge**         | CreatedOnce   | No (waits)        | No                    | Yes (if target exists)    | No (never creates)         | Yes                   |
| **CreateOrMerge** | Periodic      | Yes               | Yes (at interval)     | Yes                       | Yes (immediate)            | Yes                   |
| **CreateOrMerge** | OnChange      | Yes               | Only on ES change     | Yes                       | Yes (immediate)            | Yes                   |
| **CreateOrMerge** | CreatedOnce   | Yes               | No                    | Yes                       | Yes (immediate)            | Yes                   |
| **None**          | any           | No (no-op)        | No                    | No                        | No                         | Yes (nothing created) |

Column meanings:

- **Reflect source change**: a value changing in the provider is pulled into the
  Secret. ESO is pull-based and does not watch providers, so only `Periodic` polls
  (at each interval). `OnChange` re-syncs only when the `ExternalSecret` spec or
  metadata changes, or when you set the `force-sync` annotation. `CreatedOnce` never
  re-syncs.
- **Overwrite on ES re-create**: deleting and recreating the `ExternalSecret` object
  resets its status, so the next sync overwrites the ES-managed keys of an existing
  target (keys owned by others are preserved). This is independent of `refreshPolicy`;
  only `spec.target.immutable: true` prevents it.
- **Recreate if Secret deleted**: you delete the target Secret while the
  `ExternalSecret` still exists. `Owner` treats a missing Secret as invalid and
  recreates it immediately via the Secret watch. `Orphan` treats a missing Secret as
  valid, so it only recreates when a refresh is triggered (a `Periodic` interval);
  under `CreatedOnce` or `OnChange` it stays deleted. `Merge` never creates.
- **Retain on ES delete**: whether the target Secret survives deletion of the
  `ExternalSecret`. Only `Owner` sets an `ownerReference`, so only `Owner` is
  garbage-collected.

Modifiers that cut across every row:

- **`spec.target.immutable: true`**: once the target Secret exists, its data is never
  rewritten, so "Reflect source change" and "Overwrite on ES re-create" become No. It
  does not affect "Create if missing" or "Recreate if Secret deleted", a Secret that
  does not exist has no data to protect, so it is created or recreated fresh.
- **`refreshInterval: 0` with `Periodic`** behaves like `CreatedOnce`.
- **`deletionPolicy`** (Retain, Delete, Merge) only acts when the provider returns no
  data, which is only detected on a re-sync. So under `CreatedOnce` a provider-side
  deletion is never detected, under `OnChange` only on an `ExternalSecret` change, and
  under `Periodic` at the next interval.
