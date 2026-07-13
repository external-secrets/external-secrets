<!--
SYNC IMPACT REPORT
==================
Version change: 1.0.0 → 1.1.0 (MINOR — Principle V replaced with new Security and
Compliance principle; documentation/contracts obligation absorbed into Engineering Standards)

Modified principles:
  - V. Documentation and Contracts Stay in Sync → V. Security and Compliance
    (prior documentation obligation moved to Engineering Standards section)

Added sections:
  - None (Security and Compliance replaces an existing principle slot)

Removed sections:
  - None

Templates requiring updates:
  ✅ .specify/templates/plan-template.md — "Documentation and contracts" Constitution
     Check gate updated to "Security and compliance"
  ✅ .specify/templates/spec-template.md — QR-005 security/compliance requirement added;
     QR-003 documentation requirement retained
  ✅ .specify/templates/tasks-template.md — Security compliance task added to Foundational
     phase; Polish phase security hardening task expanded
  ⚠  No .specify/templates/commands/ directory found — no command templates to check

Follow-up TODOs:
  - TODO(COMMANDS_DIR): .specify/templates/commands/ does not exist; verify or create when
    command templates are added to the project.
-->

# External Secrets Constitution

## Core Principles

### I. Code Quality Is a Release Gate

All production changes MUST be small enough to review, explicit in intent, and
maintainable by contributors who did not author them. Changes MUST prefer
existing repository patterns over new abstractions, MUST include clear error
handling for non-happy paths, and MUST update user-facing or maintainer-facing
documentation when behavior changes. Dead code, speculative abstractions, and
silent behavior changes are not acceptable trade-offs for speed.

Rationale: External Secrets is a long-lived Kubernetes operator with many
providers and contributors. Readability and predictable maintenance cost are
required to keep changes safe across controllers, providers, CRDs, and release
artifacts.

### II. Testing Standards

Every behavior change MUST be verified by automated tests at the lowest useful
level and expanded to integration, contract, or end-to-end coverage when the
change crosses package, API, controller, CRD, or provider boundaries. Bug fixes
MUST begin with a test that reproduces the failure before the fix is written.
A task list or implementation plan is incomplete if it cannot state how the
changed behavior will be proven before merge. Tests are not optional.

Rationale: Reconciliation logic and provider integrations fail in ways that are
easy to miss in review alone. Risk-based automated testing is the minimum
standard for preventing regressions in cluster behavior and secret delivery.

### III. Consistency Beats Novelty

New APIs, CRD fields, controller flows, metrics, events, logging, package
layout, and user-facing terminology MUST follow established repository
conventions unless a documented exception is approved in the implementation
plan. Equivalent concepts MUST use equivalent names and behaviors across
providers and controllers. Deviations for convenience, cleverness, or local
style preference are not sufficient justification.

Rationale: Users and maintainers interact with External Secrets through many
entry points. Consistency reduces operator error, keeps documentation reliable,
and limits maintenance overhead across a broad surface area.

### IV. Performance Requirements Are Part of the Spec

Any feature or change that can affect reconciliation frequency, API traffic,
latency, memory, startup cost, or large-cluster behavior MUST define measurable
performance expectations in the spec or plan. Plans MUST document the expected
cost profile, relevant constraints, and the validation approach. Changes that
may introduce unbounded retries, repeated external calls, excessive list/watch
operations, or avoidable allocations MUST be rejected or redesigned before
implementation.

Rationale: Controller and provider inefficiencies scale directly into cluster
load, secret latency, and cloud API pressure. Performance is a correctness
concern, not a post-release optimization task.

### V. Security and Compliance

This project handles production credentials. Every change that touches secret
retrieval, transmission, storage, logging, or access control MUST satisfy
the following non-negotiable rules:

- Secret and credential values MUST NOT appear in logs, events, status
  conditions, or error messages at any log level.
- TLS certificate validation MUST NOT be disabled (`InsecureSkipVerify: true`
  or equivalent) on any code path reachable in a production deployment. An
  exemption requires an approved issue and MUST be isolated behind an explicit
  opt-in field on the `SecretStore` spec.
- RBAC permissions for the operator's `ServiceAccount` MUST follow
  least-privilege. Any PR that adds or broadens a ClusterRole or Role rule
  MUST include a threat model justification and requires ≥2 maintainer
  approvals.
- All new Go module dependencies MUST be checked for known CVEs before merge.
  CVEs with CVSS ≥7.0 in any direct or transitive dependency MUST be remediated
  within 14 calendar days of public disclosure; CVSS <7.0 within 30 days.
- Container images and release artifacts MUST be signed and include a SBOM and
  provenance attestation, consistent with the project's existing release pipeline.
- Changes that add new external API integrations MUST document the credential
  scope required, where credentials are stored in-memory, and when they are
  discarded.

Rationale: A compromised secrets operator is a full cluster compromise. Security
and compliance obligations are invariants, not review suggestions — violations
block merge regardless of feature value.

## Engineering Standards

- Plans MUST state the impacted packages, APIs, CRDs, docs, and generated
  artifacts.
- Plans MUST document the test strategy, including which unit, integration,
  contract, or end-to-end coverage is required and why.
- Plans MUST document performance goals or explicitly state why performance is
  not materially affected.
- Plans MUST identify secret-handling paths, TLS configurations, RBAC changes,
  and new dependencies introduced by the change, and confirm compliance with
  Principle V.
- Features that change public APIs, CRDs, or provider behavior MUST identify
  backward-compatibility expectations and migration considerations.
- Documentation, examples, and generated assets that become inconsistent with
  the implementation are treated as defects. They MUST be updated in the same
  change or in an explicitly linked follow-up merged in the same release.
- Complexity exceptions MUST be recorded in the plan with the simpler rejected
  alternative and the reason it is insufficient.

## Delivery Workflow & Review

- Specs MUST describe independently testable user scenarios and measurable
  outcomes.
- Implementation plans MUST pass the constitution check before research or
  design proceeds.
- Task lists MUST include work for tests, documentation, consistency-sensitive
  integration points, performance validation, and security compliance when those
  concerns are in scope.
- Reviews MUST block merges that lack adequate automated verification, introduce
  undocumented inconsistency, omit required documentation or generated file
  updates, or violate Principle V security rules.
- Before merge, contributors MUST validate the changed behavior with the
  documented test and verification steps and record any deferred follow-up work
  explicitly.

## Governance

This constitution supersedes conflicting local habits, ad hoc workflow
preferences, and feature-level shortcuts. Amendments MUST be documented in this
file, MUST include a clear rationale, and MUST update any affected Speckit
templates in the same change.

**Amendment procedure**:
1. Open a GitHub issue describing the proposed change and its rationale.
2. Allow a minimum 7-day comment period (14 days for MAJOR version changes).
3. Obtain approval from ≥2 maintainers.
4. Update this file, increment the version per the versioning policy below, and
   set `LAST_AMENDED_DATE` to the amendment date.
5. Propagate changes to dependent templates (plan, spec, tasks) in the same PR.

**Versioning policy** uses semantic versioning:
- MAJOR: Removes or materially redefines a governing principle or review gate.
- MINOR: Adds a new principle, mandatory section, or materially stronger rule.
- PATCH: Clarifies wording without changing expected behavior.

**Compliance review expectations**:
- Every implementation plan MUST record how it satisfies the constitution.
- Every review MUST check testing, consistency, documentation, performance, and
  security obligations relevant to the change.
- Exceptions are allowed only when documented in the plan and accepted during
  review with a stated expiration or follow-up.

**Version**: 1.1.0 | **Ratified**: 2026-05-14 | **Last Amended**: 2026-05-15
