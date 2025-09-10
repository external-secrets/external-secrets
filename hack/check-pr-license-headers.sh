#!/bin/bash

# Copyright © 2022 ESO Maintainer Team
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

set -euo pipefail

REPO_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)

# Function to get files added in the current PR
get_added_files() {
    # Get the base branch (usually main)
    local base_ref="${1:-origin/main}"
    
    # Check if the base reference exists
    if ! git rev-parse --verify "$base_ref" >/dev/null 2>&1; then
        # Try FETCH_HEAD if origin/main doesn't exist
        if git rev-parse --verify FETCH_HEAD >/dev/null 2>&1; then
            base_ref="FETCH_HEAD"
        else
            echo "Warning: Cannot find base reference. No files to check."
            return 0
        fi
    fi
    
    # Get files that have been added (A) in this PR
    git diff --name-only --diff-filter=A "$base_ref"...HEAD 2>/dev/null || echo ""
}

# Function to check if a file needs license header based on extension
needs_license_check() {
    local file="$1"
    local extension="${file##*.}"
    
    case "$extension" in
        go)
            return 0
            ;;
        *)
            return 1
            ;;
    esac
}

# Function to check license header for a specific file
check_license_header() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        echo "File not found: $file"
        return 1
    fi
    
    # Read the first 15 lines of the file to check for license
    local file_header
    file_header=$(head -n 15 "$file" 2>/dev/null || echo "")
    
    # Check for Apache License text
    if echo "$file_header" | grep -q "Licensed under the Apache License, Version 2.0"; then
        return 0
    else
        return 1
    fi
}

# Main function
main() {
    local base_ref="${1:-origin/main}"
    local exit_code=0
    
    echo "Checking license headers for files added in PR (compared to $base_ref)..."
    
    # Get all added files
    local added_files
    readarray -t added_files < <(get_added_files "$base_ref")
    
    if [[ ${#added_files[@]} -eq 0 ]]; then
        echo "No files added in this PR."
        exit 0
    fi
    
    local files_to_check=()
    local skipped_files=()
    
    # Filter files that need license check
    for file in "${added_files[@]}"; do
        if [[ -n "$file" ]]; then
            if needs_license_check "$file"; then
                files_to_check+=("$file")
            else
                skipped_files+=("$file")
            fi
        fi
    done
    
    echo "Added files in PR: ${#added_files[@]}"
    echo "Files requiring license check: ${#files_to_check[@]}"
    echo "Files skipped (no license required): ${#skipped_files[@]}"
    echo ""
    
    # Check license headers for relevant files
    local failed_files=()
    for file in "${files_to_check[@]}"; do
        if check_license_header "$file"; then
            echo "✅ $file"
        else
            echo "❌ $file"
            failed_files+=("$file")
            exit_code=1
        fi
    done
    
    # Print summary
    echo ""
    if [[ $exit_code -eq 0 ]]; then
        echo "✅ All added files have correct license headers!"
    else
        echo "❌ ${#failed_files[@]} added file(s) are missing license headers:"
        printf '  %s\n' "${failed_files[@]}"
        echo ""
        echo "Please add the Apache License 2.0 header to these files."
        echo "You can use the template from hack/boilerplate.go.txt"
    fi
    
    exit $exit_code
}

main "$@"