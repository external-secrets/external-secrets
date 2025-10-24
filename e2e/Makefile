MAKEFLAGS   += --warn-undefined-variables
SHELL       := /bin/bash
.SHELLFLAGS := -euo pipefail -c

KIND_IMG       = "kindest/node:v1.33.4@sha256:25a6018e48dfcaee478f4a59af81157a437f15e6e140bf103f85a2e7cd0cbbf2"
DOCKER_BUILD_ARGS     ?=

export E2E_IMAGE_NAME ?= ghcr.io/external-secrets/external-secrets-e2e
export GINKGO_LABELS ?= !managed
export TEST_SUITES ?= provider generator flux argocd

export OCI_IMAGE_NAME = oci.external-secrets.io/external-secrets/external-secrets

start-kind: ## Start kind cluster
	kind create cluster \
	  --name external-secrets \
	  --config kind.yaml \
	  --retain \
	  --image "$(KIND_IMG)"

stop-kind: ## Stop kind cluster
	kind delete cluster \
		--name external-secrets \

test: e2e-image ## Run e2e tests against current kube context
	$(MAKE) -C ../ docker.build \
		IMAGE_NAME=$(IMAGE_NAME) \
		VERSION=$(VERSION) \
		ARCH=amd64 \
		DOCKER_BUILD_ARGS="${DOCKER_BUILD_ARGS} --build-arg TARGETARCH=amd64 --build-arg TARGETOS=linux"
	$(MAKE) -C ../ docker.build \
		IMAGE_NAME=$(OCI_IMAGE_NAME) \
		VERSION=$(VERSION) \
		ARCH=amd64 \
		DOCKER_BUILD_ARGS="${DOCKER_BUILD_ARGS} --build-arg TARGETARCH=amd64 --build-arg TARGETOS=linux"
	kind load docker-image --name="external-secrets" $(IMAGE_NAME):$(VERSION)
	kind load docker-image --name="external-secrets" $(OCI_IMAGE_NAME):$(VERSION)
	kind load docker-image --name="external-secrets" $(E2E_IMAGE_NAME):$(VERSION)
	./run.sh

test.managed: e2e-image ## Run e2e tests against current kube context
	$(MAKE) -C ../ docker.build \
		VERSION=$(VERSION) \
		ARCH=amd64 \
		DOCKER_BUILD_ARGS="${DOCKER_BUILD_ARGS} --build-arg TARGETARCH=amd64 --build-arg TARGETOS=linux"
	$(MAKE) -C ../ docker.build \
		IMAGE_NAME=$(OCI_IMAGE_NAME) \
		VERSION=$(VERSION) \
		ARCH=amd64 \
		DOCKER_BUILD_ARGS="${DOCKER_BUILD_ARGS} --build-arg TARGETARCH=amd64 --build-arg TARGETOS=linux"
	$(MAKE) -C ../ docker.push \
		VERSION=$(VERSION)
	$(MAKE) -C ../ docker.push \
		IMAGE_NAME=$(OCI_IMAGE_NAME) \
		VERSION=$(VERSION)
	$(MAKE) -C ../ docker.push \
		IMAGE_NAME=$(E2E_IMAGE_NAME) \
		VERSION=$(VERSION)
	./run.sh


e2e-bin: install-ginkgo
	   CGO_ENABLED=0 ginkgo build ./suites/...

e2e-image: e2e-bin
	-rm -rf ./k8s/deploy
	mkdir -p k8s
	$(MAKE) -C ../ helm.generate
	cp -r ../deploy ./k8s
	docker build $(DOCKER_BUILD_ARGS) -t $(E2E_IMAGE_NAME):$(VERSION) -f Dockerfile ..

GINKGO_VERSION := $(shell grep 'github.com/onsi/ginkgo/v2' go.mod | awk '{print $$2}')
install-ginkgo:
	   @echo "Installing ginkgo version $(GINKGO_VERSION) from go.mod"
	   go install github.com/onsi/ginkgo/v2/ginkgo@$(GINKGO_VERSION)

help: ## displays this help message
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_\/-]+:.*?## / {printf "\033[34m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST) | \
		sort | \
		grep -v '#'
