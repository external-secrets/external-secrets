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

FROM alpine:3.19@sha256:51b67269f354137895d43f3b3d810bfacd3945438e94dc5ac55fdac340352f48
COPY requirements.txt /
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
RUN pip3 install --break-system-packages -r /requirements.txt
# Disable the top-level directory owner check
# https://github.com/git/git/commit/8959555cee7ec045958f9b6dd62e541affb7e7d9
RUN git config --system --add safe.directory '*'
