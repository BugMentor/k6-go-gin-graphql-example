import { check, sleep } from 'k6';
import http from 'k6/http';
import { Trend, Rate, Gauge } from 'k6/metrics';
import {
  setupTestEntities,
  teardownTestEntities,
  printScalingBox,
  executeGraphQL,
} from './_shared.js';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const TARGET_VUS = parseInt(__ENV.TARGET_VUS || '5000');
const RAMP_TIME = __ENV.RAMP_TIME || '15m';
const HOLD_TIME = __ENV.HOLD_TIME || '45m';

const maxErrors = new Rate('max_capacity_errors');
const maxWalletTransfer = new Trend('max_capacity_wallet_transfer');
const maxBatchDuration = new Trend('max_capacity_batch_duration');
const maxGraphQLDuration = new Trend('max_capacity_graphql_duration');
const activeVUs = new Gauge('max_capacity_active_vus');

export const options = {
  executor: 'ramping-vus',
  stages: [
    { target: 50, duration: '2m' },
    { target: 200, duration: '3m' },
    { target: 500, duration: '3m' },
    { target: 1000, duration: '3m' },
    { target: 2000, duration: '2m' },
    { target: 3000, duration: '1m' },
    { target: 4000, duration: '30s' },
    { target: 5000, duration: '30s' },
    { target: 5000, duration: '45m' },
    { target: 0, duration: '5m' },
  ],
  thresholds: {
    http_req_duration: ['p(95)<10000', 'p(99)<20000'],
    http_req_failed: ['rate<0.25'],
    max_capacity_wallet_transfer: ['p(95)<8000'],
    max_capacity_query_duration: ['p(95)<10000'],
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
      const transferRes = http.post(`${BASE_URL}/v1/payments/wallet-transfer`, JSON.stringify({
        walletId: walletId,
        merchantId: merchantId,
        amount: Math.round(Math.random() * 50 * 100) / 100 + 0.01,
      }), { headers: { 'Content-Type': 'application/json' } });

      maxWalletTransfer.add(transferRes.timings.duration);
      check(transferRes, { 'transfer ok': (r) => r.status >= 200 && r.status < 300 }) || maxErrors.add(1);

    } else if (scenario < 0.55) {
      const batchSize = 20 + Math.floor(Math.random() * 10);
      const batch = [];
      for (let i = 0; i < batchSize; i++) {
        batch.push({
          userId: userId,
          merchantId: merchantId,
          amount: Math.round(Math.random() * 100 * 100) / 100 + 0.01,
          type: 'DEBIT',
        });
      }

      const batchRes = http.post(`${BASE_URL}/v1/payments/batch`, JSON.stringify(batch), {
        headers: { 'Content-Type': 'application/json' },
      });

      maxBatchDuration.add(batchRes.timings.duration);
      check(batchRes, { 'batch ok': (r) => r.status >= 200 && r.status < 300 }) || maxErrors.add(1);

    } else if (scenario < 0.70) {
      const mutation = `
        mutation($walletId: String!, $merchantId: String!, $amount: Float!) {
          walletTransfer(walletId: $walletId, merchantId: $merchantId, amount: $amount) {
            id
            status
          }
        }
      `;
      const res = executeGraphQL(mutation, {
        walletId: walletId,
        merchantId: merchantId,
        amount: Math.round(Math.random() * 20 * 100) / 100 + 0.01,
      });
      maxGraphQLDuration.add(res.timings.duration);
      check(res, { 'graphql ok': (r) => r.status === 200 }) || maxErrors.add(1);

    } else if (scenario < 0.85) {
      const topUpRes = http.post(`${BASE_URL}/v1/payments/wallets/${walletId}/topup`, JSON.stringify({
        amount: 1000.0,
      }), { headers: { 'Content-Type': 'application/json' } });

      check(topUpRes, { 'topup ok': (r) => r.status >= 200 && r.status < 300 }) || maxErrors.add(1);

    } else {
      const getRes = http.get(`${BASE_URL}/v1/payments/user/${userId}?limit=5`);
      check(getRes, { 'get ok': (r) => r.status >= 200 && r.status < 300 }) || maxErrors.add(1);
    }
  } catch (e) {
    maxErrors.add(1);
    console.error(`Max capacity error: ${e.message}`);
  }

  activeVUs.add(-1);
  sleep(0.05);
}

export function teardown(data) {
  teardownTestEntities(BASE_URL, data);
}

export function handleSummary(data) {
  printScalingBox('MAX CAPACITY TEST', {
    'Target VUs': TARGET_VUS,
    'Total Requests': data.metrics.http_reqs.values.count,
    'P95 Duration': `${data.metrics.http_req_duration.values['p(95)']?.toFixed(2) || 'N/A'}ms`,
    'Error Rate': `${(data.metrics.max_capacity_errors?.values.rate * 100 || 0).toFixed(2)}%`,
    'Avg Transfer': `${data.metrics.max_capacity_wallet_transfer?.values.avg?.toFixed(2) || 'N/A'}ms`,
    'Avg GraphQL': `${data.metrics.max_capacity_graphql_duration?.values.avg?.toFixed(2) || 'N/A'}ms`,
  });

  return {
    'results/max-capacity-summary.json': JSON.stringify(data, null, 2),
  };
}
