```yaml
---
title: ExternalSecretSet CRD
version: v1alpha1
authors: Enzo Venturi 
creation-date: 2026-08-11
status: draft
---
```

# Generate One ExternalSecret Per Discovered Source (ExternalSecretSet)

## Table of Contents

<!-- toc -->
// autogen please
<!-- /toc -->

## Summary

`ExternalSecretSet` is a namespaced controller that discovers provider-native
secret identities and materializes one `ExternalSecret` per discovered
source.

This design keeps the existing `ExternalSecret` reconciliation model intact.
It does not change `dataFrom.find`, and it does not try to flatten multiple
discovered secrets into a single Kubernetes `Secret`.

The key idea is source fan-out, not data fan-in.

## Motivation

Today ESO has two related but different behaviors:

1. `dataFrom.find` can discover multiple provider secrets, but the result is
   merged into one Kubernetes `Secret`.
2. `ClusterExternalSecret` can fan out one spec into many namespaces, but it
   still produces one `ExternalSecret` per namespace, not one per discovered
   source.

The missing primitive is a controller that can discover a changing set of
external secrets and turn each discovered source into its own generated `ExternalSecret` resource.

That is a different problem from "fetch many values". It is a problem of
resource discovery, naming, ownership, cleanup, and observability.

## Goals

* Discover external secret identities through provider-native selectors.
* Create, update, and delete one `ExternalSecret` per discovered source.
* Preserve the logical identity of each discovered source in the generated
  Kubernetes `Secret`.
* Keep generated `ExternalSecret` resources explicit and debuggable.
* Make provider support explicit instead of assuming every store behaves like a
  filesystem with paths.
* Keep the existing `ExternalSecret` controller and APIs unchanged.

## Non-Goals

* Do not turn `dataFrom.find` into a one-to-many resource generator.
* Do not assume every provider supports hierarchical paths.
* Do not scan an entire provider by default when a provider cannot filter
  efficiently.
* Do not introduce a free-form templating language for generated `ExternalSecret` manifests.
* Do not support `target.manifest` or any non-Secret target materialization in
  v1.
* Do not support per-item custom transforms inside a single rule in v1.
* Do not introduce a cluster-scoped companion in v1.
* Do not replace the existing `ExternalSecret` workflow for users that do not
  need discovery fan-out.

## Proposal

`ExternalSecretSet` is a new namespaced CRD that references a single
`SecretStore` or `ClusterSecretStore` and one or more named rules.

Each rule asks a provider adapter to return a list of discovered sources. The
controller then renders one `ExternalSecret` per discovered source and
lets the existing ESO reconciliation path create the final Kubernetes `Secret`.

### User Stories

https://github.com/external-secrets/external-secrets/issues/2670

### API

The exact schema can be adjusted during implementation, but the shape should be
structured, explicit, and conservative.

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: ExternalSecretSet
metadata:
  name: app-secrets
  namespace: platform
spec:
  secretStoreRef:
    name: app-store
    kind: SecretStore
  refreshInterval: 10m
  maxExternalSecrets: 200
  rules:
    - name: prod-apps
      match:
        key:
          prefix: teamone/app/
        tags:
          env: prod
      exclude:
        - key:
            regexp: .*-(bootstrap|admin)$
      externalSecret:
        refreshInterval: 1h
        data:
          secretKey: value
        target:
          deletionPolicy: Delete
          template:
            type: Opaque
```

This is intentionally not a full text template.

For a discovered source whose payload is treated as a property set, the rule
uses the other union member:

```yaml
rules:
  - name: structured-secrets
    match:
      key:
        prefix: teamone/structured/
    externalSecret:
      properties: {}
      target:
        deletionPolicy: Delete
```

`ExternalSecretSet.spec.refreshInterval` controls how often the parent
repeats discovery. `rules[].externalSecret.refreshInterval` is passed through to
the generated `ExternalSecret` and controls how often that resource refreshes
its value from the provider. The two intervals are independent.
`ExternalSecretSet.spec.refreshInterval: 0` disables periodic parent discovery;
the controller still reconciles on create/update events and explicit requeues,
but it does not schedule time-based rediscovery.

In this RFC, the entries under `rules` are named declarative rules. Each rule
defines both membership (which provider sources belong to the rule) and the
desired `ExternalSecret` shape for every retained source. Rule names are stable
identifiers and are independent of list order. In the CRD schema, `rules` is a
map-style list keyed by `name` (`listType=map`, `listMapKey=name`) so
Server-Side Apply and merge operations preserve rule identity. `exclude` has no
independent item identity in v1 and is treated as an atomic list.

The rule syntax is intentionally small:

* `match` means "retain sources that satisfy this selector"
* `all: {}` means "retain every eligible source in the referenced store"
* `match` and `all` are mutually exclusive
* terms inside one `match` are ANDed
* entries inside `exclude` are ORed

For a provider adapter that exposes multiple source classes, `all: {}` means
the complete set of eligible source identities that the adapter advertises for
that store. If the adapter exposes more than one `Type`, it must document
whether `all: {}` spans one class or all of them.

A whole-store rule uses the explicit `all: {}` form:

```yaml
rules:
  - name: all-prod-apps
    all: {}
    externalSecret:
      data:
        secretKey: value
      target:
        deletionPolicy: Delete
```

The `externalSecret` field describes the desired shape of each generated
`ExternalSecret` for that rule. It is intentionally narrower than a complete
`ExternalSecretSpec`: the set controller owns the source binding and the
ownership fields that are intrinsic to fan-out.

There is no separate public source-binding field in v1. The controller binds
`DiscoveredSecret.Key` intrinsically when rendering each generated
`ExternalSecret`. The bound key becomes `remoteRef.key` when `data` is selected,
or `dataFrom.extract.key` when `properties` is selected. Discovery decides
which source exists; `externalSecret` describes how that source should appear
through the generated `ExternalSecret`.

In v1, exactly one of `externalSecret.data` or
`externalSecret.properties` must be set. Both union members are individually
optional in the schema; validation enforces exactly one and neither member is
defaulted.

`externalSecret.data.secretKey` declares that the discovered source is exposed
as one key in the target Kubernetes `Secret`. It renders one
`ExternalSecret.spec.data` entry whose `remoteRef.key` is controller-assigned
from `DiscoveredSecret.Key`.

`externalSecret.properties: {}` declares that the discovered source is exposed
as a property set. It renders one `ExternalSecret.spec.dataFrom.extract` entry
whose `extract.key` is controller-assigned from `DiscoveredSecret.Key`.

The controller does not infer `data` versus `properties` from payload shape,
provider metadata, or key format. If different source classes need different
desired `ExternalSecret` shapes, they belong in different named rules.

The controller may surface a small, typed set of discovery metadata in
controller-owned names and provenance, such as the discovered source name,
canonical source key, and provider metadata needed for selection. It does not
introduce a free-form per-item templating language.

### Example: Input and Output

This is the shape we want to make obvious in the design.

Input to the `ExternalSecretSet` controller:

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: ExternalSecretSet
metadata:
  name: app-secrets
  namespace: platform
spec:
  secretStoreRef:
    name: aws-teamone
    kind: ClusterSecretStore
  rules:
    - name: prod-apps
      match:
        key:
          prefix: teamone/app/
      externalSecret:
        data:
          secretKey: value
        target:
          deletionPolicy: Delete
```

Discovered items:

```yaml
- key: teamone/app/db
  name: db
  type: Secret
- key: teamone/app/cache
  name: cache
  type: Secret
- key: teamone/app/api
  name: api
  type: Secret
```

Rendered `ExternalSecret` resources:

```text
app-secrets-teamone-app-db
app-secrets-teamone-app-cache
app-secrets-teamone-app-api
```

One rendered `ExternalSecret` looks like this:

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: app-secrets-teamone-app-db
  namespace: platform
spec:
  secretStoreRef:
    name: aws-teamone
    kind: ClusterSecretStore
  target:
    name: db
    creationPolicy: Owner
    deletionPolicy: Delete
  data:
    - secretKey: value
      remoteRef:
        key: teamone/app/db
```

That is the concrete behavior this CRD should produce for one retained source.

The source binding is intrinsic to the generated `ExternalSecret`. It is not exposed as a
user-configurable field in the `ExternalSecretSet` API. `creationPolicy: Owner`
is also an invariant of generated `ExternalSecret` resources in v1 rather than a user-configurable
field on the `externalSecret` configuration.

### Discovery Contract

`ExternalSecretSet` should not reuse `GetAllSecrets` for discovery. That API
returns a flattened `map[string][]byte` and loses the identity and metadata of
each source.

The `ExternalSecretSet` controller needs discovered items, not secret payloads.

A provider adapter should expose a discovery-oriented contract that returns a
structured list of items, for example:

```go
type DiscoveryResult struct {
    Items []DiscoveredSecret
    State DiscoveryState
}

type DiscoveryClient interface {
    Discover(ctx context.Context, rule SecretDiscoveryRule, remainingBudget int) (DiscoveryResult, error)
}

type DiscoveryState string

const (
    DiscoveryStateComplete       DiscoveryState = "Complete"
    DiscoveryStateTooManyResults DiscoveryState = "TooManyResults"
    DiscoveryStateIncomplete     DiscoveryState = "Incomplete"
)

type DiscoveredSecret struct {
    Key     string
    Name    string
    Version string
    Labels  map[string]string
    Type    string
}
```

> `SecretDiscoveryRule` is the provider-facing discovery portion of a set rule; it does not include the `externalSecret` desired `externalSecret` configuration.

The important part is not the exact Go signature. The important part is that
discovery returns stable identities and metadata, not secret values. The state
machine has exactly three valid outcomes. `Complete` means the retained result
set is exhaustive for the rule. `TooManyResults` means the retained result set
exceeded the configured limit and the adapter knows there are more retained
results beyond those returned. `Incomplete` means the adapter could not prove
completeness. Any other state is invalid. A scan budget can be exhausted
without proving there are more retained results; in that case the adapter must
return `Incomplete`, not `TooManyResults`.

The rule name is part of the contract because status, events, and collision
reporting are keyed per rule. Rule names must be stable, unique within the set,
and simple enough to survive as status and provenance identifiers.

Providers that cannot satisfy a discovery selector should reject it explicitly.
An explicit whole-store discovery mode is still valid when the user wants every
discoverable source from the referenced store. The controller should not widen
scope beyond the referenced store to emulate an unsupported selector.

The adapter is responsible for evaluating the rule's `match` and `exclude`
clauses as one discovery contract. It must return exactly one discovery state
for the rule. `DiscoveryStateComplete` means the adapter has exhausted the
retained result set for that rule. `DiscoveryStateTooManyResults` means the
retained result set exceeded the configured limit and there are more retained
results beyond those returned. `DiscoveryStateIncomplete` means the adapter
could not prove completeness. Any other state is invalid. `remainingBudget` is
only a scan hint for retained items; returning exactly `remainingBudget` items
is not enough to claim completeness. A scan budget can be exhausted without
proving there are more retained results; in that case the adapter must return
`DiscoveryStateIncomplete`, not `DiscoveryStateTooManyResults`.

The discovery adapter is constructed from the same resolved provider store that
the normal `SecretsClient` would use. It is a per-reconcile object, not shared
mutable state, and it should be closed after discovery completes.

If the referenced store object is updated in place and keeps the same UID, the
controller treats it as the same source root. It must run a fresh discovery
cycle, but it does not treat the store edit itself as a new identity.

For v1, the canonical source identity is the resolved store UID plus the
discovered type plus the discovered key. `Name` and `Labels` are metadata.
`Version` is observational only in v1; the controller does not pin versions.
If a future API wants pinning, it needs an explicit field for it. The `Type`
strings that participate in identity are canonical provider API values and
must remain stable across compatible adapter upgrades.

`DiscoveredSecret.Key` is the canonical fetch key. It must be safe to copy
verbatim into the generated `ExternalSecret` source key for the referenced
store. If a provider needs type information to address the object, that
information must already be encoded in `Key`.

`Name` alone is metadata and does not change source identity. A rename may
update status or annotations if the provider exposes it. Only the canonical key
or type changing alters source identity. In v1, the target naming policy always
derives the final `Secret` name from the discovered `Name`, so a `Name` change
changes the rendered `ExternalSecret` spec and triggers replacement even though source
identity stays the same.

Provider metadata other than the canonical identity fields is used only for
selection or explicit rendering. The controller does not copy provider metadata
into status or events by default.

If the same canonical source identity is discovered twice with equivalent
metadata, the controller deduplicates it. If the identity is the same but the
metadata differs in a way that affects selection or rendering, that is a
provider inconsistency and reconciliation fails closed. The discovery
fingerprint used for dedupe and conflict detection includes only metadata that
can affect selection or rendered `ExternalSecret` shape; observational fields such as
`Version` are excluded.

Rule identity is provenance, not generated `ExternalSecret` identity. If a source stops matching one
rule and starts matching another, the generated `ExternalSecret` is re-rendered from the rule that
matched it in the latest complete discovery cycle. If that render changes
immutable fields, the normal replacement path applies. If it renders the same
`ExternalSecret` spec, the rendered spec is a no-op even though provenance annotations may
still update.

If only the rule provenance changes, the controller updates the provenance
annotation even when the rendered `ExternalSecret` spec is unchanged. That keeps the
observed source of truth aligned with the latest complete cycle.

`Type` is the provider-reported class of the discovered item. A provider that
only exposes one class may use a single constant type for all items. A provider
that exposes multiple classes must include the type in identity and document
what `all: {}` covers for that adapter. Changing the `Type` strings across a
compatible release is a breaking change because `Type` participates in
canonical identity.

The controller stores the full canonical source identity in an annotation and
a fixed-length hash of that identity in a label. It does not rely on labels for
the raw identity because keys can be long, sensitive, or invalid as label
values.

### Controller Sketch

The controller needs to reconcile the desired generated `ExternalSecret`
resources against the generated `ExternalSecret` resources it currently owns.
A small sketch makes that visible:

```go
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    set := &v1alpha1.ExternalSecretSet{}
    if err := r.Get(ctx, req.NamespacedName, set); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    storeUID := r.resolvedStoreUID(set)
    desired := make(map[string]*esv1.ExternalSecret)
    seenSource := make(map[string]struct {
        ruleName    string
        fingerprint string
    })
    seenExternalSecret := make(map[string]string)
    seenTarget := make(map[string]string)

    for _, rule := range set.Spec.Rules {
        remainingBudget := set.Spec.MaxExternalSecrets - len(desired)
        if remainingBudget < 0 {
            remainingBudget = 0
        }

        discovered, err := r.discovery.Discover(ctx, rule, remainingBudget)
        if err != nil {
            return r.failDiscovery(set, rule, err)
        }

        switch discovered.State {
        case DiscoveryStateComplete:
        case DiscoveryStateTooManyResults:
            return r.failTooManyResults(set)
        case DiscoveryStateIncomplete:
            return r.failDiscovery(set, rule, fmt.Errorf("discovery incomplete for rule %q", rule.Name))
        default:
            return r.failDiscovery(set, rule, fmt.Errorf("invalid discovery state %q for rule %q", discovered.State, rule.Name))
        }

        for _, item := range discovered.Items {
            sourceID := r.canonicalSourceIdentity(storeUID, item)
            sourceFingerprint := r.discoveryFingerprint(item)

            if prev, exists := seenSource[sourceID]; exists {
                if prev.ruleName == rule.Name && prev.fingerprint == sourceFingerprint {
                    continue
                }
                if prev.ruleName != rule.Name {
                    return r.failRuleOverlap(set, rule, item, prev.ruleName)
                }
                return r.failDiscovery(set, rule, fmt.Errorf("conflicting metadata for source %q", sourceID))
            }

            externalSecret, err := r.renderExternalSecret(set, rule, item)
            if err != nil {
                return r.failExternalSecret(set, rule, item, err)
            }

            if prevSource, exists := seenExternalSecret[externalSecret.Name]; exists {
                return r.failExternalSecretNameCollision(set, rule, item, prevSource)
            }

            targetName := externalSecret.Spec.Target.Name
            if prevSource, exists := seenTarget[targetName]; exists {
                return r.failTargetCollision(set, rule, item, prevSource)
            }

            seenSource[sourceID] = struct {
                ruleName    string
                fingerprint string
            }{
                ruleName:    rule.Name,
                fingerprint: sourceFingerprint,
            }
            seenExternalSecret[externalSecret.Name] = sourceID
            seenTarget[targetName] = sourceID
            desired[externalSecret.Name] = externalSecret

            if len(desired) > set.Spec.MaxExternalSecrets {
                return r.failTooManyResults(set)
            }
        }
    }

    return r.reconcileExternalSecrets(ctx, set, desired)
}
```

The important part is the control flow:

* resolve the referenced store first
* discover one rule at a time
* render second
* diff third
* create or update desired `ExternalSecret` resources first
* prune stale generated `ExternalSecret` resources last

That order is the default for non-conflicting resources. If a desired generated
`ExternalSecret` would reuse an immutable target name that is still held by a
stale generated `ExternalSecret`, the controller treats that as a replacement:
it deletes the stale resource first and creates the replacement in a later
reconcile.

If any create or update in the desired set fails, the controller stops before
pruning stale generated `ExternalSecret` resources. A partial apply never prunes
in the same reconcile.

No part of that flow needs to flatten sources into a single `Secret`.

### Provider Model

The CRD should not define a universal path hierarchy.

Selectors such as `prefix`, `keyRegexp`, `tags`, or `labels` are provider-native
concepts. A provider adapter maps them to whatever query model that provider
supports.

That means:

* `/` is not special in the CRD.
* `path` is not assumed to exist.
* prefix discovery only works where the provider can express it.
* tag and label discovery only work where the provider can filter on metadata.

`maxExternalSecrets` is a hard ceiling on the retained desired `ExternalSecret` set for the whole `ExternalSecretSet` after exclusions and
deduplication. The controller may pass the remaining global budget to each rule
as a scan hint, but the limit is enforced on the union of all retained results,
not per rule. A provider may honor that hint early if it can do so without
producing false negatives after exclusions. The hint is for cost control only;
it must not suppress a retained item that the adapter can already return merely
to stay under budget, because the controller still needs enough information to
classify overlaps and collisions.

This avoids the mistake of pretending that all secret stores behave the same.

### Naming

The generated `ExternalSecret` and the final Kubernetes `Secret` should
be named deterministically.

Suggested rules:

* The generated `ExternalSecret` name should be derived from the set name plus a
  stable hash or suffix from the canonical source identity.
* The final Kubernetes `Secret` name should preserve the discovered logical
  name, sanitized for Kubernetes rules.
* v1 does not support per-item target-name overrides; the target name comes
  from the discovered logical name only.
* If sanitization causes a collision, reconciliation must fail closed.
* If two rules match the same source, that overlap should be treated
  as a configuration error unless the design later adds explicit precedence.
* The naming algorithm must be deterministic, order-independent, collision-
  resistant, DNS-compliant, and stable across upgrades for a given v1 source
  identity.

The default collision policy should be `Error`, not "guess a name and hope".

The design distinguishes four different things:

* Source identity: resolved store UID + discovered type + discovered key.
* Rule identity: the stable `name` on the rule that produced the item.
* Generated `ExternalSecret` identity: the generated object name and namespace.
* Target identity: the final Kubernetes `Secret` name rendered by the generated `ExternalSecret`.

Changing the store changes source identity. Changing the target name does not
change source identity, but it does change the rendered `ExternalSecret` spec
and may require replacement because the generated target name is treated as
immutable in v1.

The controller never parses the discovered key to derive the logical target
name. It uses the discovered `Name` field, sanitized for Kubernetes rules. If
a provider does not expose a separate logical name, the adapter must synthesize
one and place it in `Name` explicitly.

### `externalSecret` Configuration

#### Intrinsic Source Binding

Each rule produces one generated `ExternalSecret` for each retained
canonical source identity.

In v1, the controller binds `DiscoveredSecret.Key` intrinsically when it
renders the generated `ExternalSecret`. The user does not configure a second
source reference, and the controller does not derive the binding from payload
shape, tags, or naming heuristics.

`externalSecret.data` means the discovered source is exposed as one key in the
target Kubernetes `Secret`:

```yaml
externalSecret:
  data:
    secretKey: value
```

The renderer produces one `ExternalSecret.spec.data` entry and sets
`remoteRef.key` to the discovered canonical fetch key.

`externalSecret.properties: {}` means the discovered source is exposed as a
property set:

```yaml
externalSecret:
  properties: {}
```

The renderer produces one `ExternalSecret.spec.dataFrom.extract` entry and sets
`extract.key` to the discovered canonical fetch key.

Exactly one of `data` or `properties` is required. A generated
`ExternalSecret` may not introduce a second provider-backed source, and v1 does
not expose a per-property selector list.

Each rule should describe one homogeneous desired `ExternalSecret` shape. If a
selector covers sources that need different shapes, split it into multiple
named rules. The controller does not branch on secret content or guess
transformations per item.

That gives the v1 API a positive, narrow shape:

* one named rule for one class of sources
* one `externalSecret` configuration for every source retained by that rule
* one provider-backed source identity per generated `ExternalSecret`
* exactly one of `externalSecret.data` or `externalSecret.properties`

The v1 `externalSecret` schema exposes only fields the user can meaningfully
control for every generated `ExternalSecret`: refresh behavior, the desired
data shape, and supported target configuration. It does not embed an arbitrary
`ExternalSecretSpec` and does not expose `secretStoreRef`, `sourceRef`,
`dataFrom.find`, generator refs, `target.manifest`, or `creationPolicy`.

The renderer maps the selected `data` or `properties` shape onto the existing
`ExternalSecret` reconciliation paths and always renders
`creationPolicy: Owner`. The generated target is a Kubernetes `Secret`;
non-Secret targets are out of scope for v1.

### Behavior

The reconcile loop should be:

1. Resolve the referenced store and capture its UID.
2. Validate that the resolved store supports the selected discovery strategies.
3. Discover one rule at a time through the provider adapter.
4. Require the adapter to evaluate the rule's `match` and `exclude` clauses
   before returning results, then deduplicate exact repeated identities.
5. Reject duplicate or colliding results before creating new generated `ExternalSecret` resources.
6. Diff the desired set of generated `ExternalSecret` resources against the existing owned ones.
7. Create or update desired `ExternalSecret` resources first, then prune stale ones.
8. Update bounded aggregate status and the `Reconciled` condition.

Reconciliation is a complete discovery cycle. The controller only prunes stale
generated `ExternalSecret` resources after discovery completes successfully and the full desired set
can be rendered. The cycle is not a point-in-time transaction over the
provider; different rules may be observed sequentially, and the controller only
treats the resulting complete cycle as authoritative for pruning. An incomplete
discovery result, pagination failure, timeout, auth error, or
`maxExternalSecrets` overflow is treated as a reconciliation failure, not as
evidence that previously discovered sources disappeared.

Discovery identifies stable source identities within the referenced store. The
controller ignores ordering and deduplicates repeated results that resolve to
the same canonical identity. Distinct identities that collapse to the same
generated `ExternalSecret` name or target name are collisions and must fail closed. If the same
canonical identity appears more than once with conflicting metadata, that is an
error, not a tie-break. Metadata such as labels, tags, disabled flags, or
expiration markers may help the provider adapter filter or annotate results,
but they do not replace identity and they do not by themselves force deletion
or recreation.

If the same canonical identity is repeated within the same rule or across pages
of the same rule, that is a duplicate and should be deduplicated. If two
distinct rules produce the same canonical identity in the same reconcile, that
is overlap and must fail closed.

The `ExternalSecretSet` controller does not infer payload shape from discovery metadata. A
source becoming renamed, disabled, expired, or otherwise reclassified is only
meaningful if the provider adapter exposes that state as part of a complete
discovery cycle. Otherwise the controller treats it as ordinary provider state
and leaves the generated-resource mapping alone. A rename that keeps the canonical key and
type is metadata only for source identity, but in v1 the target naming policy
derives the target name from `Name`, so the rendered `ExternalSecret` spec changes and
replacement is required when the target field is immutable.

If supported provider metadata participates in controller-owned rendering,
changes to that metadata update the rendered `ExternalSecret` spec. Metadata used only for
selection affects membership but not the rendered `ExternalSecret` spec.

The controller is level-based. If the same source rotates 20 times between two
reconciles, only the latest successful discovery cycle matters. If the source
changes only in value, the generated `ExternalSecret` spec stays the same and
the existing `ExternalSecret` controller reconciles the updated value into the
target `Secret` according to that resource's own refresh semantics. If the
source changes identity or disappears from a complete discovery cycle, the old
generated `ExternalSecret` is removed, and a new one is created only when the
next successful discovery cycle says that source is present.

The controller should re-check the parent generation and resolved store UID
after discovery and before any apply or prune. A stale reconcile must not
mutate generated `ExternalSecret` resources if the parent spec or referenced store changed while discovery
was running.

A change to `secretStoreRef` changes source identity and requires a new desired
set. A change to a rendered target name does not change source identity, but it
is still a replacement event because the generated target name is treated as
immutable in v1. The controller handles that as delete-then-recreate rather
than an in-place mutation.

The rendered `ExternalSecret` is the unit of reconciliation. Manual edits to
controller-owned fields on a generated `ExternalSecret` are converged back to
the rendered spec. Manual edits to the final Kubernetes `Secret` are governed by the
existing `ExternalSecret` semantics, not by the set controller. The set
controller does not manage the target `Secret` directly.

The generated `ExternalSecret` spec is controller-owned end to end. The
controller records the parent UID, rule name, canonical source identity, and
`renderedExternalSecretRevision` in controller-managed labels or annotations so
it can rebuild desired state even if status is lost.
`renderedExternalSecretRevision` is a deterministic hash of the normalized
controller-owned rendered `ExternalSecret` spec after the intrinsic source
binding, selected `data` or `properties` shape, and target name are applied. It
does not include bookkeeping-only provenance annotations, which may update
independently. The generated `ExternalSecret` name is not the only source of
truth.

In v1, pruning the generated `ExternalSecret` is also how the final Kubernetes
`Secret` is removed, because the controller always renders
`creationPolicy: Owner` on the generated `ExternalSecret`. That parent-pruning
path is authoritative for removing a source from the desired set and therefore
removing the generated resource. The generated `ExternalSecret`'s own
`deletionPolicy` independently governs what it does when it observes a remote
source disappear before the parent has pruned it. The two paths are separate and may
converge on the same eventual cleanup, but they do not have identical timing or
scope.

The controller should be conservative:

* If discovery reports `DiscoveryStateTooManyResults`, stop and report a
  `TooManyResults` reason on the `Reconciled` condition. The limit is counted on
  the retained desired set after exclusions and deduplication, but the adapter
  should honor the budget hint early if it can.
* If discovery reports `DiscoveryStateIncomplete`, stop and report
  `DiscoveryFailed`.
* If one item fails to render, report the failure through the bounded status,
  condition message, and an event without expanding a per-resource collection in
  parent status.
* If rendering fails before the desired set is complete, abort before any
  create, update, or delete. Partial sync only applies after a complete desired
  set has been rendered and the apply phase then fails for some generated `ExternalSecret` resources.
* If discovery or rendering fails before the desired set is complete, do not
  prune existing generated `ExternalSecret` resources.
* If some desired `ExternalSecret` resources cannot be converged,
  report `PartialSync` rather than hiding the failure.
* If two rules, name normalizations, or target policies collide, fail
  closed rather than guessing which object should win.
* If the rendered target `Secret` already exists and is not owned by a
  generated `ExternalSecret` of this `ExternalSecretSet`, fail closed and do
  not adopt it. If the target is owned by a stale generated `ExternalSecret` of
  this set, the replacement path applies instead.
* If a desired generated `ExternalSecret` name already exists but is not owned by the current set,
  treat that as a collision and do not adopt it implicitly.
* If two `ExternalSecretSet` objects converge on the same exclusive target,
  reject the conflict when it is observed and do not attempt cross-set
  arbitration. This is a best-effort exclusivity contract, not a cluster-wide
  lock.
* If the user deletes the `ExternalSecretSet`, owned generated `ExternalSecret`
  resources should be cleaned up by owner references. Add a finalizer only if
  the controller needs cleanup that garbage collection cannot provide.
* Create and update desired `ExternalSecret` resources before
  pruning stale ones. If a generated `ExternalSecret` needs replacement because
  an immutable field changed, delete the old resource and let a later reconcile
  recreate it.
* If the parent generation or referenced store UID changes before mutation
  begins, abort the reconcile and do not advance `observedGeneration` or mark
  the attempt as completed.

The `ExternalSecretSet` controller should manage the generated `ExternalSecret` objects, not the
final Kubernetes `Secret` objects directly.

That keeps the existing `ExternalSecret` ownership and deletion semantics in
one place.

### Version Handling

If the discovery adapter can resolve a provider version, that version may be
stored in the discovered item for observability. The default generated
`ExternalSecret` leaves the version unset so it follows the provider's normal
current/latest behavior.

If the provider does not expose a version concept, the generated
`ExternalSecret` leaves the version unset and lets the provider's default
semantics apply.

The controller should not invent a version, and v1 does not provide a version
pinning mode. If a later API wants pinning, it needs an explicit field and a
separate compatibility story.

### Status

Status should summarize the currently observed reconciliation state without
expanding the generated `ExternalSecret` collection into the parent object.

Recommended status shape:

* `observedGeneration`: the `metadata.generation` most recently observed by a
  completed reconciliation of the `ExternalSecretSet` spec
* `conditions`: standard `metav1.Condition` values, with a top-level
  `Reconciled` condition and reasons such as `PartialSync`, `DiscoveryFailed`,
  and `TooManyResults`
* `discoveredSources`: number of retained discovered source identities observed
  for the reported reconciliation
* `desiredExternalSecrets`: number of generated `ExternalSecret` objects in the
  desired set after normalization and deduplication
* `managedExternalSecrets`: number of desired `ExternalSecret` objects
  currently owned by the set and converged to the rendered controller-owned
  spec
* `failedExternalSecrets`: number of desired `ExternalSecret` objects that
  could not be converged in the reported reconciliation

The conditions field uses the standard Kubernetes condition schema and
map-style list semantics:

```go
// +listType=map
// +listMapKey=type
// +optional
Conditions []metav1.Condition `json:"conditions,omitempty"`
```

Per-`ExternalSecret` detail lives on the generated resources rather than being
copied into parent status. Generated `ExternalSecret` resources carry
controller-managed labels and annotations for parent identity, rule provenance,
canonical source identity hash, and rendered revision, so users and tooling can
drill down with ordinary Kubernetes list/select operations.

`Reconciled=True` means the latest complete discovery cycle was rendered and
the resulting generated `ExternalSecret` specs were converged successfully.
It does not mean the remote provider stayed unchanged after that cycle, and it
does not aggregate the readiness or freshness of each generated
`ExternalSecret`'s target `Secret`.

When `Reconciled=False`, the `reason` explains whether reconciliation failed due
to `DiscoveryFailed`, `PartialSync`, `TooManyResults`, or another explicit
failure mode. The design does not use a phase enum.

Counts that depend on a complete discovery cycle are authoritative only when
discovery completed successfully. The status remains bounded by design.
Historical, per-attempt detail belongs in Events, logs, and metrics rather than
an ever-growing status structure.

Concrete status example:

```yaml
status:
  observedGeneration: 17
  conditions:
    - type: Reconciled
      status: "False"
      observedGeneration: 17
      reason: PartialSync
      message: "2 of 3 desired ExternalSecrets are managed"
      lastTransitionTime: "2026-08-11T12:30:00Z"
  discoveredSources: 3
  desiredExternalSecrets: 3
  managedExternalSecrets: 2
  failedExternalSecrets: 1
```

### Interaction With Other ESO Concepts

This resource should stay namespaced.

If users need cluster-wide fan-out plus source discovery, that should be a
separate design decision. In v1, `ExternalSecretSet` should not grow into a
`ClusterExternalSecretSet`.

Likewise, this feature should not change the semantics of standard
`ExternalSecret`, `ClusterExternalSecret`, or `PushSecret`.

## Consequences

* **Better separation of concerns**: discovery fan-out becomes its own
  controller, while `ExternalSecret` keeps owning one target Secret per
  object.
* **More explicit provider behavior**: each provider must advertise which
  discovery strategies it supports.
* **Less magical templating**: the design avoids nested template engines and
  keeps the per-rule generated `ExternalSecret` shape explicit.
* **Better observability**: the controller can report aggregate counts while
  preserving per-`ExternalSecret` drill-down through generated resources.
* **More controller code**: discovery, diffing, and cleanup are new moving
  parts.

## Drawbacks

* Not every provider can support discovery efficiently.
* Some users will need multiple rules instead of one broad wildcard.
* Heterogeneous secret sets are not automatically handled inside one rule.
* A broad selector can still generate too many resources, so the design needs
  limits and clear failure modes.
* A valid but empty discovery cycle can still prune many generated
  `ExternalSecret` resources; v1 does not add a second in-API `maxPrune` gate.
* Explicit whole-store discovery can enumerate every visible source name, so
  providers must document that capability carefully.
* The fan-out can create a thundering herd unless providers and controllers
  add their own rate limiting and jitter.
* Immutable target replacement can create a brief availability gap for the
  final `Secret` when the old generated `ExternalSecret` must be removed before
  the replacement can be admitted.
* The feature is more operationally expensive than a single `ExternalSecret`,
  because the system now manages a set of generated `ExternalSecret` resources.

## Acceptance Criteria

* behavior:
  * `ExternalSecret` behavior remains unchanged.
  * `ExternalSecretSet.spec.refreshInterval` controls discovery cadence for the
    parent controller, while `rules[].externalSecret.refreshInterval` controls the
    refresh cadence of the generated `ExternalSecret`.
  * `ExternalSecretSet.spec.refreshInterval: 0` disables periodic parent
    discovery; the controller still reconciles on create/update events and
    explicit requeues, but it does not schedule time-based rediscovery.
  * `maxExternalSecrets` is a hard ceiling on the retained desired
    `ExternalSecret` set for the whole `ExternalSecretSet`, not per rule.
  * `rules` is a map-style list keyed by rule `name`; rule identity is stable
    across list reordering and Server-Side Apply updates.
  * `match` and `all` are mutually exclusive discovery variants, and whole-store
    discovery uses the explicit `all: {}` form.
  * `rules[].externalSecret` contains a typed union in v1; exactly one of `data`
    or `properties` must be set and neither variant is defaulted.
  * `externalSecret.data` renders one `ExternalSecret.spec.data` entry;
    `externalSecret.properties` renders one `ExternalSecret.spec.dataFrom.extract`
    entry. The controller does not infer the choice from payload shape or key format.
  * every generated `ExternalSecret` has an intrinsic source binding derived from the
    discovered key; it is not separately configurable by the user.
  * every provider-backed `remoteRef.key` or `dataFrom.extract.key` in the
    generated `ExternalSecret` equals the discovered key; generator-backed refs are not
    exposed by the v1 `externalSecret` schema.
  * the v1 `externalSecret` configuration is a positive, narrow schema and does not embed an
    arbitrary `ExternalSecretSpec`; it does not expose a `ExternalSecret` store override,
    `sourceRef`, `dataFrom.find`, `generatorRef`, `target.manifest`, or
    `creationPolicy`.
  * generated `ExternalSecret` resources are always rendered with `creationPolicy: Owner` in v1.
  * discovery is a complete discovery cycle, and only complete, successful
    cycles may drive pruning.
  * partial discovery failures, page failures, timeouts, auth errors, and
    `maxExternalSecrets` overflow never delete existing generated `ExternalSecret` resources.
  * discovery results report exactly one state: `Complete`, `TooManyResults`, or
    `Incomplete`.
  * `TooManyResults` is reported as reason `TooManyResults`, while `Incomplete`
    is reported as `DiscoveryFailed`.
  * discovery results are deterministic and order-independent once a complete
    normalized discovery cycle has been collected.
  * source identity is scoped to the resolved store UID plus the discovered type
    plus the discovered key, not just the raw key.
  * `Type` values that participate in canonical identity are stable canonical
    provider API values and must not change across compatible adapter upgrades.
  * `DiscoveredSecret.Key` is a canonical fetch key; if a provider needs type
    information to address an object, that information is already encoded in
    `Key`.
  * source `Name` changes alone do not change source identity; in v1 target
    naming always uses `Name`, so such changes alter the rendered `ExternalSecret` and
    trigger replacement.
  * source key or type changes become delete-plus-create only after the next
    successful discovery cycle.
  * value-only changes do not alter the generated `ExternalSecret` spec; the existing
    `ExternalSecret` controller propagates them according to its own refresh
    semantics.
  * v1 does not provide version pinning; discovered versions are observational
    only and are excluded from the discovery fingerprint.
  * provider adapters validate unsupported discovery strategies explicitly.
  * rules have stable, unique names and are reconciled independently.
  * repeated canonical identities within one rule are deduplicated; the same
    canonical identity produced by two rules in the same reconcile is overlap
    and fails closed.
  * naming collisions and overlapping rules fail closed.
  * target names of generated `ExternalSecret` resources are derived from the discovered logical name
    and sanitized for Kubernetes rules; the controller does not parse the key to
    invent a target name. v1 does not support per-item target-name overrides.
  * if the rendered target Secret already exists and is not owned by a generated `ExternalSecret` of
    this `ExternalSecretSet`, the controller fails closed and does not adopt it.
    If the target is owned by a stale generated `ExternalSecret` of this set, the replacement path
    applies instead.
  * desired generated `ExternalSecret` names that already exist but are not owned by the current set
    are collisions and are never adopted implicitly.
  * conflicting `ExternalSecretSet` objects do not coordinate ownership of the
    same exclusive target Secret; observed conflicts are rejected rather than
    adopted.
  * controller-owned fields on generated `ExternalSecret` objects,
    including the entire generated `ExternalSecret` spec, converge back to desired state.
  * partial apply failures never prune stale generated `ExternalSecret` resources in the same reconcile.
  * manual edits to the final `Secret` continue to follow existing ESO
    semantics.
  * generated `ExternalSecret` cleanup is idempotent.
  * `Reconciled=True` means the latest complete discovery cycle was rendered and
    the desired generated `ExternalSecret` specs were converged successfully; it does not aggregate
    generated `ExternalSecret` readiness.
  * `status.conditions` uses `metav1.Condition` and map-style list semantics
    keyed by condition `type`.
  * `status.observedGeneration` is the `metadata.generation` most recently
    observed by a completed reconciliation of the parent spec.
  * `observedGeneration` does not advance for a reconcile that aborts before
    mutation because the parent generation or resolved store UID changed.
  * parent status remains bounded and uses explicit counters such as
    `discoveredSources`, `desiredExternalSecrets`, `managedExternalSecrets`, and
    `failedExternalSecrets`; per-`ExternalSecret` detail lives in generated `ExternalSecret` resources.
* rollout:
  * the feature should be gated or alpha until at least one provider adapter and
    the status model are implemented.
  * docs must list which discovery strategies are supported by each provider.
* tests:
  * controller tests for discovery, diffing, collision detection, exclusions,
    cleanup, and list-order independence.
  * regression tests for partial sync, standard condition handling, and status
    reporting.
  * e2e coverage for at least one provider with explicit discovery support.
  * e2e coverage for a successful empty discovery cycle that prunes all
    previously managed `ExternalSecret` resources.
* docs:
  * API/CRD spec inline documentation.
  * design doc in `design/`.
  * user guide for discovery fan-out and its limitations.

## Alternatives

### Extend `dataFrom.find` To Create Many Resources

Rejected.

`dataFrom.find` already means "many inputs to one secret". Changing it to emit
resources would break the current mental model and overload a field that is
already well understood.

### Add Wildcards Directly To `ExternalSecret.remoteRef.key`

Rejected.

That would still leave the controller with one target `Secret`, which is the
wrong level for the problem. It also keeps the design tied to a fake path
hierarchy that does not exist uniformly across providers.

### Generate `Secret` Directly

Rejected.

That duplicates the existing `ExternalSecret` reconciliation logic and skips
the common ownership, templating, and lifecycle behavior already present in
ESO.

### Add A Cluster-Scoped Variant First

Rejected for v1.

Cluster scope adds another axis of complexity before the core discovery model
is proven. The namespaced controller is enough to validate the resource model
and the provider contract.
