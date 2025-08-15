# External Secrets Contributor Ladder

This document defines the roles, responsibilities, advancement criteria, inactivity process, and interim role policies for ESO contributors.
It extends the standard Kubernetes-style ladder with **specialty tracks** so contributors can grow within their focus areas.

---

## Roles

### 1) Contributor
Anyone making contributions of any kind (code, docs, tests, CI configs, security reports, reviews, triage, discussions).

**Requirements**
- Sign the CNCF CLA.
- Follow the ESO Code of Conduct.

**Privileges**
- Recognized as an active community member.
- Eligible for nomination to **Member**.

---

### 2) Member
Regular contributors engaged with the project for at least **3 months**.

**Requirements**
- Nominated by a **Member** or above.
- Sponsored by **two Maintainers**.
- ~**8+** substantive contributions (code, docs, tests, reviews, triage) in the last 3 months.

**Privileges**
- Added to the GitHub `members` team.
- Can be assigned issues and PRs.
- Eligible for **Reviewer** nomination.

---

### 3) Reviewer (per Specialty)
Experienced contributors who review changes in **one or more specialties**.

**Requirements**
- Member for at least **3 months**.
- Regular, high-quality reviews in the specialty.
- ~**20+** meaningful PR reviews over the last 3 months (or equivalent specialty output).

**Privileges**
- Listed as `reviewer` in the relevant `OWNERS` files.
- May use `/lgtm` on PRs within the specialty.
- Eligible for **Approver** nomination.

---

### 4) Approver (per Specialty)
Trusted contributors with **merge/approve authority** in their specialty.

**Requirements**
- Reviewer for at least **3 months**.
- Multiple significant approvals/landed changes in the specialty.
- Demonstrated understanding of project-wide implications.

**Privileges**
- Listed as `approver` in relevant `OWNERS` files.
- Can approve and merge PRs within the specialty.
- Eligible for **Maintainer** nomination.

---

### 5) Maintainer (Project-Wide)
Project leaders with governance, release, and cross-specialty responsibility.

**Requirements**
- Approver for at least **6 months** in one or more specialties.
- Demonstrated leadership, reliability, and constructive collaboration.
- Nominated and approved by a **supermajority** of Maintainers.

**Privileges**
- GitHub admin rights as needed.
- Release management authority.
- Representation in CNCF processes.

> A contributor may hold different roles across specialties (e.g., **Approver** in Providers and **Reviewer** in CI).

---

## Specialty Tracks

Specialties define scope for `reviewer`/`approver` permissions and expectations. Ownership is documented in `OWNERS` files.

### CI / Infrastructure
Focus: GitHub Actions, build/test pipelines, images, release automation.
- **Reviewer**: Reviews CI changes, enforces reproducibility, flags flaky tests.
- **Approver**: Owns pipeline stability, approves release workflow changes.

### Testing
Focus: Unit/integration/E2E tests, frameworks, fixtures, test data.
- **Reviewer**: Ensures adequate coverage and quality in PRs, promotes testability.
- **Approver**: Sets testing strategy, guards test harness stability and performance.

### Core Controllers
Focus: CRDs, reconciliation logic, API evolution, performance.
- **Reviewer**: Reviews controller/CRD changes, ensures API consistency and backward compatibility.
- **Approver**: Owns core designs affecting controller behavior and APIs.

### Providers
Focus: Provider integrations (AWS, Vault, GCP, Azure, CyberArk, etc.).
- **Reviewer**: Reviews provider-specific code and conformance to provider guidelines.
- **Approver**: Owns lifecycle and quality of one or more providers; coordinates breaking changes.

### Security
Focus: Vulnerability handling, dependency hygiene, threat modeling, secure coding.
- **Reviewer**: Reviews PRs for security impact; flags risky patterns.
- **Approver**: Leads security releases and coordinates disclosure/patch processes.

---

## Interim Roles

In some cases, Maintainers may create **interim roles** for **Member** or **Reviewer** positions in a given specialty.  
These are **temporary training-oriented roles** designed to help contributors gain the experience needed to meet the full role requirements.

### Purpose
- Provide **on-the-job training** and mentorship.
- Allow contributors to participate with limited privileges while building their track record.
- Reduce barriers for new contributors to join governance roles.

### Scope
- Available only for **Member** and **Reviewer** levels (including per-specialty Reviewers).
- Limited to a **maximum of three (3) months**.
- Specialty and scope are explicitly documented in `OWNERS` files and/or a public tracking issue.
- Interim roles per specialty can accumulate (e.g. a contributor can be an interim reviewer on CI and on Providers at the same time). This is to allow a fast path to upskill future project-wide maintainers.

#### Approver exclusion

Approvers on any specialty may never be interim. This is to prevent abuse of the interim role system, and to consolidate the needed trust between the other maintainers & approvers for the different elements of the codebase.

### Limits
- There can only be a maximum of two interim roles per specialty (two CI members; two CI reviewers; two CI interim approvers; two provider Members; two provider Reviewers; two provider interim Approvers).

### Examples
- **Interim Member**: Has fewer than 8 substantive contributions but commits to achieve them within 3 months.
- **Interim Reviewer**: Has not yet reviewed 20+ PRs but is actively being mentored to do so.

### Process
1. Maintainers discuss and vote on the need for an interim role (lazy consensus).
2. Scope and duration are defined clearly (specialty, responsibilities, expected milestones).
3. Nomination of interim roles is done by lazy consensus.
4. Interim roles are granted for a maximum of three months.
5. Interim status is reviewed at the end of the period:
   - If requirements are met → promotion to the permanent role.
   - If not met → revert to previous role; contributor may try again later.

### Handling abuse

Interim roles are not a substitute for permanent roles. If a contributor is abusing the interim role system, they may be demoted to their previous role.

---

## Advancement Process

1. **Nomination** by an eligible community member (Member+).
2. **Sponsorship** by two role holders at the **target level or higher** (within the specialty where applicable).
3. **Review** of activity and behavior (quality, reliability, collaboration, responsiveness).
4. **Decision** by lazy consensus of the relevant group (or **supermajority** if contested).

---

## Inactivity

A role holder may be considered inactive if they have not actively contributed in their specialty or general project responsibilities for **six (6) consecutive months**.

### Measurement Sources
- GitHub activity: merged PRs, PR reviews, issue triage/comments.
- Participation in community calls or asynchronous design discussions.
- Specialty-specific metrics (e.g., CI job maintenance, provider updates, security disclosures).

### Triggering Process
1. **Detection**  
   - Activity is reviewed at least quarterly by Maintainers or via automation.  
   - Any Maintainer may propose an inactivity review for a role holder.
2. **Notification**  
   - A public issue is opened in a `community`/`governance` space (or email if sensitive).  
   - The individual is tagged/emailed and given **30 days** to respond.
3. **Grace Period**  
   - If the contributor indicates intent to return, no change is made.  
   - If there is no response or no significant activity within the grace period, proceed.
4. **Decision**  
   - Demotion is decided by **lazy consensus** of Maintainers, or **supermajority** if contested.
5. **Documentation**  
   - Update `OWNERS`, GitHub teams, and governance records.  
   - Former Maintainers are listed as **Emeritus**.

### Reinstatement
A contributor can be reinstated at their previous level via the standard advancement process. Prior history is considered favorably.

---

## Conduct & CLA

All contributors must follow the CNCF Code of Conduct and sign the CNCF CLA (where applicable) before contributions are merged.

---

## Cross-References

- Project governance and decision-making: see [`GOVERNANCE.md`](./GOVERNANCE.md)
- Proposal process and template: `design/000-template.md`
- Specialty ownership: `OWNERS` files per directory
