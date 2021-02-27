# Copyright 2019 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

DOCKER ?= docker
MKDOCS_IMAGE ?= github.com/external-secrets-mkdocs:latest
MKDOCS ?= mkdocs
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
		--mount type=bind,source=$(SRCDIR),target=/docs \
		--mount type=bind,source=$(CFGFILE),target=/mkdocs.yml \
		--mount type=bind,source=$(GENROOT),target=/site \
		--sig-proxy=true \
		--rm \
		$(MKDOCS_IMAGE) \
		/bin/bash -c "cd / && $(MKDOCS) build; find /site -exec chown $(UID):$(GID) {} \;"

.PHONY: generate
generate:
	./generate.sh $(SRCDIR)/spec.md

.PHONY: clean
clean:
	rm -r $(GENROOT)/* || true

# serve runs mkdocs as a local webserver for interactive development.
# This will serve the live copy of the docs on 127.0.0.1:8000.
.PHONY: serve
serve:
	$(DOCKER) run \
		-it \
		--sig-proxy=true \
		--mount type=bind,source=$(SRCDIR),target=/docs \
		--mount type=bind,source=$(CFGFILE),target=/mkdocs.yml \
		--mount type=bind,source=$(GENROOT),target=/site \
		-p $(SERVE_BIND_ADDRESS):8000:8000 \
		--rm \
		$(MKDOCS_IMAGE) \
		/bin/bash -c "cd / && $(MKDOCS) serve -a 0.0.0.0:8000"
