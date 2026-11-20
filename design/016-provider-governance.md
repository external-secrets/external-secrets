```yaml
---
title: Provider Governance and Lifecycle
version: v1alpha1
authors: Jean-Philippe Evrard
creation-date: 2026-10-15
status: draft
---
```

# Provider Governance and Lifecycle

## Summary

ESO cannot reasonably maintain and test every provider without help from
provider experts. This proposal makes provider responsibility explicit while
keeping ESO accountable for code shipped in its binary.

For version 1, every provider has a `provider-<name>-reviewers` team with at
least one active Provider Representative. Substantive changes require:

1. provider acknowledgement that the backend implementation is correct; and
2. normal ESO approval that the change meets project security, architecture,
   compatibility, and quality goals.

A public registry becomes the source of truth for ownership, support, features,
flags, SDKs, tests, and generated documentation.

For version 2, once a production-ready generic provider mechanism exists, new
providers are external by default. ESO does not govern their internal code; it
defines the contract, runs compatibility tests on advertised providers,
and controls their public listing and support status.

Listing is never an endorsement, security certification, recommendation, or
warranty by ESO, CNCF, or The Linux Foundation.

## Motivation

Provider integrations require backend-specific knowledge of APIs, SDKs,
authentication, service deprecations, and operational constraints. The core team
cannot retain this knowledge for an unlimited number of providers.

Today, provider ownership, runtime maintenance status, CODEOWNERS, tests, and
public documentation can drift. Nearly every implementation declares itself
`Maintained`, but that status is not derived from active ownership, test health,
or dependency maintenance. The weekly dependency update also permits individual
module failures without creating provider-specific accountability.

This proposal combines these precedents:

- **cert-manager:** unlimited in-tree integrations are impractical; define
  responsible reviewers and inactivity rules.
- **CoreDNS:** use integration-specific owners and independent core review; once
  plugins are external, leave their internal governance to their projects.
- **Secrets Store CSI Driver:** determine support through ESO-owned conformance
  tests and remove persistently incompatible providers from the supported set.
- **Cluster API:** publish a provider contract and migration guidance.

### Goals

- Give every provider an active, publicly identified representative.
- Define objective maintenance and support requirements.
- Keep SDKs, tests, documentation, and ESO practices current.
- Notify representatives of provider-impacting core changes.
- Generate public provider status and feature documentation from one registry.
- Define gradual inactivity, deprecation, disablement, and removal processes.
- Prepare for independently distributed version 2 providers.

### Non-goals

- Vendor ownership or veto over ESO.
- Requiring vendor employment, meaningless commits, or every optional feature.
- Automatic promotion to project-wide roles.
- Immediately externalizing version 1 providers.
- Selecting the future generic provider protocol.
- Replacing ESO's security response or deprecation policies.

## Principles

1. ESO remains accountable for code it builds, signs, and ships.
2. Individuals hold roles based on knowledge and contribution, not employment.
3. Stability, maintenance, compatibility, and listing are independent.
4. Public status must be objective and auditable.
5. Activity means useful maintenance, not commit frequency.
6. Optional features may be unsupported or inapplicable.
7. No provider may indefinitely block core or security work.
8. The strictest applicable governance, deprecation, or semantic-versioning
   requirement wins.
9. Representatives may request reasonable extensions or step down without
   penalty.

## Provider Representative

ESO defines one provider-specific role: **Provider Representative**.

A representative must be an identifiable individual who:

- follows the Code of Conduct and applicable CLA;
- uses two-factor authentication;
- understands the backend, SDK, or implementation;
- publicly discloses their affiliation;
- accepts this policy; and
- is approved through ESO's governance process.

Vendor employees are welcome but not required. Companies, organization accounts,
and bots do not satisfy the human representative requirement.

Every provider has its own team and at least one representative; two are
recommended, especially for stable providers. ESO-owned reference providers
such as `fake`, `crd`, `kubernetes`, and `webhook` also have dedicated teams.
They are maintained _solely_ by (permanent and ad-interim) maintainers by default.

Representatives:

- review backend behavior, SDK use, authentication, and terminology;
- triage provider bugs and dependency updates;
- maintain documentation, capabilities, and known limitations;
- respond to provider-impact notices; and
- participate in six-month health reviews.

They may record provider acknowledgement and receive scoped triage permissions.
The role alone grants no project-wide `/lgtm`, merge, release, security, or CNCF
representation authority.

In other words, providers and ESO maintainers work in tandem:

- The **provider representative** gives opinion about whether changes
  are correct for the backend API, SDK, authentication model,
  and terminology (share domain expertise)
- The **ESO reviewers** give their opinion about overall security
  and alignment with ESO architecture, compatibility, shared practices,
  tests, and project direction (share project expertise).

Sustained provider work counts toward the normal Contributor Ladder. Promotion
to general Reviewer or Maintainer is optional and not automatic.

A representative becomes inactive after stepping down, losing role
requirements, or completing no qualifying activity for six months and failing
to answer a 30-day activity check. Qualifying activity includes reviews,
triage, dependency decisions, documentation, testing, impact acknowledgement,
and health attestation; it need not include a commit.

Representatives may not:

- Update their group members autonomously (this is a maintainer duty).

Provider team therefore contains:

- ad-interim and permanent maintainers (for ownership management)
- provider representatives

## Provider Registry and its public documentation

A (machine-readable) registry (text file) exists in our project
current active branch to record provider's metadata.

The text file records, per provider family:

- identifier (codename), full name;
- location (in-tree folder location for v1, url of source for v2);
- representatives (responsible people);
- Maintenance (Maintained, Unmaintained, Deprecated) status;
- Introduction date;
- Compatibility with current version (by default Supported in v1, unless provider is explicitly Unmaintained/Deprecated);
- Capabilities, unsupported features, flags, and feature gates;
- known limitations;
- deprecation and removal dates where applicable.

Some of this data will be directly generated from the codebase in our 'v1'.
In v2, everything will have to be submitted by a human in a PR.

The documentation page for support matrix includes a disclaimer equivalent to:

> Listing identifies an integration known to work, or to have worked, with ESO.
> It is not endorsement by ESO, CNCF, or The Linux Foundation. Support status
> only describes compliance with ESO's published requirements; it is not a
> security certification, commercial recommendation, or warranty.

Vendor claims must be identified as such.

Each provider documentation page will link to the support matrix page.

## What does "maintained" mean for our users: The user-facing expectations

While the code is not provided with any guarantees whatsoever, the term
**Maintained** provider encompasses the following:

1. it has a team with at least an active representative;
2. its published documentation for status is up-to-date;
3. its tests in ESO codebase pass or have an approved temporary
   exception;
4. its capabilities and limitations are accurate;
5. its SDK and upstream API are supported and acceptably secure;
6. dependency updates and provider-impact PRs are managed by its representatives;
7. its health review was successful in the last 6 months.

Lack of recent code changes does not impact the _maintained_ state of
a provider.

## What does "maintained" mean for the core team: The ESO maintainer team expectations

The maintainer team expects the provider to:

- implement our expected interfaces, for example the complete `SecretsClient`;
- declare current capabilities;
- respect our policy for contributions;
- respect our code expectations (error/context/timeout management/);
- credential and namespace-boundary validation;
- use standard secret, service-account, CA, metadata, and metrics helpers;
- provide documentation of features and behaviour;
- provide unit and conformance tests, ideally isolated e2e tests;
- maintain codebase, especially upon changes of upstream SDKs or ESO core.

## The specific case of dependency updates

ESO bumps the dependencies of ALL its modules weekly.

In v1, the providers in our codebase are expected to help on this duty.
When a failure of the dependency update is attributed to a provider failure,
ESO maintainer team will create an issue requesting the provider team to
provide a solution. Representatives have a 30 calendar days period to fix
the issue by merging, replacing, postponing the dependency update
with a reason and review date.

In other words, providers can request to "freeze" (=exempt from weekly bump)
their required SDKs. Yet, this is *not recommended* by the ESO maintenance team.
Any issue caused by freezing is treated like issues when bumping:
Representatives have 30 calendar days to fix the issue.

Those 30 calendar days deadline is **NOT** valid for security issues.
Those remain controlled by the security process.

Failure to reply in the 30 calendar days results in a transition from the 
provider to its "Unmaintained" state (see below).

## The specific case of ESO breaking changes or new interfaces

Any change to shared interfaces, helpers, authentication, metrics, errors, or
controller behavior will be labelled with impacted providers if possible, and
the review will be requested to the provider-<name>-reviewers.

ESO will highlight:
- mandatory, optional, or informational impact;
- compatibility and user impact;
- migration instructions and, where practical, a reference implementation;
- common tests;
- target release and dates; and
- the owner of repository-wide migration work.

An issue _may_ be created to highlight the change.
Upon merge or issue creation, a response windows is started, measuring
provider acknowledgement.

We expect the provider to respond within 30 calendar days (for routine requests) or
within 10 calendar days (for release impacting requests). Again, security
issues are following our security policy in terms of timing.

## The duty from ESO maintenance team: A regular audit

An automated health review runs at least every six months and checks ownership,
tests, dependencies, advisories, upstream deprecations, docs, features, flags,
and significant bugs. One dashboard or tracking issue should cover all
providers and avoid repetitive issues when checks are green.

## New Provider Admission

Until the version 2 generic provider mechanism is production ready, a new
provider may be accepted in-tree only when it:

- demonstrates user demand and useful functionality distinct from existing
  providers;
- has a dedicated team, at least one Provider Representative, and an ESO
  sponsor before merge;
- has complete registry metadata, documentation, capabilities, flags, known
  limitations, and SDK support information;
- complies with the provider contract and includes unit and conformance tests;
- includes required E2E coverage or an approved plan and deadline for it;
- covers implementation, API, registration, docs, snippets, and tests in its
  ownership scope; and
- is approved by an ESO reviewer unaffiliated with the submitting organization.

ESO may decline an integration whose security, architectural, testing,
or long-term maintenance cost is unreasonable.

Once the generic provider mechanism is production ready, new providers are
external by default as described in [Version 2 External
Providers](#version-2-external-providers). An in-tree exception then requires a
separate design proposal.

## The impact of not respecting expectations

A provider enters "Unmaintained" state when it has no representative, misses two
different deadlines, has required tests broken for 30 days, uses unsupported
SDK/API versions, misses its health review after the 30-day reminder,
or materially misrepresents support.

The lack of maintenance is documented in the support matrix with the
deprecation date. Upon editing the support matrix, the ESO maintainers will
also add warnings for their users.
Warnings appear in admission responses, Kubernetes events, release notes,
documentation, controller logs, and the provider directory.

ESO provider will be disable after 3 months of deprecation.
A disabled provider:

- makes no backend calls;
- creates, updates, or deletes no target data;
- reports a clear resource condition and event;
- rejects new uses where safely distinguishable; and
- links users to migration instructions.

Existing synchronized Secrets remain unchanged and may become stale; ESO does
not silently delete them. The technical disablement mechanism requires a
separate implementation design.

If no change happens to make the provider more maintained,
Providers' code will be purged 6 months after deprecation date if no support
progress was made. This will be joined by a release notes and a major bump
of ESO version.

### Restoring to "Maintained" state

Support can be restored after ownership, health, tests, registry data, and ESO
review are restored. Reversing deprecation or disablement additionally requires
a governance decision.

## Drawbacks

The health reviews and pings add administration for the ESO team.
Public status downgrades may impact user perception.

These costs are preferable to claiming support ESO cannot reasonably provide:
We consider transparency is better than security problems in this project, so
those drawbacks are acceptable.

## Acceptance Criteria

- We have updated our matrix for providers
- We have audited all providers
- We have added provider representatives to all providers (or started their
  deprecation cycle)
- We have added ad-interim maintainers to all the provider-_name_-reviewers
  teams.
- We create an issue to adapt this document for v2's "compliance level".
  The issue will be closed when v2's interface will be made public and its
  e2e conformance test will be described.

## References

- [cert-manager DNS provider policy](https://cert-manager.io/docs/contributing/dns-providers/)
- [cert-manager external issuers](https://cert-manager.io/docs/contributing/external-issuers/)
- [cert-manager governance](https://github.com/cert-manager/community/blob/main/GOVERNANCE.md)
- [CoreDNS governance](https://github.com/coredns/coredns/blob/master/GOVERNANCE.md)
- [CoreDNS plugin ownership](https://github.com/coredns/coredns/blob/master/CODEOWNERS)
- [CoreDNS plugin inclusion guidance](https://github.com/coredns/coredns/blob/master/plugin.md#qualifying-for-main-repo)
- [Secrets Store CSI Driver provider criteria](https://github.com/kubernetes-sigs/secrets-store-csi-driver/blob/main/docs/book/src/providers.md)
- [Cluster API provider contract](https://cluster-api.sigs.k8s.io/developer/providers/contracts/overview)
- [Cluster API provider migrations](https://cluster-api.sigs.k8s.io/developer/providers/migrations/overview)
- [ESO governance](../GOVERNANCE.md)
- [ESO Contributor Ladder](../CONTRIBUTOR_LADDER.md)
- [ESO deprecation process](../DEPRECATING.md)
- [ESO provider versioning strategy](007-provider-versioning-strategy.md)
