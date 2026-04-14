MAKEFLAGS   += --warn-undefined-variables
SHELL       := /usr/bin/env bash
.SHELLFLAGS := -euo pipefail -c

KIND_IMG ?= kindest/node:v1.33.4@sha256:25a6018e48dfcaee478f4a59af81157a437f15e6e140bf103f85a2e7cd0cbbf2
KIND_CLUSTER_NAME ?= external-secrets
KIND_CONTEXT ?= kind-$(KIND_CLUSTER_NAME)
DOCKER_BUILD_ARGS     ?=
SKIP_PROVIDER_KUBERNETES_BUILD ?= false

export E2E_IMAGE_NAME ?= ghcr.io/external-secrets/external-secrets-e2e
export GINKGO_LABELS ?= !managed && !v2
export V2_GINKGO_LABELS ?= v2
export TEST_SUITES ?= provider generator flux argocd
export GOCACHE ?= $(CURDIR)/.cache/go-build
export GOMODCACHE ?= $(CURDIR)/.cache/go-mod

export OCI_IMAGE_NAME = ghcr.io/external-secrets/external-secrets
export IMAGE_NAME ?= $(OCI_IMAGE_NAME)
TEST_V2_PROVIDER_KUBERNETES_BUILD_CMD = $(MAKE) -C ../ docker.build.provider.kubernetes VERSION=$(VERSION) ARCH=amd64 DOCKER_BUILD_ARGS="${DOCKER_BUILD_ARGS} --build-arg TARGETARCH=amd64 --build-arg TARGETOS=linux"
TEST_V2_PROVIDER_AWS_BUILD_CMD = $(MAKE) -C ../ docker.build.provider.aws VERSION=$(VERSION) ARCH=amd64 DOCKER_BUILD_ARGS="${DOCKER_BUILD_ARGS} --build-arg TARGETARCH=amd64 --build-arg TARGETOS=linux"
TEST_V2_PROVIDER_FAKE_BUILD_CMD = $(MAKE) -C ../ docker.build.provider.fake VERSION=$(VERSION) ARCH=amd64 DOCKER_BUILD_ARGS="${DOCKER_BUILD_ARGS} --build-arg TARGETARCH=amd64 --build-arg TARGETOS=linux"

ifeq ($(shell git tag),)
export VERSION ?= $(shell echo "v0.0.0-$$(git rev-list HEAD --count)-g$$(git describe --dirty --always)" | sed 's/-/./2' | sed 's/-/./2')
else
export VERSION ?= $(shell git describe --dirty --always --tags --exclude 'helm*' | sed 's/-/./2' | sed 's/-/./2')
endif

start-kind: ## Start kind cluster
	@if kind get clusters | grep -qx "$(KIND_CLUSTER_NAME)"; then \
		kind export kubeconfig --name "$(KIND_CLUSTER_NAME)"; \
		if kubectl --context "$(KIND_CONTEXT)" --request-timeout=5s get --raw=/readyz >/dev/null 2>&1; then \
			echo "kind cluster $(KIND_CLUSTER_NAME) is ready"; \
		else \
			echo "kind cluster $(KIND_CLUSTER_NAME) exists but is unhealthy, recreating"; \
			kind delete cluster --name "$(KIND_CLUSTER_NAME)"; \
			kind create cluster \
			  --name "$(KIND_CLUSTER_NAME)" \
			  --config kind.yaml \
			  --retain \
			  --image "$(KIND_IMG)"; \
		fi; \
	else \
		kind create cluster \
		  --name "$(KIND_CLUSTER_NAME)" \
		  --config kind.yaml \
		  --retain \
		  --image "$(KIND_IMG)"; \
	fi
	kind export kubeconfig --name "$(KIND_CLUSTER_NAME)"

stop-kind: ## Stop kind cluster
	kind delete cluster \
		--name "$(KIND_CLUSTER_NAME)" \

test: start-kind e2e-image ## Run e2e tests against current kube context
	$(MAKE) -C ../ docker.build.controller.e2e \
		IMAGE_NAME=$(OCI_IMAGE_NAME) \
		VERSION=$(VERSION) \
		ARCH=amd64 \
		DOCKER_BUILD_ARGS="${DOCKER_BUILD_ARGS} --build-arg TARGETARCH=amd64 --build-arg TARGETOS=linux"
	kind load docker-image --name="$(KIND_CLUSTER_NAME)" $(OCI_IMAGE_NAME):$(VERSION)
	kind load docker-image --name="$(KIND_CLUSTER_NAME)" $(E2E_IMAGE_NAME):$(VERSION)
	KUBECTL_CONTEXT="$(KIND_CONTEXT)" E2E_SKIP_HELM_DEPENDENCY_UPDATE="true" ./run.sh

test.v2: start-kind e2e-image ## Run v2 e2e tests against current kube context
	$(MAKE) -C ../ docker.build.controller.e2e \
		IMAGE_NAME=$(IMAGE_NAME) \
		VERSION=$(VERSION) \
		ARCH=amd64 \
		DOCKER_BUILD_ARGS="${DOCKER_BUILD_ARGS} --build-arg TARGETARCH=amd64 --build-arg TARGETOS=linux"
	$(if $(filter true,$(SKIP_PROVIDER_KUBERNETES_BUILD)),,$(TEST_V2_PROVIDER_KUBERNETES_BUILD_CMD))
	$(TEST_V2_PROVIDER_AWS_BUILD_CMD)
	$(TEST_V2_PROVIDER_FAKE_BUILD_CMD)
	$(MAKE) -C ../ docker.build.controller.e2e \
		IMAGE_NAME=$(OCI_IMAGE_NAME) \
		VERSION=$(VERSION) \
		ARCH=amd64 \
		DOCKER_BUILD_ARGS="${DOCKER_BUILD_ARGS} --build-arg TARGETARCH=amd64 --build-arg TARGETOS=linux"
	kind load docker-image --name="$(KIND_CLUSTER_NAME)" $(IMAGE_NAME):$(VERSION)
	kind load docker-image --name="$(KIND_CLUSTER_NAME)" $(OCI_IMAGE_NAME):$(VERSION)
	kind load docker-image --name="$(KIND_CLUSTER_NAME)" $(E2E_IMAGE_NAME):$(VERSION)
	kind load docker-image --name="$(KIND_CLUSTER_NAME)" ghcr.io/external-secrets/provider-kubernetes:$(VERSION)
	kind load docker-image --name="$(KIND_CLUSTER_NAME)" ghcr.io/external-secrets/provider-aws:$(VERSION)
	kind load docker-image --name="$(KIND_CLUSTER_NAME)" ghcr.io/external-secrets/provider-fake:$(VERSION)
	KUBECTL_CONTEXT="$(KIND_CONTEXT)" GINKGO_LABELS="$(V2_GINKGO_LABELS)" E2E_PROVIDER_MODE="v2" TEST_SUITES="provider" E2E_SKIP_HELM_DEPENDENCY_UPDATE="true" ./run.sh

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
	E2E_SKIP_HELM_DEPENDENCY_UPDATE="true" ./run.sh


e2e-bin: install-ginkgo
	   CGO_ENABLED=0 ginkgo build ./suites/...

e2e-image: e2e-bin
	-rm -rf ./k8s/deploy
	mkdir -p k8s
	$(MAKE) -C ../ helm.generate
	../hack/helm.dependency.ensure.sh ../deploy/charts/external-secrets
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
