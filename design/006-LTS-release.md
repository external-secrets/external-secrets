# Long Term Support Policy

WRT: https://github.com/external-secrets/external-secrets/issues/2044

We want to provide security patches and critical bug fixes in a timely manner to our users.
To do so, we offer long-term support for our latest two (N, N-1) software releases.
We aim for a 2-3 month **minor** release cycle, i.e. a given release is supported for about 4-6 months.

We want to cover the following cases:

- weekly image rebuilds to update OS dependencies
- weekly go dependency updates
- backport bug fixes on demand

Note: features cut off on a minor release will not be backported to older releases.

## Automatic Updates

We have set up a Github Action (GHA) which will automatically update the `go.mod` dependencies once per week or on request.
The GHA will make the necessary code changes and opens a PR. Once approved and merged into `main` or `release-x.y` our build pipelines
will build and push the artifact to ghcr.

## Manual Updates

Bug Fixes will be merged onto each release branch individually.
This is achieved by creating separate PRs from a corresponding branch of the release
(e.g. bug fixes targetting `release-1.0` should be created from `release-1.0` branch).
Once approved and merged into `main` or `release-x.y`, ou build pipeline will build and push the artifact to ghcr

## Process

### Branch Management

When a new **minor release** is cut and merged into `main`, we must branch off to `release-{major}.{minor}`.
This is the long-lived release branch that will get dependency updates and bug fixes.
In case we do a `patch` release we **must also merge** into the correct `release-{major}.{minor}` branch.

### Release Issue Template

We'll have a release issue template that gives the release lead a task list to work through all the steps needed to create a release.

#### Release Preparation Tasks

- [ ] ask in `#external-secrets-dev` if we're ready for a release cut-off or if something needs to get urgently in
- [ ] docs: [stability & support page](https://external-secrets.io/main/introduction/stability-support/) is up to date
  - [ ] version table
  - [ ] Provider Stability and Support table
  - [ ] Provider Feature Support table
- [ ] docs: update [roadmap page](https://external-secrets.io/main/contributing/roadmap/)
- [ ] tidy up [Project Board](https://github.com/orgs/external-secrets/projects/2)
  - [ ] move issues to next milestone
  - [ ] close milestone

#### Release Execution

- [ ] Follow the [Release Process guide](https://external-secrets.io/main/contributing/release/)

#### After Release Tasks

- [ ] Announce release on `#external-secrets` in Slack
