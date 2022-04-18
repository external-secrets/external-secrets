## Project Management
The Code, our TODOs and Documentation is maintained on
[GitHub](https://github.com/external-secrets/external-secrets). All Issues
should be opened in that repository.

## Issues

Features, bugs and any issues regarding the documentation should be filed as
[GitHub Issue](https://github.com/external-secrets/external-secrets/issues) in
our repository. We use labels like `kind/feature`, `kind/bug`, `area/aws` to
organize the issues. Issues labeled `good first issue` and `help wanted` are
especially good for a first contribution. If you want to pick up an issue just
leave a comment.

## Submitting a Pull Request

This project uses the well-known pull request process from GitHub. To submit a
pull request, fork the repository and push any changes to a branch on the copy,
from there a pull request can be made in the main repo. Merging a pull request
requires the following steps to be completed before the pull request will
be merged:

* ideally, there is an issue that documents the problem or feature in depth.
* code must have a reasonable amount of test coverage
* tests must pass
* PR needs be reviewed and approved

Once these steps are completed the PR will be merged by a code owner.
We're using the pull request `assignee` feature to track who is responsible
for the lifecycle of the PR: review, merging, ping on inactivity, close.
We close pull requests or issues if there is no response from the author for
a period of time. Feel free to reopen if you want to get back on it.

### Triggering e2e tests

We have an extensive set of e2e tests that test the integration with *real* cloud provider APIs.
Maintainers must trigger these kind of tests manually for PRs that come from forked repositories. These tests run inside a `kind` cluster in the GitHub Actions runner:

```
/ok-to-test sha=xxxxxx
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

There's another suite of e2e tests that integrate with managed Kuberentes offerings.
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
```

Both tests can run in parallel. Once started they add a dynamic GitHub check `integration-managed-(gcp|aws)` to the PR that triggered the test.


### Executing Managed Kubernetes e2e tests locally

You have to prepare your shell environment with the necessary variables so the e2e
test runner knows what credentials to use. See `.github/workflows/e2e-managed.yml`
for the variables that are passed in. If you e.g. want to test AWS integration make
sure set all variables containing `AWS_*` and `TF_VAR_AWS_*` mentioned in that file.

Then execute `tf.apply.aws` or `tf.apply.gcp` to create the infrastructure.

```
make tf.apply.aws
```

Then run the `managed` testsuite. You will need push permissions to the external-secrets ghcr repository. You can set `IMAGE_REGISTRY` to control which image registry is used to store the controller and e2e test images in.

You also have to setup a proper Kubeconfig so the e2e test pod gets deployed into the managed cluster.

```
aws eks update-kubeconfig --name ${AWS_CLUSTER_NAME}
or
gcloud container clusters get-credentials ${GCP_GKE_CLUSTER} --region europe-west1-b
```

Use [ginkgo labels](https://onsi.github.io/ginkgo/#spec-labels) to select the tests
you want to execute.

```
# you may have to set IMAGE_REGISTRY=docker.io/your-user/external-secrets
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

## Cutting Releases

The external-secrets project is released on a as-needed basis. Feel free to open a issue to request a release. Details on how to cut a release can be found in the `RELEASE.md` file in the repo.
