#!/bin/bash
set -euo pipefail

# Run a single test plan against a specific container port
# Usage: ./scripts/run-suite.sh --plan <path> --port <port>

PLAN=""
PORT=""
RESULTS_DIR="test-results"

while [[ $# -gt 0 ]]; do
    case $1 in
        --plan) PLAN="$2"; shift 2 ;;
        --port) PORT="$2"; shift 2 ;;
        *) echo "Unknown arg: $1"; exit 1 ;;
    esac
done

if [[ -z "$PLAN" || -z "$PORT" ]]; then
    echo "Usage: $0 --plan <path-to-plan.md> --port <port>"
    exit 1
fi

SUITE_NAME=$(basename "$PLAN" .md)
SESSION_NAME="suite-${SUITE_NAME}-port${PORT}"
RESULT_FILE="${RESULTS_DIR}/${SUITE_NAME}.result"
LOG_FILE="${RESULTS_DIR}/${SUITE_NAME}.log"

mkdir -p "$RESULTS_DIR"

# Verify container is healthy
echo "[$SUITE_NAME] Checking container health on port $PORT..."
for i in $(seq 1 30); do
    if curl -sf "http://localhost:${PORT}/health" > /dev/null 2>&1; then
        echo "[$SUITE_NAME] Container healthy."
        break
    fi
    if [ "$i" -eq 30 ]; then
        echo "[$SUITE_NAME] Container not healthy on port $PORT"
        echo "FAIL" > "$RESULT_FILE"
        exit 1
    fi
    sleep 2
done

PLAN_CONTENT=$(cat "$PLAN")

PROMPT="You are running browser tests against http://localhost:${PORT} using playwright-cli.

IMPORTANT: Use session name '${SESSION_NAME}' for all playwright-cli commands (pass -s=${SESSION_NAME}).
Replace any references to localhost:8080 with localhost:${PORT}.

Execute ALL test cases in the following test plan. For each test case:
1. Perform the described steps using playwright-cli
2. Verify the expected outcomes
3. Report PASS or FAIL with details

Test Plan:
${PLAN_CONTENT}

After completing all tests, output a summary in this exact format:
SUITE: ${SUITE_NAME}
PASSED: <count>
FAILED: <count>
RESULT: PASS or FAIL"

echo "[$SUITE_NAME] Starting test run on port $PORT..."
echo "RUNNING" > "$RESULT_FILE"

# Run claude agent with playwright-cli access
if claude --allowedTools "Bash(playwright-cli:*)" --print "$PROMPT" > "$LOG_FILE" 2>&1; then
    # Parse result from output
    if grep -q "RESULT: PASS" "$LOG_FILE"; then
        echo "PASS" > "$RESULT_FILE"
        echo "[$SUITE_NAME] PASSED"
    else
        echo "FAIL" > "$RESULT_FILE"
        echo "[$SUITE_NAME] FAILED"
    fi
else
    echo "FAIL" > "$RESULT_FILE"
    echo "[$SUITE_NAME] FAILED (agent error)"
fi

# Cleanup playwright session
playwright-cli quit -s="$SESSION_NAME" 2>/dev/null || true
