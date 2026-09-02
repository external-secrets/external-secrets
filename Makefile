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
	@GOWORK=off go -C hack/tools/gen-crd-api-reference-docs mod tidy
	@go mod tidy
	@cd e2e/ && go mod tidy
	@cd apis/ && go mod tidy
	@cd runtime/ && go mod tidy
	@for provider in providers/v1/*/; do (cd $$provider && go mod tidy); done
	@for generator in generators/v1/*/; do (cd $$generator && go mod tidy); done

check-diff: reviewable ## Ensure branch is clean.
	@$(INFO) checking that branch is clean
	@status="$$(git status --porcelain)" && test -z "$$status" || (printf '%s\n' "$$status" && $(FAIL))
	@$(OK) branch is clean

UPDATECLI_ACTION ?= apply
UPDATECLI_KIND ?=
UPDATECLI_PUBLISH ?= false
UPDATECLI_CONFIG = .updatecli.d$(if $(UPDATECLI_KIND),/$(UPDATECLI_KIND).yaml)
update-deps: updatecli ## Update dependencies; use UPDATECLI_KIND=<kind> to limit scope, UPDATECLI_ACTION=diff to preview, or UPDATECLI_PUBLISH=true to publish PRs.
	@test -e "$(UPDATECLI_CONFIG)" || (echo "unknown dependency kind: $(UPDATECLI_KIND)" >&2; exit 1)
	@set -e; \
	publish="$(UPDATECLI_PUBLISH)"; \
	case "$$publish" in true|false) ;; *) echo "UPDATECLI_PUBLISH must be true or false" >&2; exit 1 ;; esac; \
	token="$${GITHUB_TOKEN:-$$(gh auth token 2>/dev/null)}"; \
	test -n "$$token" || (echo "GITHUB_TOKEN is required; set it or run 'gh auth login'" >&2; exit 1); \
	values="{scm: {enabled: false}}"; \
	if test "$$publish" = true; then \
		slug="$${GITHUB_REPOSITORY:-$$(gh repo view --json nameWithOwner --jq .nameWithOwner 2>/dev/null)}"; \
		test -n "$$slug" || (echo "GITHUB_REPOSITORY is required when the repository cannot be detected with gh" >&2; exit 1); \
		owner="$${slug%%/*}"; repository="$${slug#*/}"; \
		values="{scm: {enabled: true, owner: $$owner, repository: $$repository}}"; \
	fi; \
	GITHUB_TOKEN="$$token" $(LOCALBIN)/updatecli pipeline $(UPDATECLI_ACTION) --config $(UPDATECLI_CONFIG) \
		--values-inline "$$values" \
		--disable-changelog --disable-version-check

.PHONY: update-deps update-deps-github-actions update-deps-containers update-deps-tools update-deps-helm update-deps-python update-deps-terraform
update-deps-github-actions: UPDATECLI_KIND=github-actions
update-deps-github-actions: update-deps ## Update GitHub Actions only.
update-deps-containers: UPDATECLI_KIND=docker
update-deps-containers: update-deps ## Update container images only.
update-deps-tools: UPDATECLI_KIND=tools
update-deps-tools: update-deps ## Update development tools only.
update-deps-helm: UPDATECLI_KIND=helm
update-deps-helm: update-deps ## Update Helm dependencies only.
update-deps-python: UPDATECLI_KIND=python
update-deps-python: update-deps ## Update Python dependencies only.
update-deps-terraform: UPDATECLI_KIND=terraform
update-deps-terraform: update-deps ## Update Terraform dependencies only.

.PHONY: license.check
license.check:
	$(DOCKER) run --rm -u $(shell id -u) -v $(shell pwd):/github/workspace docker.io/apache/skywalking-eyes:0.9.0@sha256:b503ab18f5b29d7e04abe05668faf42f1023f9347b095d97592a5208102588ef header check

# ====================================================================================
# Golang

.PHONY: go-work ## Creates go workspace and syncs it
go-work:
	@$(INFO) creating go workspace
	@rm -rf go.work go.work.sum
	@go work init
	@go work use -r .
	@go work edit -dropuse ./e2e -dropuse ./hack/tools/gen-crd-api-reference-docs
	@go work sync
	@$(OK) created go workspace

.PHONY: test
test: generate envtest ## Run tests
	@$(INFO) go test unit-tests
	@set -e; \
	snap=$$(mktemp); \
	./hack/modfiles.sh snapshot "$$snap"; \
	trap "./hack/modfiles.sh restore $$snap" EXIT INT TERM; \
	$(MAKE) go-work; \
	KUBEBUILDER_ASSETS="$$($(LOCALBIN)/setup-envtest use "$(ENVTEST_KUBERNETES_VERSION)" -p path --bin-dir $(LOCALBIN))" \
	    go test -tags $(PROVIDER) work -v -race -coverprofile cover.out
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
	@$(INFO) $(LOCALBIN)/cty test tests
	$(LOCALBIN)/cty test tests
	@$(OK) No breaking CRD changes detected

.PHONY: test.crds.update
test.crds.update: cty crds.generate.tests ## Update the snapshots used by the CRD tests
	@$(INFO) $(LOCALBIN)/cty test tests -u
	$(LOCALBIN)/cty test tests -u
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

.PHONY: provider-replaces.check
provider-replaces.check: ## Ensure cross-provider dependencies use local replacements
	@./hack/check-provider-replaces.sh

lint: golangci-lint provider-replaces.check ## Run golangci-lint (set LINT_TARGET to run on specific module, LINT_JOBS for parallel jobs)
	@if [ -n "$(LINT_TARGET)" ]; then \
		$(INFO) Running golangci-lint on $(LINT_TARGET); \
		(cd $(LINT_TARGET) && $(LOCALBIN)/golangci-lint run ./...) || exit 1; \
		$(OK) Finished linting $(LINT_TARGET); \
	else \
		$(INFO) Running golangci-lint on all modules in parallel; \
		JOBS=$${LINT_JOBS:-1}; \
		TMPDIR=$$(mktemp -d); \
		GOLANGCI=$(LOCALBIN)/golangci-lint; \
		trap "rm -rf $$TMPDIR" EXIT; \
		export TMPDIR GOLANGCI; \
		find . -name go.mod -not -path "*/vendor/*" -not -path "*/e2e/*" -not -path "*/hack/tools/*" -not -path "*/node_modules/*" -exec dirname {} \; | \
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

generate: controller-gen ## Generate code and crds
	@CONTROLLER_GEN=$(LOCALBIN)/controller-gen ./hack/crd.generate.sh $(BUNDLE_DIR) $(CRD_DIR)
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
	$(DOCKER) run --rm -v $(shell pwd)/$(HELM_DIR):/helm-docs -u $(shell id -u) docker.io/jnorwood/helm-docs:v1.14.2@sha256:7e562b49ab6b1dbc50c3da8f2dd6ffa8a5c6bba327b1c6335cc15ce29267979c

HELM_VERSION ?= $(shell helm show chart $(HELM_DIR) | grep '^version:' | sed 's/version: //g')

helm.build: helm.generate ## Build helm chart
	@$(INFO) helm package
	@helm package $(HELM_DIR) --dependency-update --destination $(OUTPUT_DIR)/chart
	@mv $(OUTPUT_DIR)/chart/external-secrets-$(HELM_VERSION).tgz $(OUTPUT_DIR)/chart/external-secrets.tgz
	@$(OK) helm package

HELM_SCHEMA_PLUGIN_VERSION := 2.2.1
HELM_SCHEMA_PLUGIN_URL := https://github.com/losisin/helm-values-schema-json.git
HELM_UNITTEST_PLUGIN_VERSION := 1.0.0
HELM_UNITTEST_PLUGIN_URL := https://github.com/helm-unittest/helm-unittest.git

define install_helm_plugin
	@version=$$(helm plugin list | awk '$$1 == "$(1)" { print $$2 }'); \
	if test "$$version" != "$(2)"; then \
		if test -n "$$version"; then helm plugin remove "$(1)"; fi; \
		helm plugin install --version "$(2)" "$(3)"; \
	fi; \
	echo "Helm plugin $(1) $(2) is ready"
endef

helm.schema.plugin:
	$(call install_helm_plugin,schema,$(HELM_SCHEMA_PLUGIN_VERSION),$(HELM_SCHEMA_PLUGIN_URL))

helm.unittest.plugin:
	$(call install_helm_plugin,unittest,$(HELM_UNITTEST_PLUGIN_VERSION),$(HELM_UNITTEST_PLUGIN_URL))

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
	echo $(DOCKER) buildx build -f $(DOCKERFILE) . $(DOCKER_BUILD_ARGS) -t $(IMAGE_NAME):$(IMAGE_TAG)
	$(DOCKER) buildx build -f $(DOCKERFILE) . $(DOCKER_BUILD_ARGS) -t $(IMAGE_NAME):$(IMAGE_TAG)
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

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $@

## Tool Binaries
LINT_TARGET ?= ""
TOOL_OS := $(shell uname -s | tr '[:lower:]' '[:upper:]')
TOOL_ARCH := $(shell uname -m | sed -e 's/^x86_64$$/AMD64/' -e 's/^amd64$$/AMD64/' -e 's/^aarch64$$/ARM64/' -e 's/^arm64$$/ARM64/')
TOOL_PLATFORM := $(TOOL_OS)_$(TOOL_ARCH)

ENVTEST_KUBERNETES_VERSION := 1.33.x

CONTROLLER_GEN_VERSION := v0.19.0
CONTROLLER_GEN_REPOSITORY := kubernetes-sigs/controller-tools
CONTROLLER_GEN_FORMAT := binary
CONTROLLER_GEN_DARWIN_AMD64_ASSET := controller-gen-darwin-amd64
CONTROLLER_GEN_DARWIN_AMD64_BINARY := controller-gen-darwin-amd64
CONTROLLER_GEN_DARWIN_AMD64_SHA256 := 20ba6eab9c769c7de27b85d3a0bd115c2e03f81d06d2a522d9c28500853a5e06
CONTROLLER_GEN_DARWIN_ARM64_ASSET := controller-gen-darwin-arm64
CONTROLLER_GEN_DARWIN_ARM64_BINARY := controller-gen-darwin-arm64
CONTROLLER_GEN_DARWIN_ARM64_SHA256 := 09f2f22eaffb179374cab99ab55953236a0daea64ef61b573818ac9f329bc48b
CONTROLLER_GEN_LINUX_AMD64_ASSET := controller-gen-linux-amd64
CONTROLLER_GEN_LINUX_AMD64_BINARY := controller-gen-linux-amd64
CONTROLLER_GEN_LINUX_AMD64_SHA256 := 5df5d2cced0621d7d8d8040ef20482f5c6e2ced32f1b1ad825f1bdf52f433161
CONTROLLER_GEN_LINUX_ARM64_ASSET := controller-gen-linux-arm64
CONTROLLER_GEN_LINUX_ARM64_BINARY := controller-gen-linux-arm64
CONTROLLER_GEN_LINUX_ARM64_SHA256 := 50dc28b10d4080e0bc547f8de91b26054fd58d51147f5cb36fd632400a3c5f4f

SETUP_ENVTEST_VERSION := v0.24.1
SETUP_ENVTEST_REPOSITORY := kubernetes-sigs/controller-runtime
SETUP_ENVTEST_FORMAT := binary
SETUP_ENVTEST_DARWIN_AMD64_ASSET := setup-envtest-darwin-amd64
SETUP_ENVTEST_DARWIN_AMD64_BINARY := setup-envtest-darwin-amd64
SETUP_ENVTEST_DARWIN_AMD64_SHA256 := 3fb17f2b1b0f09b7e5395180bd2bcb1d53bb78d72bb0415106b7ae8bf64e23d0
SETUP_ENVTEST_DARWIN_ARM64_ASSET := setup-envtest-darwin-arm64
SETUP_ENVTEST_DARWIN_ARM64_BINARY := setup-envtest-darwin-arm64
SETUP_ENVTEST_DARWIN_ARM64_SHA256 := 7e59a0d526f6946aa2f114d34b2e309639c811f3a4f83d56f37b6e3197c6fdfb
SETUP_ENVTEST_LINUX_AMD64_ASSET := setup-envtest-linux-amd64
SETUP_ENVTEST_LINUX_AMD64_BINARY := setup-envtest-linux-amd64
SETUP_ENVTEST_LINUX_AMD64_SHA256 := a9a78fadfc338a38188332f36863c76877f1c86df1a83d2241d2bfc3935297d2
SETUP_ENVTEST_LINUX_ARM64_ASSET := setup-envtest-linux-arm64
SETUP_ENVTEST_LINUX_ARM64_BINARY := setup-envtest-linux-arm64
SETUP_ENVTEST_LINUX_ARM64_SHA256 := c5d8968ec3f2a120b66bc13bd36f80fe4150c34aae7cc491bf9624c8680296c7

GOLANGCI_LINT_VERSION := v2.12.2
GOLANGCI_LINT_REPOSITORY := golangci/golangci-lint
GOLANGCI_LINT_FORMAT := tar.gz
GOLANGCI_LINT_DARWIN_AMD64_ASSET := golangci-lint-$(patsubst v%,%,$(GOLANGCI_LINT_VERSION))-darwin-amd64.tar.gz
GOLANGCI_LINT_DARWIN_AMD64_BINARY := golangci-lint-$(patsubst v%,%,$(GOLANGCI_LINT_VERSION))-darwin-amd64/golangci-lint
GOLANGCI_LINT_DARWIN_AMD64_SHA256 := f6f06d94b6241521c53d15450c5209b028270bf966f842afb11c030c79f5bc16
GOLANGCI_LINT_DARWIN_ARM64_ASSET := golangci-lint-$(patsubst v%,%,$(GOLANGCI_LINT_VERSION))-darwin-arm64.tar.gz
GOLANGCI_LINT_DARWIN_ARM64_BINARY := golangci-lint-$(patsubst v%,%,$(GOLANGCI_LINT_VERSION))-darwin-arm64/golangci-lint
GOLANGCI_LINT_DARWIN_ARM64_SHA256 := a9c54498731b3128f79e090be6110f3e5fffccc617b08142ed244d4126c73f29
GOLANGCI_LINT_LINUX_AMD64_ASSET := golangci-lint-$(patsubst v%,%,$(GOLANGCI_LINT_VERSION))-linux-amd64.tar.gz
GOLANGCI_LINT_LINUX_AMD64_BINARY := golangci-lint-$(patsubst v%,%,$(GOLANGCI_LINT_VERSION))-linux-amd64/golangci-lint
GOLANGCI_LINT_LINUX_AMD64_SHA256 := 8df580d2670fed8fa984aac0507099af8df275e665215f5c7a2ae3943893a553
GOLANGCI_LINT_LINUX_ARM64_ASSET := golangci-lint-$(patsubst v%,%,$(GOLANGCI_LINT_VERSION))-linux-arm64.tar.gz
GOLANGCI_LINT_LINUX_ARM64_BINARY := golangci-lint-$(patsubst v%,%,$(GOLANGCI_LINT_VERSION))-linux-arm64/golangci-lint
GOLANGCI_LINT_LINUX_ARM64_SHA256 := 44cd40a8c76c86755375adfeea52cfd3533cb43d7bd647771e0ae065e166df3a

DLV_VERSION := v1.26.3
DLV_REPOSITORY := go-delve/delve
DLV_FORMAT := tar.gz
DLV_DARWIN_AMD64_ASSET := dlv_$(patsubst v%,%,$(DLV_VERSION))_darwin_amd64.tar.gz
DLV_DARWIN_AMD64_BINARY := dlv
DLV_DARWIN_AMD64_SHA256 := 6827a438473167a1e0805b4546e5bf2d53401530f694deb35e41c6e7b46e27c8
DLV_DARWIN_ARM64_ASSET := dlv_$(patsubst v%,%,$(DLV_VERSION))_darwin_arm64.tar.gz
DLV_DARWIN_ARM64_BINARY := dlv
DLV_DARWIN_ARM64_SHA256 := 7f28483a42f0a911f29b236aa40d24d7099f1b0ec54c56c4d439a6903d478a3d
DLV_LINUX_AMD64_ASSET := dlv_$(patsubst v%,%,$(DLV_VERSION))_linux_amd64.tar.gz
DLV_LINUX_AMD64_BINARY := dlv
DLV_LINUX_AMD64_SHA256 := cdd4d6b2a638d8f26468d82a76b766df594641490bea566629305d90fbccc06e
DLV_LINUX_ARM64_ASSET := dlv_$(patsubst v%,%,$(DLV_VERSION))_linux_arm64.tar.gz
DLV_LINUX_ARM64_BINARY := dlv
DLV_LINUX_ARM64_SHA256 := 5b03fd74895d676c4435bec1aade0863be1489a4be1bb5c9269c6ef389bf5d2d

CRANE_VERSION := v0.22.0
CRANE_REPOSITORY := google/go-containerregistry
CRANE_FORMAT := tar.gz
CRANE_DARWIN_AMD64_ASSET := go-containerregistry_Darwin_x86_64.tar.gz
CRANE_DARWIN_AMD64_BINARY := crane
CRANE_DARWIN_AMD64_SHA256 := 7ece3bde64b8eb14f8f1d33affc70f7e0b9b2fe0b2e115c46d246a17e73ed622
CRANE_DARWIN_ARM64_ASSET := go-containerregistry_Darwin_arm64.tar.gz
CRANE_DARWIN_ARM64_BINARY := crane
CRANE_DARWIN_ARM64_SHA256 := 0d6955caf5168b163824317b3b0caed699b3ca3ca9523dd4442e3227e83dd1ef
CRANE_LINUX_AMD64_ASSET := go-containerregistry_Linux_x86_64.tar.gz
CRANE_LINUX_AMD64_BINARY := crane
CRANE_LINUX_AMD64_SHA256 := edb74d53fad9a596860f59d1c5d04a43dfb5f441dc71f57060dd0bf39483c833
CRANE_LINUX_ARM64_ASSET := go-containerregistry_Linux_arm64.tar.gz
CRANE_LINUX_ARM64_BINARY := crane
CRANE_LINUX_ARM64_SHA256 := 971e8e2de7e09172330527a9dfb10689e344b160fd81bac7fb91adcb109bad37

TILT_VERSION := v0.33.21
TILT_REPOSITORY := tilt-dev/tilt
TILT_FORMAT := tar.gz
TILT_DARWIN_AMD64_ASSET := tilt.$(patsubst v%,%,$(TILT_VERSION)).mac.x86_64.tar.gz
TILT_DARWIN_AMD64_BINARY := tilt
TILT_DARWIN_AMD64_SHA256 := 40d49940a17333020614bf7fc8b3cf210811fd2fef6cbc203ee165e7fc0106bf
TILT_DARWIN_ARM64_ASSET := tilt.$(patsubst v%,%,$(TILT_VERSION)).mac.arm64.tar.gz
TILT_DARWIN_ARM64_BINARY := tilt
TILT_DARWIN_ARM64_SHA256 := 3038f8fbfb815d3c65f2b79cf55b524b46081b21e784a843ff058e06cc433172
TILT_LINUX_AMD64_ASSET := tilt.$(patsubst v%,%,$(TILT_VERSION)).linux.x86_64.tar.gz
TILT_LINUX_AMD64_BINARY := tilt
TILT_LINUX_AMD64_SHA256 := 10696e07eacfee5489f66ed1934b53eddabd3144c36b3533427f528e3a6dce89
TILT_LINUX_ARM64_ASSET := tilt.$(patsubst v%,%,$(TILT_VERSION)).linux.arm64.tar.gz
TILT_LINUX_ARM64_BINARY := tilt
TILT_LINUX_ARM64_SHA256 := d76162001028091ca33389ae2f5f3808c8ddcd105204e32d79135494e947fa39

CTY_VERSION := v1.1.5
CTY_REPOSITORY := Skarlso/crd-to-sample-yaml
CTY_FORMAT := tar.gz
CTY_DARWIN_AMD64_ASSET := cty_darwin_amd64.tar.gz
CTY_DARWIN_AMD64_BINARY := cty
CTY_DARWIN_AMD64_SHA256 := 9de1bec65bdc54a74e81f0424fdd9211488d9da38c73e8e0ccf0986535df4db1
CTY_DARWIN_ARM64_ASSET := cty_darwin_arm64.tar.gz
CTY_DARWIN_ARM64_BINARY := cty
CTY_DARWIN_ARM64_SHA256 := eb73fd2e4223b70ab1d69a0e1ed9208e228c5ce8ec3df063fbb5bf6791047c02
CTY_LINUX_AMD64_ASSET := cty_linux_amd64.tar.gz
CTY_LINUX_AMD64_BINARY := cty
CTY_LINUX_AMD64_SHA256 := 9a9b57c46f10d8e3ac66f8e5fcc6e99cfacf4184fc1fcb1b526b9dc63ea7315b
CTY_LINUX_ARM64_ASSET := cty_linux_arm64.tar.gz
CTY_LINUX_ARM64_BINARY := cty
CTY_LINUX_ARM64_SHA256 := 472250183a3f70bf89de2ca388984f05608638bf4fd5fba09a4b98bdb01343ab

UPDATECLI_VERSION := v0.120.1
UPDATECLI_REPOSITORY := updatecli/updatecli
UPDATECLI_FORMAT := tar.gz
UPDATECLI_DARWIN_AMD64_ASSET := updatecli_Darwin_x86_64.tar.gz
UPDATECLI_DARWIN_AMD64_BINARY := updatecli
UPDATECLI_DARWIN_AMD64_SHA256 := 75f2d2c174c7d85086fd7982b1d35e8450ab24e44a0dfcc3050480a524a90ebd
UPDATECLI_DARWIN_ARM64_ASSET := updatecli_Darwin_arm64.tar.gz
UPDATECLI_DARWIN_ARM64_BINARY := updatecli
UPDATECLI_DARWIN_ARM64_SHA256 := df4c9bdbe7c3003a4f382cee97a78080cb97189e6650d5ee85857de59057eb0d
UPDATECLI_LINUX_AMD64_ASSET := updatecli_Linux_x86_64.tar.gz
UPDATECLI_LINUX_AMD64_BINARY := updatecli
UPDATECLI_LINUX_AMD64_SHA256 := e2a17372253bcfcd0759ed6ddaaffef07f26ff2d5e8205d1cb1baa2cf4efab2c
UPDATECLI_LINUX_ARM64_ASSET := updatecli_Linux_arm64.tar.gz
UPDATECLI_LINUX_ARM64_BINARY := updatecli
UPDATECLI_LINUX_ARM64_SHA256 := 52fb346c469f4f3f7d084edccc8ad4ebf90cc8f68dfa4d924384b533904faa36

define install_tool
	@set -eu; \
	asset='$($(1)_$(TOOL_PLATFORM)_ASSET)'; \
	checksum='$($(1)_$(TOOL_PLATFORM)_SHA256)'; \
	binary='$($(1)_$(TOOL_PLATFORM)_BINARY)'; \
	if test -z "$$asset" || test -z "$$checksum"; then echo "$(2) does not support $(TOOL_OS)/$(TOOL_ARCH)" >&2; exit 1; fi; \
	mkdir -p "$(dir $@)"; \
	tmp=$$(mktemp -d "$(dir $@).$(2)-XXXXXX"); \
	trap 'rm -rf "$$tmp"' EXIT; \
	url="https://github.com/$($(1)_REPOSITORY)/releases/download/$($(1)_VERSION)/$$asset"; \
	echo "Downloading $(2) $($(1)_VERSION)"; \
	curl --fail --location --retry 3 --silent --show-error --output "$$tmp/asset" "$$url"; \
	if command -v sha256sum >/dev/null 2>&1; then actual=$$(sha256sum "$$tmp/asset"); else actual=$$(shasum -a 256 "$$tmp/asset"); fi; \
	actual=$${actual%% *}; \
	if test "$$actual" != "$$checksum"; then echo "checksum mismatch for $$asset: got $$actual, want $$checksum" >&2; exit 1; fi; \
	case '$($(1)_FORMAT)' in \
		binary) source="$$tmp/asset" ;; \
		tar.gz) tar -xzf "$$tmp/asset" -C "$$tmp" "$$binary"; source="$$tmp/$$binary" ;; \
		*) echo "unsupported format: $($(1)_FORMAT)" >&2; exit 1 ;; \
	esac; \
	cp "$$source" "$$tmp/.installed"; \
	chmod 0755 "$$tmp/.installed"; \
	mv "$$tmp/.installed" "$@"; \
	echo "$(2) $($(1)_VERSION) installed successfully"
endef

.PHONY: envtest
envtest: $(LOCALBIN)/setup-envtest
$(LOCALBIN)/setup-envtest: Makefile ## Download setup-envtest locally if necessary.
	$(call install_tool,SETUP_ENVTEST,setup-envtest)

.PHONY: controller-gen
controller-gen: $(LOCALBIN)/controller-gen
$(LOCALBIN)/controller-gen: Makefile ## Download controller-gen locally if necessary.
	$(call install_tool,CONTROLLER_GEN,controller-gen)

.PHONY: gen-crd-api-reference-docs
gen-crd-api-reference-docs: $(LOCALBIN)/gen-crd-api-reference-docs
$(LOCALBIN)/gen-crd-api-reference-docs: Makefile hack/tools/gen-crd-api-reference-docs/go.mod hack/tools/gen-crd-api-reference-docs/go.sum ## Build gen-crd-api-reference-docs locally if necessary.
	mkdir -p $(dir $@)
	GOWORK=off go -C hack/tools/gen-crd-api-reference-docs build -mod=readonly -o $(abspath $@) github.com/ahmetb/gen-crd-api-reference-docs

.PHONY: golangci-lint
golangci-lint: $(LOCALBIN)/golangci-lint
$(LOCALBIN)/golangci-lint: Makefile ## Download golangci-lint locally if necessary.
	$(call install_tool,GOLANGCI_LINT,golangci-lint)

.PHONY: dlv
dlv: $(LOCALBIN)/dlv
$(LOCALBIN)/dlv: Makefile ## Download Delve locally for the Tilt debug image.
	$(call install_tool,DLV,dlv)

.PHONY: crane
crane: $(LOCALBIN)/crane
$(LOCALBIN)/crane: Makefile ## Download crane locally if necessary.
	$(call install_tool,CRANE,crane)

.PHONY: tilt
tilt: $(LOCALBIN)/tilt
$(LOCALBIN)/tilt: Makefile ## Download tilt locally if necessary.
	$(call install_tool,TILT,tilt)

.PHONY: cty
cty: $(LOCALBIN)/cty
$(LOCALBIN)/cty: Makefile ## Download cty locally if necessary.
	$(call install_tool,CTY,cty)

.PHONY: updatecli
updatecli: $(LOCALBIN)/updatecli
$(LOCALBIN)/updatecli: Makefile ## Download Updatecli locally if necessary.
	$(call install_tool,UPDATECLI,updatecli)
