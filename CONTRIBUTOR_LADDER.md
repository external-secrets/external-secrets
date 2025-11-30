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
- Self nomination by the contributor via a Github Issue.
- Must be Sponsored by **two Maintainers** (sponsorship ask must happen within the github issue by tagging the sponsors).
- substantive contributions (code, docs, tests, reviews, triage) in the last 3 months.

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
- several meaningful PR reviews over the last 3 months (or equivalent specialty output).

**Privileges**
- Listed as `reviewer` in the relevant `OWNERS` files.
- May use `/lgtm` on PRs within the specialty.
- Eligible for **Maintainer** nomination.

> A contributor may hold different roles across specialties (e.g., **Reviewer** in CI, **Member** in Core Controllers).

---

### 4) Maintainer (Project-Wide)
Project leaders with governance, release, and cross-specialty responsibility.

**Requirements**
- Reviewer for at least **6 months** in one or more specialties.
- Demonstrated leadership, reliability, and constructive collaboration.
- Nominated and approved by a **supermajority** of Maintainers.

**Privileges**
- GitHub admin rights as needed.
- Release management authority.
- OSC / FOSSA Administrator
- Representation within CNCF.

---

## Specialty Tracks

Specialties define scope for `reviewer` permissions and expectations. 

### CI / Infrastructure
Focus: GitHub Actions, build/test pipelines, images, release automation.
Activity: Reviews CI changes, enforces reproducibility, flags flaky tests.

### Testing
Focus: Unit/integration/E2E tests, frameworks, fixtures, test data.
Activity: Ensures adequate coverage and quality in PRs, promotes testability.

### Core Controllers
Focus: CRDs, reconciliation logic, API evolution, performance.
Activity: Reviews controller/CRD changes, ensures API consistency and backward compatibility.

### Providers
Focus: Provider integrations (AWS, Vault, GCP, Azure, CyberArk, etc.).
Activity: Reviews provider-specific code and conformance to provider guidelines; coordinates breaking changes (for providers that aren't `stable` graded).

### Documentation
Focus: Update documentation in the project. This could be the following: blog posts, provider documentation, tutorials, examples, enhanced developer guides, etc.
Activity: Reviews and create documents and descriptions of the project.

### Community
Focus: Community nurturing. Help with issues, handling community meetings, help fostering the face of external secrets operator, potentially partaking in events and promoting this project.
Activity: Help on issues, monitor the slack channel, organize community talks and events, promote ESO, create demos, etc.

---

## Interim Roles

In some cases, Maintainers may create **interim roles** for **Members**, **Reviewers** in a given specialty or **Maintainers**.  
These are **temporary training-oriented roles** designed to help contributors gain the experience needed to meet the full role requirements.

### Purpose
- Provide **on-the-job training** and mentorship.
- Allow contributors to participate with limited privileges while building their track record.
- Reduce barriers for new contributors to join governance roles.

### Scope
- Limited to a **maximum of three (3) months** for **Members** and **Reviewers**, and **a maximum of six (6) months** for **Maintainers**.
- Specialty and scope are explicitly documented in `OWNERS` files and/or a public tracking issue.
- Interim roles per specialty can accumulate (e.g. a contributor can be an interim reviewer on CI and on Providers at the same time). This is to allow a fast path to upskill future project-wide maintainers.
- Interim roles are not eligible for advancement to permanent roles:
   - Interim Member is not elegible for a permanent reviewer role;
   - Interim reviewer is not elegible for a permanent maintainer role; 
   This is to prevent abuse of the interim system to get permanent roles easily by promoting interim members to them
- Anyone can be nominated to an interim position.
- Anyone effectivated after its interim period does not need to cover requirements from lower positions (e.g. an effectivated interim reviewer on CI does not need to have the criteria of being an effective CI member)
- Any time spent on an interim role counts as participation to the project towards the requirements of the permanent role.

### Limits
- There can only be a maximum of two interim roles per specialty (two CI members; two CI reviewers; two provider Members; two provider Reviewers; two interim Maintainers).

#### Interim Maintainer Role Limits
- An Interim Maintainer election needs super majority approval of Permanent Maintainers.
- Interim Maintainers will not have access to projects' infrastructure credentials
- Interim Maintainers will not have access to projects' Open Source Collective
- Interim Maintainers cannot represent external-secrets for CNCF (they will not be registered as maintainers with the CNCF)
- Interim Maintainers will not have ownership over the organization's github repository.
- While discritionary, an interim maintainer nomination must still be approved by a super majority of permanent maintainers.
Criteria for an interim maintainer position includes contribution/maintenance on other CNCF projects and a commitment to the external-secrets project.

### Examples
- **Interim Member**: Has fewer than 8 substantive contributions but commits to achieve them within 3 months.
- **Interim Reviewer**: Has not yet reviewed 20+ PRs but is actively being mentored to do so.
- **Interim Maintainer**: Key contributor from other CNCF projects willing to help out; has not yet met the criteria for a permanent role but is committed to the project. Emeritus maintainers wanting to return to the project.

### Process
1. Maintainers discuss and vote on the need for an interim role (lazy consensus).
2. Scope and duration are defined clearly (specialty, responsibilities, expected milestones).
3. Nomination of interim roles is done by lazy consensus.
4. Interim roles are granted for its maximum duration.
5. Interim status is reviewed at the end of the period:
   - If requirements are met → promotion to the permanent role.
   - If not met → revert to previous role; contributor may try again later.

### Handling abuse

Interim roles are not a substitute for permanent roles. If a contributor is abusing the interim role system, they may be demoted to their previous role.

---

## Member Abuse

Abuse of project resources is a serious violation of our community standards and will not be tolerated. This includes but is not limited to:

* Using project infrastructure for unauthorized activities like cryptocurrency mining.
* Misusing project funds or financial resources.
* Gaining unauthorized access to or damaging project infrastructure.
* Willingly engaging in activities that are against the Code of Conduct.
* Willingly introducing malwares or viruses to the project's infrastructure or codebase.
* Any other activity that jeopardizes the project's resources, reputation, or community members.

### Procedure for Handling Abuse

1.  **Immediate Revocation of Privileges**: If abuse is suspected, any permanent maintainer can  immediately revoke the member's access to all project infrastructure and resources to prevent further damage. This is a precautionary measure and not a final judgment.

2.  **Investigation**: The maintainers will conduct a private investigation to gather all relevant facts and evidence. The accused member will be given an opportunity to respond to the allegations.

3.  **Decision**: Based on the investigation, the maintainers will determine if a violation has occurred.

4.  **Consequences**: If a violation is confirmed, consequences will be applied, which may include:
    *   Permanent removal from the project.
    *   Reporting the user to GitHub and other relevant platforms.
    *   In cases of financial misuse or illegal activities, reporting to law enforcement authorities.

All actions taken will be documented. The privacy of all individuals involved will be respected throughout the process.


---

## Advancement Process

1. **Nomination** by an eligible community member (Member or Higher) via a github issue.
2. **Sponsorship** by two role holders at the **target level or higher** (within the specialty where applicable).
3. **Review** of activity and behavior (quality, reliability, collaboration, responsiveness).
4. **Decision** by lazy consensus of the relevant group (or **supermajority** if contested).

---

## Inactivity

A **Reviewer** or **Maintainer** role holder may be considered inactive if they have not actively contributed or performed general project responsibilities for **six (6) consecutive months**.

### Measurement Sources
- GitHub activity: merged PRs, PR reviews, issue triage/comments.
- Participation in community calls or asynchronous design discussions.

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
5. **Scope**
   - Demotion via inactivity fully removes the role holder from the organization.
5. **Documentation**  
   - Update `OWNERS`, GitHub teams, and governance records.  
   - Former Members may be listed as **Emeritus**.

### Reinstatement
A contributor can be reinstated at their previous level via the standard advancement process. Prior history is considered favorably.

---

## Emeritus Status
Emeritus status recognizes former Maintainers, Reviewers, or Members who have made substantial and lasting contributions to the External Secrets project but are stepping down from active responsibilities.

### Eligibility

* Must have held a permanent role (Member, Reviewer, or Maintainer) **for at least twelve (12) consecutive months**.
* Demonstrated sustained, high-quality contributions and collaboration.
* Voluntarily stepping down, retiring, or transitioning out of active participation on good terms.

### Privileges

* Public recognition on project website, documentation, and GitHub OWNERS or CONTRIBUTORS files.
* Optionally listed in governance and community records as Emeritus Maintainer / Reviewer / Member.
* Maintains access to project communications for discussion and mentorship purposes (read-only on GitHub teams if desired).
* Eligible to provide mentorship or advisory support to new contributors.
* Invited to participate in major project decisions informally, without voting authority.

### Limitations
* No administrative or write access to repositories, releases, or infrastructure.
* Emeritus status is honorary and does not confer any formal responsibilities or authority.

### Purpose
* Honor and recognize long-term contributions.
* Preserve institutional knowledge and mentorship potential.
* Encourage continued engagement with the community without requiring full role responsibilities.

---

## Conduct & CLA

All contributors must follow the CNCF Code of Conduct and sign the CNCF CLA (where applicable) before contributions are merged.

---

## Cross-References

- Project governance and decision-making: see [`GOVERNANCE.md`](./GOVERNANCE.md)
- Proposal process and template: `design/000-template.md`
- Specialty ownership: `OWNERS` files per directory
