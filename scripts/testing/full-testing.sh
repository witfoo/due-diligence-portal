#!/usr/bin/env bash
# Orchestrates all test phases.
# Usage: ./scripts/testing/full-testing.sh [--from-step STEP] [--verbose]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../lib/common.sh"

FROM_STEP="${FROM_STEP:-1}"
VERBOSE=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        --from-step) FROM_STEP="$2"; shift 2 ;;
        --verbose) VERBOSE="--verbose"; shift ;;
        *) shift ;;
    esac
done

log_header "Full Testing Pipeline"

TOTAL_PASSED=0
TOTAL_FAILED=0

run_step() {
    local step_num=$1
    local step_name=$2
    local script=$3

    if [ "$step_num" -lt "$FROM_STEP" ]; then
        log_info "Skipping step $step_num: $step_name"
        return 0
    fi

    log_header "Step $step_num: $step_name"
    if bash "$script" $VERBOSE; then
        TOTAL_PASSED=$((TOTAL_PASSED + 1))
        log_success "Step $step_num: $step_name PASSED"
    else
        TOTAL_FAILED=$((TOTAL_FAILED + 1))
        log_fail "Step $step_num: $step_name FAILED"
    fi
}

run_step 1 "Linting" "${SCRIPT_DIR}/linter.sh"
run_step 2 "Unit Tests" "${SCRIPT_DIR}/unit-tests.sh"
run_step 3 "Security Scan" "${SCRIPT_DIR}/security.sh"

echo ""
log_header "Final Results"
echo -e "${GREEN}  Steps Passed: ${TOTAL_PASSED}${NC}"
echo -e "${RED}  Steps Failed: ${TOTAL_FAILED}${NC}"

if [ "$TOTAL_FAILED" -gt 0 ]; then
    echo -e "\n${RED}  PIPELINE: FAILED${NC}\n"
    exit 1
else
    echo -e "\n${GREEN}  PIPELINE: PASSED${NC}\n"
    exit 0
fi
