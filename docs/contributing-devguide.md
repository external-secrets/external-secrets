## Getting Started

You must have a working [Go environment](https://golang.org/doc/install) and
then clone the repo:

```shell
git clone https://github.com/external-secrets/external-secrets.git
cd external-secrets
```

If you want to run controller tests you also need to install kubebuilder's `envtest`:

```
export KUBEBUILDER_TOOLS_VERSION='1.20.2' # check for latest version or a version that has support to what you are testing

# Using Linux
curl -sSLo envtest-bins.tar.gz "https://storage.googleapis.com/kubebuilder-tools/kubebuilder-tools-$KUBEBUILDER_TOOLS_VERSION-linux-amd64.tar.gz"

# Using MacOS (OSX)
#curl -sSLo envtest-bins.tar.gz "https://storage.googleapis.com/kubebuilder-tools/kubebuilder-tools-$KUBEBUILDER_TOOLS_VERSION-darwin-amd64.tar.gz"

# Using ARM based processors
#curl -sSLo envtest-bins.tar.gz "https://storage.googleapis.com/kubebuilder-tools/kubebuilder-tools-$KUBEBUILDER_TOOLS_VERSION-linux-arm64.tar.gz"


sudo mkdir -p /usr/local/kubebuilder
sudo tar -C /usr/local/kubebuilder --strip-components=1 -zvxf envtest-bins.tar.gz
```

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

We use [mkdocs material](https://squidfunk.github.io/mkdocs-material/) to generate this
documentation. See `/docs` for the source code and `/hack/api-docs` for the build process.

When writing documentation it is advised to run the mkdocs server with livereload:

```shell
make serve-docs
```

Run the following command to run a complete build. The rendered assets are available under `/site`.

```shell
make docs
make serve-docs
```

Open `http://localhost:8000` in your browser.
