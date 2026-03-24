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

# Capabilities that must never appear in any package.
FORBIDDEN_CAPS=(
    CAPABILITY_NETWORK
    CAPABILITY_CGO
    CAPABILITY_EXEC
)

capslock_to_baseline() {
    "$CAPSLOCK" -packages=./... -output=package -granularity=package \
        | jq -r 'to_entries | sort_by(.key) | .[] | .key + ": " + (.value | sort | join(", "))'
}

generate() {
    capslock_to_baseline > "$BASELINE"
    echo "Wrote $BASELINE"
}

check() {
    if [ ! -f "$BASELINE" ]; then
        echo "ERROR: $BASELINE not found. Run '$0 generate' first."
        exit 1
    fi

    current=$(mktemp)
    trap 'rm -f "$current"' EXIT

    capslock_to_baseline > "$current"

    failed=0

    # Verify go-toml source never directly imports "unsafe".
    # Capslock may report CAPABILITY_UNSAFE_POINTER due to stdlib internals
    # (e.g. reflect -> unsafe), which is a false positive. Instead of relying
    # on capslock for this, we check the source directly.
    unsafe_imports=$(find . -name '*.go' -not -name '*_test.go' -not -path './vendor/*' \
        -exec grep -l '"unsafe"' {} +) || true
    if [ -n "$unsafe_imports" ]; then
        echo "FORBIDDEN: direct unsafe import found in:"
        echo "$unsafe_imports"
        failed=1
    fi

    # Check for forbidden capabilities in current output.
    for cap in "${FORBIDDEN_CAPS[@]}"; do
        if grep -q "$cap" "$current"; then
            echo "FORBIDDEN capability found: $cap"
            grep "$cap" "$current"
            failed=1
        fi
    done

    # Extract all unique capability names from baseline and current.
    # Exclude CAPABILITY_UNSAFE_POINTER from comparison — capslock reports it
    # as a false positive from stdlib internals (reflect, sync, etc. use
    # unsafe.Pointer internally). Go 1.26+ triggers this due to changes in
    # how capslock traces through unclassified reflect functions. The direct
    # source check above is the real guard against unsafe usage.
    baseline_caps=$(grep -oE 'CAPABILITY_[A-Z_]+' "$BASELINE" | grep -v CAPABILITY_UNSAFE_POINTER | sort -u)
    current_caps=$(grep -oE 'CAPABILITY_[A-Z_]+' "$current" | grep -v CAPABILITY_UNSAFE_POINTER | sort -u)

    # Check for new capability names not in the baseline.
    new_caps=$(comm -13 <(echo "$baseline_caps") <(echo "$current_caps"))
    if [ -n "$new_caps" ]; then
        echo "NEW capabilities detected (not in baseline):"
        echo "$new_caps"
        failed=1
    fi

    # Check for new per-package capabilities (a package gained a capability it didn't have before).
    while IFS=': ' read -r pkg caps; do
        baseline_pkg_caps=$(grep "^${pkg}:" "$BASELINE" 2>/dev/null | sed 's/^[^:]*: //' || true)
        if [ -z "$baseline_pkg_caps" ]; then
            echo "NEW package with capabilities: $pkg: $caps"
            failed=1
            continue
        fi
        # Check each capability in current for this package
        for cap in $(echo "$caps" | tr ', ' '\n' | grep -v '^$' | grep -v CAPABILITY_UNSAFE_POINTER); do
            if ! echo "$baseline_pkg_caps" | grep -q "$cap"; then
                echo "NEW capability for $pkg: $cap"
                failed=1
            fi
        done
    done < "$current"

    if [ "$failed" -eq 1 ]; then
        echo ""
        echo "FAILED: capabilities have grown."
        echo "If this is intentional, run '$0 generate' and commit the updated $BASELINE."
        exit 1
    fi

    echo "OK: no new capabilities detected."
}

case "${1:-}" in
    generate) generate ;;
    check)    check ;;
    *)
        echo "Usage: $0 {generate|check}"
        exit 1
        ;;
esac
