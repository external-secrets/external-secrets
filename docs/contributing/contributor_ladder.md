<!-- reference: What is our contributor ladder? Dry description. -->
# External Secrets Contributor Ladder

This document defines the roles, responsibilities, advancement criteria, inactivity process, and interim role policies for ESO contributors.
It extends the standard Kubernetes-style ladder with **specialty tracks** so contributors can grow within their focus areas.

## Permanent Roles

### 1) Users

People who use ESO and engage via GitHub, Slack, or mailing lists.

### 2) Contributor
Anyone making contributions of any kind (code, docs, tests, CI configs, security reports, reviews, triage, discussions).

**Requirements**:

- Sign the CNCF CLA.
- Follow the ESO Code of Conduct.

**Privileges**:

- Recognized as an active community member.
- Eligible for nomination to **Member**.

// TODO EVRARDJP WITH SKARLSO: Verify if that's true.

**Election Process**:

- Automatically granted by GitHub

---

### 3) Member
Regular contributors engaged with the project for at least **3 months**.

**Requirements**:

- Self nomination by the contributor via a GitHub Issue.
- substantive contributions (code, docs, tests, reviews, triage) in the last 3 months.

**Privileges**:

- Added to the GitHub `members` team.
- Can be assigned issues and PRs.
- Eligible for **Reviewer** nomination.

**Election Process**:

- Must be Sponsored by **two Maintainers** (sponsorship ask must happen within the GitHub issue by tagging the sponsors).

---

### 4) Reviewer (per Specialty)
Experienced contributors who review changes in **one or more specialties**.
A specialty define scope for `reviewer` permissions and expectations.

> A contributor may hold different roles across specialties (e.g., **Reviewer** in CI, **Member** in Core Controllers).

**Requirements**:

- Member for at least **3 months**.
- Regular, high-quality reviews in the specialty.
- several meaningful PR reviews over the last 3 months (or equivalent specialty output).

**Privileges**:

- Listed as `reviewer` in the relevant `OWNERS` files. // TODO EVRARDJP: REPLACE OWNERS WITH CODEOWNERS
- May use `/lgtm` on PRs within the specialty.
- may `/approve` and merge within their specialty if they are also `approvers`
- Eligible for **Maintainer** nomination.

**Expectations**:

- Review pull requests, triage issues, and fix bugs in their areas of expertise.
- Participate in ESO community meetings and asynchronous design/review discussions.
- Follow the decision-making processes described in our governance document.
- //TODO EVRARDJP: Mention the ok-to-test?
- //TODO EVRARDJP: ASK SKARLSO IF THAT MAKES SENSE.

**Election process**:

- Must be nominated by **one Maintainer**?.

// TODO EVRARDJP: ASK SKARLSO IF THAT MAKES SENSE.

#### Reviewer specialties

| Name                | Focus                                                                                                                                                                                | Activities                                                                                                                                      | Name of `CODEOWNERS` group              |
|---------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------|-----------------------------------------|
| CI / Infrastructure | GitHub Actions, build/test pipelines, images, release automation                                                                                                                     | Reviews CI changes, enforces reproducibility, flags flaky tests                                                                                 | `@external-secrets/ci-reviewers`        |
| Testing             | Unit/integration/E2E tests, frameworks, fixtures, test data                                                                                                                          | Ensures adequate coverage and quality in PRs, promotes testability                                                                              | `@external-secrets/testing-reviewers`   |
| Core Controllers    | CRDs, reconciliation logic, API evolution, performance                                                                                                                               | Reviews controller/CRD changes, ensures API consistency and backward compatibility                                                              | `@external-secrets/core-reviewers`      |
| Providers           | Provider integrations (AWS, Vault, GCP, Azure, CyberArk, etc.)                                                                                                                       | Reviews provider-specific code and conformance to provider guidelines; coordinates breaking changes (for providers that aren't `stable` graded) | `@external-secrets/providers-reviewers` |
| Documentation       | Update documentation in the project. This could be the following: blog posts, provider documentation, tutorials, examples, enhanced developer guides, etc.                           | Reviews and create documents and descriptions of the project                                                                                    |                                         |
| Community           | Community nurturing. Help with issues, handling community meetings, help fostering the face of external secrets operator, potentially partaking in events and promoting this project | Help on issues, monitor the slack channel, organize community talks and events, promote ESO, create demos, etc.                                 |                                         |


// TODO EVRARDJP: ASK SKARLSO WHAT WE DO FOR: SECURITY (NO CODE OWNERS, NOT LISTED HERE) + Docs (NO CODEOWNERS, OK TO FIX THIS?)
---

### 5) Maintainer (Project-Wide)

They are project leaders responsible for overall health, governance, technical direction, cross-specialty knowledge and release management.

// TODO: EVRARDJP: SPLIT BETWEEN REQUIREMENTS AND EXPECTATIONS

**Requirements**:

- Reviewer for at least **6 months** in one or more specialties.
- Demonstrated leadership, reliability, and constructive collaboration.

TODO: EVRARDJP -> link to decision-making processes

**Privileges**:

- GitHub admin rights as needed.
- Release management authority.
- OSC / FOSSA Administrator
- Representation within CNCF.
- may `/approve` and merge commits.

**Expectations**:

- Manage milestones and assign issues on the GitHub project
  - Apply labels on issues according to our [Label policy](labels_reference.md)
- Review pull requests, triage issues, and fix bugs in their areas of expertise.
- Respond rapidly to time-sensitive security issues.
- Participate in ESO community meetings and asynchronous design/review discussions.
- Follow the decision-making processes described in our governance document.

**Election process**:

- **Addition**: A candidate is nominated by an existing maintainer and elected by a **supermajority** of current maintainers.
- **Removal**: Removal requires a **supermajority** of current maintainers.

---

## Interim Roles

In some cases, Maintainers may create **interim roles** for **Members**, **Reviewers** in a given specialty or **Maintainers**.
These are **temporary training-oriented roles** designed to help contributors gain the experience needed to meet the full role requirements.

Criteria for an interim maintainer position includes contribution/maintenance on other CNCF projects and a commitment to the external-secrets project.

**Examples**:

- **Interim Member**: Has fewer than 8 substantive contributions but commits to achieve them within 3 months.
- **Interim Reviewer**: Has not yet reviewed 20+ PRs but is actively being mentored to do so.
- **Interim Maintainer**: Key contributor from other CNCF projects willing to help out; has not yet met the criteria for a permanent role but is committed to the project. Emeritus maintainers wanting to return to the project.

### Purpose

- Provide **on-the-job training** and mentorship.
- Allow contributors to participate with limited privileges while building their track record.
- Reduce barriers for new contributors to join governance roles.

### Eligibility

Anyone can be nominated to an interim position. // TODO EVRARDJP: Any CONTRIBUTOR?

### Scope / Privileges

- Specialty and scope are explicitly documented in `OWNERS` files and/or a public tracking issue.
- A person may have multiple ad-interim roles across specialties (e.g. a contributor can be an interim reviewer on CI and on Providers at the same time). This is to allow a fast path to upskill future project-wide maintainers.
- Anyone effectivated after its interim period does not need to cover requirements from lower positions (e.g. an effectivated interim reviewer on CI does not need to have the criteria of being an effective CI member) // TODO EVRARDJP: CLARIFY WITH SKARLSO
- Any time spent on an interim role counts as participation to the project towards the requirements of the permanent role.

### Limitations

- Limited to a **maximum of three (3) months** for **Members** and **Reviewers**, and **a maximum of six (6) months** for **Maintainers**.
- There can only be a maximum of two interim roles per specialty (two CI/Infra members; two CI/Infra reviewers; two provider Members; two provider Reviewers; two interim Maintainers; etc.).
- Interim Maintainer election follows the same process as Maintainer: it needs **supermajority** approval of Permanent Maintainers.
- Interim Maintainers will not have access to projects' infrastructure credentials
- Interim Maintainers will not have access to projects' Open Source Collective
- Interim Maintainers cannot represent external-secrets for CNCF (they will not be registered as maintainers with the CNCF)
- Interim Maintainers will not have ownership over the organization's GitHub repository.
- Interim roles are not eligible for advancement to permanent roles:
    - Interim Member is not eligible for a permanent reviewer role;
    - Interim Reviewer is not eligible for a permanent maintainer role;
      This is to prevent abuse of the interim system to get permanent roles easily by promoting interim members to them

### Election Process

1. Maintainers discuss and vote on the need for an interim role (lazy consensus).
2. Scope and duration are defined clearly (specialty, responsibilities, expected milestones).
3. Nomination of interim roles is done by lazy consensus.
4. Interim roles are granted for its maximum duration.
5. Interim status is reviewed at the end of the period:
   - If requirements are met → promotion to the permanent role.
   - If not met → revert to previous role; contributor may try again later.

### Abuse handling

Interim roles cannot be considered as substitute for permanent roles.
If a contributor is abusing the interim role system, they may be demoted to their previous role.

---

## Emeritus Status

Emeritus status recognizes former Maintainers, Reviewers, or Members who have made substantial and lasting contributions to the External Secrets project but are stepping down from active responsibilities.

### Purpose

- Honor and recognize long-term contributions.
- Preserve institutional knowledge and mentorship potential.
- Encourage continued engagement with the community without requiring full role responsibilities.

### Eligibility

- Must have held a permanent role (Member, Reviewer, or Maintainer) **for at least twelve (12) consecutive months**.
- Demonstrated sustained, high-quality contributions and collaboration.
- Voluntarily stepping down, retiring, or transitioning out of active participation on good terms.

### Scope / Privileges

- Public recognition on project website, documentation, and GitHub OWNERS or CONTRIBUTORS files.
- Optionally listed in governance and community records as Emeritus Maintainer / Reviewer / Member.
- Maintains access to project communications for discussion and mentorship purposes (read-only on GitHub teams if desired).
- Eligible to provide mentorship or advisory support to new contributors.
- Invited to participate in major project decisions informally, without voting authority.

### Limitations
* No administrative or write access to repositories, releases, or infrastructure.
* Emeritus status is honorary and does not confer any formal responsibilities or authority.


---

### Election Process

This is automatically granted upon step down request, if requirements are met.

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

## Moving up/down the ladder

### Climbing up the ladder

1. **Nomination** by an eligible community member (Member or Higher) via a github issue.
2. **Sponsorship** by two role holders at the **target level or higher** (within the specialty where applicable).
3. **Review** of activity and behavior (quality, reliability, collaboration, responsiveness).
4. **Decision** by lazy consensus of the relevant group (or **supermajority** if contested).

---

### Going down the ladder for inactivity

A **Reviewer** or **Maintainer** role holder may be considered inactive if they have not actively contributed or performed general project responsibilities for **six (6) consecutive months**.

#### Measurement Sources
- GitHub activity: merged PRs, PR reviews, issue triage/comments.
- Participation in community calls or asynchronous design discussions.

#### Process

Step 1, Detection:

* Activity is reviewed at least quarterly by Maintainers or via automation.
* Any Maintainer may propose an inactivity review for a role holder.

Step 2, Notification:

   * A public issue is opened in a `community`/`governance` space (or email if sensitive).
   * The individual is tagged/emailed and given **30 days** to respond.

Step 3, Waiting until the end of grace period:

   * If the contributor indicates intent to return, no change is made.
   * If there is no response or no significant activity within the grace period, proceed.

Step 4, Decision:

   * Demotion is decided by **lazy consensus** of Maintainers, or **supermajority** if contested.

Step 5, Scope/Privileges removal:

   * Demotion via inactivity fully removes the role holder from the organization.
   * Update of the `OWNERS`, GitHub teams, and governance records.
   * Optional: Former Members may be listed as **Emeritus**.

#### Reinstatement
A contributor can be reinstated at their previous level via the standard advancement process. Prior history is considered favorably.

---

## Voting rights summary

Voting rights vary by decision type:

| Decision Type                                    | Eligible Voters                              |
|--------------------------------------------------|----------------------------------------------|
| **Governance changes**                           | Permanent Maintainers                        |
| **Adding/removing Maintainers**                  | Permanent Maintainers                        |
| **Interim role appointments (Member/Reviewer)**  | Permanent Maintainers                        |
| **Interim role appointment (Maintainer)**        | Permanent Maintainers                        |
| **Project-wide technical direction**             | All Maintainers                              |
| **Security incident decisions**                  | All Maintainers                              |
| **Technical decisions within a specialty**       | All Reviewers and Maintainers                |

**Notes:**

- Interim role holders do **not** vote in role promotions/demotions.
- Company voting limits apply: maintainers/reviewers from the same declared employer have **two** combined votes - regardless if they contribute individually or representing their employer. An exception to this rule is for voting to add/remove maintainers. In this case, a company voting limit is of **one** vote.
- If more than two maintainers/reviewers from the same declared employer cannot reach consensus for their vote (e.g. 3 in favour, 2 against), both the employer's votes are recorded as **abstain**.


## Cross-References

TODO EVRARDJP: CLARIFY WITH SKARLSO v (current leadership pointer to codeowners + how-to propose someone as maintainer?)

- Project governance and decision-making: see [policy](policy.md)
- Proposal process and template: `design/000-template.md`
- Specialty ownership: `OWNERS` files per directory
