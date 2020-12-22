MAKEFLAGS     += --warn-undefined-variables
SHELL         := /bin/bash
.SHELLFLAGS   := -euo pipefail -c
.DEFAULT_GOAL := all

# Image URL to use all building/pushing image targets
IMG ?= controller:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"
HELM_DIR    ?= deploy/charts/external-secrets
CRD_DIR     ?= config/crd/bases

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: build

.PHONY: test
test: generate manifests ## Run tests
	go test ./... -coverprofile cover.out

.PHONY: build
build: generate fmt ## Build binary
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt manifests
	go run ./main.go

# Install CRDs into a cluster
install: manifests
	kustomize build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests
	kustomize build config/crd | kubectl delete -f -

.PHONY: deploy
deploy: manifests ## Deploy controller in the Kubernetes cluster of current context
	cd config/manager && kustomize edit set image controller=${IMG}
	kustomize build config/default | kubectl apply -f -

manifests: controller-gen ## Generate manifests e.g. CRD, RBAC etc.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

lint/check: # Check install of golanci-lint
	@if ! golangci-lint --version > /dev/null 2>&1; then \
		echo -e "\033[0;33mgolangci-lint is not installed: run \`\033[0;32mmake lint-install\033[0m\033[0;33m\` or install it from https://golangci-lint.run\033[0m"; \
		exit 1; \
	fi

lint-install: # installs golangci-lint to the go bin dir
	@if ! golangci-lint --version > /dev/null 2>&1; then \
		echo "Installing golangci-lint"; \
		curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $(GOBIN) v1.33.0; \
	fi

lint: lint/check ## run golangci-lint
	@if ! golangci-lint run; then \
		echo -e "\033[0;33mgolangci-lint failed: some checks can be fixed with \`\033[0;32mmake fmt\033[0m\033[0;33m\`\033[0m"; \
		exit 1; \
	fi

fmt: lint/check ## ensure consistent code style
	go mod tidy
	go fmt ./...
	golangci-lint run --fix > /dev/null 2>&1 || true

generate: controller-gen ## Generate code
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

docker-build: test ## Build the docker image
	docker build . -t ${IMG}

docker-push: ## Push the docker image
	docker push ${IMG}

helm-docs: ## Generate helm docs
	cd $(HELM_DIR); \
	docker run --rm -v $(shell pwd)/$(HELM_DIR):/helm-docs -u $(shell id -u) jnorwood/helm-docs:latest

crds-to-chart: # Copy crds to helm chart directory
	cp $(CRD_DIR)/*.yaml $(HELM_DIR)/templates/crds/; \
	for i in $(HELM_DIR)/templates/crds/*.yaml; do \
		sed -i '1s/.*/{{- if .Values.installCRDs }}/;$$a{{- end }}' $$i; \
	done

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.1 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

help: ## displays this help message
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_\/-]+:.*?## / {printf "\033[34m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST) | \
		sort | \
		grep -v '#'
