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

### 3) Reviewer
Experienced contributors who review changes.

**Requirements**
- Member for at least **3 months**.
- Regular, high-quality reviews in the specialty.
- ~**20+** meaningful PR reviews over the last 3 months (or equivalent specialty output).

**Privileges**
- Listed as `reviewer` in the relevant `OWNERS` files.
- May use `/lgtm` on PRs.
- Eligible for **Approver** nomination.

---

### 4) Approver
Trusted contributors with **merge/approve authority**.

**Requirements**
- Reviewer for at least **3 months**.
- Multiple significant approvals/landed changes.
- Demonstrated understanding of project-wide implications.

**Privileges**
- Listed as `approver` in relevant `OWNERS` files.
- Can approve and merge PRs within the specialty.
- Eligible for **Maintainer** nomination.

---

### 5) Maintainer (Project-Wide)
Project leaders with governance, release, and cross-specialty responsibility.

**Requirements**
- Approver for at least **6 months**.
- Demonstrated leadership, reliability, and constructive collaboration.
- Nominated and approved by a **supermajority** of Maintainers.

**Privileges**
- GitHub admin rights as needed.
- Release management authority.
- Representation in CNCF processes.

---

## Interim Roles

In some cases, Maintainers may create **interim roles** for **Member**, **Reviewer** or **Approver** positions.  

These are **temporary training-oriented roles** designed to help contributors gain the experience needed to meet the full role requirements.

### Purpose
- Provide **on-the-job training** and mentorship.
- Allow contributors to participate with limited privileges while building their track record.
- Reduce barriers for new contributors to join governance roles.

### Scope
- Limited to a **maximum of three (3) months**.
- **Approver** interim roles must be elected by super majority.
- Non Renewable.

### Limits
- There can only be a maximum of three interim roles.

### Examples
- **Interim Member**: Has fewer than 8 substantive contributions but commits to achieve them within 3 months.
- **Interim Reviewer**: Has not yet reviewed 20+ PRs but is actively being mentored to do so.

### Process
1. Maintainers discuss and vote on the need for an interim role (lazy consensus).
2. Scope and duration are defined clearly (specialty, responsibilities, expected milestones).
3. Nomination of interim roles is done by lazy consensus (except for interim approvers).
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
