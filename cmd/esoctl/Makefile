# set the shell to bash always
SHELL         := /bin/bash

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

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

.PHONY: build
build: ## Build binary for the specified arch
	go build -o '$(LOCALBIN)/esoctl' -trimpath -ldflags="-s -w -X 'main.version=$(VERSION)'" .

.PHONY: binaries
binaries: ## Build release binaries for all major OSs.
	@rm -fr dist
	@mkdir -p dist
	GOOS=linux GOARCH=amd64 go build -o dist/esoctl-linux-amd64 -trimpath -ldflags="-s -w -X 'main.version=$(VERSION)'" .
	GOOS=darwin GOARCH=amd64 go build -o dist/esoctl-darwin-amd64 -trimpath -ldflags="-s -w -X 'main.version=$(VERSION)'" .
	GOOS=windows GOARCH=amd64 go build -o dist/esoctl-windows-amd64.exe -trimpath -ldflags="-s -w -X 'main.version=$(VERSION)'" .
