# Copyright 2019 The Kubernetes Authors.
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

# Pinned release: alpine:3.24.3 (Alpine; tracking 3.x)
FROM alpine@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a
RUN apk add -U --no-cache \
    bash \
    diffutils \
    gcc \
    git \
    git-fast-import \
    musl-dev \
    openssh \
    py3-pip \
    python3 \
    python3-dev

ENV PATH=$PATH:/.venv/bin
COPY build-requirements.txt requirements.txt /
RUN python3 -m venv .venv && \
    source .venv/bin/activate && \
    pip3 install --require-hashes --only-binary=:all: -r /build-requirements.txt && \
    pip3 install --require-hashes --only-binary=:all: --no-binary=watchdog --no-build-isolation -r /requirements.txt

# Disable the top-level directory owner check
# https://github.com/git/git/commit/8959555cee7ec045958f9b6dd62e541affb7e7d9
RUN git config --system --add safe.directory '*'

# Keep the image non-root by default; the docs Makefile overrides the UID/GID
# with the host user's IDs when bind-mounting the repository.
ENV HOME=/tmp
USER 1000:1000
