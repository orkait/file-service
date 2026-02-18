import http from "k6/http";
import { check, sleep } from "k6";
import { Rate, Trend } from "k6/metrics";
import { FormData } from "https://jslib.k6.io/formdata/0.0.2/index.js";

const errorRate = new Rate("errors");
const batchUploadLatency = new Trend("batch_upload_latency", true);
const batchDownloadLatency = new Trend("batch_download_latency", true);

// Batch operations stress test — most memory-intensive on 1GB RAM
export const options = {
  scenarios: {
    batch_stress: {
      executor: "ramping-vus",
      startVUs: 0,
      stages: [
        { duration: "20s", target: 3 }, // Very gentle start — batch ops are heavy
        { duration: "1m", target: 10 }, // Moderate — each VU sends multiple files
        { duration: "2m", target: 25 }, // High load — memory pressure
        { duration: "1m", target: 50 }, // Peak — OOM danger zone on 1GB
        { duration: "30s", target: 0 }, // Cool down
      ],
      gracefulRampDown: "15s",
    },
  },
  thresholds: {
    http_req_duration: ["p(95)<5000", "p(99)<10000"], // Batch ops are slow
    errors: ["rate<0.2"],
    http_req_failed: ["rate<0.2"],
  },
};

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";
const BATCH_SIZE = parseInt(__ENV.BATCH_SIZE || "5"); // Files per batch
const FILE_SIZE_KB = parseInt(__ENV.FILE_SIZE_KB || "50"); // Smaller files for batch

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
  const res = http.get(`${BASE_URL}/ping`);
  check(res, { "server is up": (r) => r.status === 200 });
  return { fileData: generateTestFile(FILE_SIZE_KB) };
}

export default function (data) {
  const folderName = `batch-stress-${__VU}/`;

  // ── Test 1: Batch Upload ──────────────────────────────────────────
  const fd = new FormData();
  fd.append("path", folderName);

  const uploadedFileNames = [];
  for (let i = 0; i < BATCH_SIZE; i++) {
    const fileName = `batch-${__VU}-${__ITER}-${i}.bin`;
    uploadedFileNames.push(`${folderName}${fileName}`);
    fd.append("files", http.file(data.fileData, fileName, "text/plain"));
  }

  const uploadRes = http.post(`${BASE_URL}/batch-upload`, fd.body(), {
    headers: { "Content-Type": `multipart/form-data; boundary=${fd.boundary}` },
    timeout: "60s",
  });

  const uploadOk = check(uploadRes, {
    "batch upload status 200": (r) => r.status === 200,
    "batch upload success": (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.status === "Success";
      } catch (e) {
        return false;
      }
    },
    "batch upload time < 10s": (r) => r.timings.duration < 10000,
  });

  errorRate.add(!uploadOk);
  batchUploadLatency.add(uploadRes.timings.duration);

  sleep(0.5);

  // ── Test 2: Batch Download ────────────────────────────────────────
  const dlPayload = JSON.stringify({ paths: uploadedFileNames });
  const dlRes = http.post(`${BASE_URL}/batch-download`, dlPayload, {
    headers: { "Content-Type": "application/json" },
    timeout: "30s",
  });

  const dlOk = check(dlRes, {
    "batch download status 200": (r) => r.status === 200,
    "batch download success": (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.status === "Success";
      } catch (e) {
        return false;
      }
    },
    "batch download time < 5s": (r) => r.timings.duration < 5000,
  });

  errorRate.add(!dlOk);
  batchDownloadLatency.add(dlRes.timings.duration);

  sleep(0.5);
}

export function teardown() {
  // Attempt cleanup — delete batch stress folders
  for (let vu = 1; vu <= 50; vu++) {
    http.del(`${BASE_URL}/delete-folder?path=batch-stress-${vu}/`);
  }
  console.log("Cleanup complete");
}

export function handleSummary(data) {
  return {
    stdout: textSummary(data, { indent: " ", enableColors: true }),
    "stress-tests/results/batch_results.json": JSON.stringify(data, null, 2),
  };
}

import { textSummary } from "https://jslib.k6.io/k6-summary/0.0.1/index.js";
