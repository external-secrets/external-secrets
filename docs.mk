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

# Build rules for the documentation
#
# `make help` for a summary of top-level targets.

DOCKER ?= docker
MKDOCS_IMAGE ?= github.com/external-secrets-mkdocs:latest
MKDOCS ?= mkdocs
SERVE_BIND_ADDRESS ?= 127.0.0.1

# TOP is the current directory where this Makefile lives.
TOP := $(dir $(firstword $(MAKEFILE_LIST)))
# DOCROOT is the root of the mkdocs tree.
DOCROOT := $(abspath $(TOP))
# GENROOT is the root of the generated documentation.
GENROOT := $(DOCROOT)/docs

# Grab the uid/gid to fix permissions due to running in a docker container.
GID := $(shell id -g)
UID := $(shell id -u)

# SOURCES is a list of a source files used to generate the documentation.
SOURCES := $(shell find $(DOCROOT)/docs-src -name \*.md)
SOURCES += mkdocs.yml docs.mk

# entrypoint
all: .mkdocs.timestamp

# Support docker images.
.PHONY: images
images: .mkdocs.dockerfile.timestamp

# build the image for mkdocs
.mkdocs.dockerfile.timestamp: mkdocs.dockerfile
	docker build -t $(MKDOCS_IMAGE) -f mkdocs.dockerfile .
	date > $@

# generate the `docs` from SOURCES
.mkdocs.timestamp: images $(SOURCES)
	$(DOCKER) run \
		--mount type=bind,source=$(DOCROOT),target=/d \
		--sig-proxy=true \
		--rm \
		$(MKDOCS_IMAGE) \
		/bin/bash -c "cd /d && $(MKDOCS) build; find /d/docs -exec chown $(UID):$(GID) {} \;"
	# Remove sitemap as it contains the timestamp, which is a source of a lot
	# of merge conflicts.
	rm docs/sitemap.xml docs/sitemap.xml.gz
	date > $@

.PHONY: verify
# verify that the docs have been properly updated.
#
# docs-src/.mkdocs-exclude contains a list of `diff -X` files to
# exclude from the verification diff.
verify: images
	$(DOCKER) run \
		--mount type=bind,source=$(DOCROOT),target=/d \
		--sig-proxy=true \
		--rm \
		$(MKDOCS_IMAGE) \
		/bin/bash -c "cd /d && $(MKDOCS) build --site-dir=/tmp/d && diff -X /d/docs-src/.mkdocs-exclude -qr /d/docs /tmp/d"

# clean deletes generated files
.PHONY: clean
clean:
	test -f .mkdocs.timestamp && rm .mkdocs.timestamp || true
	test -f .mkdocs.dockerfile.timestamp && rm .mkdocs.dockerfile.timestamp || true
	rm -r $(GENROOT)/* || true

# serve runs mkdocs as a local webserver for interactive development.
# This will serve the live copy of the docs on 127.0.0.1:8000.
.PHONY: images serve
serve:
	$(DOCKER) run \
		-it \
		--sig-proxy=true \
		--mount type=bind,source=$(DOCROOT),target=/d \
		-p $(SERVE_BIND_ADDRESS):8000:8000 \
		--rm \
		$(MKDOCS_IMAGE) \
		/bin/bash -c "cd /d && $(MKDOCS) serve -a 0.0.0.0:8000"

# help prints usage for this Makefile.
.PHONY: help
help:
	@echo "Usage:"
	@echo ""
	@echo "make        Build the documentation"
	@echo "make help   Print this help message"
	@echo "make clean  Delete generated files"
	@echo "make serve  Run the webserver for live editing (ctrl-C to quit)"

# init creates a new mkdocs template. This is included for completeness.
.PHONY: images init
init:
	$(DOCKER) run \
		--mount type=bind,source=$(DOCROOT),target=/d \
		--sig-proxy=true \
		--rm \
		$(MKDOCS_IMAGE) \
		/bin/bash -c "$(MKDOCS) new d; find /d -exec chown $(UID):$(GID) {} \;"
