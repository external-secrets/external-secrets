ESO and the ESO Helm Chart have two distinct lifecycles and can be released independently. Helm Chart releases are named `external-secrets-x.y.z`.

The external-secrets project is released on a as-needed basis. Feel free to open a issue to request a release.

## Multi-Module Versioning

External Secrets Operator uses a multi-module structure with the following modules:
- `/apis` - CRD types and interfaces
- `/runtime` - Shared utilities
- `/providers/v1/*` - Individual provider modules
- `/generators/v1/*` - Individual generator modules
- `/` (root) - Main module with controllers and binary

**All modules share the same version tag.** When releasing version `v0.x.y`, a single git tag is created that applies to all modules in the repository. Go's module system automatically handles this, and consumers can reference any module using the same version tag.

For example:
```go
require (
    github.com/external-secrets/external-secrets/apis v0.10.0
    github.com/external-secrets/external-secrets/runtime v0.10.0
    github.com/external-secrets/external-secrets/providers/v1/aws v0.10.0
)
```

**Important:** When updating dependencies that consume ESO modules, ensure all module references use the same version to maintain compatibility.

## CHANGELOG and Release Notes

### The Policy: Why we automate (with a human touch)

To reduce toil and ensure consistency across releases, the External Secrets project utilizes GitHub's automated release notes feature to generate our changelog.

We chose this hybrid approach of automation plus a "human touch" for three main reasons:

1. **Less maintenance burden** — Contributors should not have to update multiple places (PR description and a CHANGELOG file). The release workflow gathers PRs into categories automatically, making the changelog a natural by-product.
2. **GitHub is the source of truth** — Our `CHANGELOG.md` mirrors what GitHub generates. Keeping a single source of truth avoids drift.
3. **Human Quality Control** — The automation generates a *baseline draft* based on PR labels and titles. Before any release is finalized, maintainers review this draft to rewrite confusing titles, highlight `⚠️ Breaking Changes`, and ensure the overall quality and readability of the release notes.

> **Note:** Breaking changes require manual writing. While we automate the gathering and categorization, the context of each release note comes from the PR body. Contributors must write clear upgrade notes for any breaking change.

### The Practice: How to ensure great release notes

For the automation to work effectively, it relies on two main components: clear Pull Request titles and accurate labeling. Every PR becomes a changelog entry. The automation handles the grouping; you handle the content.

#### 1. For Maintainers: Labels determine the category

The `.github/release.yml` file defines the mapping. The category a PR falls into during the release draft generation is entirely dictated by its labels. When reviewing and merging a PR, ensure the correct `kind/*` or `area/*` labels are applied:

| Label(s) | Category in release notes | When to use |
|----------|---------------------------|-----|
| `breaking-change` | **⚠️ Breaking Changes / Urgent Upgrade Notes** | API changes, removed fields, changed upgrade behavior. This label must be added manually to PRs that introduce breaking changes. |
| `kind/feature`, `kind/improvement` | 🚀 Features & Improvements | New functionality or improvements from the user's perspective. |
| `kind/bug` | 🐛 Bug Fixes | Bug fixes, not just code bugs but also configs or infrastructure that were wrong. |
| `area/documentation` | 📖 Documentation | Docs-only changes. |
| `security` | 🔒 Security | Security-related fixes or advisories. |
| `kind/chore`, `kind/refactor`, `kind/maintenance` | 🧹 Maintenance & Refactoring | Internal cleanup, refactoring, or maintenance tasks. |
| `area/dependencies`, `dependencies` | Dependencies | Dependency updates. |

*Note: If a PR should not be included in the public changelog (e.g., a minor CI tweak), apply the `skip-changelog` label.*

#### 2. For Contributors: Writing a good changelog entry

Each PR title becomes the changelog entry. Follow these guidelines:

- **Write the PR title as a changelog entry** — it should make sense to end users without reading the full PR body.
- **Use the conventional commit format** in the PR title: `feat(scope): description` or `fix(scope): description`. The scope is optional.
- **Describe the impact, not the implementation** — "Add AWS Secrets Manager integration" not "Add AWS SDK call to provider".
- **Reference the issue** — use `Fixes #...` in the PR body.

#### Breaking changes

If your PR introduces a breaking change:

1. Add the `breaking-change` label.
2. Fill in the **"Breaking Changes / Upgrade Notes"** section in the PR template with:
   - What changed and why.
   - What users need to do to upgrade.
   - Any migration steps or deprecation timeline.
3. Write the PR title so the breaking change is clearly visible.

PRs with the `breaking-change` label appear in a dedicated section at the very top of the release notes, making them easy to spot for anyone performing an upgrade.

## Release ESO

When doing a release it's best to start with the ["Create Release" issue template](https://github.com/external-secrets/external-secrets/issues/new?assignees=&labels=area%2Frelease&projects=&template=create_release.md&title=Release+x.y), it has a checklist to go over.

⚠️ Note: when releasing multiple versions, make sure to first release the "old" version, then the newer version.
Otherwise the `latest` documentation will point to the older version. Also avoid to release both versions at the same time to avoid race conditions in the CI pipeline (updating docs, GitHub Release, helm chart release).

1. Make sure the [stability & support page](https://external-secrets.io/latest/introduction/stability-support/) is up to date. The new version should be listed in the version table before proceeding with the release.
1. Make sure there is no pending CI jobs running. This is to avoid promoting a stale image to a new version (we need to rely on _existing_ pushed images for release).
1. Run `Create Release` Action to create a new release, pass in the desired version number to release.
    1. choose the right `branch` to execute the action: use `main` when creating a new release.
    2. ⚠️ make sure that CI on the relevant branch has completed the docker build/push jobs. Otherwise an old image will be promoted.
2. GitHub Release, Changelog will be created by the `release.yml` workflow which also promotes the container image.
3. update Helm Chart, see below

## Release Helm Chart

1. Update `version` and/or `appVersion` in `Chart.yaml` and run `make helm.docs helm.update.appversion helm.test.update docs.update test.crds.update`
2. push to branch and open pr
3. run `/ok-to-test-managed` commands for all cloud providers
4. merge PR if everything is green
5. CI picks up the new chart version and creates a new GitHub Release for it

The following things are updated with those commands:
1. Update helm docs
2. Update the apiVersion in the snapshots for the helm tests
3. Update all the helm tests with potential added values
4. Update the stability docs with the latest minor version if exists
5. Update the CRD conformance tests

The branch to create this release should be `release-chart-x.y.z`. Though be aware that release branches are _immutable_.
This means that if there is anything that needs to be fixed, a new branch will need to be created.

Also, keep an eye on `main` so nothing is merged while the chart branch is running the e2e tests. If that happens,
the chart PR CANNOT be merged because we don't allow not up-to-date pull requests to be merged. And you can't update
because the branch is immutable.
