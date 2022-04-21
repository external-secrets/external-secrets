MAKEFLAGS   += --warn-undefined-variables
SHELL       := /bin/bash
.SHELLFLAGS := -euo pipefail -c

KIND_IMG       = "kindest/node:v1.23.3@sha256:0df8215895129c0d3221cda19847d1296c4f29ec93487339149333bd9d899e5a"
BUILD_ARGS     ?=

export E2E_IMAGE_REGISTRY ?= ghcr.io/external-secrets/external-secrets-e2e
export GINKGO_LABELS ?= !managed
export TEST_SUITES ?= provider argocd

start-kind: ## Start kind cluster
	kind create cluster \
	  --name external-secrets \
	  --config kind.yaml \
	  --retain \
	  --image "$(KIND_IMG)"

test: e2e-image ## Run e2e tests against current kube context
	$(MAKE) -C ../ docker.build \
		IMAGE_REGISTRY=$(IMAGE_REGISTRY) \
		VERSION=$(VERSION) \
		ARCH=amd64 \
		BUILD_ARGS="${BUILD_ARGS} --build-arg TARGETARCH=amd64 --build-arg TARGETOS=linux"
	kind load docker-image --name="external-secrets" $(IMAGE_REGISTRY):$(VERSION)
	kind load docker-image --name="external-secrets" $(E2E_IMAGE_REGISTRY):$(VERSION)
	./run.sh

test.managed: e2e-image ## Run e2e tests against current kube context
	$(MAKE) -C ../ docker.build \
		VERSION=$(VERSION) \
		ARCH=amd64 \
		BUILD_ARGS="${BUILD_ARGS} --build-arg TARGETARCH=amd64 --build-arg TARGETOS=linux"
	$(MAKE) -C ../ docker.push \
		VERSION=$(VERSION)
	$(MAKE) -C ../ docker.push \
		IMAGE_REGISTRY=$(E2E_IMAGE_REGISTRY) \
		VERSION=$(VERSION)
	./run.sh

e2e-bin:
	CGO_ENABLED=0 go run github.com/onsi/ginkgo/v2/ginkgo build ./suites/...

e2e-image: e2e-bin
	-rm -rf ./k8s/deploy
	mkdir -p k8s
	$(MAKE) -C ../ helm.generate
	cp -r ../deploy ./k8s
	docker build $(BUILD_ARGS) -t $(E2E_IMAGE_REGISTRY):$(VERSION) .

stop-kind: ## Stop kind cluster
	kind delete cluster \
		--name external-secrets \

help: ## displays this help message
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_\/-]+:.*?## / {printf "\033[34m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST) | \
		sort | \
		grep -v '#'
