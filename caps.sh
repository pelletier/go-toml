#!/usr/bin/env bash
#
# Generates or checks the capability baseline for go-toml.
#
# Usage:
#   ./caps.sh generate   # regenerate capability_baseline.txt
#   ./caps.sh check      # check that capabilities haven't grown
#
# Requires: go, capslock (go install github.com/google/capslock/cmd/capslock@latest)

set -euo pipefail

BASELINE="capability_baseline.txt"
CAPSLOCK="${CAPSLOCK:-capslock}"

generate() {
    "$CAPSLOCK" -packages=./... -output=package -granularity=package \
        | jq -r 'to_entries | sort_by(.key) | .[] | .key + ": " + (.value | sort | join(", "))' \
        > "$BASELINE"
    echo "Wrote $BASELINE"
}

check() {
    if [ ! -f "$BASELINE" ]; then
        echo "ERROR: $BASELINE not found. Run '$0 generate' first."
        exit 1
    fi

    current=$(mktemp)
    trap 'rm -f "$current"' EXIT

    "$CAPSLOCK" -packages=./... -output=package -granularity=package \
        | jq -r 'to_entries | sort_by(.key) | .[] | .key + ": " + (.value | sort | join(", "))' \
        > "$current"

    if diff -u "$BASELINE" "$current"; then
        echo "OK: capabilities unchanged."
    else
        echo ""
        echo "FAILED: capabilities have changed."
        echo "If this is intentional, run '$0 generate' and commit the updated $BASELINE."
        exit 1
    fi
}

case "${1:-}" in
    generate) generate ;;
    check)    check ;;
    *)
        echo "Usage: $0 {generate|check}"
        exit 1
        ;;
esac
