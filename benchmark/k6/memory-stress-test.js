import { check, sleep } from 'k6';
import http from 'k6/http';
import { Trend, Rate, Gauge } from 'k6/metrics';
import {
  setupTestEntities,
  teardownTestEntities,
  buildRampStages,
  printScalingBox,
} from './_shared.js';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const TARGET_VUS = parseInt(__ENV.TARGET_VUS || '100');
const TEST_DURATION = __ENV.TEST_DURATION || '5m';

const memErrors = new Rate('mem_stress_errors');
const largePayloadDuration = new Trend('mem_stress_large_payload_duration');
const concurrentHolding = new Trend('mem_stress_concurrent_holding');
const activeVUs = new Gauge('mem_stress_active_vus');

export const options = {
  stages: [
    { target: Math.floor(TARGET_VUS * 0.3), duration: '30s' },
    { target: Math.floor(TARGET_VUS * 0.6), duration: '30s' },
    { target: TARGET_VUS, duration: '60s' },
    { target: TARGET_VUS, duration: TEST_DURATION },
    { target: 0, duration: '60s' },
  ],
  thresholds: {
    http_req_duration: ['p(95)<5000', 'p(99)<10000'],
    http_req_failed: ['rate<0.15'],
    mem_stress_large_payload_duration: ['p(95)<8000'],
  },
};

export function setup() {
  return setupTestEntities(BASE_URL);
}

export default function (data) {
  activeVUs.add(1);

  const walletId = data.walletId;
  const merchantId = data.merchantId;
  const userId = data.userId;

  try {
    const scenario = Math.random();

    if (scenario < 0.35) {
      const batchCount = 100;
      const largeMetadata = 'x'.repeat(1024);
      const batch = [];
      for (let i = 0; i < batchCount; i++) {
        batch.push({
          userId: userId,
          merchantId: merchantId,
          amount: Math.round(Math.random() * 500 * 100) / 100 + 0.01,
          type: 'DEBIT',
        });
      }

      const batchRes = http.post(`${BASE_URL}/v1/payments/batch`, JSON.stringify(batch), {
        headers: { 'Content-Type': 'application/json' },
      });

      largePayloadDuration.add(batchRes.timings.duration);
      check(batchRes, { 'batch ok': (r) => r.status >= 200 && r.status < 300 }) || memErrors.add(1);

    } else if (scenario < 0.60) {
      const responses = http.batch([
        ['POST', `${BASE_URL}/v1/payments/wallet-transfer`, JSON.stringify({
          walletId: walletId,
          merchantId: merchantId,
          amount: 0.01,
        }), { headers: { 'Content-Type': 'application/json' } }],
        ['POST', `${BASE_URL}/v1/payments/wallet-transfer`, JSON.stringify({
          walletId: walletId,
          merchantId: merchantId,
          amount: 0.01,
        }), { headers: { 'Content-Type': 'application/json' } }],
        ['POST', `${BASE_URL}/v1/payments/wallet-transfer`, JSON.stringify({
          walletId: walletId,
          merchantId: merchantId,
          amount: 0.01,
        }), { headers: { 'Content-Type': 'application/json' } }],
        ['POST', `${BASE_URL}/v1/payments/wallet-transfer`, JSON.stringify({
          walletId: walletId,
          merchantId: merchantId,
          amount: 0.01,
        }), { headers: { 'Content-Type': 'application/json' } }],
        ['POST', `${BASE_URL}/v1/payments/wallet-transfer`, JSON.stringify({
          walletId: walletId,
          merchantId: merchantId,
          amount: 0.01,
        }), { headers: { 'Content-Type': 'application/json' } }],
        ['POST', `${BASE_URL}/v1/payments/wallet-transfer`, JSON.stringify({
          walletId: walletId,
          merchantId: merchantId,
          amount: 0.01,
        }), { headers: { 'Content-Type': 'application/json' } }],
        ['POST', `${BASE_URL}/v1/payments/wallet-transfer`, JSON.stringify({
          walletId: walletId,
          merchantId: merchantId,
          amount: 0.01,
        }), { headers: { 'Content-Type': 'application/json' } }],
        ['POST', `${BASE_URL}/v1/payments/wallet-transfer`, JSON.stringify({
          walletId: walletId,
          merchantId: merchantId,
          amount: 0.01,
        }), { headers: { 'Content-Type': 'application/json' } }],
      ]);

      concurrentHolding.add(responses.length);
      const allOk = responses.every((r) => r.status >= 200 && r.status < 300);
      if (!allOk) memErrors.add(1);

    } else if (scenario < 0.80) {
      const listRes = http.get(`${BASE_URL}/v1/payments/user/${userId}?status=SUCCESS&limit=100`);

      check(listRes, { 'list ok': (r) => r.status >= 200 && r.status < 300 }) || memErrors.add(1);

    } else if (scenario < 0.95) {
      const topUpRes = http.post(`${BASE_URL}/v1/payments/wallets/${walletId}/topup`, JSON.stringify({
        amount: 1000.0,
      }), { headers: { 'Content-Type': 'application/json' } });

      check(topUpRes, { 'topup ok': (r) => r.status >= 200 && r.status < 300 }) || memErrors.add(1);

    } else {
      const searchRes = http.get(`${BASE_URL}/v1/payments/search?status=SUCCESS&page=0&size=100`);

      check(searchRes, { 'search ok': (r) => r.status >= 200 && r.status < 300 }) || memErrors.add(1);
    }
  } catch (e) {
    memErrors.add(1);
    console.error(`Memory stress error: ${e.message}`);
  }

  activeVUs.add(-1);
  sleep(0.1);
}

export function teardown(data) {
  teardownTestEntities(BASE_URL, data);
}

export function handleSummary(data) {
  printScalingBox('MEMORY STRESS TEST', {
    'Target VUs': TARGET_VUS,
    'Total Requests': data.metrics.http_reqs.values.count,
    'P95 Duration': `${data.metrics.http_req_duration.values['p(95)']?.toFixed(2) || 'N/A'}ms`,
    'Error Rate': `${(data.metrics.mem_stress_errors?.values.rate * 100 || 0).toFixed(2)}%`,
    'Avg Large Payload': `${data.metrics.mem_stress_large_payload_duration?.values.avg?.toFixed(2) || 'N/A'}ms`,
  });

  return {
    'results/memory-stress-summary.json': JSON.stringify(data, null, 2),
  };
}
