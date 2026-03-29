#!/bin/bash
set -e

# fetch_update.sh - Manage mod updates based on GitHub repository rules

# Default settings
COMMAND=""
TARGET_RULE=""
DB_URL="${DATABASE_URL:-}"
DRY_RUN=false
VERBOSE=false
BUILD=true
INTERACTIVE=false
BUILD_SOURCE="module"
BUILD_VERSION="latest"

usage() {
    echo "Usage: $0 [command] [options]"
    echo ""
    echo "Commands:"
    echo "  check      Check for available updates without applying them"
    echo "  update     Update mods to the latest version (default)"
    echo "  list       List configured rules"
    echo ""
    echo "Options:"
    echo "  --rule <file>   Target a specific rule file"
    echo "  --db <url>      Database connection string (or set DATABASE_URL)"
    echo "  --dry-run       Show what would happen without making changes"
    echo "  --no-build      Skip building tools (if already built)"
    echo "  --build-source <source> Set build source: 'local' or 'module' (default: module)"
    echo "  --build-version <ver> Set build version for module source (default: latest)"
    echo "  --interactive   Enable interactive selection of releases"
    echo "  --verbose       Enable verbose output"
    echo "  --help          Show this help message"
    exit 1
}

log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $*"
}

debug() {
    if [ "$VERBOSE" = true ]; then
        log "DEBUG: $*"
    fi
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        check)
            COMMAND="check"
            shift
            ;;
        update)
            COMMAND="update"
            shift
            ;;
        list)
            COMMAND="list"
            shift
            ;;
        --rule)
            TARGET_RULE="$2"
            shift 2
            ;;
        --db)
            DB_URL="$2"
            shift 2
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --no-build)
            BUILD=false
            shift
            ;;
        --build-source)
            BUILD_SOURCE="$2"
            shift 2
            ;;
        --build-version)
            BUILD_VERSION="$2"
            shift 2
            ;;
        --interactive)
            INTERACTIVE=true
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        --help|-h)
            usage
            ;;
        *)
            # Handle unknown options or positional args as command if not set
            if [[ "$1" == -* ]]; then
                echo "Unknown option: $1"
                usage
            else
                COMMAND="$1"
                shift
            fi
            ;;
    esac
done

# Validate environment
if [ "$COMMAND" != "list" ] && [ -z "$DB_URL" ] && [ "$DRY_RUN" = false ]; then
    echo "Error: Database URL is required for '$COMMAND' (unless --dry-run is used)."
    echo "Set DATABASE_URL environment variable or use --db option."
    exit 1
fi

# Export DB_URL for subprocesses
if [ -n "$DB_URL" ]; then
    export DATABASE_URL="$DB_URL"
fi

# Move to project root
cd "$(dirname "$0")/.." || exit 1

# Ensure jq is installed
if ! command -v jq &> /dev/null; then
    echo "Error: jq is required but not installed."
    exit 1
fi

# Build tools
if [ "$BUILD" = true ] && [ "$COMMAND" != "list" ]; then
    log "Building tools (Source: $BUILD_SOURCE)..."
    mkdir -p bin
    
    if [ "$BUILD_SOURCE" == "module" ]; then
        # Use go install with GOBIN for module source
        # shellcheck disable=SC2155
        export GOBIN="$(pwd)/bin"
        go install "github.com/ikafly144/au_mod_installer/cmd/fetch-gh-release@$BUILD_VERSION"
        go install "github.com/ikafly144/au_mod_installer/cmd/mus-mgr@$BUILD_VERSION"
    else
        # Use go build with local path
        go build -o bin/fetch-gh-release ./cmd/fetch-gh-release
        go build -o bin/mus-mgr ./cmd/mus-mgr
    fi
fi

process_rule() {
    local rule_file="$1"
    local mode="$2" # check or update

    if [ ! -f "$rule_file" ]; then
        log "Warning: Rule file '$rule_file' not found."
        return
    fi

    log "Processing $rule_file..."

    # Interactive Release Selection (only for update/check modes when specific rule is targeted or interactive mode)
    local target_tag=""
    if [ "$INTERACTIVE" = true ]; then
        log "Fetching available releases..."
        local releases_json
        if ! releases_json=$(./bin/fetch-gh-release -rule "$rule_file" -list); then
             log "Error: Failed to fetch releases list"
             return
        fi
        
        # Parse releases
        local tags=()
        local names=()
        while IFS= read -r line; do
            tags+=("$line")
        done < <(echo "$releases_json" | jq -r '.[].tag_name')
        
        while IFS= read -r line; do
            names+=("$line")
        done < <(echo "$releases_json" | jq -r '.[].name')

        if [ ${#tags[@]} -eq 0 ]; then
            log "No releases found."
            return
        fi

        echo "Available Releases:"
        for i in "${!tags[@]}"; do
            echo "$((i+1)). ${tags[$i]} (${names[$i]})"
        done
        
        read -p "Select release (default: 1): " selection
        selection=${selection:-1}
        
        if [[ "$selection" =~ ^[0-9]+$ ]] && [ "$selection" -ge 1 ] && [ "$selection" -le "${#tags[@]}" ]; then
            target_tag="${tags[$((selection-1))]}"
            log "Selected release: $target_tag"
        else
            log "Invalid selection, using latest: ${tags[0]}"
            target_tag="${tags[0]}"
        fi
    fi

    # 1. Fetch release info
    debug "Fetching release info for $rule_file (tag: ${target_tag:-latest})..."
    local output
    local fetch_args=(-rule "$rule_file")
    if [ -n "$target_tag" ]; then
        fetch_args+=(-tag "$target_tag")
    fi
    
    if ! output=$(./bin/fetch-gh-release "${fetch_args[@]}"); then
        log "Error: Failed to fetch release info for $rule_file"
        return
    fi
    
    if [ -z "$output" ]; then
        log "Error: No output from fetch-gh-release"
        return
    fi

    # 2. Parse fetched info
    local mod_id
    local version_id
    mod_id=$(echo "$output" | jq -r '.mod_id')
    version_id=$(echo "$output" | jq -r '.version_id')

    if [ "$mod_id" == "null" ] || [ -z "$mod_id" ]; then
        log "Error: Invalid mod_id in output for $rule_file"
        return
    fi

    debug "Latest version for $mod_id is $version_id"

    # 3. Check if version exists in DB
    local version_exists=false
    if [ -n "$DB_URL" ]; then
        if ./bin/mus-mgr version list "$mod_id" 2>/dev/null | grep -q "^$version_id$"; then
            version_exists=true
        fi
    fi

    if [ "$version_exists" = true ]; then
        log "  $mod_id: Version $version_id is already registered."
        return
    else
        log "  $mod_id: New version available: $version_id"
    fi

    # 4. Perform Action
    if [ "$mode" == "check" ] || [ "$DRY_RUN" = true ]; then
        log "  [Dry Run] Would register version $version_id"
        return
    fi

    if [ "$mode" == "update" ]; then
        log "  Registering version $version_id..."
        
        # Construct arguments
        local args=("--mod-id" "$mod_id" "--version-id" "$version_id" "--set-latest")
        
        # Add files
        # Use a temporary file to handle while loop input safely
        echo "$output" | jq -r '.files[]? // empty' > /tmp/mus-mgr-files.tmp
        while IFS= read -r file; do
            if [ -n "$file" ] && [ "$file" != "null" ]; then
                args+=("--file" "$file")
            fi
        done < /tmp/mus-mgr-files.tmp
        rm /tmp/mus-mgr-files.tmp

        # Add dependencies
        echo "$output" | jq -r '.dependencies[]? // empty' > /tmp/mus-mgr-deps.tmp
        while IFS= read -r dep; do
            if [ -n "$dep" ] && [ "$dep" != "null" ]; then
                args+=("--dependency" "$dep")
            fi
        done < /tmp/mus-mgr-deps.tmp
        rm /tmp/mus-mgr-deps.tmp

        # Add features
        echo "$output" | jq -r '.features[]? // empty' > /tmp/mus-mgr-features.tmp
        while IFS= read -r feature; do
            if [ -n "$feature" ] && [ "$feature" != "null" ]; then
                args+=("--feature" "$feature")
            fi
        done < /tmp/mus-mgr-features.tmp
        rm /tmp/mus-mgr-features.tmp
        
        # Execute
        if ./bin/mus-mgr version add "${args[@]}"; then
            log "  Successfully registered $mod_id version $version_id"
        else
            log "  Error: Failed to register version"
        fi
    fi
}

list_rules() {
    echo "Configured Rules:"
    echo "-----------------"
    printf "%-30s %-20s %s\n" "Rule File" "Mod ID" "GitHub Repo"
    
    for rule_file in rules/*.rule.json; do
        if [ -f "$rule_file" ]; then
            local mod_id
            local repo
            mod_id=$(jq -r '.mod_id // "unknown"' "$rule_file")
            repo=$(jq -r '.github_repo // "unknown"' "$rule_file")
            printf "%-30s %-20s %s\n" "$(basename "$rule_file")" "$mod_id" "$repo"
        fi
    done
}

# Main Execution Logic
case $COMMAND in
    list)
        list_rules
        ;;
    check|update)
        if [ -n "$TARGET_RULE" ]; then
            process_rule "$TARGET_RULE" "$COMMAND"
        else
            # If interactive and no rule specified, ask user to select a rule
            if [ "$INTERACTIVE" = true ]; then
                # Gather available rules
                rules=(rules/*.rule.json)
                if [ ${#rules[@]} -eq 0 ] || [ ! -e "${rules[0]}" ]; then
                    echo "No rules found in rules/ directory."
                    exit 1
                fi
                
                echo "Available Rules:"
                for i in "${!rules[@]}"; do
                    # Nice display name
                    mod_id=$(jq -r '.mod_id // "unknown"' "${rules[$i]}")
                    filename=$(basename "${rules[$i]}")
                    echo "$((i+1)). $filename ($mod_id)"
                done
                
                read -p "Select rule (0 for all): " selection
                if [ "$selection" = "0" ]; then
                    for rule_file in "${rules[@]}"; do
                        process_rule "$rule_file" "$COMMAND"
                        echo "----------------------------------------"
                    done
                elif [[ "$selection" =~ ^[0-9]+$ ]] && [ "$selection" -ge 1 ] && [ "$selection" -le "${#rules[@]}" ]; then
                     process_rule "${rules[$((selection-1))]}" "$COMMAND"
                else
                    echo "Invalid selection."
                    exit 1
                fi
            else
                # Process all rules
                for rule_file in rules/*.rule.json; do
                    if [ -f "$rule_file" ]; then
                        process_rule "$rule_file" "$COMMAND"
                        echo "----------------------------------------"
                    fi
                done
            fi
        fi
        ;;
    *)
        usage
        ;;
esac

log "Operation completed."
