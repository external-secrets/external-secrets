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
BOILERPLATE_FILE="${REPO_ROOT}/hack/boilerplate.go.txt"

# Function to check if a file has the correct license header
check_license_header() {
    local file="$1"
    local extension="${file##*.}"
    
    # Only check Go files for now (can be extended for other file types)
    case "$extension" in
        go)
            # Extract the expected license text (without the /* */ wrapper)
            local expected_license
            expected_license=$(sed '1d;$d' "$BOILERPLATE_FILE")
            
            # Check if file starts with license header
            local file_header
            file_header=$(head -n 15 "$file" 2>/dev/null || echo "")
            
            # Check if the license text is present in the header
            if echo "$file_header" | grep -q "Licensed under the Apache License, Version 2.0"; then
                return 0
            else
                return 1
            fi
            ;;
        *)
            # Skip non-Go files for now
            return 0
            ;;
    esac
}

# Main function to check license headers
main() {
    local exit_code=0
    local files_to_check=()
    
    # If called with arguments, check those specific files
    if [[ $# -gt 0 ]]; then
        files_to_check=("$@")
    else
        # Otherwise, check all Go files
        while IFS= read -r -d '' file; do
            files_to_check+=("$file")
        done < <(find "$REPO_ROOT" -name "*.go" -not -path "*/vendor/*" -not -path "*/.git/*" -print0)
    fi
    
    echo "Checking license headers for ${#files_to_check[@]} files..."
    
    local failed_files=()
    for file in "${files_to_check[@]}"; do
        if [[ -f "$file" ]]; then
            if ! check_license_header "$file"; then
                failed_files+=("$file")
                echo "❌ Missing or incorrect license header: $file"
                exit_code=1
            else
                echo "✅ License header OK: $file"
            fi
        fi
    done
    
    if [[ $exit_code -eq 0 ]]; then
        echo "✅ All files have correct license headers!"
    else
        echo ""
        echo "❌ ${#failed_files[@]} file(s) are missing or have incorrect license headers:"
        printf '%s\n' "${failed_files[@]}"
        echo ""
        echo "Please add the correct license header to these files."
        echo "You can use the template from hack/boilerplate.go.txt"
    fi
    
    exit $exit_code
}

main "$@"