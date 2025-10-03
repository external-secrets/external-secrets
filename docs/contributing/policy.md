<!-- Explanation. We explain why we do things the way we do here. -->
<!-- If you want to contribute to this page: -->
<!-- Analyse whether you want to express Can you teach me to x? (Search for the appropriate tutorial) -->
<!-- Analyse whether you want to express HOW DO I follow process x (Search for the appropriate how-to page) -->
<!-- Analyse whether you want to express WHY we have/do process x unrelated to contributions (build another explanation page) -->
<!-- Analyse whether you want to express WHAT IS contributing process x (Search for the appropriate reference page) -->

# Contributing and governance Policy

This document describes the policies and guidelines for contributing to the External Secrets project.
It covers practices ranging from code contributions to our voting processes.
Security policy is in a separate document

## Code of Conduct

We expect all contributors to adhere to our [Code of Conduct](CODE_OF_CONDUCT.md).
This is mandatory for all contributors, regardless of the nature of their contribution (code, documentation, community support, etc.).

Next to that, as a CNCF project, you must also follow the CNCF Code of Conduct.

## Contributor License Agreement

As a CNCF project, any contributor must (where applicable) sign the CNCF Contributor License Agreement (CLA) before the project accept their contribution.

## Code contributions must be tracked until completion, no stragglers

The work is not over once your code is pushed.

We are using the pull request `assignee` feature to track who is responsible
for the lifecycle of the PR: review, merging, ping on inactivity, close.

As an author or an `assignee`, you are responsible for bringing the code to completion.
It is a community effort to bring new features or bug fixes in. It is not solely on the shoulders of the maintainers.

The maintainers will close pull requests or issues if there is no follow-up or response from the author for  a period of time.
Feel free to reopen if you want to get back on it.

## Code contributions must include documentation updates where applicable

Because docs quality is as important as code quality, it is expected to maintain documentation in `/docs` when contributing features or changing existing behavior.
Read the `Setting up the development environment for documentation contributions` section in `CONTRIBUTING.md` for more information.

## Long term impact of each code contribution

> A "no" from a maintainer is temporary, a "yes" if most likely forever.

Ideally, all code contributions should be:

- In conformance with the community long-term vision (use a proposal if required)
- Linked to a triaged issue (documenting the problem or feature in depth)
- Passing all the tests (do not break current expectations)
- Approved by owners (follow our openness principles)

## Large code contributions

Pull requests that are labelled with _size/l_ and above _MUST_ have at least **TWO**
approvers for it to be merged.
We need this to ensure the quality of code (long-term maintainability) in this project.

### Use of design proposals for impactful changes

Every "extra large" or "with user impact" work needs a proposal.
This is our way to record large redesigns without losing context over time.
It makes it possible to **gather feedback from the community to ensure that we progress in the right direction**
before we develop and release big changes.

Significant changes include for example:

* creating new custom resources
* proposing breaking changes
* changing the behavior of the controller significantly

These are proposals under the repository’s `design/` directory.
Each proposal go through the following cycle: **New** → **Reviewing** → **Accepted** or **Rejected**.

The maintainers team will move the status of the proposal during community meeting.
The project's roadmap is shaped by **Accepted** proposals.

### Policy changes are considered impactful

Substantive changes to this document require a **supermajority** of maintainers. This is the way we keep up with our quality standards.

### Deprecations are considered impactful

Any change in our deprecation policy must come with a proposal.
The proposal must clearly outline the rationale for inclusion, the impact on users, stability, long term maintenance plan, or any day-to-day activities.
For more technical details about WHAT is covered by this deprecation policy, check our [Deprecation Policy Reference](deprecation_policy.md)

While the project is currently considered as in *Beta* state (see table below), we are considering deprecations very thoroughly.

// TODO EVRARDJP: CHECK WITH SKARLSO

| Version | Status    | Notes                                                                                                                                                                                                                       |
|---------|-----------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| v0.9.x  | **Alpha** | Experimental features; API may change without notice. New features may be added, removed, or changed without a major version bump. Use only in short-lived test clusters unless you are prepared for manual migration work. |
| v1.0.x  | **Beta**  | Features are stable enough for day-to-day use. Production‑ready for most workloads; incompatible changes only in minor releases. Backwards-compatible changes are not the norm, but migration docs will be provided.        |
| ≥v2.0   | **GA**    | The API surface is considered stable for many releases in a row; major upgrades are rare and well‑documented in advance. Upgrades are typically non-disruptive. If they are disruptive, a migration guidance is provided.   |


## LLM Policies

### Using LLMs for AI Generated Content

We're not against using AI tools.
They can be genuinely helpful for drafting code, catching bugs, or exploring ideas.

What we don't want is the obvious copy-paste output that hasn't been reviewed, understood, or adapted to our project.
You can tell when something was generated by an LLM and submitted without a second thought.

The overly formal language, the generic explanations that don't quite fit the context, the boilerplate comments that add nothing, the solutions that technically work but ignore our existing patterns. That's what we're trying to avoid.
If you use an LLM to help with your contribution, that's fine.
Just make sure you actually read what it produced, verify it works, check it follows our coding standards, and adjust it to match how we do things here.
Add your own context. Remove the fluff. Make it yours.

The same goes for issues. Don't submit LLM-generated bug reports or feature requests that are verbose, generic, and obviously haven't been thought through.
If an AI helped you articulate something, great, but the issue should still sound like it came from someone who actually encountered a problem or has a real use case.

We value contributions from humans who understand what they're submitting, even if they had some algorithmic assistance along the way.
The goal is quality and genuine engagement with the project, not quantity of AI-generated content.

### Using LLMs for Reviews

We welcome the use of AI to review work, as it might help us follow our policy here above.

We are currently evaluating CodeRabbit https://app.coderabbit.ai/login and GitHub Copilot for our contributions.
Be aware of [algorithm bias](https://en.wikipedia.org/wiki/Algorithmic_bias) when reviewing code.

## Contributor ladder and CNCF perspective

ESO governance aims for open, transparent, and vendor-neutral operation consistent with CNCF expectations.
The [Contributor Ladder](contributor_ladder.md) provides clear pathways for community members to grow into leadership, as recommended by the CNCF.

### Limitation of impact: Company voting in project roles

Votes to nominate maintainers by contributors belonging to the same employer count as **one** collective vote.
If they cannot reach internal consensus, that employer’s vote is recorded as **abstain**.

The intent is to prevent becoming a "single company/vendor" project, impacting the sustainability of the external-secrets ecosystem.
Next to that, more votes of a unique company could technically lead to unbalancing the project towards the company.

## Decisions and tie-breaking

ESO strives for **consensus** via open discussion. When consensus cannot be reached, any eligible voter may call a vote.

- **Supermajority**: Two-thirds (⅔) of eligible voters in the group.
- **Venues**: Votes may occur on GitHub, email, Slack, community meetings, or a suitable voting service.
- **Ballots**: “Agree/+1”, “Disagree/-1”, or “Abstain” (counts as no vote).

TODO: EVRARDJP WITH SKARLSO: Deprecations: "A majority vote of maintainers is required to approve the deprecation." this is not the supermajority

### Lazy consensus principle

ESO uses [Lazy Consensus](https://community.apache.org/committers/decisionMaking.html#lazy-consensus) for most decisions.

- PRs and proposals should allow **at least five (5) working days** for comments and consider global holidays.
- Other maintainers may request additional time when justified and commit to a timely review.

**Exclusions** (lazy consensus does **not** apply):
- Removal of maintainers.
- Any substantive governance changes.

## Cutting Releases

The external-secrets project is released on an as-needed basis.
Nobody is entitled to new releases. Our community work is best-effort.
Feel free to open an issue to request a release and help on such release.
Details on how to cut a release can be found in the [Contributing](CONTRIBUTING.md#releasing) page.

## Handling community events

We welcome you to spread the love about external secrets.

External Secrets being a CNCF project, you have to follow the CNCF guidelines for the use of marketing materials.
When in doubt, contact the maintainers through in slack (#external-secrets) if you want to organize a community event
(e.g., meetup, webinar, workshop, etc.) around External Secrets.

Sadly, as mentioned in our [Contributing Journey](journey.md), we do not have marketing material available.

## Financial contributions

TODO

## Security policy

The list of our security processes is in our [Security Policy Reference](security_policy.md).
