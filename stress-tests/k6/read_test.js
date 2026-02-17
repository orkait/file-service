import http from "k6/http";
import { check, sleep } from "k6";
import { Rate, Trend } from "k6/metrics";

const errorRate = new Rate("errors");
const listLatency = new Trend("list_latency", true);
const downloadLatency = new Trend("download_latency", true);

// Stress test for list + download — read-heavy workload
export const options = {
  scenarios: {
    read_stress: {
      executor: "ramping-vus",
      startVUs: 0,
      stages: [
        { duration: "20s", target: 10 },
        { duration: "1m", target: 50 },
        { duration: "2m", target: 150 }, // High concurrent reads
        { duration: "1m", target: 300 }, // Peak read stress
        { duration: "30s", target: 0 },
      ],
      gracefulRampDown: "10s",
    },
  },
  thresholds: {
    http_req_duration: ["p(95)<1000", "p(99)<3000"],
    errors: ["rate<0.1"],
    http_req_failed: ["rate<0.1"],
  },
};

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";
const TEST_FOLDER = __ENV.TEST_FOLDER || ""; // Root folder by default

export default function () {
  // Test 1: List files
  const listRes = http.get(`${BASE_URL}/list?path=${TEST_FOLDER}&pageSize=10`);

  const listOk = check(listRes, {
    "list status 200": (r) => r.status === 200,
    "list has data": (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.status === "Success" && body.data !== null;
      } catch (e) {
        return false;
      }
    },
    "list time < 500ms": (r) => r.timings.duration < 500,
  });

  errorRate.add(!listOk);
  listLatency.add(listRes.timings.duration);

  // Test 2: Try to download first file from listing (if available)
  try {
    const listBody = JSON.parse(listRes.body);
    if (listBody.data && listBody.data.data && listBody.data.data.length > 0) {
      const firstFile = listBody.data.data.find((f) => !f.isFolder);
      if (firstFile) {
        const dlRes = http.get(
          `${BASE_URL}/download?path=${encodeURIComponent(firstFile.name)}`,
        );

        check(dlRes, {
          "download status 200": (r) => r.status === 200,
          "download has url": (r) => {
            try {
              const body = JSON.parse(r.body);
              return body.data && body.data.url;
            } catch (e) {
              return false;
            }
          },
          "download time < 1s": (r) => r.timings.duration < 1000,
        });

        errorRate.add(dlRes.status !== 200);
        downloadLatency.add(dlRes.timings.duration);
      }
    }
  } catch (e) {
    // If parsing fails, just continue
  }

  // Test 3: List folders
  const foldersRes = http.get(`${BASE_URL}/list-folders?path=${TEST_FOLDER}`);

  check(foldersRes, {
    "folders status 200": (r) => r.status === 200,
    "folders time < 500ms": (r) => r.timings.duration < 500,
  });

  sleep(0.2);
}

export function handleSummary(data) {
  return {
    stdout: textSummary(data, { indent: " ", enableColors: true }),
    "stress-tests/results/read_results.json": JSON.stringify(data, null, 2),
  };
}

import { textSummary } from "https://jslib.k6.io/k6-summary/0.0.1/index.js";
