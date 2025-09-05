## Getting Started

You must have a working [Go environment](https://golang.org/doc/install) and
then clone the repo:

```shell
git clone https://github.com/external-secrets/external-secrets.git
cd external-secrets
```

_Note: many of the `make` commands use [yq](https://github.com/mikefarah/yq), version 4.2X.X or higher._

Our helm chart is tested using `helm-unittest`. You will need it to run tests locally if you modify the helm chart. Install it with the following command:

```
$ helm plugin install https://github.com/helm-unittest/helm-unittest
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
```shell
make test
make lint # OR
docker run --rm -v $(pwd):/app -w /app golangci/golangci-lint:v1.49.0 golangci-lint run
```

Build the documentation:
```shell
make docs
```

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
helm upgrade --install external-secrets ./deploy/charts/external-secrets/ \
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
