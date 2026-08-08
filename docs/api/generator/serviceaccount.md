`ServiceAccountToken` issues a short-lived token for a Kubernetes ServiceAccount through the
[TokenRequest API](https://kubernetes.io/docs/reference/kubernetes-api/authentication-resources/token-request-v1/).

It is the counterpart to the legacy `kubernetes.io/service-account-token` Secret, which never
expires and is never rotated. Combined with `PushSecret`, it lets a token be minted in one
cluster and delivered to another without a long-lived credential existing anywhere.

## Output Keys and Values

| Key                   | Description                                                                    |
|-----------------------|--------------------------------------------------------------------------------|
| token                 | The issued bearer token.                                                        |
| expirationTimestamp   | RFC3339 instant at which the API server stops accepting the token.              |

## Request Parameters

| Field                          | Description                                                              |
|--------------------------------|--------------------------------------------------------------------------|
| `serviceAccountRef.name`       | ServiceAccount to issue the token for. Required.                          |
| `serviceAccountRef.audiences`  | The `aud` claim to request. Defaults to the API server's own audience.    |
| `expirationSeconds`            | Requested lifetime. The issuer may return less — see below.               |

`serviceAccountRef.namespace` is **rejected**, not ignored. A token is always issued in the
namespace the generator is evaluated in; being able to name another namespace would let the
generator reach beyond it.

## Lifetime and refreshInterval are independent

The API server decides the token's actual lifetime. It enforces its own bounds, and
`--service-account-max-token-expiration` caps a request without reporting that it did — so
`expirationSeconds` is a request, and the returned `expirationTimestamp` is the fact.

Regeneration is driven **only** by the `refreshInterval` of the `ExternalSecret` or
`PushSecret` referencing this generator. Nothing reconciles the two. If `refreshInterval` is
at or above the token's real lifetime, the Secret holds an expired token from the moment it
dies until the next refresh — see
[#6690](https://github.com/external-secrets/external-secrets/issues/6690), where the same
shape bites the GCR generator.

Set `refreshInterval` comfortably below the lifetime you actually receive, and read
`expirationTimestamp` rather than assuming you got what you asked for.

## Permissions

Issuing a token requires `create` on the `serviceaccounts/token` subresource.

**The chart already grants this by default**, cluster-wide within ESO's scope, because
`serviceAccountRef`-based authentication needs it (Vault Kubernetes auth, Conjur JWT auth).
This generator therefore adds no permission the operator does not already hold. The risk
that comes with that default, and how to scope it down, is described under
[Scoping ServiceAccount Token Creation](../../guides/security-best-practices.md#scoping-serviceaccount-token-creation).

If you have set `rbac.serviceAccountTokenCreate: false`, the blanket permission is gone and
you must delegate token creation for each ServiceAccount this generator names:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: eso-token-argocd-remote
  namespace: argocd
rules:
  - apiGroups: ['']
    resources: ['serviceaccounts/token']
    resourceNames: ['argocd-remote']
    verbs: ['create']
```

## Example Manifest

```yaml
{% include 'generator-serviceaccount.yaml' %}
```

Example `ExternalSecret` that references the ServiceAccountToken generator:
```yaml
{% include 'generator-serviceaccount-example.yaml' %}
```
