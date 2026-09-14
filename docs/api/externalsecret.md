The `ExternalSecret` describes what data should be fetched, how the data should
be transformed and saved as a `Kind=Secret`:

* tells the operator what secrets should be synced by using `spec.data` to
  explicitly sync individual keys or use `spec.dataFrom` to get **all values**
  from the external API.
* you can specify how the secret should look like by specifying a
  `spec.target.template`

## Template

When the controller reconciles the `ExternalSecret` it will use the `spec.template` as a blueprint to construct a new `Kind=Secret`. You can use golang templates to define the blueprint and use template functions to transform secret values. You can also pull in `ConfigMaps` that contain golang-template data using `templateFrom`. See [advanced templating](../guides/templating.md) for details.

## Update behavior with 3 different refresh policies

You can control how and when the `ExternalSecret` is refreshed by setting the `spec.refreshPolicy` field. If not specified, the default behavior is `Periodic`.

### CreatedOnce

With `refreshPolicy: CreatedOnce`, the controller syncs the `Kind=Secret` once per
`ExternalSecret` object and then stops:

- Runs a single sync on the first reconcile of the `ExternalSecret`, creating or
  overwriting the target `Kind=Secret`
- Does not refresh on a schedule or when the source data changes
- Still re-syncs if the target `Kind=Secret` is changed or deleted while the same
  `ExternalSecret` object still exists
- Useful for immutable credentials or when you want to manage updates manually

The "once" is tracked on the `ExternalSecret`'s own status (`syncedResourceVersion`
and `refreshTime`), not on whether the target Secret already exists.

!!! warning "Recreating the ExternalSecret re-syncs the target Secret"
    `CreatedOnce` does not check whether the target Secret already exists. If the
    `ExternalSecret` object itself is deleted and recreated (for example by a GitOps
    controller that prunes and re-applies it), its status resets and the controller
    performs a fresh sync that overwrites the existing target Secret. With a
    stateless generator such as the Password generator this produces a brand new
    value, so the target Secret and any credential already derived from the old
    value diverge with no way to recover the original.

    `creationPolicy` does not guard against this: `Owner`, `Orphan`, `Merge` and
    `CreateOrMerge` all rewrite the managed keys on that sync. The only field that prevents an existing
    target Secret's data from being overwritten is `spec.target.immutable: true`,
    which skips the data update whenever the Secret already exists. Combine
    `refreshPolicy: CreatedOnce` with `spec.target.immutable: true` for credentials
    that must be generated once and never change.

Example:
```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: example
spec:
  refreshPolicy: CreatedOnce
  # other fields...
```

To generate a credential exactly once and never overwrite it afterward, even if
the `ExternalSecret` object is recreated, combine `refreshPolicy: CreatedOnce` with
`spec.target.immutable: true`. This is the correct pattern for a credential that an
application persists on first run (for example a Keycloak admin password written
into the application database at bootstrap): regenerating the value later would
leave the target Secret out of sync with the live credential. `creationPolicy:
Orphan` keeps the target Secret in place if the `ExternalSecret` is deleted, and
`immutable: true` stops any later reconcile (including one triggered by recreating
the `ExternalSecret`) from rewriting the existing data.

```yaml
apiVersion: generators.external-secrets.io/v1alpha1
kind: Password
metadata:
  name: keycloak-admin-password
spec:
  length: 32
  digits: 5
  symbols: 5
  allowRepeat: true
---
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: keycloak-admin
spec:
  refreshPolicy: CreatedOnce
  target:
    name: keycloak-admin
    # keep the Secret if this ExternalSecret is ever deleted
    creationPolicy: Orphan
    # never rewrite the Secret data once it exists, even on recreation
    immutable: true
  dataFrom:
    - sourceRef:
        generatorRef:
          apiVersion: generators.external-secrets.io/v1alpha1
          kind: Password
          name: keycloak-admin-password
```

### Periodic

With `refreshPolicy: Periodic` (the default behavior), the controller will:

- Create the `Kind=Secret` if it doesn't exist
- Update the `Kind=Secret` regularly based on the `spec.refreshInterval` duration
- When `spec.refreshInterval` is set to zero, it will only create the secret once and not update it afterward
- When `spec.refreshInterval` is set to a value greater than zero, the controller will update the `Kind=Secret` at the specified interval or when the `ExternalSecret` specification changes

Example:
```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: example
spec:
  refreshPolicy: Periodic
  refreshInterval: 1h0m0s  # Update every hour
  # other fields...
```

### OnChange

With `refreshPolicy: OnChange`, the controller will:

- Create the `Kind=Secret` if it doesn't exist
- Update the `Kind=Secret` only when the `ExternalSecret`'s metadata or specification changes
- This policy is independent of the `refreshInterval` value
- Useful when you want to manually control when the secret is updated, by modifying the `ExternalSecret` resource

Example:
```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: example
spec:
  refreshPolicy: OnChange
  # other fields...
```

## Manual Refresh

If supported by the configured `refreshPolicy`, you can manually trigger a refresh of the `Kind=Secret` by updating the annotations of the `ExternalSecret`:

```
kubectl annotate es my-es force-sync=$(date +%s) --overwrite
```

## SyncWindows

`syncWindows` restricts **when** periodic refreshes may occur. It is evaluated in UTC and applies only to the `Periodic` refresh policy (or when `refreshPolicy` is unset). `OnChange` and `CreatedOnce` policies are unaffected.

A sync-windows block carries a shared `kind` and a list of `schedule + duration` entries:

- `kind: allow` -- periodic syncs are permitted **only** while at least one window is active; all other times are blocked.
- `kind: deny` -- periodic syncs are **blocked** while any window is active; all other times proceed normally.

Each entry in `windows` uses a standard 5-field cron `schedule` (UTC) and a `duration` string (e.g. `8h`, `30m`). The window stays open for `duration` after each schedule firing. A window entry with an unparseable `schedule` is silently ignored and treated as inactive, so a typo does not permanently block syncs.

### Example: allow syncs only during business hours (Mon-Fri 09:00-17:00 UTC)

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: example
spec:
  refreshInterval: 1h
  syncWindows:
    kind: allow
    windows:
      - schedule: "0 9 * * 1-5"  # weekdays at 09:00 UTC
        duration: 8h              # window open until 17:00 UTC
```

### Example: block syncs during a Saturday maintenance window (02:00-04:00 UTC)

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: example
spec:
  refreshInterval: 30m
  syncWindows:
    kind: deny
    windows:
      - schedule: "0 2 * * 6"  # Saturdays at 02:00 UTC
        duration: 2h            # block until 04:00 UTC
```

### Multiple windows

You can list several entries under `windows`. For `kind: allow`, the sync is permitted when **any** window is active. For `kind: deny`, the sync is blocked when **any** window is active.

### Interaction with refreshInterval

`syncWindows` only suppresses sync operations -- it does not change how often the controller checks. The controller still requeues at `refreshInterval` regardless of whether a sync was blocked. This means that if `refreshInterval` is longer than `window.duration`, a window could open and close entirely between two consecutive checks and the sync would be missed for that occurrence. This is by design: `refreshInterval` is the primary driver; `syncWindows` is a gate on top of it. To ensure no window occurrence is missed, set `refreshInterval` to a value shorter than the smallest `window.duration`.

## Features

Individual features are described in the [Guides section](../guides/introduction.md):

* [Find many secrets / Extract from structured data](../guides/getallsecrets.md)
* [Templating](../guides/templating.md)
* [Using Generators](../guides/generator.md)
* [Secret Ownership and Deletion](../guides/ownership-deletion-policy.md)
* [Key Rewriting](../guides/datafrom-rewrite.md)
* [Filtering Keys with Select](../guides/datafrom-select.md)
* [Decoding Strategy](../guides/decoding-strategy.md)

## Example

Take a look at an annotated example to understand the design behind the
`ExternalSecret`.

``` yaml
{% include 'full-external-secret.yaml' %}
```
