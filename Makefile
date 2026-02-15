# set the shell to bash always
SHELL         := /usr/bin/env bash

# set make and shell flags to exit on errors
MAKEFLAGS     += --warn-undefined-variables
.SHELLFLAGS   := -euo pipefail -c

ARCH ?= amd64 arm64 ppc64le
BUILD_ARGS ?= CGO_ENABLED=0
DOCKER_BUILD_ARGS ?=
DOCKERFILE ?= Dockerfile
DOCKER ?= docker
# default target is build
.DEFAULT_GOAL := all
.PHONY: all
all: $(addprefix build-,$(ARCH))

# Image registry for build/push image targets
export IMAGE_REGISTRY ?= ghcr.io
export IMAGE_REPO     ?= external-secrets/external-secrets
export IMAGE_NAME ?= $(IMAGE_REGISTRY)/$(IMAGE_REPO)

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

TAG_SUFFIX ?=
export IMAGE_TAG ?= $(VERSION)$(TAG_SUFFIX)

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

reviewable: generate docs manifests helm.generate helm.schema.update helm.docs lint license.check helm.test.update test.crds.update tf.fmt ## Ensure a PR is ready for review.
	@go mod tidy
	@cd e2e/ && go mod tidy
	@cd apis/ && go mod tidy
	@cd runtime/ && go mod tidy
	@for provider in providers/v1/*/; do (cd $$provider && go mod tidy); done
	@for generator in generators/v1/*/; do (cd $$generator && go mod tidy); done

check-diff: reviewable ## Ensure branch is clean.
	@$(INFO) checking that branch is clean
	@test -z "$$(git status --porcelain)" || (echo "$$(git status --porcelain)" && $(FAIL))
	@$(OK) branch is clean

update-deps: ## Update dependencies across all modules (root, apis, runtime, e2e, providers, generators)
	@./hack/update-deps.sh

.PHONY: license.check
license.check:
	$(DOCKER) run --rm -u $(shell id -u) -v $(shell pwd):/github/workspace apache/skywalking-eyes:0.6.0 header check

# ====================================================================================
# Golang

.PHONY: go-work ## Creates go workspace and syncs it
go-work:
	@$(INFO) creating go workspace
	@rm -rf go.work go.work.sum
	@go work init
	@go work use -r .
	@go work edit -dropuse ./e2e
	@go work sync
	@$(OK) created go workspace

.PHONY: test
test: generate envtest go-work ## Run tests
	@$(INFO) go test unit-tests
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(KUBERNETES_VERSION) -p path --bin-dir $(LOCALBIN))" go test -tags $(PROVIDER) work -v -race -coverprofile cover.out
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

.PHONY: test.crds
test.crds: cty crds.generate.tests ## Test CRDs for modification and backwards compatibility
	@$(INFO) $(CTY) test tests
	$(CTY) test tests
	@$(OK) No breaking CRD changes detected

.PHONY: test.crds.update
test.crds.update: cty crds.generate.tests ## Update the snapshots used by the CRD tests
	@$(INFO) $(CTY) test tests -u
	$(CTY) test tests -u
	@$(OK) Successfully updated all test snapshots

.PHONY: build
build: $(addprefix build-,$(ARCH)) ## Build binary

PROVIDER ?= all_providers
.PHONY: build-%
build-%: generate ## Build binary for the specified arch
	@$(INFO) go build $*
	$(BUILD_ARGS) GOOS=linux GOARCH=$* \
		go build -tags $(PROVIDER) -o '$(OUTPUT_DIR)/external-secrets-linux-$*' main.go
	@$(OK) go build $*

lint: golangci-lint ## Run golangci-lint (set LINT_TARGET to run on specific module, LINT_JOBS for parallel jobs)
	@if [ -n "$(LINT_TARGET)" ]; then \
		$(INFO) Running golangci-lint on $(LINT_TARGET); \
		(cd $(LINT_TARGET) && $(GOLANGCI_LINT) run ./...) || exit 1; \
		$(OK) Finished linting $(LINT_TARGET); \
	else \
		$(INFO) Running golangci-lint on all modules in parallel; \
		JOBS=$${LINT_JOBS:-20}; \
		TMPDIR=$$(mktemp -d); \
		GOLANGCI=$(GOLANGCI_LINT); \
		trap "rm -rf $$TMPDIR" EXIT; \
		export TMPDIR GOLANGCI; \
		find . -name go.mod -not -path "*/vendor/*" -not -path "*/e2e/*" -not -path "*/node_modules/*" -exec dirname {} \; | \
		xargs -n 1 -P $$JOBS sh -c ' \
			module="$$0"; \
			name=$$(echo "$$module" | sed "s/[\/\.]/_/g"); \
			echo "Linting $$module"; \
			if (cd "$$module" && $$GOLANGCI run ./... 2>&1); then \
				echo "✓ $$module" > "$$TMPDIR/$$name.success"; \
			else \
				echo "✗ $$module" > "$$TMPDIR/$$name.failed"; \
				exit 1; \
			fi \
		'; \
		FAILED=$$(find $$TMPDIR -name "*.failed" 2>/dev/null | wc -l | tr -d " "); \
		SUCCESS=$$(find $$TMPDIR -name "*.success" 2>/dev/null | wc -l | tr -d " "); \
		echo "Results: $$SUCCESS passed, $$FAILED failed"; \
		if [ $$FAILED -ne 0 ]; then \
			echo "Failed modules:"; \
			cat $$TMPDIR/*.failed 2>/dev/null || true; \
			$(ERR) Linting failed in $$FAILED module\(s\); \
			exit 1; \
		fi; \
		$(OK) Finished linting all modules; \
	fi

generate: ## Generate code and crds
	@./hack/crd.generate.sh $(BUNDLE_DIR) $(CRD_DIR)
	@$(OK) Finished generating deepcopy and crds

# ====================================================================================
# Local Utility

# This is for running out-of-cluster locally, and is for convenience.
# For more control, try running the binary directly with different arguments.
run: generate ## Run app locally (without a k8s cluster)
	go run -tags $(PROVIDER) ./main.go

manifests: helm.generate ## Generate manifests from helm chart
	mkdir -p $(OUTPUT_DIR)/deploy/manifests
	helm dependency build $(HELM_DIR)
	helm template external-secrets $(HELM_DIR) -f deploy/manifests/helm-values.yaml > $(OUTPUT_DIR)/deploy/manifests/external-secrets.yaml

crds.install: generate ## Install CRDs into a cluster. This is for convenience
	kubectl apply -f $(BUNDLE_DIR) --server-side

crds.uninstall: ## Uninstall CRDs from a cluster. This is for convenience
	kubectl delete -f $(BUNDLE_DIR)

crds.generate.tests:
	./hack/test.crds.generate.sh $(BUNDLE_DIR) tests/crds
	@$(OK) Finished generating crds for testing

tilt-up: tilt manifests ## Generates the local manifests that tilt will use to deploy the controller's objects.
	$(LOCALBIN)/tilt up

# ====================================================================================
# Helm Chart

helm.docs: ## Generate helm docs
	@cd $(HELM_DIR); \
	$(DOCKER) run --rm -v $(shell pwd)/$(HELM_DIR):/helm-docs -u $(shell id -u) docker.io/jnorwood/helm-docs:v1.14.2

HELM_VERSION ?= $(shell helm show chart $(HELM_DIR) | grep '^version:' | sed 's/version: //g')

helm.build: helm.generate ## Build helm chart
	@$(INFO) helm package
	@helm package $(HELM_DIR) --dependency-update --destination $(OUTPUT_DIR)/chart
	@mv $(OUTPUT_DIR)/chart/external-secrets-$(HELM_VERSION).tgz $(OUTPUT_DIR)/chart/external-secrets.tgz
	@$(OK) helm package

# install_helm_plugin is for installing the provided plugin, if it doesn't exist
# $1 - plugin name
# $2 - plugin version
# $3 - plugin url
define install_helm_plugin
@v=$$(helm plugin list | awk '$$1=="$(1)"{print $$2}'); \
if [ -z "$$v" ]; then \
	$(INFO) "Installing $(1) v$(2)"; \
	helm plugin install --version $(2) $(3); \
	$(OK) "Installed $(1) v$(2)"; \
elif [ "$$v" != "$(2)" ]; then \
	$(INFO) "Found $(1) $$v. Reinstalling v$(2)"; \
	helm plugin remove $(1); \
	helm plugin install --version $(2) $(3); \
	$(OK) "Reinstalled $(1) v$(2)"; \
else \
	$(OK) "$(1) already at v$(2)"; \
fi
endef

HELM_SCHEMA_NAME := schema
HELM_SCHEMA_VER  := 2.2.1
HELM_SCHEMA_URL  := https://github.com/losisin/helm-values-schema-json.git
helm.schema.plugin:
	$(call install_helm_plugin,$(HELM_SCHEMA_NAME),$(HELM_SCHEMA_VER), $(HELM_SCHEMA_URL))

HELM_UNITTEST_PLUGIN_NAME := unittest
HELM_UNITTEST_PLUGIN_VER := 1.0.0
HELM_UNITTEST_PLUGIN_URL := https://github.com/helm-unittest/helm-unittest.git
helm.unittest.plugin:
	$(call install_helm_plugin,$(HELM_UNITTEST_PLUGIN_NAME),$(HELM_UNITTEST_PLUGIN_VER), $(HELM_UNITTEST_PLUGIN_URL))

helm.schema.update: helm.schema.plugin
	@$(INFO) Generating values.schema.json
	@helm schema -f $(HELM_DIR)/values.yaml -o $(HELM_DIR)/values.schema.json
	@$(OK) Generated values.schema.json

helm.generate:
	./hack/helm.generate.sh $(BUNDLE_DIR) $(HELM_DIR)
	@$(OK) Finished generating helm chart files

helm.test: helm.unittest.plugin helm.generate
	@helm unittest deploy/charts/external-secrets/

helm.test.update: helm.unittest.plugin helm.generate
	@helm unittest -u deploy/charts/external-secrets/

helm.update.appversion:
	@chartversion=$$(yq .version ./deploy/charts/external-secrets/Chart.yaml) ; \
	chartappversion=$$(yq .appVersion ./deploy/charts/external-secrets/Chart.yaml) ; \
	chartname=$$(yq .name ./deploy/charts/external-secrets/Chart.yaml) ; \
	$(INFO) Update chartname and chartversion string in test snapshots.; \
	sed -s -i "s/^\([[:space:]]\+helm\.sh\/chart:\).*/\1 $${chartname}-$${chartversion}/" ./deploy/charts/external-secrets/tests/__snapshot__/*.yaml.snap ; \
	sed -s -i "s/^\([[:space:]]\+app\.kubernetes\.io\/version:\).*/\1 $${chartappversion}/" ./deploy/charts/external-secrets/tests/__snapshot__/*.yaml.snap ; \
	sed -s -i "s/^\([[:space:]]\+image: ghcr\.io\/external-secrets\/external-secrets:\).*/\1$${chartappversion}/" ./deploy/charts/external-secrets/tests/__snapshot__/*.yaml.snap ; \
	$(OK) "Version strings updated"

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

DOCS_VERSION ?= $(VERSION)
.PHONY: docs.check
docs.check: ## Check docs
	$(MAKE) -C ./hack/api-docs check DOCS_VERSION=$(DOCS_VERSION)

.PHONY: docs.update
docs.update: ## Update docs
	$(MAKE) -C ./hack/api-docs stability-support.update DOCS_VERSION=$(DOCS_VERSION)

# ====================================================================================
# Build Artifacts

.PHONY: build.all
build.all: docker.build helm.build ## Build all artifacts (docker image, helm chart)

.PHONY: docker.image
docker.image:  ## Emit IMAGE_NAME:IMAGE_TAG
	@echo $(IMAGE_NAME):$(IMAGE_TAG)

.PHONY: docker.imagename
docker.imagename:  ## Emit IMAGE_NAME
	@echo $(IMAGE_NAME)

.PHONY: docker.tag
docker.tag:  ## Emit IMAGE_TAG
	@echo $(IMAGE_TAG)

.PHONY: docker.build
docker.build: $(addprefix build-,$(ARCH)) ## Build the docker image
	@$(INFO) $(DOCKER) build
	echo $(DOCKER) build -f $(DOCKERFILE) . $(DOCKER_BUILD_ARGS) -t $(IMAGE_NAME):$(IMAGE_TAG)
	DOCKER_BUILDKIT=1 $(DOCKER) build -f $(DOCKERFILE) . $(DOCKER_BUILD_ARGS) -t $(IMAGE_NAME):$(IMAGE_TAG)
	@$(OK) $(DOCKER) build

.PHONY: docker.push
docker.push: ## Push the docker image to the registry
	@$(INFO) $(DOCKER) push
	@$(DOCKER) push $(IMAGE_NAME):$(IMAGE_TAG)
	@$(OK) $(DOCKER) push

# RELEASE_TAG is tag to promote. Default is promoting to main branch, but can be overriden
# to promote a tag to a specific version.
RELEASE_TAG ?= $(IMAGE_TAG)
SOURCE_TAG ?= $(VERSION)$(TAG_SUFFIX)

.PHONY: docker.promote
docker.promote: ## Promote the docker image to the registry
	@$(INFO) promoting $(SOURCE_TAG) to $(RELEASE_TAG)
	$(DOCKER) manifest inspect --verbose $(IMAGE_NAME):$(SOURCE_TAG) > .tagmanifest
	for digest in $$(jq -r 'if type=="array" then .[] | select(.Descriptor.platform.architecture != "unknown") | .Descriptor.digest else .Descriptor.digest end' < .tagmanifest); do \
		$(DOCKER) pull $(IMAGE_NAME)@$$digest; \
	done
	$(DOCKER) manifest create $(IMAGE_NAME):$(RELEASE_TAG) \
		$$(jq -j 'if type=="array" then [.[] | select(.Descriptor.platform.architecture != "unknown")] | map("--amend $(IMAGE_NAME)@" + .Descriptor.digest) | join(" ") else "--amend $(IMAGE_NAME)@" + .Descriptor.digest end' < .tagmanifest)
	$(DOCKER) manifest push $(IMAGE_NAME):$(RELEASE_TAG)
	@$(OK) $(DOCKER) push $(RELEASE_TAG) \

# ====================================================================================
# Terraform

define run_terraform
	@cd $(TF_DIR)/$1/infrastructure && \
	terraform init && \
	$2 && \
	cd ../kubernetes && \
	terraform init && \
	$3
endef

tf.plan.%:
	$(call run_terraform,$*,terraform plan,terraform plan)

tf.apply.%:
	$(call run_terraform,$*,terraform apply -auto-approve,terraform apply -auto-approve)

tf.destroy.%:
	@cd $(TF_DIR)/$*/kubernetes && \
	terraform init && \
	terraform destroy -auto-approve && \
	cd ../infrastructure && \
	terraform init && \
	terraform destroy -auto-approve

tf.fmt:
	@cd $(TF_DIR) && \
	terraform fmt -recursive

# ====================================================================================
# Help

.PHONY: help
# only comments after make target name are shown as help text
help: ## Displays this help message
	@echo -e "$$(grep -hE '^\S+:.*##' $(MAKEFILE_LIST) | sed -e 's/:.*##\s*/|/' -e 's/^\(.\+\):\(.*\)/\\x1b[36m\1\\x1b[m:\2/' | column -c2 -t -s'|' | sort)"


.PHONY: clean
clean:  ## Clean bins
	@$(INFO) clean
	@rm -f $(OUTPUT_DIR)/external-secrets-linux-*
	@$(OK) go build $*

# ====================================================================================
# Build Dependencies

detected_OS := $(shell uname -s)
real_OS := $(detected_OS)
arch := $(shell uname -m)
ifeq ($(detected_OS),Darwin)
        detected_OS := mac
        real_OS := darwin
endif
ifeq ($(detected_OS),Linux)
        detected_OS := linux
	real_OS := linux
endif

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
TILT ?= $(LOCALBIN)/tilt
CTY ?= $(LOCALBIN)/cty
ENVTEST ?= $(LOCALBIN)/setup-envtest
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint
LINT_TARGET ?= ""
## Tool Versions
GOLANGCI_VERSION := 2.4.0
KUBERNETES_VERSION := 1.33.x
TILT_VERSION := 0.33.21
CTY_VERSION := 1.1.3

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: golangci-lint
.PHONY: $(GOLANGCI_LINT)
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	test -s $(LOCALBIN)/golangci-lint && $(LOCALBIN)/golangci-lint version | grep -q $(GOLANGCI_VERSION) || \
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(LOCALBIN) v$(GOLANGCI_VERSION)

.PHONY: tilt
.PHONY: $(TILT)
tilt: $(TILT) ## Download tilt locally if necessary. Architecture is locked at x86_64.
$(TILT): $(LOCALBIN)
	test -s $(LOCALBIN)/tilt || curl -fsSL https://github.com/tilt-dev/tilt/releases/download/v$(TILT_VERSION)/tilt.$(TILT_VERSION).$(detected_OS).$(arch).tar.gz | tar -xz -C $(LOCALBIN) tilt

.PHONY: cty
.PHONY: $(CTY)
cty: $(CTY) ## Download cty locally if necessary. Architecture is locked at x86_64.
$(CTY): $(LOCALBIN)
	test -s $(LOCALBIN)/cty || curl -fsSL https://github.com/Skarlso/crd-to-sample-yaml/releases/download/v$(CTY_VERSION)/cty_$(real_OS)_amd64.tar.gz | tar -xz -C $(LOCALBIN) cty
