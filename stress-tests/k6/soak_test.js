import http from "k6/http";
import { check, sleep } from "k6";
import { Rate } from "k6/metrics";

const errorRate = new Rate("errors");

// Soak test: sustained moderate load over a long period
// Purpose: detect memory leaks, goroutine leaks, cache bloat on 1GB RAM
export const options = {
  scenarios: {
    soak_test: {
      executor: "constant-vus",
      vus: 20, // Moderate steady load
      duration: "15m", // 15 minutes sustained
    },
  },
  thresholds: {
    http_req_duration: ["p(95)<500"],
    errors: ["rate<0.05"], // Tighter error threshold for soak
    http_req_failed: ["rate<0.05"],
  },
};

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";

export default function () {
  // Mix of read operations to simulate real traffic
  const ops = [
    () => http.get(`${BASE_URL}/ping`),
    () => http.get(`${BASE_URL}/list?path=&pageSize=10`),
    () => http.get(`${BASE_URL}/list-folders?path=`),
  ];

  const op = ops[Math.floor(Math.random() * ops.length)];
  const res = op();

  check(res, {
    "status is 200": (r) => r.status === 200,
    "response time < 300ms": (r) => r.timings.duration < 300,
  });

  errorRate.add(res.status !== 200);
  sleep(0.5);
}

// Periodically check memory via /metrics/memory
export function checkMemory() {
  const res = http.get(`${BASE_URL}/metrics/memory`);
  if (res.status === 200) {
    try {
      const mem = JSON.parse(res.body);
      console.log(
        `[Memory] Alloc: ${mem.alloc_mb.toFixed(1)}MB | Sys: ${mem.sys_mb.toFixed(1)}MB | Goroutines: ${mem.goroutines} | GC: ${mem.num_gc}`,
      );
      // Alert if memory exceeds 700MB (leaving 300MB for OS on 1GB machine)
      if (mem.sys_mb > 700) {
        console.warn(
          `WARNING: Memory usage ${mem.sys_mb.toFixed(1)}MB exceeds 700MB threshold!`,
        );
      }
    } catch (e) {
      // ignore parse errors
    }
  }
}

export function handleSummary(data) {
  return {
    stdout: textSummary(data, { indent: " ", enableColors: true }),
    "stress-tests/results/soak_results.json": JSON.stringify(data, null, 2),
  };
}

import { textSummary } from "https://jslib.k6.io/k6-summary/0.0.1/index.js";
