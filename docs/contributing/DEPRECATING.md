# HOW-TO deal with impactful removals

## Remind yourself of our policy

Read the [deprecation policy reference](deprecation_policy.md) first.

## Showcase to community

The proposing maintainer must present the proposed deprecation to the maintainer group. This can be done synchronously during a community meeting or asynchronously, through a GitHub Pull Request.

## Voting

The maintainers will vote to the accept the deprecation, following our [policy](policy.md#decisions-and-tie-breaking)

## Implementation of the first stage: deprecation

Upon approval, the proposing maintainer may now implement the changes required to mark the feature as deprecated.

This includes:

* Updating the codebase with deprecation warnings where applicable.
* Documenting the deprecation in release notes and relevant documentation.
* Updating APIs, metrics, or behaviors per the Kubernetes Deprecation Policy if in scope.
* If the feature is entirely deprecated (e.g., OLM-specific builds), archival of any associated repositories.

The rest of the implementation follows our usual process (reviews, release, ...)

## Implementation of the final stage: Removal

After the amount of time/releases planned in the proposal, ANY contributor can implement the removal changes.

This includes:

* Deleting the relevant parts of the codebase.
* Updating the old feature in the documentation to an archive
* Updating APIs, metrics, or behaviors per the Kubernetes Deprecation Policy.
* If the feature is entirely deprecated (e.g., OLM-specific builds), archival of any associated repositories.

When a deprecated feature is finally removed, it must be communicated in the relevant release's notes.
