#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BENCHMARK_DIR="${SCRIPT_DIR}"
K6_DIR="${BENCHMARK_DIR}/k6"
RESULTS_DIR="${K6_DIR}/results"

mkdir -p "${RESULTS_DIR}"

BASE_URL="${BASE_URL:-http://localhost:8080}"
TARGET_VUS="${VUS:-50}"
TEST_DURATION="${DURATION:-5m}"

export BASE_URL
export TARGET_VUS
export TEST_DURATION

TIMESTAMP=$(date +%Y%m%d_%H%M%S)

run_cpu_stress() {
  echo ""
  echo "==================================="
  echo "  CPU Stress Test"
  echo "==================================="
  echo ""

  k6 run \
    --out json="${RESULTS_DIR}/cpu-stress_${TIMESTAMP}.json" \
    --out csv="${RESULTS_DIR}/cpu-stress_${TIMESTAMP}.csv" \
    "${K6_DIR}/cpu-stress-test.js" 2>&1 | tee "${RESULTS_DIR}/cpu-stress_${TIMESTAMP}.log"
}

run_memory_stress() {
  echo ""
  echo "==================================="
  echo "  Memory Stress Test"
  echo "==================================="
  echo ""

  k6 run \
    --out json="${RESULTS_DIR}/memory-stress_${TIMESTAMP}.json" \
    --out csv="${RESULTS_DIR}/memory-stress_${TIMESTAMP}.csv" \
    "${K6_DIR}/memory-stress-test.js" 2>&1 | tee "${RESULTS_DIR}/memory-stress_${TIMESTAMP}.log"
}

run_max_capacity() {
  echo ""
  echo "==================================="
  echo "  Max Capacity Test (Breaking Point)"
  echo "==================================="
  echo ""

  k6 run \
    --out json="${RESULTS_DIR}/max-capacity_${TIMESTAMP}.json" \
    --out csv="${RESULTS_DIR}/max-capacity_${TIMESTAMP}.csv" \
    "${K6_DIR}/max-capacity-test.js" 2>&1 | tee "${RESULTS_DIR}/max-capacity_${TIMESTAMP}.log"
}

run_load_test() {
  echo ""
  echo "==================================="
  echo "  Baseline Load Test (50 VUs, 5min)"
  echo "==================================="
  echo ""

  k6 run \
    --out json="${RESULTS_DIR}/load_${TIMESTAMP}.json" \
    --out csv="${RESULTS_DIR}/load_${TIMESTAMP}.csv" \
    "${K6_DIR}/payment-service-load-test.js" 2>&1 | tee "${RESULTS_DIR}/load_${TIMESTAMP}.log"
}

run_all() {
  echo "==========================================="
  echo "  Full Benchmark Suite (Go + Gin + GraphQL)"
  echo "==========================================="
  echo "  BASE_URL: ${BASE_URL}"
  echo "  Timestamp: ${TIMESTAMP}"
  echo "==========================================="
  echo ""

  run_load_test
  run_cpu_stress
  run_memory_stress
  run_max_capacity

  echo ""
  echo "==================================="
  echo "  All benchmarks completed!"
  echo "  Results: ${RESULTS_DIR}/"
  echo "==================================="
}

case "${1:-}" in
  "cpu")
    run_cpu_stress
    ;;
  "mem")
    run_memory_stress
    ;;
  "tp"|"max")
    run_max_capacity
    ;;
  "load")
    run_load_test
    ;;
  "all"|"")
    run_all
    ;;
  *)
    echo "Usage: $0 {all|cpu|mem|tp|load}"
    echo ""
    echo "  all   - Run full benchmark suite"
    echo "  cpu   - CPU stress test (HPA trigger)"
    echo "  mem   - Memory stress test (OOM trigger)"
    echo "  tp    - Max capacity / breaking point test"
    echo "  load  - Baseline load test (50 VUs, 5min)"
    echo ""
    echo "Environment variables:"
    echo "  BASE_URL       - Service URL (default: http://localhost:8080)"
    echo "  TARGET_VUS     - Virtual users (default: 50)"
    echo "  TEST_DURATION  - Test duration (default: 5m)"
    exit 1
    ;;
esac
