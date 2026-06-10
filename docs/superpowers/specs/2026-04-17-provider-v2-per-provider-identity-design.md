# Provider V2 Per-Provider Identity TLS Design

**Date:** 2026-04-17

## Goal

Replace the current namespace-wide shared provider PKI model with full per-provider identity isolation for v2 providers, while preserving support for arbitrary provider addresses and keeping the common chart-managed in-cluster path automatic.

## Scope

In scope:
- Defining a per-provider TLS identity model for v2 `Provider` and `ClusterProvider`.
- Supporting both chart-managed in-cluster providers and arbitrary provider addresses.
- Separating issuer, server, and client credentials so provider pods do not receive CA private keys or reusable client keys.
- Defining the API shape, runtime lookup rules, chart wiring, and cert-controller responsibilities.
- Describing validation, migration, rotation, and test strategy for the new model.

Out of scope:
- Redesigning the general provider CRD model beyond TLS-related additions.
- Reworking provider business logic unrelated to TLS identity and transport security.
- Solving external CA / cert-manager integration in this phase.
- Tightening all provider RBAC in this spec beyond what is necessary to enforce the new TLS secret boundaries.

## Current Problems

1. All provider pods in a namespace share one TLS secret and one client identity.
2. Provider pods can currently gain access to CA private key material and reusable client credentials.
3. The runtime resolves provider TLS secrets implicitly from a namespace and fixed secret name, which is incompatible with full per-provider identity separation.
4. DNS-derived lookup is insufficient for arbitrary provider addresses because the dial target may not map cleanly to an ESO-managed in-cluster Service.

These issues mean transport-layer trust is broader than the logical trust boundary of a single provider deployment.

## Requirements

1. Each managed provider deployment must have its own transport identity and trust root.
2. A compromised provider pod must not be able to impersonate sibling providers.
3. Arbitrary provider addresses must remain supported.
4. The chart-managed in-cluster path should remain automatic by default.
5. Runtime TLS lookup must be explicit and deterministic; DNS may help with hostname verification, but it must not be the sole source of trust identity.
6. `Provider` namespace boundaries must remain strict.
7. `ClusterProvider` must continue to support cross-namespace authentication material where the API model already allows it.

## Approaches Considered

### Recommended: Per-provider CA with managed-or-explicit TLS mode

Each provider identity gets its own issuer, server, and client secrets. Chart-managed providers use controller-managed secret derivation. Arbitrary addresses use an explicit client `secretRef`.

Why this is the recommendation:
- It removes lateral trust between sibling providers.
- It keeps the common Helm path automatic.
- It gives arbitrary addresses an explicit trust anchor instead of relying on DNS heuristics.
- It is understandable operationally: every provider has one isolated trust domain.

### Alternative: Shared CA with per-provider leaf certs

Keep a namespace-wide CA but mint per-provider server/client leaf certificates.

Rejected because:
- A leaked client key still authenticates broadly inside the shared CA trust domain.
- The cert-controller still remains a single shared root of trust for all sibling providers in the namespace.
- It materially improves the current state, but it does not satisfy the requested full per-provider identity separation.

### Alternative: Explicit TLS secret references everywhere

Require every `Provider` and `ClusterProvider` to specify all TLS material explicitly.

Rejected for now because:
- It makes the common chart-managed in-cluster flow much harder to operate.
- It pushes too much ceremony onto users for the default ESO-managed deployment path.
- It gives up a valuable safe default that Helm can provide automatically.

## Design

### 1. Identity model

The transport trust boundary is the individual v2 provider deployment, not the namespace.

Each provider identity gets three distinct secrets:

- **Issuer secret**
  - Readable only by the cert-controller.
  - Contains `ca.crt` and `ca.key`.
  - Used only to mint and rotate that provider's server and client leaf certificates.

- **Server secret**
  - Mounted only into the matching provider deployment.
  - Contains `ca.crt`, `tls.crt`, and `tls.key`.
  - Used by the provider gRPC server for mTLS.

- **Client secret**
  - Read by the ESO runtime only when dialing that specific provider.
  - Contains `ca.crt`, `client.crt`, and `client.key`.
  - Used by the controller-side gRPC client.

This model ensures a provider pod never receives CA private key material and never receives a reusable client certificate for sibling providers.

### 2. API shape

Add a TLS block to v2 provider connection config:

```yaml
spec:
  config:
    address: provider.example.com:8443
    providerRef:
      apiVersion: provider.external-secrets.io/v2alpha1
      kind: SecretsManager
      name: aws-backend
    tls:
      mode: Managed | SecretRef
      secretRef:
        name: provider-aws-client-tls
        namespace: external-secrets
```

Semantics:

- `Managed`
  - The provider's client and server TLS assets are controller-managed.
  - Intended for chart-managed in-cluster provider deployments.
  - The runtime derives the client secret from the provider identity, not from a fixed namespace-wide secret.

- `SecretRef`
  - The referenced secret is the exact client TLS bundle used by ESO to dial the provider.
  - Required for arbitrary provider addresses unless the operator explicitly wires managed identity end-to-end.
  - Secret contents are `ca.crt`, `client.crt`, and `client.key`.

If `tls` is omitted:

- For chart-managed in-cluster provider deployments, treat it as equivalent to `Managed`.
- For arbitrary addresses, validation should fail unless a valid `SecretRef` is configured.

### 3. Managed mode secret naming and ownership

Managed mode needs deterministic names per provider identity.

The naming key should be the provider deployment identity derived from the provider chart and provider resource association, not just namespace or DNS address. The exact secret names should be stable and chart-generated from provider fullname/service helpers.

For each managed provider deployment:

- issuer secret name: derived, stable, unique per provider
- server secret name: derived, stable, unique per provider
- client secret name: derived, stable, unique per provider

Ownership rules:

- Cert-controller owns issuer, server, and client secret content.
- Provider deployment mounts only the server secret.
- ESO runtime reads only the client secret.
- No provider service account should need access to issuer or client secrets.

### 4. Managed mode cert-controller targets

The current cert-controller input model is too coarse because it only accepts one namespace and a flat list of service names. It must be replaced with a set of per-provider TLS targets.

Each target should include:

- provider identity key
- namespace
- service name
- deployment name
- issuer secret name
- server secret name
- client secret name

Responsibilities:

- reconcile issuer secret existence and validity for that provider identity
- issue or rotate the server secret for that provider service DNS identity
- issue or rotate the client secret for ESO's dial path to that provider
- restart only the matching provider deployment when the server secret changes
- avoid restarting unrelated provider deployments when another provider rotates

This makes certificate lifecycle management align with the actual trust boundary.

### 5. Runtime TLS lookup

The runtime must stop treating namespace plus fixed secret name as the identity lookup mechanism.

When dialing a v2 provider:

- `Managed`
  - Resolve the client secret from the provider identity carried by the `Provider` or `ClusterProvider` object.
  - Load `ca.crt`, `client.crt`, and `client.key` from that secret.

- `SecretRef`
  - Load exactly the referenced client secret.
  - Do not infer secret identity from address or namespace fallback rules.

`address` remains relevant for hostname verification:

- If the address includes a hostname, use it for `ServerName` where appropriate.
- If the address is a Kubernetes service DNS name, it should match the SANs minted into the managed server certificate.
- DNS parsing may still help populate `ServerName`, but it must not decide which secret is trusted.

### 6. Arbitrary address support

Arbitrary addresses remain supported by design.

Rules:

- If the address is not a chart-managed in-cluster ESO provider service, operators should use `tls.mode=SecretRef`.
- The referenced client secret must be explicit and reviewable.
- ESO does not attempt to guess secret identity for arbitrary endpoints from hostnames.

This avoids brittle magic while keeping the external-provider path fully functional.

### 7. Namespace and scope rules

`Provider` and `ClusterProvider` must keep different trust rules.

For namespaced `Provider`:

- `tls.mode=Managed`
  - Derived secrets must remain in the provider namespace.
- `tls.mode=SecretRef`
  - Cross-namespace secret refs are forbidden.
- `spec.config.providerRef.namespace` remains constrained to the same namespace as the `Provider`.

For cluster-scoped `ClusterProvider`:

- `tls.mode=Managed`
  - The managed secret namespace must be explicit and deterministic for that cluster-scoped identity.
- `tls.mode=SecretRef`
  - Cross-namespace refs are allowed because cluster-scoped auth material is already part of the API model.
- Existing authentication-scope semantics continue to apply independently of TLS identity.

### 8. Validation

Validation should fail early for invalid TLS wiring.

Required checks:

- `Managed` and `SecretRef` are the only valid modes.
- `SecretRef` requires `name`; namespace rules depend on `Provider` vs `ClusterProvider`.
- Arbitrary addresses without explicit `SecretRef` must fail validation unless they are recognized as managed chart-owned in-cluster providers.
- Managed mode must reject unresolved or ambiguous provider identity mapping.
- Server certificate SANs in managed mode must match the target provider service DNS names.

These checks belong in both admission validation where available and runtime guardrails for defense in depth.

### 9. Rotation model

Rotation should be isolated per provider identity.

- Leaf rotation:
  - Server and client leaf certs rotate under the same provider-specific CA.
  - Rotating one provider must not affect siblings.

- Server leaf change:
  - Restart only the matching provider deployment.

- Client leaf change:
  - No provider restart required.
  - Existing pooled connections may continue until naturally recycled.

- CA rollover:
  - Also per-provider, not namespace-wide.
  - Should be treated as a controlled path, not an opportunistic rewrite of all secrets during routine reconciliation.
  - Connection pool lifetime and rollover sequencing must be considered so old and new trust chains do not create avoidable outages.

### 10. RBAC expectations

This design assumes narrower secret access than the current namespace-wide shared secret model.

Minimum required boundary:

- Provider service accounts must not be able to read issuer secrets.
- Provider service accounts must not be able to read client secrets for sibling providers.
- ESO runtime must be able to read only the client secrets needed for configured providers.
- Cert-controller must be the only component that can read or write issuer secrets.

This spec does not attempt to redesign all provider RBAC, but the managed TLS path is not secure unless these secret boundaries are enforced.

## Migration

Migration should be staged to avoid breaking existing v2 deployments.

Phase 1:

- Introduce the new TLS API and managed target model.
- Support both the legacy shared-secret path and the new per-provider path behind compatibility logic.
- Prefer the new path for newly rendered provider chart resources.

Phase 2:

- Migrate managed chart users to per-provider secrets automatically.
- Keep explicit `SecretRef` available for arbitrary endpoints.

Phase 3:

- Remove or deprecate the legacy fixed `external-secrets-provider-tls` assumption once consumers have migrated.

Migration guardrails:

- Legacy and new secrets must not be confused by the runtime lookup logic.
- Rollouts must be per provider, not namespace-wide.
- Upgrade docs must explain how arbitrary-address users supply the new `SecretRef`.

## Testing Strategy

1. Unit tests for cert-controller target reconciliation:
   - per-provider secret naming
   - isolated issuer/server/client generation
   - targeted deployment restart behavior

2. Unit tests for runtime TLS lookup:
   - managed mode resolves the correct derived client secret
   - `SecretRef` mode loads the exact referenced secret
   - arbitrary address without explicit `SecretRef` is rejected

3. Unit tests for validation:
   - `Provider` cross-namespace `SecretRef` rejection
   - `ClusterProvider` allowed `SecretRef.namespace`
   - invalid mode / missing secret ref / ambiguous managed identity failures

4. Chart tests:
   - one server secret mount per provider deployment
   - provider-specific secret names are rendered deterministically
   - cert-controller receives the expected provider target list

5. Integration or focused e2e coverage:
   - two providers in one namespace have distinct trust material
   - compromise-style negative tests prove one provider cannot authenticate as another
   - arbitrary-address provider works with explicit `SecretRef`

## Risks And Guardrails

- The managed identity derivation must be unambiguous. If the runtime cannot identify exactly one managed client secret, it should fail closed.
- CA rollover logic is easy to get wrong with long-lived pooled connections. The first implementation should keep rollover conservative.
- Supporting both legacy and new paths temporarily increases complexity, so the compatibility boundary must be explicit and time-bounded.
- Chart and controller code must agree on identity naming rules exactly; any drift creates hard-to-debug TLS failures.

## Success Criteria

- Each managed provider deployment has isolated issuer, server, and client TLS material.
- Provider pods no longer receive CA private keys or sibling client keys.
- ESO can dial arbitrary provider addresses using explicit client `SecretRef` configuration.
- Managed chart users keep an automatic in-cluster path without manual TLS wiring.
- Runtime and chart logic no longer rely on a namespace-wide fixed provider TLS secret.
