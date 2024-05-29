# set the shell to bash always
SHELL         := /bin/bash

# set make and shell flags to exit on errors
MAKEFLAGS     += --warn-undefined-variables
.SHELLFLAGS   := -euo pipefail -c

ARCH ?= amd64 arm64 ppc64le
BUILD_ARGS ?= CGO_ENABLED=0
DOCKER_BUILD_ARGS ?=
DOCKERFILE ?= Dockerfile

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

reviewable: generate docs manifests helm.generate helm.docs lint ## Ensure a PR is ready for review.
	@go mod tidy
	@cd e2e/ && go mod tidy

check-diff: reviewable ## Ensure branch is clean.
	@$(INFO) checking that branch is clean
	@test -z "$$(git status --porcelain)" || (echo "$$(git status --porcelain)" && $(FAIL))
	@$(OK) branch is clean

update-deps:
	go get -u
	cd e2e && go get -u
	@go mod tidy
	@cd e2e/ && go mod tidy

# ====================================================================================
# Golang

.PHONY: test
test: generate envtest ## Run tests
	@$(INFO) go test unit-tests
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(KUBERNETES_VERSION) -p path --bin-dir $(LOCALBIN))" go test -race -v $(shell go list ./... | grep -v e2e) -coverprofile cover.out
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
	$(BUILD_ARGS) GOOS=linux GOARCH=$* \
		go build -o '$(OUTPUT_DIR)/external-secrets-linux-$*' main.go
	@$(OK) go build $*

lint: golangci-lint ## Run golangci-lint
	@if ! $(GOLANGCI_LINT) run; then \
		echo -e "\033[0;33mgolangci-lint failed: some checks can be fixed with \`\033[0;32mmake fmt\033[0m\033[0;33m\`\033[0m"; \
		exit 1; \
	fi
	@$(OK) Finished linting

fmt: golangci-lint ## Ensure consistent code style
	@go mod tidy
	@cd e2e/ && go mod tidy
	@go fmt ./...
	@$(GOLANGCI_LINT) run --fix
	@$(OK) Ensured consistent code style

generate: ## Generate code and crds
	@./hack/crd.generate.sh $(BUNDLE_DIR) $(CRD_DIR)
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

tilt-up: tilt manifests ## Generates the local manifests that tilt will use to deploy the controller's objects.
	$(LOCALBIN)/tilt up

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
	./hack/helm.generate.sh $(BUNDLE_DIR) $(HELM_DIR)
	@$(OK) Finished generating helm chart files

helm.test: helm.generate
	@helm unittest --file tests/*.yaml --file 'tests/**/*.yaml' deploy/charts/external-secrets/

helm.test.update: helm.generate
	@helm unittest -u --file tests/*.yaml --file 'tests/**/*.yaml' deploy/charts/external-secrets/

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
	@$(INFO) docker build
	echo docker build -f $(DOCKERFILE) . $(DOCKER_BUILD_ARGS) -t $(IMAGE_NAME):$(IMAGE_TAG)
	DOCKER_BUILDKIT=1 docker build -f $(DOCKERFILE) . $(DOCKER_BUILD_ARGS) -t $(IMAGE_NAME):$(IMAGE_TAG)
	@$(OK) docker build

.PHONY: docker.push
docker.push: ## Push the docker image to the registry
	@$(INFO) docker push
	@docker push $(IMAGE_NAME):$(IMAGE_TAG)
	@$(OK) docker push

# RELEASE_TAG is tag to promote. Default is promoting to main branch, but can be overriden
# to promote a tag to a specific version.
RELEASE_TAG ?= $(IMAGE_TAG)
SOURCE_TAG ?= $(VERSION)$(TAG_SUFFIX)

.PHONY: docker.promote
docker.promote: ## Promote the docker image to the registry
	@$(INFO) promoting $(SOURCE_TAG) to $(RELEASE_TAG)
	docker manifest inspect --verbose $(IMAGE_NAME):$(SOURCE_TAG) > .tagmanifest
	for digest in $$(jq -r 'if type=="array" then .[].Descriptor.digest else .Descriptor.digest end' < .tagmanifest); do \
		docker pull $(IMAGE_NAME)@$$digest; \
	done
	docker manifest create $(IMAGE_NAME):$(RELEASE_TAG) \
		$$(jq -j '"--amend $(IMAGE_NAME)@" + if type=="array" then .[].Descriptor.digest else .Descriptor.digest end + " "' < .tagmanifest)
	docker manifest push $(IMAGE_NAME):$(RELEASE_TAG)
	@$(OK) docker push $(RELEASE_TAG) \

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

ifeq ($(OS),Windows_NT)     # is Windows_NT on XP, 2000, 7, Vista, 10...
    detected_OS := windows
    arch := x86_64
else
    detected_OS := $(shell uname -s)
    arch := $(shell uname -m)
    ifeq ($(detected_OS),Darwin)
    	detected_OS := mac
    endif
    ifeq ($(detected_OS),Linux)
    	detected_OS := linux
    endif
endif

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
TILT ?= $(LOCALBIN)/tilt
ENVTEST ?= $(LOCALBIN)/setup-envtest
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint

## Tool Versions
GOLANGCI_VERSION := 1.57.2
KUBERNETES_VERSION := 1.30.x
TILT_VERSION := 0.33.10

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: golangci-lint
.PHONY: $(GOLANGCI_LINT)
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	test -s $(LOCALBIN)/golangci-lint && $(LOCALBIN)/golangci-lint version --format short | grep -q $(GOLANGCI_VERSION) || \
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(LOCALBIN) v$(GOLANGCI_VERSION)

.PHONY: tilt
.PHONY: $(TILT)
tilt: $(TILT) ## Download tilt locally if necessary. Architecture is locked at x86_64.
$(TILT): $(LOCALBIN)
	test -s $(LOCALBIN)/tilt || curl -fsSL https://github.com/tilt-dev/tilt/releases/download/v$(TILT_VERSION)/tilt.$(TILT_VERSION).$(detected_OS).$(arch).tar.gz | tar -xz -C $(LOCALBIN) tilt
