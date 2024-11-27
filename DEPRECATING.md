# External Secrets Operator Deprecation Policy

This document defines the Deprecation Policy for External Secrets Operator components.

## Overview

**External Secrets Operator** is a Kubernetes operator that integrates external
secret management systems like [AWS Secrets
Manager](https://aws.amazon.com/secrets-manager/), [HashiCorp
Vault](https://www.vaultproject.io/), [Google Secrets
Manager](https://cloud.google.com/secret-manager), [Azure Key
Vault](https://azure.microsoft.com/en-us/services/key-vault/), [CyberArk Conjur](https://www.conjur.org) and many more. The
operator reads information from external APIs and automatically injects the
values into a [Kubernetes
Secret](https://kubernetes.io/docs/concepts/configuration/secret/).

## Deprecation Policy

We follow the [Kubernetes Deprecation Policy](https://kubernetes.io/docs/reference/using-api/deprecation-policy/) and [API Versioning Scheme](https://kubernetes.io/docs/reference/using-api/#api-versioning): alpha, beta, GA.

The project is currently in `beta` state. Please try the `beta` features and provide feedback. After the features exits beta, it may not be practical to make more changes.

* alpha
    * The support for a feature may be dropped at any time without notice.
    * The API may change in incompatible ways in a later software release without notice.
    * The software is recommended for use only in short-lived testing clusters, due to increased risk of bugs and lack of long-term support.

* beta
    * The software is well tested. Enabling a feature is considered safe. Features are enabled by default.
    * The support for a feature will not be dropped, though the details may change.
    * The schema and/or semantics of objects may change in incompatible ways in a subsequent beta or stable release. When this happens, migration instructions are provided. Schema changes may require deleting, editing, and re-creating API objects. The editing process may not be straightforward. The migration may require downtime for applications that rely on the feature.
    * The software is not recommended for production uses. Subsequent releases may introduce incompatible changes. If you have multiple clusters which can be upgraded independently, you may be able to relax this restriction.
* GA
    * The stable versions of features appear in released software for many subsequent versions.
    * Use it in production ;)

## API Surface

We define the following scope that is covered by our deprecation policy. We follow the [9 Rules of the Kubernetes Deprecation Policy](https://kubernetes.io/docs/reference/using-api/deprecation-policy/).

### Scope
* API Objects and fields: `.Spec`, `.Status` and `.Status.Conditions[]`
* Enums and constant values
* Controller Configuration: CLI flags & environment variables
* Metrics as defined in the [Kubernetes docs](https://kubernetes.io/docs/reference/using-api/deprecation-policy/#deprecating-a-metric)
* The following features or specific behavior:
    * `ExternalSecret` [update mechanics](http://localhost:8000/api-externalsecret/#update-behavior)

### Non-Scope
Everything not listed in scope is not subject to this deprecation policy and it is subject to breaking changes, updates at any point in time, and deprecation - **as long as it follows the Deprecation Process listed below**.

This includes, but isn't limited to :
* Any feature / specific behavior not in Scope.
* Source code imports
* Helm Charts
* Release process
* Docker Images (including multi-arch builds)
* Image Signature (including provenance, providers, keys)
* OLM-specific builds

## Including features and behaviors to the Deprecation Policy
Any maintainer may propose including a feature, component, or behavior out of scope to be in scope of the deprecation policy. 

The proposal must clearly outline the rationale for inclusion, the impact on users, stability, long term maintenance plan, and day-to-day activities, if such.

The proposal must be formalized by submitting a `design` document as a Pull Request.

## Deprecation Process
### Nomination of Deprecation

Any maintainer may propose deprecating a feature, component, or behavior (both in and out of scope). In Scope changes must abide to the Deprecation Policy above.

The proposal must clearly outline the rationale for deprecation, the impact on users, and any alternatives, if such.

The proposal must be formalized by submiting a `design` document as a Pull Request.

### Showcase to Maintainers

The proposing maintainer must present the proposed deprecation to the maintainer group. This can be done synchronously during a community meeting or asynchronously, through a GitHub Pull Request.

### Voting

A majority vote of maintainers is required to approve the deprecation.
Votes may be conducted asynchronously, with a reasonable deadline for responses (e.g., one week). Lazy Consensus applies if the reasonable deadline is extended, with a minimal of at least one other maintainer approving the changes.

### Implementation

Upon approval, the proposing maintainer is responsible for implementing the changes required to mark the feature as deprecated. This includes:

* Updating the codebase with deprecation warnings where applicable.
* Documenting the deprecation in release notes and relevant documentation.
* Updating APIs, metrics, or behaviors per the Kubernetes Deprecation Policy if in scope.
* If the feature is entirely deprecated (e.g., OLM-specific builds), archival of any associated repositories.

### Deprecation Notice in Release

Deprecation must be introduced in the next release. The release must follow semantic versioning:
* If the project is in the 0.x stage, a minor version bump is required.
* For projects 1.x and beyond, a major version bump is required.

The release notes must prominently include:
* A deprecation notice for the feature.
* The expected timeline for removal (if applicable).

### Full Deprecation and Removal

When a deprecated feature is removed, it must be communicated in the release notes of the removal version.
The removal must follow standard Kubernetes deprecation timelines if the feature is in scope.