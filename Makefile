# set the shell to bash always
SHELL         := /bin/bash

# set make and shell flags to exit on errors
MAKEFLAGS     += --warn-undefined-variables
.SHELLFLAGS   := -euo pipefail -c

ARCH = amd64 arm64
BUILD_ARGS ?=

# default target is build
.DEFAULT_GOAL := all
.PHONY: all
all: $(addprefix build-,$(ARCH))

# Image registry for build/push image targets
export IMAGE_REGISTRY ?= ghcr.io/external-secrets/external-secrets

#Valid licenses for license.check
LICENSES ?= Apache-2.0|MIT|BSD-3-Clause|ISC|MPL-2.0|BSD-2-Clause|Unknown
BUNDLE_DIR     ?= deploy/crds
CRD_DIR     ?= config/crds

HELM_DIR    ?= deploy/charts/external-secrets
TF_DIR ?= terraform

OUTPUT_DIR  ?= bin

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# check if there are any existing `git tag` values
ifeq ($(shell git tag),)
# no tags found - default to initial tag `v0.0.0`
export VERSION := $(shell echo "v0.0.0-$$(git rev-list HEAD --count)-g$$(git describe --dirty --always)" | sed 's/-/./2' | sed 's/-/./2')
else
# use tags
export VERSION := $(shell git describe --dirty --always --tags --exclude 'helm*' | sed 's/-/./2' | sed 's/-/./2')
endif

# ====================================================================================
# Colors

BLUE         := $(shell printf "\033[34m")
YELLOW       := $(shell printf "\033[33m")
RED          := $(shell printf "\033[31m")
GREEN        := $(shell printf "\033[32m")
CNone        := $(shell printf "\033[0m")

# ====================================================================================
# Logger

TIME_LONG	= `date +%Y-%m-%d' '%H:%M:%S`
TIME_SHORT	= `date +%H:%M:%S`
TIME		= $(TIME_SHORT)

INFO	= echo ${TIME} ${BLUE}[ .. ]${CNone}
WARN	= echo ${TIME} ${YELLOW}[WARN]${CNone}
ERR		= echo ${TIME} ${RED}[FAIL]${CNone}
OK		= echo ${TIME} ${GREEN}[ OK ]${CNone}
FAIL	= (echo ${TIME} ${RED}[FAIL]${CNone} && false)

# ====================================================================================
# Conformance

reviewable: generate helm.generate lint ## Ensure a PR is ready for review.
	@go mod tidy

golicenses.check: ## Check install of go-licenses
	@if ! go-licenses >> /dev/null 2>&1; then \
		echo -e "\033[0;33mgo-licenses is not installed: run go install github.com/google/go-licenses@latest" ; \
		exit 1; \
	fi

license.check: golicenses.check
	@$(INFO) running dependency license checks
	@ok=0; go-licenses csv github.com/external-secrets/external-secrets 2>/dev/null | \
	 grep -v -E '${LICENSES}' | \
	 tr "," " " | awk '{print "Invalid License " $$3 " for dependency " $$1 }'|| ok=1; \
	 if [[ $$ok -eq 1 ]]; then $(OK) dependencies are compliant; else $(FAIL); fi
	 
check-diff: reviewable ## Ensure branch is clean.
	@$(INFO) checking that branch is clean
	@test -z "$$(git status --porcelain)" || (echo "$$(git status --porcelain)" && $(FAIL))
	@$(OK) branch is clean

# ====================================================================================
# Golang

.PHONY: test
test: generate ## Run tests
	@$(INFO) go test unit-tests
	go test -race -v $(shell go list ./... | grep -v e2e) -coverprofile cover.out
	@$(OK) go test unit-tests

.PHONY: test.e2e
test.e2e: generate ## Run e2e tests
	@$(INFO) go test e2e-tests
	$(MAKE) -C ./e2e test
	@$(OK) go test e2e-tests

.PHONY: test.e2e.managed
test.e2e.managed: generate ## Run e2e tests managed
	@$(INFO) go test e2e-tests-managed
	$(MAKE) -C ./e2e test.managed
	@$(OK) go test e2e-tests-managed

.PHONY: build
build: $(addprefix build-,$(ARCH)) ## Build binary

.PHONY: build-%
build-%: generate ## Build binary for the specified arch
	@$(INFO) go build $*
	@CGO_ENABLED=0 GOOS=linux GOARCH=$* \
		go build -o '$(OUTPUT_DIR)/external-secrets-linux-$*' main.go
	@$(OK) go build $*

lint.check: ## Check install of golanci-lint
	@if ! golangci-lint --version > /dev/null 2>&1; then \
		echo -e "\033[0;33mgolangci-lint is not installed: run \`\033[0;32mmake lint.install\033[0m\033[0;33m\` or install it from https://golangci-lint.run\033[0m"; \
		exit 1; \
	fi

lint.install: ## Install golangci-lint to the go bin dir
	@if ! golangci-lint --version > /dev/null 2>&1; then \
		echo "Installing golangci-lint"; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOBIN) v1.42.1; \
	fi

lint: lint.check ## Run golangci-lint
	@if ! golangci-lint run; then \
		echo -e "\033[0;33mgolangci-lint failed: some checks can be fixed with \`\033[0;32mmake fmt\033[0m\033[0;33m\`\033[0m"; \
		exit 1; \
	fi
	@$(OK) Finished linting

fmt: lint.check ## Ensure consistent code style
	@go mod tidy
	@go fmt ./...
	@golangci-lint run --fix > /dev/null 2>&1 || true
	@$(OK) Ensured consistent code style

generate: ## Generate code and crds
	@go run sigs.k8s.io/controller-tools/cmd/controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."
	@go run sigs.k8s.io/controller-tools/cmd/controller-gen crd paths="./..." output:crd:artifacts:config=$(CRD_DIR)/bases
# Remove extra header lines in generated CRDs
	@for i in $(CRD_DIR)/bases/*.yaml; do \
  		tail -n +2 <"$$i" >"$$i.bkp" && \
  		cp "$$i.bkp" "$$i" && \
  		rm "$$i.bkp"; \
  	done
	@yq e '.spec.conversion.strategy = "Webhook" | .spec.conversion.webhook.conversionReviewVersions = ["v1"] | .spec.conversion.webhook.clientConfig.service.name = "kubernetes" | .spec.conversion.webhook.clientConfig.service.namespace = "default" |	.spec.conversion.webhook.clientConfig.service.path = "/convert"' $(CRD_DIR)/bases/*  > $(BUNDLE_DIR)/bundle.yaml
	@$(OK) Finished generating deepcopy and crds

# ====================================================================================
# Local Utility

# This is for running out-of-cluster locally, and is for convenience.
# For more control, try running the binary directly with different arguments.
run: generate ## Run app locally (without a k8s cluster)
	go run ./main.go

manifests: helm.generate ## Generate manifests from helm chart
	mkdir -p $(OUTPUT_DIR)/deploy/manifests
	helm template external-secrets $(HELM_DIR) -f deploy/manifests/helm-values.yaml > $(OUTPUT_DIR)/deploy/manifests/external-secrets.yaml

crds.install: generate ## Install CRDs into a cluster. This is for convenience
	kubectl apply -f $(BUNDLE_DIR)

crds.uninstall: ## Uninstall CRDs from a cluster. This is for convenience
	kubectl delete -f $(BUNDLE_DIR)

# ====================================================================================
# Helm Chart

helm.docs: ## Generate helm docs
	@cd $(HELM_DIR); \
	docker run --rm -v $(shell pwd)/$(HELM_DIR):/helm-docs -u $(shell id -u) jnorwood/helm-docs:v1.5.0

HELM_VERSION ?= $(shell helm show chart $(HELM_DIR) | grep 'version:' | sed 's/version: //g')

helm.build: helm.generate ## Build helm chart
	@$(INFO) helm package
	@helm package $(HELM_DIR) --dependency-update --destination $(OUTPUT_DIR)/chart
	@mv $(OUTPUT_DIR)/chart/external-secrets-$(HELM_VERSION).tgz $(OUTPUT_DIR)/chart/external-secrets.tgz
	@$(OK) helm package

helm.generate:
# Split the generated bundle yaml file to inject control flags
	@for i in $(BUNDLE_DIR)/*.yaml; do \
		yq e -Ns '"$(HELM_DIR)/templates/crds/" + .spec.names.singular' "$$i"; \
	done
# Add helm if statement for controlling the install of CRDs
	@for i in $(HELM_DIR)/templates/crds/*.yml; do \
		export CRDS_FLAG_NAME="create$$(yq e '.spec.names.kind' $$i)"; \
		cp "$$i" "$$i.bkp"; \
		if [[ "$$CRDS_FLAG_NAME" == *"Cluster"* ]]; then \
			echo "{{- if and (.Values.installCRDs) (.Values.crds.$$CRDS_FLAG_NAME) }}" > "$$i"; \
		else \
			echo "{{- if .Values.installCRDs }}" > "$$i"; \
		fi; \
		cat "$$i.bkp" >> "$$i" && \
		echo "{{- end }}" >> "$$i" && \
		rm "$$i.bkp" && \
		sed -i 's/name: kubernetes/name: {{ include "external-secrets.fullname" . }}-webhook/g' "$$i" && \
		sed -i 's/namespace: default/namespace: {{ .Release.Namespace | quote }}/g' "$$i" && \
		mv "$$i" "$${i%.yml}.yaml"; \
	done
	@$(OK) Finished generating helm chart files

# ====================================================================================
# Documentation
.PHONY: docs
docs: generate ## Generate docs
	$(MAKE) -C ./hack/api-docs build

.PHONY: docs.publish
docs.publish: generate ## Generate and deploys docs
	$(MAKE) -C ./hack/api-docs build.publish

.PHONY: docs.serve
docs.serve: ## Serve docs
	$(MAKE) -C ./hack/api-docs serve

# ====================================================================================
# Build Artifacts

build.all: docker.build helm.build ## Build all artifacts (docker image, helm chart)

docker.build: $(addprefix build-,$(ARCH)) ## Build the docker image
	@$(INFO) docker build
	@docker build . $(BUILD_ARGS) -t $(IMAGE_REGISTRY):$(VERSION)
	@$(OK) docker build

docker.push: ## Push the docker image to the registry
	@$(INFO) docker push
	@docker push $(IMAGE_REGISTRY):$(VERSION)
	@$(OK) docker push

# RELEASE_TAG is tag to promote. Default is promoting to main branch, but can be overriden
# to promote a tag to a specific version.
RELEASE_TAG ?= main
SOURCE_TAG ?= $(VERSION)

docker.promote: ## Promote the docker image to the registry
	@$(INFO) promoting $(SOURCE_TAG) to $(RELEASE_TAG)
	docker manifest inspect $(IMAGE_REGISTRY):$(SOURCE_TAG) > .tagmanifest
	for digest in $$(jq -r '.manifests[].digest' < .tagmanifest); do \
		docker pull $(IMAGE_REGISTRY)@$$digest; \
	done
	docker manifest create $(IMAGE_REGISTRY):$(RELEASE_TAG) \
		$$(jq -j '"--amend $(IMAGE_REGISTRY)@" + .manifests[].digest + " "' < .tagmanifest)
	docker manifest push $(IMAGE_REGISTRY):$(RELEASE_TAG)
	@$(OK) docker push $(RELEASE_TAG) \

docker.sign: ## Sign
	@$(INFO) signing $(IMAGE_REGISTRY):$(RELEASE_TAG)
	crane digest $(IMAGE_REGISTRY):$(RELEASE_TAG) > .digest
	cosign sign $(IMAGE_REGISTRY)@$$(cat .digest)
	@$(OK) cosign sign $(IMAGE_REGISTRY):$(RELEASE_TAG)

# ====================================================================================
# Terraform

tf.plan.%: ## Runs terrform plan for a provider
	@cd $(TF_DIR)/$*; \
	terraform init; \
	terraform plan

tf.apply.%: ## Runs terrform apply for a provider
	@cd $(TF_DIR)/$*; \
	terraform init; \
	terraform apply -auto-approve

tf.destroy.%: ## Runs terrform destroy for a provider
	@cd $(TF_DIR)/$*; \
	terraform init; \
	terraform destroy -auto-approve

tf.show.%: ## Runs terrform show for a provider and outputs to a file
	@cd $(TF_DIR)/$*; \
	terraform init; \
	terraform plan -out tfplan.binary; \
	terraform show -json tfplan.binary > plan.json

# ====================================================================================
# Help

# only comments after make target name are shown as help text
help: ## Displays this help message
	@echo -e "$$(grep -hE '^\S+:.*##' $(MAKEFILE_LIST) | sed -e 's/:.*##\s*/:/' -e 's/^\(.\+\):\(.*\)/\\x1b[36m\1\\x1b[m:\2/' | column -c2 -t -s : | sort)"
