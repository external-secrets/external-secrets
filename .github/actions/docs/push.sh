#!/bin/bash

# Copyright 2020 The Kubernetes Authors.
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
set -e
remote_repo="https://${GITHUB_ACTOR}:${GITHUB_TOKEN}@github.com/${GITHUB_REPOSITORY}.git"
git config --global user.name "$GITHUB_ACTOR"
git config --global user.email "$GITHUB_ACTOR@users.noreply.github.com"

# clone gh pages, save helm index
git clone --branch=gh-pages --depth=1 "${remote_repo}" gh-pages
cd gh-pages
temp_worktree=$(mktemp -d)
if [ -f index.yaml ]; then
  cp --force "index.yaml" "$temp_worktree/index.yaml"
fi
git rm -r .

# copy new page content, restore helm index, add cname
cp -r ../site/* .
if [ -f $temp_worktree/index.yaml ]; then
  cp "$temp_worktree/index.yaml" .
fi
echo "${CNAME}" > CNAME

# commit & push
git add .
git commit -m "Deploy GitHub Pages"
git push --force "${remote_repo}" gh-pages
