---
title: Complete Docs refactor proposal
version: v1alpha1
authors: Jean-Philippe Evrard
creation-date: 2026-03-10
status: draft
---

# Docs refactor proposal

## Summary

ESO documentation fails in (at least) three recurring ways: users cannot find setup instructions, cannot determine feature support status, and cannot follow a clear path through the site. This proposal defines five sequential steps to fix it. Steps 1 and 3 are optional accelerators; Steps 2, 4, and 5 are required.

## Motivation

ESO receives three categories of recurring support questions that documentation should answer without human intervention:

- Setup: users cannot find or follow instructions for feature X
- Support status: the feature matrix is ambiguous; users cannot determine what is alpha, beta, or stable
- Navigation: users do not know where to go next, what to read, or how to write a contribution

Undocumented decisions and an undefined content architecture also block automated content generation.

### Goals

By the end of this proposal:

- external-secrets.io serves content per persona: adopters, operators, contributors, security reviewers
- Feature support status per provider is unambiguous and machine-generated from source
- The site supports multi-project navigation across ESO, reloader, and the CLI
- Maintenance cost per release is reduced: no manual feature matrix updates, no manual structure decisions

### Non-Goals

- Documentation for sub-projects (CLI, reloader) beyond navigation and cross-linking
- Content rewrites outside the ESO main documentation

## Proposal

### Step 1 (Optional): New multi-project website

We create a single landing site for external-secrets.io modeled after kured.dev. The landing page states the value proposition, lists key features, and routes users to per-project documentation.

Top-level structure:
- Sub-project docs (ESO, reloader, CLI) as first-class sections
- Releases as a top-level page
- Community links shared across all projects

#### Actual implementation

We use Hugo with Docsy to match the CNCF ecosystem baseline. Docsy targets single-project versioning. Because we need coherent search across the current version of the active sub-project and the stable versions of the others, we customize Docsy's build process, JS, and templates.

Search constraints:
- No external search service (offline capability required)
- Local search index only
- Index must build efficiently given the large version history

Demo: https://github.com/evrardj-roche/external-secrets-website

#### Acceptance criteria

The new site is live at https://external-secrets.io, replacing the current landing page.

#### Side-effects

The release management pipeline must be updated to build and publish the new site. This must be complete before starting Step 2.

### Step 2: Agree on a documentation architecture in a community meeting

The current content architecture has no defined personas, no enforced structure per section, and provider documentation that has grown unreadably long (based on provider). Step 1 makes this visible. Step 2 fixes the foundation.

We adopt Diataxis (https://www.diataxis.fr) as the basis for content framework. Diataxis defines four documentation modes (tutorials, how-tos, explanation, reference) and **maps each to a reader goal**. It does not require a dedicated content architect to enforce.

The content/pages hierarchy does not change on day 1. The writing process changes immediately.
Important note: The hierachy might not map (sensu stricto) to Diataxis, it will depend on reader goals/personas.

#### Actual implementation

1. Run a community meeting to agree on the reader personas for ESO documentation.
2. Add a CI check that enforces documentation updates are separated by persona (one file per persona-scoped section).
3. Publish a set of LLM-based writing skills to help contributors write to **our** adapated Diataxis layout.

#### Acceptance criteria

- Community meeting minutes record the agreed persona list
- CI fails on PRs that add documentation outside a persona-scoped path
- At least one LLM writing skill is available in the contributor toolchain

#### Side-effects

Documentation will temporarily fragment across more files as new content follows our adapted Diataxis layout while old content does not. This is expected and will be resolved in Step 3.

### Step 3 (Optional): Accelerate the adaptation of our documentation based on these new foundations

Rewriting all existing content in one go is not feasible for a single contributor (reference: https://github.com/external-secrets/external-secrets/pull/5822). This step is optional: the community decides whether to resource it.

#### Actual implementation

One or more volunteers take ownership of a full content rewrite, working section by section against the Diataxis structure agreed in Step 2.

#### Acceptance criteria

- All existing pages are rewritten for their target persona
- Top-level structure is reduced to: Providers / Guides / How-tos / Reference / Tutorials

#### Side-effects

The final structure cannot be known in advance. Step 4 must not depend on it.

### Step 4: Clarify governance around features stability and their documentation

ESO contributors and users have no authoritative definition of alpha, beta, and stable for features and providers. The support matrix is manually maintained and inconsistently applied.

We define and publish a lifecycle policy for features and providers.

#### Actual implementation

1. Jean-Philippe Evrard opens a PR defining the feature lifecycle (alpha / beta / stable criteria, promotion and demotion rules).
2. The PR is discussed asynchronously and during a community meeting.

#### Acceptance criteria

- The lifecycle policy is merged into the main branch

#### Side-effects

Some providers or features will be reclassified. The reclassification must be communicated clearly to avoid unexpected user impact.

### Step 5: Automatically generate lifecycle events in our documentation, from our codebase.

With the Step 4 policy in place, the feature support matrix for each provider is generated from code, not written by hand.

#### Actual implementation

Each provider registers into a central Go registry containing its supported features and their maturity level. The docs build reads the registry via a Go client and generates the support matrix markdown for each provider. CI blocks manual edits to the generated files.

PoC (abandoned): https://github.com/evrardj-roche/external-secrets/commits/feature-flags/

This step supersedes design proposal 014: https://github.com/external-secrets/external-secrets/blob/81078c9ab6a7cf2ddbd0fe5856188a120e09e87a/design/014-feature-flag-consolidation.md

#### Acceptance criteria

- Feature support documentation is fully generated; no manual edits are possible
- CI enforces this with a diff check on the generated files

#### Side-effects

None beyond the supersession of proposal 014.

### User Stories

- As a user, I need step-by-step instructions for configuring feature X without opening a Slack thread.
- As an advocate, I need a one-page summary of what ESO does that I can share without asking the team to write one.
- As a new contributor, I need a single linear path through environment setup, triage, and first PR without chasing links across three separate pages.

### API

No API change.

### Behavior

No user behavior change.

### Drawbacks

None identified.

### Acceptance Criteria

See each step's acceptance criteria above.

## Alternatives

Doing nothing is the rejected alternative. The current state produces recurring support load, blocks automation, and degrades contributor onboarding quality with each added provider.
