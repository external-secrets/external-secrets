#!/bin/bash

## Helper tool to check contribution sizes. Not used in CI

# Script to count lines from git show output for files matching a path pattern
# Usage: ./count-lines-by-path.sh [-r] [-e exclude-pattern] <commit> <path-pattern>
# Example: ./count-lines-by-path.sh HEAD "*.go"
# Example: ./count-lines-by-path.sh abc123 "pkg/federation/*"
# Example: ./count-lines-by-path.sh -r HEAD ".*workflow.*\.go$" (regex mode)

set -e

USE_REGEX=false
EXCLUDE_PATTERN=""

# Parse flags
while getopts "re:" opt; do
    case $opt in
        r)
            USE_REGEX=true
            ;;
        e)
            EXCLUDE_PATTERN="$OPTARG"
            ;;
        \?)
            echo "Invalid option: -$OPTARG" >&2
            exit 1
            ;;
    esac
done
shift $((OPTIND-1))

if [ "$#" -lt 2 ]; then
    echo "Usage: $0 [-r] [-e exclude-pattern] <commit> <path-pattern>"
    echo ""
    echo "Options:"
    echo "  -r                  Use regex pattern instead of glob pattern"
    echo "  -e exclude-pattern  Exclude files matching this pattern (regex)"
    echo ""
    echo "Examples:"
    echo "  $0 HEAD '*.go'                                    # All Go files (glob)"
    echo "  $0 abc123 'pkg/*'                                 # All files in pkg/ (glob)"
    echo "  $0 -r HEAD '.*workflow.*\.go$'                    # Regex pattern"
    echo "  $0 -r -e 'zz_generated' HEAD '.*workflow.*\.go$'  # Exclude zz_generated files"
    exit 1
fi

COMMIT="$1"
PATTERN="$2"

echo "Counting lines in commit: $COMMIT"
echo "Path pattern: $PATTERN"
if [ "$USE_REGEX" = true ]; then
    echo "Mode: regex"
else
    echo "Mode: glob"
fi
if [ -n "$EXCLUDE_PATTERN" ]; then
    echo "Exclude pattern: $EXCLUDE_PATTERN"
fi
echo ""

# Get the list of files changed in the commit that match the pattern
if [ "$USE_REGEX" = true ]; then
    FILES=$(git show --name-only --pretty=format: "$COMMIT" | grep -E "$PATTERN" || true)
else
    # Convert glob to regex: * -> .*, ? -> .
    REGEX_PATTERN="${PATTERN//\*/.*}"
    REGEX_PATTERN="${REGEX_PATTERN//\?/.}"
    FILES=$(git show --name-only --pretty=format: "$COMMIT" | grep -E "^${REGEX_PATTERN}$" || true)
fi

# Apply exclude pattern if specified
if [ -n "$EXCLUDE_PATTERN" ] && [ -n "$FILES" ]; then
    FILES=$(echo "$FILES" | grep -v -E "$EXCLUDE_PATTERN" || true)
fi

if [ -z "$FILES" ]; then
    echo "No files matching pattern '$PATTERN' found in commit $COMMIT"
    exit 0
fi

echo "Matching files:"
echo "$FILES"
echo ""

# Count added and removed lines for matching files
ADDED=0
REMOVED=0
MAX_FILENAME_LEN=0

# First pass: calculate totals and find longest filename
declare -A FILE_STATS
while IFS= read -r file; do
    if [ -n "$file" ]; then
        # Get the diff stats for this specific file
        STATS=$(git show "$COMMIT" --numstat -- "$file" 2>/dev/null | grep -E '^[0-9-]+[[:space:]]+[0-9-]+[[:space:]]' | head -n 1 || echo "0 0")
        FILE_ADDED=$(echo "$STATS" | awk '{print $1}')
        FILE_REMOVED=$(echo "$STATS" | awk '{print $2}')
        
        # Handle binary files (shown as -) or empty values
        if [ "$FILE_ADDED" = "-" ] || [ -z "$FILE_ADDED" ]; then
            FILE_ADDED=0
        fi
        if [ "$FILE_REMOVED" = "-" ] || [ -z "$FILE_REMOVED" ]; then
            FILE_REMOVED=0
        fi
        
        FILE_STATS["$file"]="$FILE_ADDED $FILE_REMOVED"
        ADDED=$((ADDED + FILE_ADDED))
        REMOVED=$((REMOVED + FILE_REMOVED))
        
        # Track longest filename
        FILE_LEN=${#file}
        if [ $FILE_LEN -gt $MAX_FILENAME_LEN ]; then
            MAX_FILENAME_LEN=$FILE_LEN
        fi
    fi
done <<< "$FILES"

TOTAL=$((ADDED + REMOVED))

# Print diff-style output
while IFS= read -r file; do
    if [ -n "$file" ]; then
        read FILE_ADDED FILE_REMOVED <<< "${FILE_STATS[$file]}"
        FILE_TOTAL=$((FILE_ADDED + FILE_REMOVED))
        
        # Calculate bar length (max 50 chars)
        if [ $TOTAL -gt 0 ]; then
            BAR_LEN=$((FILE_TOTAL * 50 / TOTAL))
            if [ $BAR_LEN -eq 0 ] && [ $FILE_TOTAL -gt 0 ]; then
                BAR_LEN=1
            fi
        else
            BAR_LEN=0
        fi
        
        # Create the bar with + and -
        BAR=""
        if [ $FILE_TOTAL -gt 0 ]; then
            PLUS_LEN=$((FILE_ADDED * BAR_LEN / FILE_TOTAL))
            MINUS_LEN=$((BAR_LEN - PLUS_LEN))
            BAR=$(printf '+%.0s' $(seq 1 $PLUS_LEN))$(printf -- '-%.0s' $(seq 1 $MINUS_LEN))
        fi
        
        printf " %-${MAX_FILENAME_LEN}s | %5d %s\n" "$file" "$FILE_TOTAL" "$BAR"
    fi
done <<< "$FILES"

# Print summary line
printf " %d files changed, %d insertions(+), %d deletions(-)\n" "$(echo "$FILES" | grep -c .)" "$ADDED" "$REMOVED"
