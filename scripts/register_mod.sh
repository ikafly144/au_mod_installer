#!/bin/bash
set -e

# register_mod.sh - Register a new mod from a rule file using mus-mgr

DB_URL="${DATABASE_URL:-}"
RULE_FILE=""
MOD_NAME=""
MOD_AUTHOR=""
MOD_DESC=""
BUILD_SOURCE="module"
BUILD_VERSION="latest"

DRY_RUN=false

usage() {
    echo "Usage: $0 --rule <rule_file> [options]"
    echo ""
    echo "Options:"
    echo "  --rule <file>    Path to rule file (required)"
    echo "  --name <name>    Mod name"
    echo "  --author <name>  Mod author"
    echo "  --desc <text>    Mod description"
    echo "  --thumbnail <url> Mod thumbnail URL"
    echo "  --db <url>       Database connection string"
    echo "  --dry-run        Print command without executing"
    echo "  --build-source <source> Set build source: 'local' or 'module' (default: module)"
    echo "  --build-version <ver> Set build version for module source (default: latest)"
    echo "  --help           Show this help message"
    exit 0
}

# Parse args
while [[ $# -gt 0 ]]; do
    case $1 in
        --rule)
            RULE_FILE="$2"
            shift 2
            ;;
        --name)
            MOD_NAME="$2"
            shift 2
            ;;
        --author)
            MOD_AUTHOR="$2"
            shift 2
            ;;
        --desc)
            MOD_DESC="$2"
            shift 2
            ;;
        --thumbnail)
            MOD_THUMBNAIL="$2"
            shift 2
            ;;
        --db)
            DB_URL="$2"
            shift 2
            ;;
        --build-source)
            BUILD_SOURCE="$2"
            shift 2
            ;;
        --build-version)
            BUILD_VERSION="$2"
            shift 2
            ;;
        --dry-run)
            DRY_RUN=true
            shift 1
            ;;
        --help|-h)
            usage
            ;;
        *)
            echo "Unknown option: $1"
            usage
            ;;
    esac
done

if [ -z "$RULE_FILE" ]; then
    echo "Error: Rule file is required."
    usage
fi

if [ ! -f "$RULE_FILE" ]; then
    echo "Error: Rule file '$RULE_FILE' not found."
    exit 1
fi

# Ensure jq
if ! command -v jq &> /dev/null; then
    echo "Error: jq is required."
    exit 1
fi

# Move to project root
cd "$(dirname "$0")/.." || exit 1

# Check mus-mgr
if [ ! -f "bin/mus-mgr" ]; then
    echo "Building mus-mgr (Source: $BUILD_SOURCE)..."
    mkdir -p bin
    
    if [ "$BUILD_SOURCE" == "module" ]; then
        # shellcheck disable=SC2155
        export GOBIN="$(pwd)/bin"
        go install "github.com/ikafly144/au_mod_installer/cmd/mus-mgr@$BUILD_VERSION"
    else
        go build -o bin/mus-mgr ./cmd/mus-mgr
    fi
fi

# Extract mod_id
MOD_ID=$(jq -r '.mod_id // empty' "$RULE_FILE")

if [ -z "$MOD_ID" ]; then
    echo "Error: Could not extract mod_id from $RULE_FILE"
    exit 1
fi

echo "Registering Mod ID: $MOD_ID"

# Interactive prompts if missing
if [ -z "$MOD_THUMBNAIL" ]; then
    MOD_THUMBNAIL=$(jq -r '.thumbnail_url // empty' "$RULE_FILE")
fi

if [ -z "$MOD_NAME" ]; then
    read -p "Enter Mod Name [$MOD_ID]: " input
    MOD_NAME="${input:-$MOD_ID}"
fi

if [ -z "$MOD_AUTHOR" ]; then
    # Try to guess author from github_repo if available
    GH_REPO=$(jq -r '.github_repo // empty' "$RULE_FILE")
    DEFAULT_AUTHOR=""
    if [ -n "$GH_REPO" ]; then
        DEFAULT_AUTHOR=$(echo "$GH_REPO" | cut -d'/' -f1)
    fi
    
    prompt="Enter Mod Author"
    if [ -n "$DEFAULT_AUTHOR" ]; then
        prompt="$prompt [$DEFAULT_AUTHOR]"
    fi
    read -p "$prompt: " input
    MOD_AUTHOR="${input:-$DEFAULT_AUTHOR}"
fi

if [ -z "$MOD_DESC" ]; then
    read -p "Enter Mod Description (optional): " input
    MOD_DESC="$input"
fi

if [ -z "$MOD_AUTHOR" ]; then
    echo "Error: Author is required."
    exit 1
fi

# Construct command
CMD=(./bin/mus-mgr)
if [ -n "$DB_URL" ]; then
    CMD+=(--db "$DB_URL")
fi

CMD+=(mod add --id "$MOD_ID" --name "$MOD_NAME" --author "$MOD_AUTHOR")

if [ -n "$MOD_DESC" ]; then
    CMD+=(--desc "$MOD_DESC")
fi

if [ -n "$MOD_THUMBNAIL" ]; then
    CMD+=(--thumbnail-url "$MOD_THUMBNAIL")
fi

echo "Executing: ${CMD[*]}"

if [ "$DRY_RUN" = true ]; then
    exit 0
fi

"${CMD[@]}"
