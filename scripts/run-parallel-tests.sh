#!/bin/bash
set -euo pipefail

# Orchestrator: run test plans in parallel across Docker containers
# Usage: ./scripts/run-parallel-tests.sh [--all | --suites "phase-3,phase-4"] [--no-teardown]

MAX_CONTAINERS=5
SUITES=""
RUN_ALL=false
NO_TEARDOWN=false
RESULTS_DIR="test-results"
PLANS_DIR="testing-plans"

while [[ $# -gt 0 ]]; do
    case $1 in
        --all) RUN_ALL=true; shift ;;
        --suites) SUITES="$2"; shift 2 ;;
        --no-teardown) NO_TEARDOWN=true; shift ;;
        *) echo "Unknown arg: $1"; exit 1 ;;
    esac
done

cleanup() {
    if [[ "$NO_TEARDOWN" == "false" ]]; then
        echo ""
        echo "Tearing down containers..."
        docker compose down -v --remove-orphans 2>/dev/null || true
    else
        echo "Skipping teardown (--no-teardown)"
    fi
}
trap cleanup EXIT

# Discover suites
declare -a PLAN_FILES=()
if [[ "$RUN_ALL" == "true" ]]; then
    while IFS= read -r f; do
        PLAN_FILES+=("$f")
    done < <(ls "$PLANS_DIR"/phase-*.md 2>/dev/null | sort)
elif [[ -n "$SUITES" ]]; then
    IFS=',' read -ra SUITE_NAMES <<< "$SUITES"
    for name in "${SUITE_NAMES[@]}"; do
        name=$(echo "$name" | xargs)  # trim whitespace
        match=$(ls "$PLANS_DIR"/${name}*.md 2>/dev/null | head -1)
        if [[ -n "$match" ]]; then
            PLAN_FILES+=("$match")
        else
            echo "Warning: no plan found for '$name'"
        fi
    done
else
    echo "Usage: $0 --all | --suites \"phase-3,phase-4\" [--no-teardown]"
    exit 1
fi

TOTAL=${#PLAN_FILES[@]}
if [[ "$TOTAL" -eq 0 ]]; then
    echo "No test plans found."
    exit 1
fi

echo "Found $TOTAL test plan(s) to run."
echo ""

# Clean results
rm -rf "$RESULTS_DIR"
mkdir -p "$RESULTS_DIR"

# Build and start containers
echo "Building Docker image..."
docker compose build

echo "Starting $MAX_CONTAINERS containers..."
docker compose up -d --wait

echo "All containers healthy."
echo ""

# Run suites in waves of MAX_CONTAINERS
WAVE=0
SUITE_IDX=0
OVERALL_PASS=true

while [[ $SUITE_IDX -lt $TOTAL ]]; do
    WAVE=$((WAVE + 1))
    echo "=== Wave $WAVE ==="

    declare -a PIDS=()
    declare -a WAVE_SUITES=()
    CONTAINER=1

    while [[ $SUITE_IDX -lt $TOTAL && $CONTAINER -le $MAX_CONTAINERS ]]; do
        PLAN="${PLAN_FILES[$SUITE_IDX]}"
        PORT=$((8080 + CONTAINER))
        SUITE_NAME=$(basename "$PLAN" .md)

        echo "  Starting: $SUITE_NAME on port $PORT"
        ./scripts/run-suite.sh --plan "$PLAN" --port "$PORT" &
        PIDS+=($!)
        WAVE_SUITES+=("$SUITE_NAME")

        SUITE_IDX=$((SUITE_IDX + 1))
        CONTAINER=$((CONTAINER + 1))
    done

    # Wait for wave to complete
    for pid in "${PIDS[@]}"; do
        wait "$pid" || true
    done

    # Reset containers between waves if more waves needed
    if [[ $SUITE_IDX -lt $TOTAL ]]; then
        echo "  Resetting containers for next wave..."
        docker compose down -v 2>/dev/null || true
        docker compose up -d --wait
    fi

    echo ""
done

# Print summary
echo "========================================="
echo "         TEST RESULTS SUMMARY"
echo "========================================="
printf "%-45s %s\n" "SUITE" "RESULT"
echo "-----------------------------------------"

PASSED=0
FAILED=0

for plan in "${PLAN_FILES[@]}"; do
    name=$(basename "$plan" .md)
    result_file="${RESULTS_DIR}/${name}.result"
    if [[ -f "$result_file" ]]; then
        result=$(cat "$result_file")
    else
        result="UNKNOWN"
    fi

    if [[ "$result" == "PASS" ]]; then
        printf "%-45s ✅ PASS\n" "$name"
        PASSED=$((PASSED + 1))
    else
        printf "%-45s ❌ FAIL\n" "$name"
        FAILED=$((FAILED + 1))
        OVERALL_PASS=false

        # Show tail of failed log
        log_file="${RESULTS_DIR}/${name}.log"
        if [[ -f "$log_file" ]]; then
            echo "  --- Last 10 lines of $name log ---"
            tail -10 "$log_file" | sed 's/^/  /'
            echo "  ---"
        fi
    fi
done

echo ""
echo "Total: $((PASSED + FAILED)) | Passed: $PASSED | Failed: $FAILED"
echo "========================================="

if [[ "$OVERALL_PASS" == "false" ]]; then
    echo "Some suites FAILED."
    exit 1
fi

echo "All suites PASSED."
