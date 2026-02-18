import http from "k6/http";
import { check, sleep } from "k6";
import { Rate, Trend } from "k6/metrics";
import { FormData } from "https://jslib.k6.io/formdata/0.0.2/index.js";

const errorRate = new Rate("errors");
const uploadLatency = new Trend("upload_latency", true);

// Stress test for file upload — critical for memory on 1GB RAM
export const options = {
  scenarios: {
    upload_stress: {
      executor: "ramping-vus",
      startVUs: 0,
      stages: [
        { duration: "20s", target: 5 }, // Warm up — few concurrent uploads
        { duration: "1m", target: 20 }, // Moderate concurrent uploads
        { duration: "2m", target: 50 }, // High load — tests memory pressure
        { duration: "1m", target: 100 }, // Peak — OOM risk zone on 1GB
        { duration: "30s", target: 0 }, // Cool down
      ],
      gracefulRampDown: "10s",
    },
  },
  thresholds: {
    http_req_duration: ["p(95)<2000", "p(99)<5000"], // Uploads are slower
    errors: ["rate<0.15"],
    http_req_failed: ["rate<0.15"],
  },
};

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";
const TEST_FILE_SIZE = parseInt(__ENV.FILE_SIZE_KB || "100"); // Default 100KB test files

// Generate random binary string of specified size (string is JSON-serializable)
function generateTestFile(sizeKB) {
  const size = sizeKB * 1024;
  let result = "";
  const chars =
    "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
  for (let i = 0; i < size; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length));
  }
  return result;
}

export function setup() {
  // Verify server is running
  const res = http.get(`${BASE_URL}/ping`);
  check(res, { "server is up": (r) => r.status === 200 });
  return { fileData: generateTestFile(TEST_FILE_SIZE) };
}

export default function (data) {
  const fileName = `stress-test-${__VU}-${__ITER}-${Date.now()}.txt`;

  const fd = new FormData();
  fd.append("file", http.file(data.fileData, fileName, "text/plain"));
  fd.append("path", "stress-test-uploads/");

  const res = http.post(`${BASE_URL}/upload`, fd.body(), {
    headers: { "Content-Type": `multipart/form-data; boundary=${fd.boundary}` },
    timeout: "30s",
  });

  check(res, {
    "upload status 200": (r) => r.status === 200,
    "upload success response": (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.status === "Success";
      } catch (e) {
        return false;
      }
    },
    "upload time < 3s": (r) => r.timings.duration < 3000,
  });

  errorRate.add(res.status !== 200);
  uploadLatency.add(res.timings.duration);

  sleep(0.5); // Half second between uploads per VU
}

export function teardown() {
  // Clean up: delete the stress test folder
  const res = http.del(`${BASE_URL}/delete-folder?path=stress-test-uploads/`);
  console.log(`Cleanup: ${res.status}`);
}

export function handleSummary(data) {
  return {
    stdout: textSummary(data, { indent: " ", enableColors: true }),
    "stress-tests/results/upload_results.json": JSON.stringify(data, null, 2),
  };
}

import { textSummary } from "https://jslib.k6.io/k6-summary/0.0.1/index.js";
