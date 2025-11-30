#!/usr/bin/env bash

# Script to add a new ESO version to the stability-support.md file
# Usage: ./add_eso_version.sh <ESO_VERSION>

# Set ROOT to the repository root (two levels up from this script)
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Check if all required arguments are provided
if [ $# -ne 1 ]; then
    echo "Usage: $0 <ESO_VERSION>"
    echo "Example: $0 '0.17.x'"
    exit 1
fi

# Assign parameters to variables
ESO_VERSION="$1"
K8S_VERSION="$(echo 1.$(cat "${ROOT}"/go.mod | grep 'k8s.io/client-go' | cut -d'v' -f2 | cut -d'.' -f2))"
RELEASE_DATE=$(date +%B\ %d,\ %Y)



# Path to the stability-support.md file
FILE_PATH="$ROOT/docs/introduction/stability-support.md"

# Check if the file exists
if [ ! -f "$FILE_PATH" ]; then
    echo "Error: File $FILE_PATH does not exist."
    exit 1
fi

echo "Checking for version: $ESO_VERSION"
current=$(cat "${ROOT}"/docs/introduction/stability-support.md | grep "$ESO_VERSION") || true
if [[ "${current}" != "" ]]; then
		echo "Version already exists. Nothing to do"
        exit 0
fi

# Set End of Life to "Release of next version"
END_OF_LIFE="Release of $(echo "${ESO_VERSION}" | awk -F. '{print $1"."$2+1}')"

# Create the new line to insert
NEW_LINE="| $ESO_VERSION      | $K8S_VERSION               | $RELEASE_DATE  | $END_OF_LIFE |"

# Create a temporary file
TEMP_FILE=$(mktemp)

# Process the file with awk to:
# 1. Add the new version after the table header
# 2. Keep the previous version's release date intact
# 3. Update only the End of Life date for the previous version
awk -v new_line="$NEW_LINE" -v release_date="$RELEASE_DATE" '
BEGIN {
    found_table = 0;
    added_new_line = 0;
    in_eso_table = 0;
    updated_first_version = 0;
}

# Match the ESO Version table specifically (to avoid affecting other tables)
/\| ESO Version \| Kubernetes Version \| Release Date[[:space:]]+\| End of Life/ {
    in_eso_table = 1;
    found_table = 1;
    print $0;
    next;
}

# Match the separator line (dashes) only in the ESO Version table
in_eso_table == 1 && /\| -+/ {
    print $0;
    print new_line;
    added_new_line = 1;
    next;
}

# For the first version line after we added our new line, update only the End of Life
added_new_line == 1 && in_eso_table == 1 && /\|.*\|.*\|.*\|/ && !updated_first_version {
    # Split the line by | and reconstruct with new End of Life (which should be the release date of the new version)
    split($0, fields, "|");
    # fields[1] is empty, fields[2] is version, fields[3] is k8s version, fields[4] is release date
    # Trim whitespace and format properly
    gsub(/^ +| +$/, "", fields[2]);
    gsub(/^ +| +$/, "", fields[3]);
    gsub(/^ +| +$/, "", fields[4]);
    printf "| %-10s | %-18s | %-14s | %s |\n", fields[2], fields[3], fields[4], release_date;
    updated_first_version = 1;
    next;
}

# Detect when we leave the ESO Version table (blank line or start of new section)
in_eso_table == 1 && (/^$/ || /^##/) {
    in_eso_table = 0;
}

# Print all other lines unchanged
{ print $0; }
' "$FILE_PATH" > "$TEMP_FILE"

# Replace the original file with the temporary file
mv "$TEMP_FILE" "$FILE_PATH"

echo "Successfully added ESO version $ESO_VERSION to $FILE_PATH"
echo "New line: $NEW_LINE"
echo "Updated previous version's End of Life to: $RELEASE_DATE"
