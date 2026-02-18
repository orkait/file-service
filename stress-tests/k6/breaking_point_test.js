/**
 * Breaking Point Test — Find the exact VU count where the server breaks
 *
 * Strategy: Ramp from 10 → 2000 VUs in steps, measuring:
 *   - P95 latency crossing 2000ms (degradation threshold)
 *   - Error rate crossing 5% (reliability threshold)
 *   - Connection failures (hard failure — server rejecting connections)
 *
 * Simulates AWS t3.micro: 1 vCPU, 1 GB RAM
 * Uses only /ping endpoint FIRST (no S3) to isolate the Go server's true limit.
 * Then repeats with /list to include S3 in the equation.
 */

import http from "k6/http";
import { check, sleep } from "k6";
import { Rate, Trend, Counter } from "k6/metrics";
import { textSummary } from "https://jslib.k6.io/k6-summary/0.0.1/index.js";

// ── Custom metrics ────────────────────────────────────────────────────────────
const errorRate = new Rate("errors");
const connectionErrors = new Counter("connection_errors");
const slowRequests = new Counter("slow_requests_over_2s");

// ── Test configuration ────────────────────────────────────────────────────────
export const options = {
  scenarios: {
    breaking_point: {
      executor: "ramping-vus",
      startVUs: 1,
      stages: [
        // Step 1: warm up
        { duration: "30s", target: 50 },

        // Step 2: light load — baseline
        { duration: "45s", target: 200 },
        { duration: "30s", target: 200 }, // hold

        // Step 3: moderate
        { duration: "45s", target: 500 },
        { duration: "30s", target: 500 }, // hold

        // Step 4: high
        { duration: "45s", target: 1000 },
        { duration: "30s", target: 1000 }, // hold

        // Step 5: very high — t3.micro stress zone
        { duration: "45s", target: 2000 },
        { duration: "30s", target: 2000 }, // hold

        // Step 6: extreme — goroutine + scheduler saturation
        { duration: "45s", target: 3000 },
        { duration: "30s", target: 3000 }, // hold

        // Step 7: near-break
        { duration: "45s", target: 4000 },
        { duration: "30s", target: 4000 }, // hold

        // Step 8: hard wall — max pressure
        { duration: "45s", target: 5000 },
        { duration: "30s", target: 5000 }, // hold

        // Recovery — does the server come back?
        { duration: "1m", target: 200 },
        { duration: "30s", target: 0 },
      ],
      gracefulRampDown: "30s",
    },
  },

  // Never abort — we want to capture the full failure curve
  thresholds: {
    http_req_duration: [
      { threshold: "p(95)<2000", abortOnFail: false },
      { threshold: "p(99)<5000", abortOnFail: false },
    ],
    http_req_failed: [{ threshold: "rate<0.20", abortOnFail: false }],
    errors: [{ threshold: "rate<0.20", abortOnFail: false }],
  },
};

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";
const ENDPOINT = __ENV.ENDPOINT || "ping"; // "ping" (no S3) or "list" (with S3)

export function setup() {
  const res = http.get(`${BASE_URL}/ping`, { timeout: "5s" });
  if (res.status !== 200) {
    console.error(
      `Server not responding at ${BASE_URL}/ping — status ${res.status}`,
    );
  }
  console.log(`Breaking point test starting — target endpoint: /${ENDPOINT}`);
  console.log(`VU ramp: 50 → 200 → 500 → 1000 → 2000 → 3000 → 4000 → 5000`);
  return {};
}

export default function (_data) {
  let url;
  if (ENDPOINT === "list") {
    url = `${BASE_URL}/list?path=stress-test/`;
  } else {
    url = `${BASE_URL}/ping`;
  }

  const startTime = Date.now();

  const res = http.get(url, {
    timeout: "10s",
    tags: { endpoint: ENDPOINT, vus: __VU },
  });

  const duration = Date.now() - startTime;

  // Track connection-level failures (refused, timeout, reset)
  if (res.error_code && res.error_code !== 0) {
    connectionErrors.add(1);
  }

  // Track requests that exceeded 2 seconds
  if (duration > 2000) {
    slowRequests.add(1);
  }

  const ok = check(res, {
    "status 200": (r) => r.status === 200,
    "latency < 2s": (r) => r.timings.duration < 2000,
    "no connection error": (r) => !r.error,
  });

  errorRate.add(!ok);

  // No sleep — maximum pressure test
}

export function handleSummary(data) {
  // Extract key metrics for easy reading
  const p95 = data.metrics["http_req_duration"]
    ? data.metrics["http_req_duration"].values["p(95)"]
    : "N/A";
  const p99 = data.metrics["http_req_duration"]
    ? data.metrics["http_req_duration"].values["p(99)"]
    : "N/A";
  const failRate = data.metrics["http_req_failed"]
    ? (data.metrics["http_req_failed"].values.rate * 100).toFixed(2)
    : "N/A";
  const totalReqs = data.metrics["http_reqs"]
    ? data.metrics["http_reqs"].values.count
    : "N/A";
  const reqRate = data.metrics["http_reqs"]
    ? data.metrics["http_reqs"].values.rate.toFixed(1)
    : "N/A";
  const connErrors = data.metrics["connection_errors"]
    ? data.metrics["connection_errors"].values.count
    : 0;
  const slowReqs = data.metrics["slow_requests_over_2s"]
    ? data.metrics["slow_requests_over_2s"].values.count
    : 0;

  console.log("\n╔══════════════════════════════════════════════════╗");
  console.log("║         BREAKING POINT TEST RESULTS              ║");
  console.log("╚══════════════════════════════════════════════════╝");
  console.log(`  Total requests  : ${totalReqs}`);
  console.log(`  Throughput      : ${reqRate} req/s`);
  console.log(
    `  P95 latency     : ${typeof p95 === "number" ? p95.toFixed(2) + " ms" : p95}`,
  );
  console.log(
    `  P99 latency     : ${typeof p99 === "number" ? p99.toFixed(2) + " ms" : p99}`,
  );
  console.log(`  Connection fails: ${connErrors}`);
  console.log(`  Slow reqs (>2s) : ${slowReqs}`);
  console.log(`  HTTP fail rate  : ${failRate}%`);
  console.log("════════════════════════════════════════════════════");

  return {
    stdout: textSummary(data, { indent: " ", enableColors: true }),
    "stress-tests/results/breaking_point_results.json": JSON.stringify(
      data,
      null,
      2,
    ),
  };
}
