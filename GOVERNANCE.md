# External Secrets Operator Governance

This document defines the project governance for ESO.

## Overview

**External Secrets Operator** is a Kubernetes operator that integrates external
secret management systems like [AWS Secrets
Manager](https://aws.amazon.com/secrets-manager/), [HashiCorp
Vault](https://www.vaultproject.io/), [Google Secrets
Manager](https://cloud.google.com/secret-manager), [Azure Key
Vault](https://azure.microsoft.com/en-us/services/key-vault/), [CyberArk Secrets Manager](https://www.cyberark.com/products/secrets-management/) and many more. The
operator reads information from external APIs and automatically injects the
values into a [Kubernetes
Secret](https://kubernetes.io/docs/concepts/configuration/secret/).

## Community Roles

- **Users**: People who use ESO and engage via GitHub, Slack, or mailing lists.
- **Contributors**: Anyone who makes contributions (code, docs, tests, reviews, triage, discussions).
- **Reviewers / Maintainers**: Governance roles with defined responsibilities, privileges, and promotion
  processes described in the [Contributor Ladder](CONTRIBUTOR_LADDER.md).

Maintainers are project leaders responsible for overall health, technical direction, and release management.

---

## Maintainers

### Expectations
Maintainers are expected to:
- Review pull requests, triage issues, and fix bugs in their areas of expertise.
- Monitor community channels and help users and contributors.
- Respond rapidly to time-sensitive security issues.
- Participate in ESO community meetings and asynchronous design/review discussions.
- Follow the decision-making processes described in this document and in the Contributor Ladder.

If a maintainer cannot fulfill these duties, they should move to **Emeritus** status. Maintainers may also be moved to
Emeritus via the decision-making process.

### Adding or Removing Maintainers
- **Addition**: A candidate is nominated by an existing maintainer and elected by a **supermajority** of current maintainers.
- **Removal**: Removal requires a **supermajority** of current maintainers.
- **Company voting**: Votes to nominate maintainers by contributors belonging to the same employer count as **one** collective vote. If they cannot reach internal consensus, that employer’s vote is recorded as **abstain**.

---

## Voting Eligibility

Voting rights vary by decision type:

| Decision Type                                  | Eligible Voters                              |
|------------------------------------------------|----------------------------------------------|
| **Governance changes**                         | Permanent Maintainers                        |
| **Adding/removing Maintainers**                | Permanent Maintainers                        |
| **Technical decisions within a specialty**     | All Reviewers and Maintainers                |
| **Project-wide technical direction**           | All Maintainers                              |
| **Security incident decisions**                | All Maintainers                              |
| **Interim role appointments (Member/Reviewer)**| Permanent Maintainers                        |
| **Interim role appointment (Maintainer)**      | Permanent Maintainers                        |

**Notes:**
- Interim role holders do **not** vote in role promotions/demotions.
- Company voting limits apply: maintainers/reviewers from the same declared employer have **two** combined votes - regardless if they contribute individually or representing their employer. An exception to this rule is for voting to add/remove maintainers. In this case, a company voting limit is of **one** vote.
- If more than two maintainers/reviewers from the same declared employer cannot reach consensus for their vote (e.g. 3 in favour, 2 against), both the employer's votes are recorded as **abstain**.

---

## Decision Making

ESO strives for **consensus** via open discussion. When consensus cannot be reached, any eligible voter may call a vote.

- **Supermajority**: Two-thirds (⅔) of eligible voters in the group.
- **Venues**: Votes may occur on GitHub, email, Slack, community meetings, or a suitable voting service.
- **Ballots**: “Agree/+1”, “Disagree/-1”, or “Abstain” (counts as no vote).

---

## Proposal Process

Large or impactful changes should begin as proposals under the repository’s `design/` directory.

**Proposal requirements**
- Objectives, use cases, and high-level technical approach.
- Open discussion prior to implementation.
- Use the proposal template at `design/000-template.md`.

**Lifecycle labels**
- **New** → **Reviewing** → **Accepted** or **Rejected**.

The project roadmap is shaped by **Accepted** proposals.

---

## Lazy Consensus

ESO uses [Lazy Consensus](http://en.osswiki.info/concepts/lazy_consensus) for most decisions.

- PRs and proposals should allow **at least five (5) working days** for comments and consider global holidays.
- Other maintainers may request additional time when justified and commit to a timely review.

**Exclusions** (lazy consensus does **not** apply):
- Removal of maintainers.
- Any substantive governance changes.

---

## Updating Governance

Substantive changes to this document require a **supermajority** of maintainers.

---

## Contributor Pathways & Specialties

Advancement pathways, responsibilities, privileges, and specialty areas (CI/Infra, Testing, Core Controllers, Providers, Security)
are defined in the [Contributor Ladder](CONTRIBUTOR_LADDER.md).

---

## Security

ESO follows responsible disclosure practices. Security-impacting issues should be reported via the documented security
contact channels (see `SECURITY.md` if present or repository Security tab). Security fixes may be handled privately until
a coordinated disclosure and release are ready.

---

## CNCF Alignment

ESO governance aims for open, transparent, and vendor-neutral operation consistent with CNCF expectations. The
[Contributor Ladder](CONTRIBUTOR_LADDER.md) provides clear pathways for community members to grow into leadership.
