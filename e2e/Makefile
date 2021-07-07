MAKEFLAGS   += --warn-undefined-variables
SHELL       := /bin/bash
.SHELLFLAGS := -euo pipefail -c

IMG_TAG     = test
IMG         = local/external-secrets-e2e:$(IMG_TAG)
K8S_VERSION = "1.20.7"

start-kind: ## Start kind cluster
	kind create cluster \
	  --name external-secrets \
	  --config kind.yaml \
	  --retain \
	  --image "kindest/node:v$(K8S_VERSION)"

test: e2e-image ## Run e2e tests against current kube context
	$(MAKE) -C ../ docker.build \
		IMAGE_REGISTRY=local/external-secrets \
		VERSION=$(IMG_TAG) \
		BUILD_ARGS="--build-arg ARCHS=amd64"
	kind load docker-image --name="external-secrets" local/external-secrets:$(IMG_TAG)
	kind load docker-image --name="external-secrets" $(IMG)
	./run.sh

e2e-bin:
	CGO_ENABLED=0 ginkgo build .

e2e-image: e2e-bin
	-rm -rf ./k8s/deploy
	mkdir -p k8s
	$(MAKE) -C ../ helm.generate
	cp -r ../deploy ./k8s
	docker build -t $(IMG) .

stop-kind: ## Stop kind cluster
	kind delete cluster \
		--name external-secrets \

help: ## displays this help message
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_\/-]+:.*?## / {printf "\033[34m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST) | \
		sort | \
		grep -v '#'
