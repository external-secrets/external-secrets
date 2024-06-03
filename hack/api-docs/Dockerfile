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

FROM alpine:3.20@sha256:77726ef6b57ddf65bb551896826ec38bc3e53f75cdde31354fbffb4f25238ebd
RUN apk add -U --no-cache \
    python3 \
    python3-dev \
    py3-pip \
    musl-dev \
    git \
    openssh \
    git-fast-import \
    bash \
    gcc \
    diffutils

ENV PATH=$PATH:/.venv/bin
COPY requirements.txt /
RUN python3 -m venv .venv && \
    source .venv/bin/activate && \
    pip3 install --upgrade pip && \
    pip3 install -r /requirements.txt

# Disable the top-level directory owner check
# https://github.com/git/git/commit/8959555cee7ec045958f9b6dd62e541affb7e7d9
RUN git config --system --add safe.directory '*'
