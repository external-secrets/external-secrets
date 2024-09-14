# Copyright External Secrets Inc. All Rights Reserved

DOCKER ?= docker
MKDOCS_IMAGE ?= github.com/external-secrets-mkdocs:latest
MKDOCS ?= mkdocs
MIKE ?= mike
DOCS_VERSION ?= main
DOCS_ALIAS ?= unstable
SERVE_BIND_ADDRESS ?= 127.0.0.1

# TOP is the current directory where this Makefile lives.
TOP := $(dir $(firstword $(MAKEFILE_LIST)))
# ROOT is the repository root.
ROOT := $(abspath $(TOP))/../../
SRCDIR := $(ROOT)/docs
CFGFILE := $(ROOT)/hack/api-docs/mkdocs.yml
# GENROOT is the root of the generated documentation.
GENROOT := $(ROOT)/site

# Grab the uid/gid to fix permissions due to running in a docker container.
GID := $(shell id -g)
UID := $(shell id -u)

# SOURCES is a list of a source files used to generate the documentation.
SOURCES := $(shell find $(SRCDIR) -name \*.md)
SOURCES += mkdocs.yml Makefile

# entrypoint
all: build

.PHONY: image
image:
	$(DOCKER) build -t $(MKDOCS_IMAGE) -f Dockerfile .

.PHONY: build
build: image generate $(SOURCES)
	mkdir -p $(GENROOT)
	$(DOCKER) run \
	    --mount type=bind,source=$(ROOT),target=/repo \
		--sig-proxy=true \
		--rm \
		--user $(UID):$(GID) \
		--env "GIT_COMMITTER_NAME=$(shell git config user.name)" \
		--env "GIT_COMMITTER_EMAIL=$(shell git config user.email)" \
		$(MKDOCS_IMAGE) \
		/bin/bash -c "cd /repo && $(MIKE) deploy --ignore --update-aliases -F hack/api-docs/mkdocs.yml $(DOCS_VERSION) $(DOCS_ALIAS);"
.PHONY: build.publish
build.publish: image generate $(SOURCES)
	mkdir -p $(GENROOT)
	$(DOCKER) run \
	    --mount type=bind,source=$(ROOT),target=/repo \
		--sig-proxy=true \
		--rm \
		--user $(UID):$(GID) \
		$(MKDOCS_IMAGE) \
		/bin/bash -c "cd /repo && $(MIKE) deploy --update-aliases -p -F hack/api-docs/mkdocs.yml $(DOCS_VERSION) $(DOCS_ALIAS);"
.PHONY: generate
generate:
	./generate.sh $(SRCDIR)/api/spec.md

.PHONY: clean
clean:
	rm -r $(GENROOT)/* || true

# serve runs mkdocs as a local webserver for interactive development.
# This will serve the live copy of the docs on 127.0.0.1:8000.
.PHONY: serve
serve: build
	$(DOCKER) run \
		-it \
		--sig-proxy=true \
	    --mount type=bind,source=$(ROOT),target=/repo \
		--user $(UID):$(GID) \
		-p $(SERVE_BIND_ADDRESS):8000:8000 \
		--rm \
		$(MKDOCS_IMAGE) \
		/bin/bash -c "cd /repo && mkdocs serve -f hack/api-docs/mkdocs.yml -a 0.0.0.0:8000"
