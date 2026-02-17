#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────
# Stress Test Runner for file-service
# Runs k6 tests, Go benchmarks, and collects profiling data
#
# Prerequisites:
#   - k6       : https://k6.io/docs/get-started/installation/
#   - go       : already installed
#   - curl/jq  : for metrics collection
#   - docker   : for constrained environment tests (optional)
#
# Usage:
#   ./stress-tests/run.sh [ping|upload|read|batch|soak|bench|all]
#   ./stress-tests/run.sh all                     # full suite
#   ./stress-tests/run.sh ping                    # single test
#   BASE_URL=http://remote:8080 ./stress-tests/run.sh all
# ─────────────────────────────────────────────────────────────
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
RESULTS_DIR="stress-tests/results"
PROFILE_DIR="stress-tests/profiles"
SCRIPT_DIR="stress-tests/k6"
PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

log()  { echo -e "${CYAN}[$(date +%H:%M:%S)]${NC} $*"; }
ok()   { echo -e "${GREEN}[✓]${NC} $*"; }
warn() { echo -e "${YELLOW}[!]${NC} $*"; }
err()  { echo -e "${RED}[✗]${NC} $*"; }

# ── Setup ────────────────────────────────────────────────────
setup() {
    mkdir -p "$RESULTS_DIR" "$PROFILE_DIR"
    log "Results → $RESULTS_DIR"
    log "Profiles → $PROFILE_DIR"
    log "Target → $BASE_URL"

    # Check server is up
    if ! curl -sf "$BASE_URL/ping" > /dev/null 2>&1; then
        err "Server not reachable at $BASE_URL/ping"
        echo "  Start the server first:"
        echo "    ENABLE_PROFILING=true DISABLE_RATE_LIMITER=true go run main.go"
        echo "  Or with Docker:"
        echo "    docker compose -f docker-compose.stress.yml --profile t3-micro up --build"
        exit 1
    fi
    ok "Server is up"

    # Check if profiling endpoints are available
    if curl -sf "$BASE_URL/metrics/memory" > /dev/null 2>&1; then
        ok "Profiling endpoints available"
    else
        warn "Profiling endpoints not available (set ENABLE_PROFILING=true)"
    fi
}

# ── Collect metrics snapshot ─────────────────────────────────
collect_metrics() {
    local label="$1"
    local file="$RESULTS_DIR/${label}_metrics.json"

    if curl -sf "$BASE_URL/metrics/memory" > /dev/null 2>&1; then
        echo "{" > "$file"
        echo '  "memory":' >> "$file"
        curl -sf "$BASE_URL/metrics/memory" >> "$file"
        echo ',' >> "$file"
        echo '  "requests":' >> "$file"
        curl -sf "$BASE_URL/metrics/requests" >> "$file"
        echo "}" >> "$file"
        log "Metrics saved → $file"
    fi
}

# ── Collect pprof profile ───────────────────────────────────
collect_profile() {
    local label="$1"
    local duration="${2:-10}"

    if curl -sf "$BASE_URL/debug/pprof/" > /dev/null 2>&1; then
        log "Collecting ${duration}s CPU profile..."
        curl -sf "$BASE_URL/debug/pprof/profile?seconds=$duration" \
            -o "$PROFILE_DIR/${label}_cpu.prof" 2>/dev/null &
        local cpu_pid=$!

        curl -sf "$BASE_URL/debug/pprof/heap" \
            -o "$PROFILE_DIR/${label}_heap.prof" 2>/dev/null
        ok "Heap profile → $PROFILE_DIR/${label}_heap.prof"

        curl -sf "$BASE_URL/debug/pprof/goroutine" \
            -o "$PROFILE_DIR/${label}_goroutine.prof" 2>/dev/null
        ok "Goroutine profile → $PROFILE_DIR/${label}_goroutine.prof"

        wait $cpu_pid 2>/dev/null || true
        ok "CPU profile → $PROFILE_DIR/${label}_cpu.prof"
    else
        warn "pprof not available, skipping profile collection"
    fi
}

# ── Reset metrics between tests ─────────────────────────────
reset_metrics() {
    curl -sf -X POST "$BASE_URL/metrics/reset" > /dev/null 2>&1 || true
}

# ── Check k6 is installed ───────────────────────────────────
check_k6() {
    if ! command -v k6 &> /dev/null; then
        err "k6 not installed"
        echo "  Install: https://k6.io/docs/get-started/installation/"
        exit 1
    fi
    ok "k6 found: $(k6 version 2>&1 | head -1)"
}

# ── Individual test runners ──────────────────────────────────
run_ping() {
    log "━━━ PING STRESS TEST ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    reset_metrics
    collect_metrics "ping_before"
    k6 run -e BASE_URL="$BASE_URL" "$SCRIPT_DIR/ping_test.js"
    collect_metrics "ping_after"
    collect_profile "ping" 5
    ok "Ping test complete"
}

run_upload() {
    log "━━━ UPLOAD STRESS TEST ━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    reset_metrics
    collect_metrics "upload_before"
    k6 run -e BASE_URL="$BASE_URL" -e FILE_SIZE_KB="${FILE_SIZE_KB:-100}" \
        "$SCRIPT_DIR/upload_test.js"
    collect_metrics "upload_after"
    collect_profile "upload" 5
    ok "Upload test complete"
}

run_read() {
    log "━━━ READ STRESS TEST ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    reset_metrics
    collect_metrics "read_before"
    k6 run -e BASE_URL="$BASE_URL" -e TEST_FOLDER="${TEST_FOLDER:-}" \
        "$SCRIPT_DIR/read_test.js"
    collect_metrics "read_after"
    collect_profile "read" 5
    ok "Read test complete"
}

run_batch() {
    log "━━━ BATCH OPERATIONS STRESS TEST ━━━━━━━━━━━━━━━━━━"
    reset_metrics
    collect_metrics "batch_before"
    k6 run -e BASE_URL="$BASE_URL" \
        -e BATCH_SIZE="${BATCH_SIZE:-5}" \
        -e FILE_SIZE_KB="${FILE_SIZE_KB:-50}" \
        "$SCRIPT_DIR/batch_test.js"
    collect_metrics "batch_after"
    collect_profile "batch" 5
    ok "Batch test complete"
}

run_soak() {
    log "━━━ SOAK TEST (15 min) ━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    reset_metrics
    collect_metrics "soak_before"
    k6 run -e BASE_URL="$BASE_URL" "$SCRIPT_DIR/soak_test.js"
    collect_metrics "soak_after"
    collect_profile "soak" 10
    ok "Soak test complete"
}

run_bench() {
    log "━━━ GO BENCHMARKS ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    cd "$PROJECT_ROOT"

    log "Running routes benchmarks..."
    go test -bench=. -benchmem -count=3 -run=^$ ./routes/ \
        | tee "$RESULTS_DIR/bench_routes.txt"

    log "Running cache benchmarks..."
    go test -bench=. -benchmem -count=3 -run=^$ ./pkg/cache/ \
        | tee "$RESULTS_DIR/bench_cache.txt"

    # Memory profiling via Go test
    log "Running cache benchmarks with memory profile..."
    go test -bench=BenchmarkCacheMemoryPressure -benchmem \
        -memprofile="$PROFILE_DIR/bench_mem.prof" \
        -run=^$ ./pkg/cache/

    ok "Go benchmarks complete"
    ok "Results → $RESULTS_DIR/bench_*.txt"
    ok "Mem profile → $PROFILE_DIR/bench_mem.prof"
    echo "  View with: go tool pprof -http=:9090 $PROFILE_DIR/bench_mem.prof"
}

# ── Print summary ───────────────────────────────────────────
print_summary() {
    echo ""
    log "━━━ SUMMARY ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "  Results:  $RESULTS_DIR/"
    echo "  Profiles: $PROFILE_DIR/"
    echo ""
    echo "  Analyze CPU profile:"
    echo "    go tool pprof -http=:9090 $PROFILE_DIR/<test>_cpu.prof"
    echo ""
    echo "  Analyze heap profile:"
    echo "    go tool pprof -http=:9090 $PROFILE_DIR/<test>_heap.prof"
    echo ""
    echo "  Compare benchmark runs:"
    echo "    go install golang.org/x/perf/cmd/benchstat@latest"
    echo "    benchstat $RESULTS_DIR/bench_cache.txt"
    echo ""

    # Show final memory state if available
    if curl -sf "$BASE_URL/metrics/memory" > /dev/null 2>&1; then
        echo "  Final memory state:"
        curl -sf "$BASE_URL/metrics/memory" | python3 -m json.tool 2>/dev/null || \
            curl -sf "$BASE_URL/metrics/memory"
        echo ""
    fi
}

# ── Main ─────────────────────────────────────────────────────
main() {
    local test="${1:-all}"

    echo ""
    echo "╔══════════════════════════════════════════════════╗"
    echo "║     file-service Stress Test Runner              ║"
    echo "╚══════════════════════════════════════════════════╝"
    echo ""

    case "$test" in
        ping)
            setup; check_k6; run_ping ;;
        upload)
            setup; check_k6; run_upload ;;
        read)
            setup; check_k6; run_read ;;
        batch)
            setup; check_k6; run_batch ;;
        soak)
            setup; check_k6; run_soak ;;
        bench)
            run_bench ;;
        all)
            setup; check_k6
            run_bench
            run_ping
            run_read
            run_upload
            run_batch
            run_soak
            ;;
        *)
            echo "Usage: $0 [ping|upload|read|batch|soak|bench|all]"
            exit 1
            ;;
    esac

    print_summary
}

main "$@"
