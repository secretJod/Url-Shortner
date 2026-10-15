// Validates the sliding-window rate limiter (backend/internal/redis/ratelimit.go)
// against the REAL public deployment — a single virtual user firing requests
// as fast as possible at the anonymous redirect endpoint, which is capped at
// 20 requests/minute per IP (backend/internal/middleware/ratelimit.go
// anonLimit). Expect the first ~20 to succeed (302) and the rest to be
// rejected (429) with a Retry-After header, within the same test run.
//
// Run: k6 run load-tests/rate-limit-validation.js
//   -e TARGET=https://<your-tunnel-url> -e SHORT_CODE=<a real short code>

import http from 'k6/http';
import { check } from 'k6';

const TARGET = __ENV.TARGET;
const SHORT_CODE = __ENV.SHORT_CODE;

export const options = {
  scenarios: {
    burst: {
      executor: 'per-vu-iterations',
      vus: 1,
      iterations: 30,
      maxDuration: '30s',
    },
  },
};

export default function () {
  const res = http.get(`${TARGET}/${SHORT_CODE}`, { redirects: 0 });
  check(res, {
    'is 302 (allowed) or 429 (rate-limited)': (r) => r.status === 302 || r.status === 429,
  });
  if (res.status === 429) {
    check(res, {
      '429 includes Retry-After header': (r) => r.headers['Retry-After'] !== undefined,
    });
  }
}
