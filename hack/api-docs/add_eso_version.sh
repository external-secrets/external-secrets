#!/bin/bash

# Script to add a new ESO version to the stability-support.md file
# Usage: ./add_eso_version.sh <ESO_VERSION>

# Check if all required arguments are provided
if [ $# -ne 1 ]; then
    echo "Usage: $0 <ESO_VERSION>"
    echo "Example: $0 '0.17.x'"
    exit 1
fi

# Assign parameters to variables
ESO_VERSION="$1"
K8S_VERSION="$(echo 1.$(cat $ROOT/go.mod | grep 'k8s.io/client-go' | cut -d'v' -f2 | cut -d'.' -f2))"
RELEASE_DATE=$(date +%B\ %d,\ %Y)



# Path to the stability-support.md file
FILE_PATH="$ROOT/docs/introduction/stability-support.md"

# Check if the file exists
if [ ! -f "$FILE_PATH" ]; then
    echo "Error: File $FILE_PATH does not exist."
    exit 1
fi

echo "Checking for version: $ESO_VERSION"
current=$(cat $ROOT/docs/introduction/stability-support.md | grep "$ESO_VERSION") || true
if [[ $current != "" ]]; then
		echo "Version already exists. Nothing to do"
        exit 0
fi

# Set End of Life to "Release of next version"
END_OF_LIFE="Release of $(echo $ESO_VERSION | awk -F. '{print $1"."$2+1}')"

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
/\| ESO Version \| Kubernetes Version \| Release Date \| End of Life/ { 
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
    # Extract the first three columns (up to and including Release Date)
    # We need to capture the version, k8s version, and release date
    match($0, /^(\| [^|]+ \| [^|]+ \| [^|]+ \|)/, parts);
    first_part = parts[1];
    
    # Construct the updated line with the first three columns and the new End of Life
    updated_line = first_part " " release_date "    |";
    print updated_line;
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
