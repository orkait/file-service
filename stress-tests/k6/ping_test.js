import http from "k6/http";
import { check, sleep } from "k6";
import { Rate, Trend } from "k6/metrics";

// Custom metrics
const errorRate = new Rate("errors");
const latency = new Trend("request_latency", true);

// Test configuration — simulates load on a 1GB RAM machine
// Stages ramp from light to heavy load and back down
export const options = {
  scenarios: {
    // Scenario 1: Gradual ramp-up stress test
    stress_test: {
      executor: "ramping-vus",
      startVUs: 0,
      stages: [
        { duration: "30s", target: 10 }, // Warm up
        { duration: "1m", target: 50 }, // Ramp to moderate load
        { duration: "2m", target: 100 }, // Sustained high load
        { duration: "1m", target: 200 }, // Peak stress
        { duration: "30s", target: 0 }, // Cool down
      ],
      gracefulRampDown: "10s",
    },
  },
  thresholds: {
    http_req_duration: ["p(95)<500", "p(99)<1000"], // 95% under 500ms, 99% under 1s
    errors: ["rate<0.1"], // Error rate below 10%
    http_req_failed: ["rate<0.1"],
  },
};

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";

export default function () {
  const res = http.get(`${BASE_URL}/ping`);

  check(res, {
    "status is 200": (r) => r.status === 200,
    "response has message": (r) => {
      const body = JSON.parse(r.body);
      return body.message === "pong";
    },
    "response time < 200ms": (r) => r.timings.duration < 200,
  });

  errorRate.add(res.status !== 200);
  latency.add(res.timings.duration);

  sleep(0.1); // 100ms between requests per VU
}

export function handleSummary(data) {
  return {
    stdout: textSummary(data, { indent: " ", enableColors: true }),
    "stress-tests/results/ping_results.json": JSON.stringify(data, null, 2),
  };
}

// k6 built-in text summary
import { textSummary } from "https://jslib.k6.io/k6-summary/0.0.1/index.js";
