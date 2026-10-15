// Measures real redirect-hot-path latency against the live deployment,
// staying UNDER the anonymous rate limit (20 req/min per IP) deliberately —
// this measures genuine application/network latency for legitimate traffic,
// not rate-limiter rejection behavior (see rate-limit-validation.js for that).
//
// Run: k6 run load-tests/redirect-latency.js
//   -e TARGET=https://<your-tunnel-url> -e SHORT_CODE=<a real short code>

import http from 'k6/http';
import { check, sleep } from 'k6';

const TARGET = __ENV.TARGET;
const SHORT_CODE = __ENV.SHORT_CODE;

export const options = {
  scenarios: {
    steady: {
      executor: 'constant-arrival-rate',
      rate: 10,            // 10 requests per timeUnit...
      timeUnit: '1m',      // ...per minute = well under the 20/min anonymous cap
      duration: '1m',
      preAllocatedVUs: 2,
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<500'],
  },
};

export default function () {
  const res = http.get(`${TARGET}/${SHORT_CODE}`, { redirects: 0 });
  check(res, { 'redirect succeeded (302)': (r) => r.status === 302 });
}
