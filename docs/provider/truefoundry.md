## TrueFoundry

Sync secrets from a TrueFoundry control plane into Kubernetes using the External Secrets Operator.

The provider authenticates with a TrueFoundry **cluster service token** and fetches each secret by its fully-qualified name (FQN) through a single HTTP GET against the control-plane API:

```
GET <baseURL>/v1/control-plane/secret?secret_ref=tfy-secret://<tenant>:<group>:<secret-name>
Authorization: Bearer <cluster-token>
```

The response is `{"value":"<secret>"}`. The `tfy-secret://` URI scheme is added by the provider; users supply only the bare FQN as `remoteRef.key`.

## Authentication

The provider uses a cluster service token — typically the token provisioned by the TrueFoundry agent on each cluster and stored in a Kubernetes Secret like `tfy-agent-internal-<env>-token` in the `tfy-agent` namespace under the key `CLUSTER_TOKEN`.

Reference that Secret directly from the `SecretStore`:

```yaml
spec:
  provider:
    truefoundry:
      secretRef:
        name: tfy-agent-internal-devtest-token
        namespace: tfy-agent
        key: CLUSTER_TOKEN
```

> **NOTE:** When using a `ClusterSecretStore`, set `namespace` in `secretRef` so ESO can locate the Secret across namespaces.

## SecretStore

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore                # or ClusterSecretStore
metadata:
  name: tfy-store
spec:
  provider:
    truefoundry:
      # Control-plane URL for your TrueFoundry installation. The provider
      # appends /v1/control-plane/secret to this.
      baseURL: https://your-cluster.tfy-usea1-ctl.example.com
      secretRef:
        name: tfy-agent-internal-devtest-token
        namespace: tfy-agent
        key: CLUSTER_TOKEN
```

## Referencing secrets

`remoteRef.key` is the TrueFoundry secret FQN. FQN format is `<tenant>:<group>:<secret-name>`. The provider wraps it in `tfy-secret://` and URL-encodes the result before sending it as the `secret_ref` query parameter.

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
        key: truefoundry:prod-app:DB_PASSWORD
    - secretKey: API_TOKEN
      remoteRef:
        key: truefoundry:prod-app:API_TOKEN
```

Each `data[]` entry produces exactly one HTTP request to the control plane.

### Selecting a nested field from a JSON-encoded value

If a single TrueFoundry secret stores JSON, use `remoteRef.property` with gjson syntax to pick out a sub-field:

```yaml
data:
  - secretKey: DB_HOST
    remoteRef:
      key: truefoundry:prod-app:DB_CONNECTION
      property: host        # gjson path applied to the secret's JSON value
```

### Treating a JSON secret as a key/value map

If a secret's value is a JSON object, you can mount each field as its own key in the target Secret via `dataFrom.extract`:

```yaml
dataFrom:
  - extract:
      key: truefoundry:prod-app:CONFIG_BLOB     # value must be a JSON object
```

This still costs **one** API call — the JSON object is split into map entries locally.

## Behavior

- **Missing secret.** A 404 from the control-plane endpoint maps to `Secret does not exist` and surfaces as an event on the `ExternalSecret`. The target Kubernetes Secret is not modified.
- **Auth failures.** A 401/403 fails the reconcile with a wrapped error containing `status 401`/`status 403`. Re-check the `CLUSTER_TOKEN` value and that the Secret it lives in is reachable from the SecretStore's namespace.
- **Retries.** Transport errors and 5xx responses are retried (3 attempts, exponential backoff with jitter). `Retry-After` is honored on 429.
- **Refreshes.** Values are re-fetched every `refreshInterval`. To force an immediate refresh, annotate the `ExternalSecret`: `kubectl annotate es <name> force-sync=$(date +%s) --overwrite`.

## Limitations

The provider is **read-only**. The following operations are not supported:

| Capability | Status | Reason |
|---|---|---|
| `PushSecret` / `DeleteSecret` / `SecretExists` | not implemented | read-only provider |
| `find.tags` / `find.name` / `find.path` (`GetAllSecrets`) | not supported | the control-plane endpoint fetches one secret by FQN — no enumeration mode. Enumerate each key explicitly in `spec.data[]`. |
| `Validate` probe | returns `Unknown` | the control-plane endpoint has no health probe; auth and connectivity errors surface naturally on the first reconcile |
| Custom CA / `caBundle` / `caProvider` | not supported | use a `baseURL` reachable via the cluster's default trust roots |

## Status

The TrueFoundry provider is actively maintained.
