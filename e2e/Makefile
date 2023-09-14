MAKEFLAGS   += --warn-undefined-variables
SHELL       := /bin/bash
.SHELLFLAGS := -euo pipefail -c

KIND_IMG       = "kindest/node:v1.24.2@sha256:1f0cee2282f43150b52dc7933183ed96abdcfc8d293f30ec07082495874876f1"
DOCKER_BUILD_ARGS     ?=

export E2E_IMAGE_NAME ?= ghcr.io/external-secrets/external-secrets-e2e
export GINKGO_LABELS ?= !managed
export TEST_SUITES ?= provider generator flux argocd

start-kind: ## Start kind cluster
	kind create cluster \
	  --name external-secrets \
	  --config kind.yaml \
	  --retain \
	  --image "$(KIND_IMG)"

test: e2e-image ## Run e2e tests against current kube context
	$(MAKE) -C ../ docker.build \
		IMAGE_NAME=$(IMAGE_NAME) \
		VERSION=$(VERSION) \
		ARCH=amd64 \
		DOCKER_BUILD_ARGS="${DOCKER_BUILD_ARGS} --build-arg TARGETARCH=amd64 --build-arg TARGETOS=linux"
	kind load docker-image --name="external-secrets" $(IMAGE_NAME):$(VERSION)
	kind load docker-image --name="external-secrets" $(E2E_IMAGE_NAME):$(VERSION)
	./run.sh

test.managed: e2e-image ## Run e2e tests against current kube context
	$(MAKE) -C ../ docker.build \
		VERSION=$(VERSION) \
		ARCH=amd64 \
		DOCKER_BUILD_ARGS="${DOCKER_BUILD_ARGS} --build-arg TARGETARCH=amd64 --build-arg TARGETOS=linux"
	$(MAKE) -C ../ docker.push \
		VERSION=$(VERSION)
	$(MAKE) -C ../ docker.push \
		IMAGE_NAME=$(E2E_IMAGE_NAME) \
		VERSION=$(VERSION)
	./run.sh

e2e-bin:
	CGO_ENABLED=0 go run github.com/onsi/ginkgo/v2/ginkgo build ./suites/...

e2e-image:
	-rm -rf ./k8s/deploy
	mkdir -p k8s
	$(MAKE) -C ../ helm.generate
	cp -r ../deploy ./k8s
	docker build $(DOCKER_BUILD_ARGS) -t $(E2E_IMAGE_NAME):$(VERSION) -f Dockerfile ..

stop-kind: ## Stop kind cluster
	kind delete cluster \
		--name external-secrets \

help: ## displays this help message
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_\/-]+:.*?## / {printf "\033[34m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST) | \
		sort | \
		grep -v '#'
