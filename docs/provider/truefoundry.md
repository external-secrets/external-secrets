## TrueFoundry

Sync secrets from [TrueFoundry's secret management](https://www.truefoundry.com/docs/apply-api-secret-management) into Kubernetes using the External Secrets Operator.

The provider talks to the TrueFoundry secret-management REST API (`<control-plane-url>/api/svc/v1/...`) using a TrueFoundry API key sent as `Authorization: Bearer <token>`.

## Authentication

The provider authenticates with a TrueFoundry API key — this can be a [Personal Access Token (PAT)](https://docs.truefoundry.com/docs/personal-access-token-rbac), a Virtual Access Token (VAT), or a service-account token. Any token TrueFoundry accepts as a Bearer credential will work.

Store the token in a Kubernetes `Secret` in the namespace where you will create the `SecretStore`:

```sh
HISTIGNORE='*kubectl*' kubectl create secret generic \
    tfy-creds \
    --from-literal=api-key="tfy-xxxxxxxxxxxx"
```

> **NOTE:** When using a `ClusterSecretStore`, set `namespace` in `auth.secretRef.apiKey` so ESO can locate the secret across namespaces.

## SecretStore

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: tfy-store
spec:
  provider:
    truefoundry:
      # Control plane URL for your TrueFoundry installation. SaaS users
      # typically use https://app.truefoundry.com; self-hosted / dedicated
      # tenants use their own URL (e.g. https://your-org.truefoundry.cloud).
      baseURL: https://app.truefoundry.com
      tenant: my-tenant            # used to build the FQN: <tenant>:<group>
      auth:
        secretRef:
          apiKey:
            name: tfy-creds
            key: api-key
```

The `tenant` field is required: TrueFoundry's search API identifies a secret group by its fully-qualified name `<tenant>:<group>`. The provider builds this FQN at request time from `tenant` plus the group name parsed out of the `ExternalSecret` reference.

## How TrueFoundry secrets are fetched

The TrueFoundry API does not return secret values from the group-lookup endpoint, so every read is a two-step flow:

1. `GET /api/svc/v1/secret-groups?fqn=<tenant>:<group>` returns the group's metadata, including an `associatedSecrets[]` array of `{ id, name }` for every secret inside.
2. `GET /api/svc/v1/secrets/{id}` returns one secret's plaintext value.

| What you request | HTTP calls per reconcile |
|---|---|
| Single key (`remoteRef.key: group/key`) | 1 search + 1 value fetch |
| Whole group (`dataFrom.extract` with bare group, or `remoteRef.key: group`) | 1 search + **N parallel** value fetches (N = secrets in the group, concurrency capped at 10) |

If any per-secret fetch fails, the whole call is aborted — partial results are never written to the target `Secret`.

> **Efficiency tip:** when you want many keys from one group, prefer `dataFrom.extract` over enumerating each key in `data[]`. Each `data[]` entry triggers its own search, so 5 entries from the same group cost 5 searches + 5 fetches, while `dataFrom.extract` costs 1 search + 5 parallel fetches.

## Referencing secrets

`remoteRef.key` accepts two forms:

| Form | What it returns |
|---|---|
| `<group>` | The entire group as a JSON object (sorted keys). Use with `GetSecret` for templating, or with `GetSecretMap`/`dataFrom` to materialise every key as its own field in the target `Secret`. |
| `<group>/<secret-name>` | A single secret value as raw bytes. |

`remoteRef.property` (gjson syntax) selects a sub-field from a JSON-encoded value.

### Single key from a group

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: app-credentials
spec:
  refreshInterval: 1m
  secretStoreRef:
    kind: SecretStore
    name: tfy-store
  target:
    name: app-credentials
  data:
    - secretKey: DB_PASSWORD
      remoteRef:
        key: prod-app/DB_PASSWORD
    - secretKey: API_TOKEN
      remoteRef:
        key: prod-app/API_TOKEN
```

### Whole group fanned out into the target Secret

Use `dataFrom` with a bare group name to mount every secret in the group as its own key:

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: app-credentials-all
spec:
  secretStoreRef:
    kind: SecretStore
    name: tfy-store
  target:
    name: app-credentials-all
  dataFrom:
    - extract:
        key: prod-app           # whole group
```

### Selecting a nested field from a JSON-encoded value

```yaml
data:
  - secretKey: DB_HOST
    remoteRef:
      key: prod-app/DB_CONNECTION
      property: host           # gjson path applied to the secret's JSON value
```

## Behavior

- **Missing group or key.** When the FQN search returns an empty result (TrueFoundry's response on a non-existent group is `200` with `data: []`, not 404), or when the named key isn't in the group's `associatedSecrets`, the provider returns `Secret does not exist`. ESO surfaces this as an event on the `ExternalSecret`, and the target k8s `Secret` is not modified.
- **Auth failures.** A `401` / `403` from TrueFoundry flips the `SecretStore` to `Invalid` and the `ExternalSecret` to `SecretSyncedError`.
- **Refreshes.** Values are re-fetched every `refreshInterval`. To force an immediate refresh, annotate the `ExternalSecret`: `kubectl annotate es <name> force-sync=$(date +%s) --overwrite`.

## Limitations

The provider is **read-only** in v1. The following operations are not supported:

| Capability | Status | Reason |
|---|---|---|
| `PushSecret` / `DeleteSecret` / `SecretExists` | not implemented | v1 is read-only. |
| `find.tags` | not supported | TrueFoundry's documented API has no tag concept. |
| `find.name` / `find.path` (`GetAllSecrets`) | not supported | The search endpoint requires an FQN; the documented API offers no enumerate-all-groups mode. Use one `SecretStore` per group, or a `dataFrom.extract` on a known group. |
| Custom CA / `caBundle` / `caProvider` | not supported | Use a `baseURL` reachable via the cluster's default trust roots. |

## Status

The TrueFoundry provider is actively maintained.
