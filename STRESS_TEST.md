# Stress Test & Performance Analysis Guide

## Overview

This branch adds stress testing and performance profiling infrastructure to the file-service. It's designed to test the service under AWS-like constrained environments (1 vCPU / 1GB RAM up to 2 vCPU / 4GB RAM).

## What Was Added

```
pkg/profiling/profiling.go    ← pprof endpoints, memory stats, GOMEMLIMIT tuning
pkg/metrics/metrics.go        ← request counter, latency tracker, per-endpoint stats
routes/routes_bench_test.go   ← Go benchmarks for handlers
pkg/cache/cache_bench_test.go ← Go benchmarks for cache (read/write/contention)
stress-tests/
  run.sh                      ← Orchestrator script (runs everything)
  k6/
    ping_test.js              ← Lightweight endpoint stress
    upload_test.js            ← File upload memory pressure
    read_test.js              ← List/download read-heavy load
    batch_test.js             ← Batch upload/download (most memory-intensive)
    soak_test.js              ← 15-min sustained load (leak detection)
Dockerfile                    ← Multi-stage build with memory tuning
docker-compose.stress.yml     ← AWS instance profiles (t3.micro/small/medium)
```

## Environment Variables (added)

| Variable | Default | Purpose |
|---|---|---|
| `ENABLE_PROFILING` | `false` | Enables `/debug/pprof/*`, `/health`, `/metrics/*` endpoints |
| `DISABLE_RATE_LIMITER` | `false` | Disables rate limiter for unrestricted load testing |
| `GOMEMLIMIT` | (auto) | Go runtime memory limit (set to ~80% of available RAM) |
| `GOGC` | `50` | GC frequency (lower = more aggressive, better for low-memory) |

## Quick Start

### 1. Install k6

Follow the official installation guide: https://k6.io/docs/get-started/installation/

### 2. Run Against Local Server

```bash
# Start server with profiling enabled
ENABLE_PROFILING=true DISABLE_RATE_LIMITER=true go run main.go

# In another terminal — run all tests
./stress-tests/run.sh all

# Or run individual tests
./stress-tests/run.sh ping
./stress-tests/run.sh upload
./stress-tests/run.sh bench
```

### 3. Run in Docker (Simulated AWS Instance)

```bash
# Simulate AWS t3.micro (1 vCPU, 1GB RAM)
docker compose -f docker-compose.stress.yml --profile t3-micro up --build

# Simulate AWS t3.small (2 vCPU, 2GB RAM)
docker compose -f docker-compose.stress.yml --profile t3-small up --build

# Simulate AWS t3.medium (2 vCPU, 4GB RAM)
docker compose -f docker-compose.stress.yml --profile t3-medium up --build

# Then run tests against it
./stress-tests/run.sh all
```

## Test Descriptions

### k6 Load Tests

| Test | What it tests | Peak VUs | Duration |
|---|---|---|---|
| `ping_test.js` | Baseline throughput, framework overhead | 200 | ~5 min |
| `upload_test.js` | File upload memory pressure (100KB files) | 100 | ~5 min |
| `read_test.js` | List + download read-heavy workload | 300 | ~5 min |
| `batch_test.js` | Batch upload/download (most memory-heavy) | 50 | ~5 min |
| `soak_test.js` | Sustained load, leak detection | 20 | 15 min |

### Go Benchmarks

| Benchmark | What it measures |
|---|---|
| `BenchmarkPingHandler` | Raw handler latency + allocs |
| `BenchmarkPingHandlerParallel` | Concurrent handler performance |
| `BenchmarkJSONSerialization` | Echo JSON encoding overhead |
| `BenchmarkBatchDownloadRequestParsing` | JSON body parsing cost |
| `BenchmarkCacheGet/Set/Parallel` | Cache read/write performance |
| `BenchmarkCacheMixedReadWrite` | Real-world 80/20 read/write mix |
| `BenchmarkCacheMemoryPressure` | Cache growth under concurrent load |
| `BenchmarkCacheClear` | Expired entry cleanup cost |

## Analyzing Results

### View CPU Profile (interactive web UI)

```bash
go tool pprof -http=:9090 stress-tests/profiles/<test>_cpu.prof
```

### View Heap/Memory Profile

```bash
go tool pprof -http=:9090 stress-tests/profiles/<test>_heap.prof
```

### Compare Benchmark Runs

```bash
go install golang.org/x/perf/cmd/benchstat@latest
benchstat stress-tests/results/bench_cache.txt
```

### Live Monitoring During Tests

```bash
# Memory usage
watch -n 1 'curl -s http://localhost:8080/metrics/memory | python3 -m json.tool'

# Request stats
watch -n 2 'curl -s http://localhost:8080/metrics/requests | python3 -m json.tool'

# Goroutine count
curl http://localhost:8080/debug/pprof/goroutine?debug=1
```

## Key Metrics to Watch on 1GB RAM

| Metric | Healthy | Warning | Critical |
|---|---|---|---|
| `sys_mb` (total Go memory) | < 400 MB | 400–700 MB | > 700 MB |
| `goroutines` | < 500 | 500–5000 | > 5000 |
| `num_gc` | Increasing steadily | — | Stalled |
| Error rate | < 5% | 5–15% | > 15% |
| P95 latency (ping) | < 50ms | 50–200ms | > 500ms |
| P95 latency (upload) | < 2s | 2–5s | > 5s |

## Potential Issues on 1GB RAM

1. **OOM on batch uploads** — 50 concurrent VUs × 5 files × 50KB = ~12.5MB in-flight buffers. At peak, multiply by upload pipeline depth.
2. **Cache bloat** — URL cache has no size limit. Thousands of entries → memory grows unbounded. Consider adding max-entries cap.
3. **Goroutine leaks** — Watch goroutine count during soak test. Should stay flat, not grow linearly.
4. **GC pressure** — With GOGC=50 on 1GB, GC runs frequently. Watch if GC pause time impacts P99 latency.
