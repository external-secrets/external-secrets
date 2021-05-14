## Getting Started

You must have a working [Go environment](https://golang.org/doc/install) and
then clone the repo:

```shell
git clone https://github.com/external-secrets/external-secrets.git
cd external-secrets
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

To install the External Secret Operator's CRDs into a Kubernetes Cluster run:

```shell
make crds.install
```

Apply the sample resources:
```shell
kubectl apply -f docs/snippets/basic-secret-store.yaml
kubectl apply -f docs/snippets/basic-external-secret.yaml
```

You can run the controller on your host system for development purposes:

```shell
make run
```

To remove the CRDs run:

```shell
make crds.uninstall
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

# Python 3
# inspect the build with this one-liner
python -m http.server 8000 --directory site

# Python 2
cd site
python -m httpSimpleServer 8000
```

Open `http://localhost:8000` in your browser.
