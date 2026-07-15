MAKEFLAGS   += --warn-undefined-variables
SHELL       := /usr/bin/env bash
.SHELLFLAGS := -euo pipefail -c

KIND_IMG       = "kindest/node:v1.33.4@sha256:25a6018e48dfcaee478f4a59af81157a437f15e6e140bf103f85a2e7cd0cbbf2"
DOCKER_BUILD_ARGS     ?=

export E2E_IMAGE_NAME ?= ghcr.io/external-secrets/external-secrets-e2e
export GINKGO_LABELS ?= !managed
export TEST_SUITES ?= provider generator flux argocd

export OCI_IMAGE_NAME = ghcr.io/external-secrets/external-secrets

# Directory where test.build writes the docker-saved image tarballs and where
# test.run loads them from. Lets CI hand images between a build job and a
# separate test job as artifacts instead of rebuilding on the test runner.
E2E_ARTIFACT_DIR ?= $(shell pwd)/image-artifacts

start-kind: ## Start kind cluster
	kind create cluster \
	  --name external-secrets \
	  --config kind.yaml \
	  --retain \
	  --image "$(KIND_IMG)"

stop-kind: ## Stop kind cluster
	kind delete cluster \
		--name external-secrets \

test: ## Build images and run the e2e suite on the current runner (local convenience)
	$(MAKE) test.build
	$(MAKE) test.run

test.build: e2e-image ## Build the e2e + controller images and export them as tarballs
	$(MAKE) -C ../ docker.build \
		IMAGE_NAME=$(OCI_IMAGE_NAME) \
		VERSION=$(VERSION) \
		ARCH=amd64 \
		DOCKER_BUILD_ARGS="${DOCKER_BUILD_ARGS} --build-arg TARGETARCH=amd64 --build-arg TARGETOS=linux"
	mkdir -p $(E2E_ARTIFACT_DIR)
	docker save -o $(E2E_ARTIFACT_DIR)/eso.tar $(OCI_IMAGE_NAME):$(VERSION)
	docker save -o $(E2E_ARTIFACT_DIR)/e2e.tar $(E2E_IMAGE_NAME):$(VERSION)

test.run: ## Load prebuilt image tarballs into kind and run the e2e suite
	kind load image-archive --name="external-secrets" $(E2E_ARTIFACT_DIR)/eso.tar
	kind load image-archive --name="external-secrets" $(E2E_ARTIFACT_DIR)/e2e.tar
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
