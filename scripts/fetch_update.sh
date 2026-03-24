#!/bin/bash

DB_URL="${1:-}"
if [ -z "$DB_URL" ] && [ -z "$DATABASE_URL" ]; then
    echo "Usage: $0 [database_url]"
    echo "Or set the DATABASE_URL environment variable."
    exit 1
fi

# Use the provided arg or fall back to the environment variable
export DATABASE_URL="${DB_URL:-$DATABASE_URL}"

# Move to the project root directory
cd "$(dirname "$0")/.." || exit 1

# Ensure jq is installed
if ! command -v jq &> /dev/null; then
    echo "jq is required but not installed. Please install jq."
    exit 1
fi

# Build the tools
echo "Building tools..."
mkdir -p bin
go build -o bin/fetch-gh-release ./cmd/fetch-gh-release
go build -o bin/mus-mgr ./cmd/mus-mgr

echo "Processing rules in rules/ directory..."

for rule_file in rules/*.rule.json; do
    if [ ! -f "$rule_file" ]; then
        continue
    fi

    echo "Processing $rule_file..."
    
    # Run fetch-gh-release to get the JSON output
    output=$(./bin/fetch-gh-release -rule "$rule_file")
    status=$?
    
    if [ $status -ne 0 ] || [ -z "$output" ]; then
        echo "Failed to fetch release for $rule_file or no output"
        continue
    fi

    # Parse the output using jq
    mod_id=$(echo "$output" | jq -r '.mod_id')
    version_id=$(echo "$output" | jq -r '.version_id')

    if [ "$mod_id" == "null" ] || [ -z "$mod_id" ]; then
        echo "Invalid mod_id in output for $rule_file"
        continue
    fi

    echo "Found release $version_id for mod $mod_id"

    # Construct arguments for mus-mgr version add
    args=("--mod-id" "$mod_id" "--version-id" "$version_id" "--set-latest")

    # Add file arguments
    while IFS= read -r file; do
        if [ -n "$file" ] && [ "$file" != "null" ]; then
            args+=("--file" "$file")
        fi
    done <<< "$(echo "$output" | jq -r '.files[]? // empty')"

    # Add dependency arguments
    while IFS= read -r dep; do
        if [ -n "$dep" ] && [ "$dep" != "null" ]; then
            args+=("--dependency" "$dep")
        fi
    done <<< "$(echo "$output" | jq -r '.dependencies[]? // empty')"

    # Run mus-mgr to register the version
    echo "Registering version $version_id for mod $mod_id..."
    ./bin/mus-mgr --db "$DATABASE_URL" version add "${args[@]}"
    
    if [ $? -eq 0 ]; then
        echo "Successfully registered $mod_id version $version_id"
    else
        echo "Failed to register $mod_id version $version_id"
    fi
    echo "----------------------------------------"
done

echo "Fetch update completed."
