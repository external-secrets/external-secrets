#!/usr/bin/env bash
set -euo pipefail

# update-deps.sh: Update Go module dependencies across all modules in the repository
#
# This script updates dependencies for all Go modules in the external-secrets multi-module repository:
# - Root module
# - APIs module
# - Runtime module
# - E2E module
# - All provider modules (providers/v1/*)
# - All generator modules (generators/v1/*)
#
# This script is used by the automated update-deps GitHub workflow, which:
# 1. Runs this script to update all dependencies
# 2. Runs `make check-diff` to regenerate any auto-generated files
# 3. Creates a PR if there are any changes
#
# Note: Some dependency updates may fail due to constraints - this is expected and
# the script will continue processing all modules.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors for output
BLUE='\033[34m'
GREEN='\033[32m'
YELLOW='\033[33m'
RED='\033[31m'
NC='\033[0m' # No Color

info() {
    echo -e "${BLUE}[INFO]${NC} $*"
}

success() {
    echo -e "${GREEN}[OK]${NC} $*"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $*"
}

error() {
    echo -e "${RED}[ERROR]${NC} $*"
}

# Update a single module's dependencies
update_module() {
    local module_path="$1"
    local module_name="$2"
    
    info "Updating dependencies for $module_name..."
    
    cd "$REPO_ROOT/$module_path"
    
    # Run go get -u to update dependencies
    # Some updates may fail due to dependency constraints - this is expected
    if go get -u 2>&1; then
        success "Updated dependencies for $module_name"
    else
        warn "Failed to update some dependencies for $module_name (continuing...)"
    fi
    
    # Run go mod tidy to clean up
    if go mod tidy 2>&1; then
        success "Tidied $module_name"
    else
        warn "Failed to tidy $module_name (continuing...)"
        return 1
    fi
    
    cd "$REPO_ROOT"
}

main() {
    info "Starting dependency update for all modules..."
    echo ""
    
    # Track failures
    failed_modules=()
    
    # 1. Update root module
    if ! update_module "." "root"; then
        failed_modules+=("root")
    fi
    echo ""
    
    # 2. Update APIs module
    if ! update_module "apis" "apis"; then
        failed_modules+=("apis")
    fi
    echo ""
    
    # 3. Update runtime module
    if ! update_module "runtime" "runtime"; then
        failed_modules+=("runtime")
    fi
    echo ""
    
    # 4. Update e2e module
    if ! update_module "e2e" "e2e"; then
        failed_modules+=("e2e")
    fi
    echo ""
    
    # 5. Update all provider modules
    info "Updating provider modules..."
    for provider_dir in "$REPO_ROOT"/providers/v1/*/; do
        if [ -f "$provider_dir/go.mod" ]; then
            provider_name=$(basename "$provider_dir")
            relative_path="providers/v1/$provider_name"
            if ! update_module "$relative_path" "provider/$provider_name"; then
                failed_modules+=("provider/$provider_name")
            fi
        fi
    done
    echo ""
    
    # 6. Update all generator modules
    info "Updating generator modules..."
    for generator_dir in "$REPO_ROOT"/generators/v1/*/; do
        if [ -f "$generator_dir/go.mod" ]; then
            generator_name=$(basename "$generator_dir")
            relative_path="generators/v1/$generator_name"
            if ! update_module "$relative_path" "generator/$generator_name"; then
                failed_modules+=("generator/$generator_name")
            fi
        fi
    done
    echo ""
    
    # Summary
    echo "=================================================="
    if [ ${#failed_modules[@]} -eq 0 ]; then
        success "All modules updated successfully!"
    else
        warn "Some modules encountered issues during update:"
        for module in "${failed_modules[@]}"; do
            echo "  - $module"
        done
        info "This may be expected due to dependency constraints."
    fi
    
    # Always return success - the workflow will check for changes with check-diff
    # Failures here are often expected and shouldn't block the update process
    return 0
}

# Run main function
main

