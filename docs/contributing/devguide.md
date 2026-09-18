## Getting Started

You must have a working [Go environment](https://golang.org/doc/install) and
then clone the repo:

```shell
git clone https://github.com/external-secrets/external-secrets.git
cd external-secrets
```

_Note: many of the `make` commands use [yq](https://github.com/mikefarah/yq), version 4.2X.X or higher._

Our helm chart is tested using `helm-unittest`. You will need it to run tests locally if you modify the helm chart.

```shell
make helm.test
make helm.test.update
```

## Building & Testing

The project uses the `make` build system. It'll run code generators, tests and
static code analysis.

Building the operator binary and docker image:

```shell
make build
make docker.build IMAGE_NAME=external-secrets IMAGE_TAG=latest
```

Run tests and lint the code:
```console
make test
make lint
```

Build the documentation:
```shell
make docs
```

## Updating dependencies

All is done with updatecli.
Run `make update-deps` to apply every Updatecli manifest in `.updatecli.d`.

Updatecli needs an authenticated GitHub token to avoid the anonymous API rate
limit. Export `GITHUB_TOKEN`, or authenticate the GitHub CLI with
`gh auth login`; the Make target uses either source.

### Current env vars in a nutshell

- `UPDATECLI_ACTION=diff` is preview mode (no actual changes on filesystem)
- `UPDATECLI_PUBLISH=true` creates a PR through update cli
  (change files, write a commit, pushes, creates a PR)
- `UPDATECLI_KIND=` restricts the automatic bump to an ecosystem subset
  (for example, only do docker container updates, or helm chart updates).

Those env vars are _cumulative_: They can be defined together.
Of course, publish true and preview mode does not make sense, so you should
be careful when using this.

### Preview mode

Preview available updates without changing files:

```shell
make update-deps UPDATECLI_ACTION=diff
```

This will run _all pipelines_ but will not store the changes on filesystem.

### Publish mode

By default, applying updates only changes the current checkout. To let Updatecli
commit, push, and create or update a pull request for all selected dependency
kind, opt in explicitly:

```shell
UPDATECLI_PUBLISH=true make update-deps
```

Publishing uses `GITHUB_REPOSITORY` when available and otherwise detects the
repository with `gh`. The token must have write access to repository contents,
pull requests, and workflows.

### Restrict by ecosystem

To limit an update to one dependency kind, use the env var `UPDATECLI_KIND`,
or use its convenience target:

```shell
make update-deps-gomodules
make update-deps-golang
make update-deps-github-actions
make update-deps-containers
make update-deps-tools
make update-deps-helm
make update-deps-python
make update-deps-terraform
```

Behind the scenes, these convenience targets are defining UPDATECLI_KIND.
See the Makefile for more details.

The current supported `UPDATECLI_KIND` values are `gomodules`, `golang`,
`github-actions`, `docker`, `tools`, `helm`, `python`, and `terraform`.

### Doing diff for a specific subsystem:

`UPDATECLI_ACTION=diff` works with either form, for example:

```shell
make update-deps-gomodules UPDATECLI_ACTION=diff
```

### containers management

The container pipeline keeps its image inventory and upstream tag policies in
`.updatecli.d/docker.yaml`. Go and kind/node track stable version tags and write
tag-only references. Other direct image references track `latest` in their
existing repositories and write digest-only pins (`image@sha256:...`); UBI 9
and distroless Debian 12 therefore stay on those distro lines.

We do this for two reasons:

- Explicit sources avoid repeated tag discovery and keep digest-only references
updatable (Dockerfile autodiscovery skips them). When adding a direct image
reference, add its file to an existing target or add a source and target here.
Preview with `make update-deps-containers UPDATECLI_ACTION=diff`.

- When defining both a tag and a digest, the tag is silently ignored by docker.
Sonarqube mentions this is a bad practice to keep both the tag and the digest,
as it leads to ppl mistakenly believe things are updated when updating the tag
without bumping the sha. While this is not our case (we bump at the same time),
we avoid the sonarqube alerts by removing the tags.

### golang toolkit and modules management

The Go dependency pipeline uses Updatecli's native `golang/module`,
`golang/gomod`, and `file` resources for version resolution and every bump. The
file target covers Go's `tool` directive, which the gomod target cannot write.
The standard-library Go generator in `hack/updatecli-gomodules` reads
the canonical modules declared by the root module plus the isolated e2e and
documentation-tool modules without invoking Git.

`make updatecli-go-manifests` renders the templates in
`hack/updatecli-gomodules/templates` as
`.updatecli.d/gomodules.yaml` and `.updatecli.d/golang.yaml`.

The Go modules manifest resolves a dependency shared by dozens of modules only
once, then native targets write that version to every module declaring it.
Ordinary indirect requirements are left to `go mod tidy`; modules backing a Go
`tool` directive remain explicit update targets. A final shell target only runs
`go mod tidy`; it does not select or bump versions. The separate generated
`golang.yaml` uses the same canonical module list so `make update-deps-golang`
updates the Go version in every known module.

This is a bit more efficient than our previous shell script, and use native
updatecli features.

### development tools

Development tool versions and platform checksums are pinned directly in the
`Tool Binaries` section of the Makefile.
`make` downloads and verifies their upstream release assets with
`curl`, `tar`, and either `sha256sum` or `shasum`, then installs them into
`bin/`. Set `LOCALBIN` env var to use another location. The installed binaries are the
Make outputs; no additional installation-state files are maintained.

#### envtest details

The `SETUP_ENVTEST_VERSION` variable pins the `setup-envtest` CLI. The
independent `ENVTEST_KUBERNETES_VERSION` variable selects the Kubernetes test
control-plane binaries downloaded by that CLI. The
`gen-crd-api-reference-docs` tool remains source-built from its isolated Go
module.

### Check your updates are fine

After applying updates, inspect and commit all changes, then run
`make updatecli-go-manifests` followed by the usual
`make test` and `make check-diff` to verify that generated files are current.

## License Headers

All Go source files must include the Apache License 2.0 header. The CI automatically checks license headers for new files added in pull requests using [Apache SkyWalking Eyes](https://github.com/apache/skywalking-eyes).

If you need to check license headers locally, you can use the SkyWalking Eyes tool directly. The configuration is in `.licenserc.yaml` at the project root.

## Using Tilt

[Tilt](https://tilt.dev) can be used to develop external-secrets. Tilt will hot-reload changes to the code and replace
the running binary in the container using a process manager of its own.

To run tilt, download the utility for your operating system and run `make tilt-up`. This will do two things:

- downloads tilt for the current OS and ARCH under `bin/tilt`
- make manifest files of your current changes and place them under `./bin/deploy/manifests/external-secrets.yaml`
- run tilt with `tilt run`

Hit `space` and you can observe all the pods starting up and track their output in the tilt UI.

## Installing

To install the External Secret Operator into a Kubernetes Cluster run:

```shell
helm repo add external-secrets https://charts.external-secrets.io
helm repo update
helm install external-secrets external-secrets/external-secrets
```

You can alternatively run the controller on your host system for development purposes:


```shell
make crds.install
make run
```

To remove the CRDs run:

```shell
make crds.uninstall
```

If you need to test some other k8s integrations and need the operator to be deployed to the actual cluster while developing, you can use the following workflow:

```shell
# Start a local K8S cluster with KinD
kind create cluster --name external-secrets

export TAG=$(make docker.tag)
export IMAGE=$(make docker.imagename)

# Build docker image
make docker.build

# Load docker image into local kind cluster
kind load docker-image $IMAGE:$TAG --name external-secrets

# (Optional) Pull the image from GitHub Repo to copy into kind
# docker pull ghcr.io/external-secrets/external-secrets:v0.8.2
# kind load docker-image ghcr.io/external-secrets/external-secrets:v0.8.2 -n external-secrets
# export TAG=v0.8.2

# Update helm charts and install to KinD cluster
make helm.generate
# $IMAGE already includes the registry host, so clear global.imageRegistry to
# stop the chart prefixing its own.
helm upgrade --install external-secrets ./deploy/charts/external-secrets/ \
--set global.imageRegistry= \
--set image.repository=$IMAGE --set image.tag=$TAG \
--set webhook.image.repository=$IMAGE --set webhook.image.tag=$TAG \
--set certController.image.repository=$IMAGE --set certController.image.tag=$TAG


# Command to delete the cluster when done
# kind delete cluster -n external-secrets
```

!!! note "Contributing Flow"
    The HOW TO guide for contributing is at the [Contributing Process](process.md) page.


## Documentation

We use [mkdocs material](https://squidfunk.github.io/mkdocs-material/) and [mike](https://github.com/jimporter/mike) to generate this
documentation. See `/docs` for the source code and `/hack/api-docs` for the build process.

When writing documentation it is advised to run the mkdocs server with livereload:

```shell
make docs.serve
```

Run the following command to run a complete build. The rendered assets are available under `/site`.

```shell
make docs
make docs.serve
```

Open `http://localhost:8000` in your browser.

Since mike uses a branch to create/update documentation, any docs operation will create a diff on your local `gh-pages` branch.

When finished writing/reviewing the docs, clean up your local docs branch changes with `git branch -D gh-pages`
