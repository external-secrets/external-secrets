## Project Management
The Code, our TODOs and Documentation is maintained on
[GitHub](https://github.com/external-secrets/external-secrets). All Issues
should be opened in that repository.
We have a [Roadmap](roadmap.md) to track progress for our road towards GA.

## Issues

Features, bugs and any issues regarding the documentation should be filed as
[GitHub Issue](https://github.com/external-secrets/external-secrets/issues) in
our repository.

## Issues and pull requests labels

We have [our github labels](https://github.com/external-secrets/external-secrets/labels) to help classify the issues and pull requests.

They are categorized as this:

| Label                       | Purpose                                                                                                                                                                                                                                                                                                                                                                                               |
|-----------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `kind/`                     | Kinds classification allows maintainer to easily spot what the issue is about (not where is the code touched). Bugs (kind/bug), features (kind/feature), maintenance items (kind/chore, kind/refactor, kind/cleanup, kind/dependency), initiatives (kind/design), performance changes (kind/performance), support-questions (kind/support).                                                                            |
| `triage/`                   | Triage labels tracks the issue's lifecycle (or change's lifecycle). Suggested mappings: `triage/pending-triage` (not triaged yet), `triage/needs-information` (missing details from reporter), `triage/not-reproducible` (cannot reproduce with provided info), `triage/confirmed` (valid/reproducible report), `triage/invalid` (incorrect bug report/PR), `triage/duplicate` (duplicate bug report) |
| `priority/`                 | Priority shows an increasing urgency from "important-long-term", "important-soon" to "urgent"                                                                                                                                                                                                                                                                                                         |
| `size/`                     | Allows maintainers to better guess the amount of work required to review the issue/PR                                                                                                                                                                                                                                                                                                                 |
| `area/`                     | Areas allow the routing of the work (issues/PR) based on our knowledge areas. It's intent is to target specific experts in their domain. Each provider has its own area. Examples: `area/aws`, `area/azure`, `area/charts`, `area/documentation`, `area/dependencies`.                                                                                                                                |
| `breaking-change`           | Highlights the impact on future users on the issue/change                                                                                                                                                                                                                                                                                                                                             |
| `cncf`                      | Highlights the impact on the CNCF (foundation work, support, maturity tracking)                                                                                                                                                                                                                                                                                                                       |
| `discuss-community-meeting` | Highlights the need to discuss this in a community meeting for consensus building                                                                                                                                                                                                                                                                                                                     |
| `release-blocker`           | Issue/PR that block next release                                                                                                                                                                                                                                                                                                                                                                      |
| `good-first-issue`          | An issue/PR good for newcomers. Do not hesitate to comment on it to mark your interest and tackle the work!                                                                                                                                                                                                                                                                                           |
| `help-wanted`               | Extra attention of the maintainers team is requested                                                                                                                                                                                                                                                                                                                                                  |
| `Stale`                     | Is used by a bot to highlight stale PRs/issues that will automatically close if no further activity happens                                                                                                                                                                                                                                                                                           |

With those labels, maintainers are able to efficiently see where they can help better.

Note: Some labels  like `size` and `kind` are applied automatically to Pull Requests by semantic conventions. See also the section below about submitting a pull request.

## Submitting a Pull Request

This project uses the well-known pull request process from GitHub. To submit a
pull request, fork the repository and push any changes to a branch on the copy,
from there you can create a pull request to our main repo.

Please note we are following the [conventional commits](https://www.conventionalcommits.org/en/v1.0.0/#summary) .
The use of the right semantics helps the maintaining team to classify your PR effectively.
**We will only merge PRs matching those conventions.**

Merging a pull request only happens when:

* there is ideally an issue that documents the problem or feature in depth.
* code has a reasonable amount of test coverage
* tests pass
* At least one review approves the change

Once these steps are completed the PR will be merged by a code owner.
We're using the pull request `assignee` feature to track who is responsible
for the lifecycle of the PR: review, merging, ping on inactivity, close.
We close pull requests or issues if there is no response from the author for
a period of time. Feel free to reopen if you want to get back on it.

_Note:_
Pull requests that are labelled with _size/l_ and above _MUST_ have at least **TWO**
approvers for it to be merged. Please respect this policy to ensure the quality
of code in this project.

### Triggering e2e tests

We have an extensive set of e2e tests that test the integration with *real* cloud provider APIs.
Maintainers must trigger these kind of tests manually for PRs that come from forked repositories. These tests run inside a `kind` cluster in the GitHub Actions runner:

```
/ok-to-test sha=<full_commit_hash>
```
Examples:
```
/ok-to-test sha=b8ca0040200a7a05d57048d86a972fdf833b8c9b
```

#### Executing e2e tests locally

You have to prepare your shell environment with the necessary variables so the e2e test
runner knows what credentials to use. See `e2e/run.sh` for the variables that are passed in.
If you e.g. want to test AWS integration make sure set all `AWS_*` variables mentioned
in that file.

Use [ginkgo labels](https://onsi.github.io/ginkgo/#spec-labels) to select the tests
you want to execute. You have to specify `!managed` to ensure that you do not
run managed tests.

```
make test.e2e GINKGO_LABELS='gcp&&!managed'
```

#### Managed Kubernetes e2e tests

There's another suite of e2e tests that integrate with managed Kubernetes offerings.
They create real infrastructure at a cloud provider and deploy the controller
into that environment.
This is necessary to test the authentication integration
([GCP Workload Identity](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity),
[EKS IRSA](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html)...).

These tests are time intensive (~20-45min) and must be triggered manually by
a maintainer when a particular provider or authentication mechanism was changed:

```
/ok-to-test-managed sha=xxxxxx provider=aws
# or
/ok-to-test-managed sha=xxxxxx provider=gcp
# or
/ok-to-test-managed sha=xxxxxx provider=azure
```

Both tests can run in parallel. Once started they add a dynamic GitHub check `integration-managed-(gcp|aws|azure)` to the PR that triggered the test.


### Executing Managed Kubernetes e2e tests locally

You have to prepare your shell environment with the necessary variables so the e2e
test runner knows what credentials to use. See `.github/workflows/e2e-managed.yml`
for the variables that are passed in. If you e.g. want to test AWS integration make
sure set all variables containing `AWS_*` and `TF_VAR_AWS_*` mentioned in that file.

Then execute `tf.apply.aws` or `tf.apply.gcp` to create the infrastructure.

```
make tf.apply.aws
```

Then run the `managed` testsuite. You will need push permissions to the external-secrets ghcr repository. You can set `IMAGE_NAME` to control which image registry is used to store the controller and e2e test images in.

You also have to setup a proper Kubeconfig so the e2e test pod gets deployed into the managed cluster.

```
aws eks update-kubeconfig --name ${AWS_CLUSTER_NAME}
or
gcloud container clusters get-credentials ${GCP_GKE_CLUSTER} --region europe-west1-b
```

Use [ginkgo labels](https://onsi.github.io/ginkgo/#spec-labels) to select the tests
you want to execute.

```
# you may have to set IMAGE_NAME=docker.io/your-user/external-secrets
make test.e2e.managed GINKGO_LABELS='gcp'
```

## Proposal Process
Before we introduce significant changes to the project we want to gather feedback
from the community to ensure that we progress in the right direction before we
develop and release big changes. Significant changes include for example:

* creating new custom resources
* proposing breaking changes
* changing the behavior of the controller significantly

Please create a document in the `design/` directory based on the template `000-template.md`
and fill in your proposal. Open a pull request in draft mode and request feedback. Once the proposal is accepted and the pull request is merged we can create work packages and proceed with the implementation.

## Release Planning

We have a [GitHub Project Board](https://github.com/orgs/external-secrets/projects/2/views/1) where we organize issues on a high level. We group issues by milestone. Once all issues of a given milestone are closed we should prepare a new feature release. Issues of the next milestone have priority over other issues - but that does not mean that no one is allowed to start working on them.

Issues must be _manually_ added to that board (at least for now, see [GH Roadmap](https://github.com/github/roadmap/issues/286)). Milestones must be assigned manually as well. If no milestone is assigned it is basically a backlog item. It is the responsibility of the maintainers to:

1. assign new issues to the GH Project
2. add a milestone if needed
3. add appropriate labels

If you would like to raise the priority of an issue for whatever reason feel free to comment on the issue or ping a maintainer.

## Support & Questions

Providing support to end users is an important and difficult task.
We have three different channels through which support questions arise:

1. Kubernetes Slack [#external-secrets](https://kubernetes.slack.com/archives/C017BF84G2Y)
2. [GitHub Issues](https://github.com/external-secrets/external-secrets/issues/new/choose)

## Cutting Releases

The external-secrets project is released on a as-needed basis. Feel free to open a issue to request a release. Details on how to cut a release can be found in the [release](release.md) page.
