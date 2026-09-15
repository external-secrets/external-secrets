```yaml
---
title: Workflows: declarative multi-step secret pipelines
version: v1alpha1
authors: alekc
creation-date: 2026-08-27
status: draft
---
```

# Workflows: declarative multi-step secret pipelines

## Table of Contents

<!-- toc -->
// autogen please
<!-- /toc -->

## Summary

Add a `Workflow` CRD that chains `pull`, `transform`, and `push` steps into
ordered jobs, so operators can express "copy these secrets from store A to
store B, remapping keys and paths along the way" declaratively, instead of
hand-maintaining one `PushSecret` rewrite rule per item or one
`ExternalSecret`/`PushSecret` pair per folder.

## Motivation

`ExternalSecret`'s `find` mode returns matched secrets keyed only by their
own name, the source path is discarded before it reaches the target
`Secret`. `PushSecret` can already write to any destination path it's
handed, the gap is specifically that `find` has nowhere to carry a source
path forward (filed as #6873). That's why mirroring a folder tree between
two stores today needs one hand-written `PushSecret` rewrite rule per
folder, our own 18-folder (40-secret) Infisical migration needed 18 such
blocks, each with the folder name baked into a regex and a template.

Beyond path preservation, this project has no answer at all for filtering
by criteria other than a path prefix (tag, label, cross-provider),
transforming values in flight, or branching on what was found. `PushSecret`
alone can't express "pull everything tagged `env:prod` from 1Password and
push it to both GitHub and Vault," it only pushes data already landed in
one `Secret`.

There's prior art worth building on: the unreleased
`external-secrets-inc/external-secrets-enterprise` repo shipped a
`Workflow`/`WorkflowRun` pair modeling exactly this problem. The shape is
good, but it has a real security defect in how it handles step output (see
Behavior), so this RFC proposes an open, narrower, corrected version rather
than adopting it as-is.

### Goals

- Express "copy/transform secrets from store A to store B" declaratively,
  without per-item hand enumeration, for the cases that don't need full
  automatic path preservation (see Non-Goals).
- Reuse existing ESO types (`SecretStoreRef`, `ExternalSecretData`,
  `ExternalSecretDataFromRemoteRef`, `PushSecretData`) rather than a
  parallel object model chart authors have to learn twice.
- Guarantee no plaintext secret material is ever persisted anywhere outside
  a real Kubernetes `Secret` object, not in a CRD's `.status`, not in logs.
- Let operators choose whether one job's failure aborts the whole
  Workflow or only its own dependency chain (see Failure Handling).

### Non-Goals

- Not a general-purpose workflow/CI engine, no arbitrary shell execution.
  Whether a scoped, sandboxed script step belongs at all is deferred.
- Not proposing to merge the Enterprise codebase as-is, it has an open
  defect (see Behavior) that needs a design answer, not a patch.
- Does not by itself solve automatic, structure-preserving recursive copy,
  that needs #6873 landing first. Until then, only explicit `data[]`
  entries (one `remoteRef` per secret) preserve source path through a
  workflow, the same trade-off `PushSecret` already has today.

## Proposal

### User Stories

- As an operator migrating between two instances of the same provider (my
  own case: Infisical cloud to self-hosted), I want to declare per-folder
  or per-secret copy rules once, with real dependency ordering and
  templated destination paths, instead of one `PushSecret` with N
  hand-written `match`/`rewrite` blocks that drift as secrets are added.
- As an operator syncing a subset of secrets by tag from one provider to
  several destinations, I want to pull once and push to N destinations
  declaratively. This is also the natural shape for a gradual
  provider-to-provider migration: keep old and new destinations in sync
  from one pipeline while consumers cut over at their own pace.
- As an operator composing a destination secret from values that live in
  different stores (a hostname in one, a password in another), I want to
  pull from each and combine them with a `transform` step. Plain
  `ExternalSecret` can only template from data pulled through one store.
- As a security reviewer, I want a guarantee that running a Workflow never
  leaves secret plaintext readable by anyone with `get` on the Workflow
  object itself.

### API

Sketched from the Enterprise prior art:

```yaml
apiVersion: workflows.external-secrets.io/v1alpha1
kind: Workflow
metadata:
  name: mirror-gpg-folder
spec:
  jobs:
    pullFromSource:
      standard:
        steps:
          - name: fetch
            pull:
              source:
                storeRef: { name: source-store }
              data:
                - secretKey: GPG_HELM_SIGNING_KEY
                  remoteRef: { key: gpg/GPG_HELM_SIGNING_KEY }
                - secretKey: GPG_HELM_SIGNING_PASSPHRASE
                  remoteRef: { key: gpg/GPG_HELM_SIGNING_PASSPHRASE }
    pushToDestination:
      dependsOn: [pullFromSource]
      standard:
        steps:
          - name: push
            push:
              secretSource: ".global.jobs.pullFromSource.fetch"
              destination:
                storeRef: { name: destination-store }
              data:
                - match:
                    secretKey: GPG_HELM_SIGNING_KEY
                    remoteRef: { remoteKey: "gpg/GPG_HELM_SIGNING_KEY" }
                - match:
                    secretKey: GPG_HELM_SIGNING_PASSPHRASE
                    remoteRef: { remoteKey: "gpg/GPG_HELM_SIGNING_PASSPHRASE" }
```

`pull.data[]` and `push.data[]` reuse `ExternalSecretData` and
`PushSecretData` verbatim; `pull.dataFrom` reuses
`ExternalSecretDataFromRemoteRef` the same way (with the #6873 caveat from
Non-Goals). A step addresses another step's output as
`.global.jobs.<job>.<step>.<key>`, used by both `secretSource` and
`transform` mappings.

`push.dataTo[]` (new in this RFC, reusing `PushSecretDataTo` verbatim) is
what a dynamic, `find`-based push needs: `dataTo[].rewrite[]` renames each
matched key before writing, without requiring the key set to be known up
front the way `data[]` does. `storeRef` is dropped, a `push` step already
names one destination in `destination.storeRef`.

Tag-based fan-out (User Stories) is `dataFrom.find` plus two `push` jobs
sharing one `pull` job, `ExternalSecretFind` already combines a path
prefix and a tag selector in one `find` block:

```yaml
apiVersion: workflows.external-secrets.io/v1alpha1
kind: Workflow
metadata:
  name: sync-prod-tagged-secrets
spec:
  jobs:
    pullTagged:
      standard:
        steps:
          - name: fetch
            pull:
              source:
                storeRef: { name: source-store }
              dataFrom:
                - find:
                    path: "app/"
                    tags:
                      env: prod
    pushToGithub:
      dependsOn: [pullTagged]
      standard:
        steps:
          - name: push-github
            push:
              secretSource: ".global.jobs.pullTagged.fetch"
              destination:
                storeRef: { name: github-store }
              dataTo:
                - {}
    pushToVault:
      dependsOn: [pullTagged]
      standard:
        steps:
          - name: push-vault
            push:
              secretSource: ".global.jobs.pullTagged.fetch"
              destination:
                storeRef: { name: vault-store }
              dataTo:
                - rewrite:
                    - regexp:
                        source: "(.*)"
                        target: "vault-mirror/app/$1"
```

`pullTagged` runs once; `pushToGithub` and `pushToVault` both depend on it
and run independently, reading the same in-memory result rather than
re-querying the source. `pushToGithub`'s `dataTo: [{}]` is the minimal
bulk push, every key under its own name; `pushToVault` rewrites the same
keys under `vault-mirror/app/`. Structure-preserving fan-out (recovering
the *source* layout automatically) still needs #6873.

Cross-store composition (User Stories) needs a fourth shape: two
independent `pull` jobs, a `transform` step waiting on both, and a `push`
reading the transform's output rather than either pull directly:

```yaml
apiVersion: workflows.external-secrets.io/v1alpha1
kind: Workflow
metadata:
  name: compose-database-url
spec:
  jobs:
    pullHost:
      standard:
        steps:
          - name: fetch
            pull:
              source:
                storeRef: { name: inventory-store }
              data:
                - secretKey: DB_HOST
                  remoteRef: { key: infra/app-db/host }
    pullCreds:
      standard:
        steps:
          - name: fetch
            pull:
              source:
                storeRef: { name: vault-store }
              data:
                - secretKey: DB_USER
                  remoteRef: { key: app-db/DB_USER }
                - secretKey: DB_PASSWORD
                  remoteRef: { key: app-db/DB_PASSWORD }
    combine:
      dependsOn: [pullHost, pullCreds]
      standard:
        steps:
          - name: merge
            transform:
              mappings:
                DATABASE_URL: >-
                  postgres://{{ .global.jobs.pullCreds.fetch.DB_USER }}:{{ .global.jobs.pullCreds.fetch.DB_PASSWORD }}@{{ .global.jobs.pullHost.fetch.DB_HOST }}:5432/app
    pushCombined:
      dependsOn: [combine]
      standard:
        steps:
          - name: push
            push:
              secretSource: ".global.jobs.combine.merge"
              destination:
                storeRef: { name: app-secrets-store }
              data:
                - match:
                    secretKey: DATABASE_URL
                    remoteRef: { remoteKey: "app/DATABASE_URL" }
```

`pullHost` and `pullCreds` have no dependency on each other and run
concurrently; `combine`'s `dependsOn` forces the wait. `transform` steps
read prior outputs through the same addressing and produce a new named
output from a Go template. `pushCombined` addresses `combine`'s output
exactly like a `push` step addresses a `pull`'s, any step with named
outputs is a valid `secretSource`. This is the concrete answer to
"`ExternalSecret` can only template from one store."

Deliberately deferred out of v1alpha1, revisit once the core pipeline
model has shipped and been used:

- `javascript` step: arbitrary code execution inside the controller is a
  separate security review (sandboxing, resource limits, supply chain of
  whatever JS runtime is embedded), it shouldn't gate the rest of this.
- `generator` step: overlaps with the existing `ClusterGenerator` +
  `PushSecret` combination; propose only if a concrete gap in that
  combination shows up in practice.

### Behavior

**No step's raw output is ever written to `.status`.** The prototype
persists every step's output there (etcd-stored, `kubectl get`-readable by
anyone with `get` on the object) as its inter-step data channel, and its
masking safeguard for sensitive values doesn't hold up in practice, so
that pattern would ship every plaintext secret a Workflow touches into
etcd, unmasked, for as long as the object exists.

This RFC keeps inter-step data in memory within a single reconcile
instead, recomputed fresh each time, the same way a plain
`ExternalSecret`'s pull result never touches `.status` today. `.status`
carries only non-sensitive metadata: phase, timing, counts, and
step-level failure messages.

### Conflict Handling

Three separate conflict questions, three separate answers:

- **Within one `push` step's `data[]` + `dataTo[]`**, reuse `PushSecret`'s
  own resolution unmodified: a key claimed by both resolves silently
  (explicit `data[]` wins); two different source keys resolving to the
  same destination is a hard reconcile error, not a silent overwrite.
- **Against a secret that already exists at the destination**, reuse
  `PushSecretSpec.UpdatePolicy` (`Replace`/`IfNotExists`) and
  `DeletionPolicy` verbatim, exposed per `push` step.
- **Across `push` steps in the same Workflow**, there's nothing to reuse:
  two independent `PushSecret`s targeting the same key today aren't
  coordinated at all, whichever reconciles last wins. That's easy to hit
  by accident inside one Workflow (two jobs against the same store, a
  copy-pasted rewrite producing the same prefix), so this RFC proposes
  rejecting the Workflow at admission when two `push` steps resolve to the
  same destination, rather than letting them race silently (see
  Acceptance Criteria).

### Failure Handling

The prototype treats any job failure as fatal to the whole Workflow, even
unrelated jobs never finish and every downstream job is left permanently
stuck, with no distinction between a transient error (rate-limited, a
network blip) and a terminal one (not found, bad auth). This RFC changes
both:

- **Retryable vs terminal**, before a job is ever marked failed: a step's
  own error is retried with backoff, the same way `ExternalSecret`/
  `PushSecret` already treat a store being briefly unreachable as
  recoverable rather than fatal. Only an error that isn't recoverable, or
  that exhausts retries, fails the job.
- **`spec.failurePolicy`** (new field, default `FailFast`) chooses what a
  terminal job failure does to the rest of the Workflow. `FailFast`
  matches the prototype's current behavior: one failure stops the whole
  Workflow immediately. `ContinueOnFailure` lets every job outside the
  failed job's dependency chain run to completion; the Workflow's overall
  phase becomes `PartiallyFailed` rather than `Failed`, and
  `status.jobStatuses` records the mix.
- **A job downstream of a failed job never runs, under either policy**,
  `dependsOn` still gates it. `failurePolicy` only changes whether
  *unrelated* jobs get to finish, not whether a dependent one does.

### Drawbacks

- New CRDs are new surface area: more to test, document, and support
  long-term. Needs a clear answer to "why not just PushSecret" so users
  aren't left guessing which mechanism to reach for.
- Overlaps conceptually with `PushSecret`'s own `data[]`/`dataTo[]`
  rewrite mechanism for the simple cases; the boundary between "add one
  more PushSecret rewrite rule" and "reach for a Workflow" needs to be
  documented clearly or this becomes two ways to do the same thing.
- The status-handling fix above is a real design decision requiring its
  own review, not a mechanical port of the prototype.
- `ContinueOnFailure` means a Workflow can end up `PartiallyFailed`, a
  third terminal state beyond `Succeeded`/`Failed` that every consumer of
  workflow status (dashboards, alerts, `kubectl` output) now has to know
  about.

### Acceptance Criteria

- Rollout: ship behind a feature flag per the existing convention
  (design/014-feature-flag-consolidation.md), additive CRD, no existing
  resource type touched, straightforward to disable.
- Tests: unit tests per step executor; both `failurePolicy` values,
  including that a downstream job stays un-run under `ContinueOnFailure`
  and that retries distinguish a recoverable provider error from a
  terminal one; and one non-negotiable regression test asserting no
  secret plaintext ever appears in a `Workflow`'s `.status`.
- Observability: `status.phase`/`jobStatuses`/`stepStatuses` for
  non-sensitive progress, events on job/step failure, run duration and
  success/failure metrics, following existing ESO conventions.
- Troubleshooting: `kubectl describe workflow` surfaces step-level failure
  messages without ever needing a secret value to diagnose what went
  wrong.
- Validation: admission-time rejection when two `push` steps in the same
  `Workflow` resolve to the same destination, per Conflict Handling,
  failing before either write happens rather than at runtime.

## Alternatives

- **Status quo.** Keep requiring hand-written `PushSecret` rewrite rules
  or one `ExternalSecret`/`PushSecret` pair per folder for any
  path-remapping copy. Doesn't scale past a handful of items, our own
  case needed 18 hand-written blocks for one migration.
- **Fix only #6873**, no Workflow layer. Solves automatic recursive copy
  for same-shape source-to-destination mirrors, doesn't help cross-provider
  syncs, tag-based selection, or any transform in between. The
  orchestration model is useful independent of that fix landing.
- **Adopt the Enterprise code as-is**, license question aside. Fastest
  path to something running, but ships the status-persistence defect
  described above. Treat that code as prior art and a source of test
  cases, not something to merge unreviewed.
