<!-- This explains "HOW do I contribute" -->
<!-- It is not the "tutorial/learning the contributor journey", (docs/contributing/journey.md) -->
<!-- It is not a place to clarify/explain the why of our processes (docs/contributing/policy.md) -->

# Contributing

Start with the [Contributing journey](journey.md) to learn how to help us in general.
Read the [Contribution Policy](policy.md) to understand why our contribution guidelines are the way they are
(code of conduct, LLM policy, PR message conventions, etc.).
Get the ball rolling by realizing the actions described in this document.

## Accept the legal requirements

You will be blocked by the CI bot if you do not sign the CNCF CLA.

Read why in our [Contribution Policy](policy.md#contributor-license-agreement).
All contributors must follow our Code of Conduct, the CNCF Code of Conduct, and sign the CNCF CLA (where applicable) before we accept any contributions.

## Making sure you are not working on something already fixed

If you found an issue, others might have too.
Start by browsing our [GitHub Issues](https://github.com/external-secrets/external-secrets/issues) for any duplicate.

### No existing ticket for your issue?

Create an issue! Simply follow our GitHub issue template.

### Found one? Comment on it

Help [triage the issue](journey.md#the-value-of-triaging-issues).
Mention in your comment whether if you can tackle the issue or not. If you cannot, mention what is required of the team to help you fix it.

## New feature? Check for alignment with the roadmap

Our [Project Board](https://github.com/orgs/external-secrets/projects/2/views/1) contains our vision for next releases.
If your change is aligned, continue further with this document.
If you don't know, you should ask on Slack first if you are afraid of doing something that might not get merged.

We welcome any change aligned with the roadmap. We will prioritize work in the next milestone before working on the longer term items.
If you would like to raise the priority of an issue, comment on the issue or ping a maintainer.

## Setting up the development environment for ALL technical contributions

To technically (i.e. not community or financial) contribute to ESO, you need the following tools installed on your machine:

- A working [Go environment](https://golang.org/doc/install) (see your branch's version in `go.mod`)
- Git
- GNU make
- [yq](https://github.com/mikefarah/yq), version 4.44.3 or higher.
- jq, version 1.6 or higher
- docker (Do not forget to add your user into docker group, which is equivalent to root rights, `sudo usermod -aG docker $USER`)
- kind
- kubectl
- helm 3.0 or above
- bash
- gnu sed
- docker and buildkit
- terraform

The following tools will be fetched and installed if they are not present on your system:

- envtest
- golangci-lint
- [tilt](https://docs.tilt.dev/install.html)
- cty

Optional tools:

- ctlptl, a companion to tilt (Read [ctlptl install guide](https://github.com/tilt-dev/ctlptl/blob/main/INSTALL.md))
- a local registry `docker run -d --restart=always -p "5000:5000" --name kind-registry registry:2`

Then fetch our code by cloning the repository:

```shell
git clone https://github.com/external-secrets/external-secrets.git
cd external-secrets
```

### Adding a new source code file

All source code files (docs, CI tools configuration and manifests are excluded) must include the Apache License 2.0 header.
The CI automatically checks license headers for new files added in pull requests using [Apache SkyWalking Eyes](https://github.com/apache/skywalking-eyes).

If you need to check license headers locally, you can use the SkyWalking Eyes tool directly:

```shell
make license.check
```

The configuration of SkyWalking Eyes is in `.licenserc.yaml` at the project root

## Setting up the development environment for general operator contributions

### Writing code

TODO: Express what we want in terms of code style, comments, tests, etc.

### Before you start on large work...

If you are to work on very large items, please submit a proposal. Please read our [policy for large contributions](policy.md#large-code-contributions) first.

For that:

1. copy the proposal template at `design/000-template.md` to `design/xxx-topic.md`, keeping the same structure.
2. Mention objectives, use cases, and high-level technical approach.
3. Open a pull request in draft mode and request feedback, prior to implementation.

Once the proposal is merged as "accepted" and contains work packages, proceed with the implementation.

### Writing ESO code with tilt (recommended)

[Tilt](https://tilt.dev) can be used to develop external-secrets. Tilt will hot-reload changes to the code and replace
the running binary in the container using a process manager of its own.

To run tilt, download the utility for your operating system and run `make tilt-up`. This will do two things:
- downloads tilt for the current OS and ARCH under `bin/tilt`
- make manifest files of your current changes and place them under `./bin/deploy/manifests/external-secrets.yaml`
- run tilt with `tilt run`

Hit `space` and you can observe all the pods starting up and track their output in the tilt UI.

### Building & Testing manually, without tilt (advanced)

The project uses the `make` build system. It will run code generators, tests and
static code analysis.

Building the operator binary and docker image:

```shell
make build
make docker.build IMAGE_NAME=external-secrets IMAGE_TAG=latest
```

Run tests and lint the code:
```shell
make lint # Alternatively you run your own `golangci-lint run` command either directly or in a container: docker run --rm -v $(pwd):/app -w /app golangci/golangci-lint:v1.49.0 golangci-lint run
make test # This will run the unittests, but not the e2e/managed tests
```

You can then run the code locally on your machine with:

```shell
make crds.install
make run
```

If you do not want to run directly but want to test ESO in-cluster, push your image directly to your kind cluster OR to your local registry.

TIP: If you create your kind cluster with `ctlptl create cluster kind --registry=kind-registry`, it will automatically be wired to your local registry.

After your work, you can clean up the CRDs with:
```shell
make crds.uninstall
```

### Commit your work and propose a PR

Make sure you have read the [Contribution Policy](policy.md) before submitting your PR.
It explains our expectations regarding commit messages, PR descriptions, LLM usage, and other important aspects of contributing.

This project uses the pull request process from GitHub. To submit a pull request,
fork the repository and push any changes to a branch on "your fork".
From there, propose a pull request in the external-secrets repository, targeting our main branch.

You do not have to worry about /ok-to-test, maintainers will execute it when necessary.

> Unless you add a new provider or change e2e test, you do not need to run e2e test for your contribution.
> The following section will help you run those tests.

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

#### Triggering e2e tests from github


We have an extensive set of e2e tests that test the integration with *real* cloud provider APIs.
Maintainers must trigger these kind of tests manually for PRs that come from forked repositories. These tests run inside a `kind` cluster in the GitHub Actions runner:

```
/ok-to-test sha=<full_commit_hash>
```
Examples:
```
/ok-to-test sha=b8ca0040200a7a05d57048d86a972fdf833b8c9b
```

### Contribute to "managed kubernetes" e2e tests

There's another suite of e2e tests that integrate with managed Kubernetes offerings.
They create real infrastructure at a cloud provider and deploy the controller
into that environment.

This is necessary to test the authentication integration
([GCP Workload Identity](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity),
[EKS IRSA](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html)...).

TODO(evrardjp): Please do not hesitate to complete this section! (moolen is our expert)

#### Executing Managed Kubernetes e2e tests locally

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

#### Triggering Managed Kubernetes e2e tests from github

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


## Setting up the development environment for helm chart contributions

Our helm chart is present in `deploy/charts/external-secrets`
(See also HELM_DIR in our Makefile).

When making changes to the helm chart, you will need to regenerate the chart `Chart.yaml` and its values `values.yaml` files.
It will be done **automatically** when you run the helm tests.

To test the helm chart locally, you must run the following commands:

```shell
make helm.test
make helm.test.update
```

These will run `helm-unittest`.

### Apply the helm chart on your cluster for last mile verification

To apply the helm chart on your cluster, run the following commands (adapt to your needs):

```shell
kind create cluster --name external-secrets # If you are using kind (named external-secrets) and need to create a cluster
export TAG=$(make docker.tag) #If you are rebuilding an image
export IMAGE=$(make docker.imagename) #If you are rebuilding an image
make docker.build #If you are rebuilding an image
kind load docker-image $IMAGE:$TAG --name external-secrets # If you are using kind (named external-secrets) and need to push an image
make helm.generate # If you did not run the helm.test command recently
helm upgrade --install external-secrets ./deploy/charts/external-secrets/ \
--set image.repository=$IMAGE --set image.tag=$TAG \
--set webhook.image.repository=$IMAGE --set webhook.image.tag=$TAG \
--set certController.image.repository=$IMAGE --set certController.image.tag=$TAG
# kind delete cluster -n external-secrets # To delete the kind cluster when done
```

## Setting up the development environment for infrastructure maintenance contributions

TODO(evrardjp): Please do not hesitate to fix this section! (moolen is our expert)


## Setting up the development environment for documentation contributions

We use [mkdocs material](https://squidfunk.github.io/mkdocs-material/) and [mike](https://github.com/jimporter/mike) to generate this
documentation. See `/docs` for the source code and `/hack/api-docs` for the build process.

Run the following command to run a complete build. The rendered assets are available under `/site`.

```shell
make docs
```

Serve the documentation locally (http://localhost:8000) with:
```shell
make docs.serve
```

When finished writing/reviewing the docs, clean up your local docs branch changes with `git branch -D gh-pages`

## Releasing

The whole releasing contributions how-to has been split into the [Releasing](release_howto.md) page
