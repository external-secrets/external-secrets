# Design: PushSecret `dataTo`

**Author:** Mohamed Rekiba
**Date:** 2026-01-20
**Status:** Proposed
**Related Issue:** [#5221 — Revamp PushSecret](https://github.com/external-secrets/external-secrets/issues/5221)
**PR:** [#5850](https://github.com/external-secrets/external-secrets/pull/5850)

---

## Motivation

PushSecret today requires an explicit `data` entry for every key you want to push. This creates three problems:

1. **Sync drift** — adding a key to a Kubernetes Secret without adding a matching PushSecret entry means that key silently never reaches the provider.
2. **Config verbosity** — a Secret with 20+ keys needs 20+ lines of boilerplate YAML that all look the same.
3. **Maintenance burden** — keys evolve alongside application code; keeping PushSecret config in sync is easy to forget.

ExternalSecret already solved the equivalent inbound problem with `dataFrom` (bulk-pull from providers). PushSecret has no equivalent outbound mechanism.

### Workarounds today

| Workaround | Drawback |
|---|---|
| Enumerate every key in `spec.data` | Verbose; falls out of sync when keys change |
| External tooling (scripts, Helm helpers) to generate PushSecret YAML | Adds build-time dependency; not declarative |
| One PushSecret per key | Explodes resource count; harder to reason about |

None of these are satisfactory for teams with dynamic secret sets that change frequently.

## Goals

1. Enable bulk pushing of all (or a filtered subset of) keys from a Kubernetes Secret to a provider without per-key enumeration.
2. Support key transformation so source key names can be rewritten before reaching the provider.
3. Scope each bulk-push entry to a specific store to prevent accidental cross-store pushes.
4. Coexist cleanly with explicit `data` entries, with explicit entries taking precedence.
5. Align PushSecret's capabilities with ExternalSecret's `dataFrom` where the push direction makes sense.

## Non-goals

- Replacing `spec.data` — explicit per-key control remains available and takes priority.
- Implementing ExternalSecret's `Extract` or `Find` — the source is always the Kubernetes Secret selected by `spec.selector`, not a provider query.
- Implementing the `Merge` rewrite — PushSecret has a single source, so there is nothing to merge.
- Adding a `RefreshPolicy` (tracked separately in #5221).
- Changing the provider interface (`SecretsClient`).

## Design

### API shape

```go
type PushSecretDataTo struct {
    StoreRef           *PushSecretStoreRef          // required — which store to push to
    RemoteKey          string                        // optional — bundle mode target
    Match              *PushSecretDataToMatch        // optional — regexp key filter
    Rewrite            []PushSecretRewrite           // optional — key transformations
    Metadata           *apiextensionsv1.JSON         // optional — provider-specific metadata
    ConversionStrategy PushSecretConversionStrategy  // optional — key name encoding
}

type PushSecretDataToMatch struct {
    RegExp string  // empty or nil = match all keys
}

type PushSecretRewrite struct {
    // Exactly one of:
    Regexp    *esv1.ExternalSecretRewriteRegexp
    Transform *esv1.ExternalSecretRewriteTransform
}
```

### Why `dataTo`?

The name mirrors ExternalSecret's `dataFrom`:

- `dataFrom` = pull data **from** the provider into K8s
- `dataTo` = push data **to** the provider from K8s

The direction is unambiguous and the symmetry aids discoverability.

### Two operating modes

| Mode | Trigger | Behavior | Use case |
|---|---|---|---|
| **Per-key** | `remoteKey` not set | Each matched key becomes its own provider secret/variable | Env-var providers (GitHub Actions, Doppler) |
| **Bundle** | `remoteKey` set | All matched keys bundled as JSON into one named provider secret | Named-secret providers (AWS SM, Vault, Azure KV, GCP SM) |

Bundle mode and `rewrite` are mutually exclusive — when keys are bundled into a JSON object, the key names inside the JSON are the source names (after conversion), not individually rewritten provider paths.

### Comparison with ExternalSecret `dataFrom`

| Aspect | ExternalSecret `dataFrom` | PushSecret `dataTo` |
|---|---|---|
| **Direction** | Provider → K8s | K8s → Provider |
| **Source** | Provider (Extract, Find, GeneratorRef) | K8s Secret (via `spec.selector`) |
| **Source discovery** | Find by tags/name in provider | Filter by regexp on K8s key names |
| **Key transformation** | Regexp, Transform, Merge | Regexp, Transform (no Merge — single source) |
| **Store targeting** | Single `secretStoreRef` per ES | Per-entry `storeRef` (required) |
| **Merge strategy** | Multiple dataFrom merged into one Secret | `dataTo` + explicit `data` merged (explicit wins) |

### Why simpler than `dataFrom`?

- **No `Extract` or `Find`** — the source is always the K8s Secret; there's nothing to query.
- **No `Merge` rewrite** — single source means no multi-source key collisions to resolve.
- **Per-entry store scoping** — prevents "push to all stores" footgun; each entry declares its target.

### Rewrite type reuse

`PushSecretRewrite` reuses the inner types from ExternalSecret:

- `esv1.ExternalSecretRewriteRegexp` (source/target regexp replacement)
- `esv1.ExternalSecretRewriteTransform` (Go template transformation)

This avoids type duplication while intentionally excluding `ExternalSecretRewriteMerge` which doesn't apply to the push direction.

The controller uses `rewriteWithKeyMapping()` instead of `esutils.RewriteMap()` because PushSecret needs a **source → destination key mapping** for conflict resolution and status tracking. `RewriteMap` operates on `map[string][]byte` (transforming the map in place), while PushSecret needs to track which original key produced which remote key. This divergence is intentional and documented.

### `storeRef` is required

Every `dataTo` entry must specify a `storeRef` with either `name` or `labelSelector`. This was added after maintainer feedback to prevent accidentally pushing to all stores when `secretStoreRefs` contains multiple entries.

### Metadata handling

Each `dataTo` entry carries its own `Metadata` field. Since different providers need structurally different metadata (e.g., AWS tags vs. Azure properties), and each entry targets a specific store via `storeRef`, users can provide per-store metadata naturally by having separate `dataTo` entries per store.

### Feature interactions

| Feature | Interaction with `dataTo` |
|---|---|
| **Template** (`spec.template`) | Template is applied *before* `dataTo` expansion. `dataTo` matches against template output keys. |
| **UpdatePolicy=IfNotExists** | Honored per-entry: if the remote secret already exists, the push is skipped. |
| **DeletionPolicy=Delete** | All `dataTo`-expanded entries are tracked in `status.syncedPushSecrets`. When the source Secret is deleted, all tracked provider secrets are cleaned up. |
| **ConversionStrategy** | Applied *before* key matching and rewriting, so regexp patterns see converted key names. |
| **Explicit `data`** | Explicit entries override `dataTo` for the same source key. Comparison uses original (unconverted) K8s key names. |

### Edge cases

| Scenario | Behavior |
|---|---|
| Empty match pattern | Matches all keys |
| No keys match | Info log, continue (not an error) |
| Invalid regexp | PushSecret enters error state with details in status |
| Duplicate remote keys (within or across entries) | Reconciliation fails listing all conflicting sources |
| Explicit `data` for same source key | `data` wins; `dataTo` entry is dropped |
| Invalid template | Fail with template parsing error |
| Both `regexp` and `transform` on a rewrite | Blocked by CRD XValidation |
| `storeRef` not in `secretStoreRefs` | Validation error |
| Source Secret deleted + DeletionPolicy=Delete | Provider secrets cleaned up via status tracking |
| `remoteKey` + `rewrite` on same entry | `rewrite` is ignored in bundle mode (documented) |

## Alternatives considered

### Alternative 1: Reuse ExternalSecret's `dataFrom` field name

Rejected because `dataFrom` implies pulling *from* a source, while PushSecret pushes *to* a destination. Using `dataFrom` on PushSecret would be semantically confusing.

### Alternative 2: Implicit "push all keys" when `data` is empty

Rejected because implicit behavior is dangerous for secrets. A typo or misconfiguration could push keys to unintended stores. Explicit opt-in via `dataTo` is safer.

### Alternative 3: Provider-keyed metadata map

Instead of per-entry metadata, use a map keyed by provider type. Rejected because `storeRef` per entry already enables per-store metadata naturally, and a provider-keyed map would require the API to enumerate provider types.

### Alternative 4: Type alias to `ExternalSecretRewrite`

Using a direct type alias would include `Merge` which doesn't apply to PushSecret. A new struct with shared inner types provides the right subset.

## Backwards compatibility

- `dataTo` is fully optional — existing PushSecrets work exactly as before.
- `data` field semantics are unchanged.
- No breaking changes to the v1alpha1 API (purely additive).
- No changes to the provider interface.
- All pre-existing tests continue to pass.
