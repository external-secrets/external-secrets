# External Secrets Operator Governance

This document defines the project governance for ESO.

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

## Community Roles

* **Users:** Members that engage with the ESO community via any medium (Slack, WeChat, GitHub, mailing lists, etc.).
* **Contributors:** Regular contributions to projects (documentation, code reviews, responding to issues, participation in proposal discussions, contributing code, etc.). 
* **Maintainers**: The ESO project leaders. They are responsible for the overall health and direction of the project; final reviewers of PRs and responsible for releases. Some Maintainers are responsible for one or more components within a project, acting as technical leads for that component. Maintainers are expected to contribute code and documentation, review PRs including ensuring quality of code, triage issues, proactively fix bugs, and perform maintenance tasks for these components.

### Maintainers

New maintainers must be nominated by an existing maintainer(e.g. via [PR](https://github.com/external-secrets/external-secrets/pull/1591)) and must be elected by a supermajority of existing maintainers. Likewise, maintainers can be removed by a supermajority of the existing maintainers or can resign by notifying one of the maintainers.

### Supermajority

A supermajority is defined as two-thirds of members in the group.
A supermajority of [Maintainers](#maintainers) is required for certain
decisions as outlined above. Voting on decisions can happen on the mailing list, GitHub, Slack, email, or via a voting service, when appropriate. Maintainers can either vote "agree, yes, +1", "disagree, no, -1", or "abstain". A vote passes when supermajority is met. An abstain vote equals not voting at all.

### Decision Making

Ideally, all project decisions are resolved by consensus. If impossible, any
maintainer may call a vote. Unless otherwise specified in this document, any
vote will be decided by a supermajority of maintainers.

Votes by maintainers belonging to the same company
will count as one vote; e.g., 4 maintainers employed by fictional company **ESOtum** will
only have **one** combined vote. If voting members from a given company do not
agree, the company's vote is determined by a supermajority of voters from that
company. If no supermajority is achieved, the company is considered to have
abstained.

## Proposal Process

One of the most important aspects in any open source community is the concept
of proposals. Large changes to the codebase and / or new features should be
preceded by a proposal in our community repo. This process allows for all
members of the community to weigh in on the concept (including the technical
details), share their comments and ideas, and offer to help. It also ensures
that members are not duplicating work or inadvertently stepping on toes by
making large conflicting changes.

The project roadmap is defined by accepted proposals.

Proposals should cover the high-level objectives, use cases, and technical
recommendations on how to implement. In general, the community member(s)
interested in implementing the proposal should be either deeply engaged in the
proposal process or be an author of the proposal.

The proposal should be documented as a separated markdown file pushed to the
`design` folder in the [external-secrets repository](https://github.com/external-secrets/external-secrets/tree/main/design)
repository via PR. The name of the file should follow the name pattern `<NUMBER-
meaningful words joined by '-'>.md`, e.g:
`000-clear-old-tags-with-policies.md`.

Use the [Proposal Template](design/000-template.md) as a starting point.

### Proposal Lifecycle

The proposal PR can be marked with different status labels to represent the
status of the proposal:

* **New**: Proposal is just created.
* **Reviewing**: Proposal is under review and discussion.
* **Accepted**: Proposal is reviewed and accepted (either by consensus or vote).
* **Rejected**: Proposal is reviewed and rejected (either by consensus or vote).

## Lazy Consensus

The concept of [Lazy Consensus](http://en.osswiki.info/concepts/lazy_consensus) is practiced. Ideas
and / or proposals should be shared by maintainers via
GitHub with the appropriate maintainer groups (e.g.,
`@external-secrets/maintainers`) tagged. Out of respect for other contributors,
major changes should also be accompanied by a ping on Slack or a note on the
ESO dev mailing list as appropriate. Author(s) of proposal, Pull Requests,
issues, etc.  will give a time period of no less than five (5) working days for
comment and remain cognizant of popular observed world holidays.

Other maintainers may chime in and request additional time for review, but
should remain cognizant of blocking progress and abstain from delaying
progress unless absolutely needed. The expectation is that blocking progress
is accompanied by a guarantee to review and respond to the relevant action(s)
(proposals, PRs, issues, etc.) in short order.


Lazy consensus does _not_ apply to the process of:

* Removal of maintainers from ESO.

## Updating Governance

All substantive changes in Governance require a supermajority agreement by all maintainers.
