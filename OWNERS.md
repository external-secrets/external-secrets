# External Secrets Owners

This document maps **specialty areas** to GitHub teams used for reviews and approvals.  
It complements the automation in [`CODEOWNERS`](./CODEOWNERS) and the roles defined in
[`CONTRIBUTOR_LADDER.md`](./CONTRIBUTOR_LADDER.md).

- **Reviewer**: may review and `/lgtm` within their specialty.
- **Approver**: may `/approve` and merge within their specialty.
- **Maintainer**: project-wide governance and release authority.

> Manage membership via GitHub Teams, not this file—keep teams stable and adjust members in org settings.

---

## Maintainers (project-wide)
- Teams: **@external-secrets/maintainers**
- Scope: entire repository; final escalation point.

## Interim Maintainers (project-wide)
- Teams: **@external-secrets/interim-maintainers**
- Scope: entire repository;

---

## Specialties & Paths

### 1) CI / Infrastructure
- **Paths**: `.github/`, `scripts/`, `build/`
- **Reviewers**: `@external-secrets/ci-reviewers`

### 2) Testing
- **Paths**: `test/`, `tests/`, `hack/`
- **Reviewers**: `@external-secrets/testing-reviewers`

### 3) Core Controllers
- **Paths**: `apis/`, `pkg/controllers/`
- **Reviewers**: `@external-secrets/core-reviewers`

### 4) Providers
- **Paths**: `pkg/provider/` (and subfolders like `aws/`, `gcp/`, `azure/`, `vault/`, `cyberark/`), and their respective API files.
- **Reviewers**: `@external-secrets/providers-reviewers`

### 5) Security
- **Paths**: `security/`, `docs/security/`
- **Reviewers**: `@external-secrets/security-reviewers`

---

## Interim Role Holders
If someone holds an **Interim Member** or **Interim Reviewer** role for a specialty, note it in a public tracking issue and (optionally) list here as informational, e.g.:
- `@username` — Interim Reviewer, Providers (until YYYY-MM-DD)

> Interim roles are time-boxed and governed by the policy in `CONTRIBUTOR_LADDER.md`.
