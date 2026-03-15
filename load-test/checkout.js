import http from 'k6/http';
import { check } from 'k6';
import { Counter } from 'k6/metrics';

const BASE_URL = __ENV.BASE_URL || 'http://localhost/order';

const ITEMS = [
  { id: 1,  price: 10.00 },
  { id: 2,  price: 25.00 },
  { id: 3,  price: 49.99 },
  { id: 4,  price:  5.00 },
  { id: 5,  price: 99.99 },
  { id: 6,  price: 15.00 },
  { id: 7,  price: 30.00 },
  { id: 8,  price: 75.00 },
  { id: 9,  price:  8.50 },
  { id: 10, price: 19.99 },
];

const ordersInitiated = new Counter('orders_initiated');
const ordersSucceeded = new Counter('orders_succeeded');
const ordersLost = new Counter('orders_lost');
const revenueLost = new Counter('revenue_lost');

/**
 * Ramp-to-millions load test.
 *
 * Uses ramping-arrival-rate to drive a controlled request rate regardless
 * of response latency, ramping from 100 RPS up to 10 000 RPS over 5 minutes.
 * Targeted total: ~1.5 M order attempts.
 *
 *   Stage  Duration  Target RPS  Avg RPS   Requests
 *   -----  --------  ----------  -------   --------
 *   1      1 min     1 000        550       33 000
 *   2      1 min     3 000       2 000      120 000
 *   3      1 min     6 000       4 500      270 000
 *   4      1 min     9 000       7 500      450 000
 *   5      1 min    10 000       9 500      570 000
 *                                    Total ~1 443 000
 *
 * Run:
 *   k6 run --out influxdb=http://localhost:8086/k6 load-test/checkout.js
 */
export const options = {
  scenarios: {
    checkout: {
      executor: 'ramping-arrival-rate',
      startRate: 100,
      timeUnit: '1s',
      preAllocatedVUs: 1000,
      maxVUs: 8000,
      stages: [
        { target: 1000,  duration: '1m' },
        { target: 3000,  duration: '1m' },
        { target: 6000,  duration: '1m' },
        { target: 9000,  duration: '1m' },
        { target: 10000, duration: '1m' },
      ],
      exec: 'checkout',
    },
  },
  thresholds: {
    'http_req_duration': ['p(95)<30000'],
    'orders_lost': ['count<500000'],
  },
};

function randomInt(min, max) {
  return Math.floor(Math.random() * (max - min + 1)) + min;
}

function genIdempotencyKey() {
  return `perf-${__VU}-${__ITER}-${Date.now()}-${randomInt(1, 1e6)}`;
}

function genRequestId() {
  const hex = '0123456789abcdef';
  let s = '';
  for (let i = 0; i < 32; i++) s += hex[randomInt(0, 15)];
  return s;
}

export function checkout() {
  const item = ITEMS[randomInt(0, ITEMS.length - 1)];
  const quantity = randomInt(1, 3);
  ordersInitiated.add(1);

  const res = http.post(
    `${BASE_URL}/checkout`,
    JSON.stringify({ itemId: item.id, quantity }),
    {
      headers: {
        'Content-Type': 'application/json',
        'Idempotency-Key': genIdempotencyKey(),
        'X-Request-ID': genRequestId(),
      },
      timeout: '35s',
    }
  );

  const ok = check(res, { 'checkout 200': (r) => r.status === 200 });
  if (ok) {
    ordersSucceeded.add(1);
  } else {
    ordersLost.add(1);
    revenueLost.add(quantity * item.price);
  }
}
