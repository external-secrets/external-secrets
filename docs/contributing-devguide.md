## Getting Started

You must have a working [Go environment](https://golang.org/doc/install) and
then clone the repo:

```shell
git clone https://github.com/external-secrets/external-secrets.git
cd external-secrets
```

_Note: many of the `make` commands use [yq](https://github.com/mikefarah/yq), version 4.2X.X or higher._

If you want to run controller tests you also need to install kubebuilder's `envtest`.

The recommended way to do so is to install [setup-envtest](https://pkg.go.dev/sigs.k8s.io/controller-runtime/tools/setup-envtest)

Here is an example on how to set it up:

```
go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

# list available versions
setup-envtest list --os $(go env GOOS) --arch $(go env GOARCH)

# To use a specific version
setup-envtest use -p path 1.20.2

#To set environment variables
source <(setup-envtest use 1.20.2 -p env --os $(go env GOOS) --arch $(go env GOARCH))

```

for more information, please see [setup-envtest docs](https://github.com/kubernetes-sigs/controller-runtime/tree/master/tools/setup-envtest)

## Building & Testing

The project uses the `make` build system. It'll run code generators, tests and
static code analysis.

Building the operator binary and docker image:

```shell
make build
make docker.build IMG=external-secrets:latest
```

Run tests and lint the code:
```shell
make test
make lint
```

Build the documentation:
```shell
make docs
```

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

If you need to test some other k8s integrations and need the operator to be deployed to the actuall cluster while developing, you can use the following workflow:

```
kind create cluster --name external-secrets

export TAG=v2
export IMAGE=eso-local

#For building in linux
docker build . -t $IMAGE:$TAG --build-arg TARGETARCH=amd64 --build-arg TARGETOS=linux

#For building in MacOS (OSX)
#docker build . -t $IMAGE:$TAG --build-arg TARGETARCH=amd64 --build-arg TARGETOS=darwin

#For building in ARM
#docker build . -t $IMAGE:$TAG --build-arg TARGETARCH=arm --build-arg TARGETOS=linux

make helm.generate
helm upgrade --install external-secrets ./deploy/charts/external-secrets/ --set image.repository=$IMAGE --set image.tag=$TAG
```

!!! note "Contributing Flow"
    The HOW TO guide for contributing is at the [Contributing Process](contributing-process.md) page.


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
