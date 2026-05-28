import { check, sleep } from 'k6';
import http from 'k6/http';
import { Trend, Rate, Gauge } from 'k6/metrics';
import {
  setupTestEntities,
  teardownTestEntities,
  buildRampStages,
  printScalingBox,
  executeGraphQL,
} from './_shared.js';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const TARGET_VUS = parseInt(__ENV.TARGET_VUS || '100');
const SCALE_STEPS = parseInt(__ENV.SCALE_STEPS || '8');
const STEP_DURATION = parseInt(__ENV.STEP_DURATION || '30');
const COOLDOWN_DURATION = parseInt(__ENV.COOLDOWN_DURATION || '60');

const cpuErrors = new Rate('cpu_stress_errors');
const walletTransferDuration = new Trend('cpu_stress_wallet_transfer_duration');
const batchDuration = new Trend('cpu_stress_batch_duration');
const searchDuration = new Trend('cpu_stress_search_duration');
const activeVUs = new Gauge('cpu_stress_active_vus');

export const options = {
  stages: buildRampStages(TARGET_VUS, SCALE_STEPS, STEP_DURATION, COOLDOWN_DURATION),
  thresholds: {
    http_req_duration: ['p(95)<3000', 'p(99)<5000'],
    http_req_failed: ['rate<0.10'],
    cpu_stress_wallet_transfer_duration: ['p(95)<4000'],
    cpu_stress_batch_duration: ['p(95)<5000'],
    cpu_stress_search_duration: ['p(95)<3000'],
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

    if (scenario < 0.30) {
      const transferRes = http.post(`${BASE_URL}/v1/payments/wallet-transfer`, JSON.stringify({
        walletId: walletId,
        merchantId: merchantId,
        amount: Math.round(Math.random() * 50 * 100) / 100 + 0.01,
      }), { headers: { 'Content-Type': 'application/json' } });

      walletTransferDuration.add(transferRes.timings.duration);
      check(transferRes, { 'transfer ok': (r) => r.status >= 200 && r.status < 300 }) || cpuErrors.add(1);

    } else if (scenario < 0.55) {
      const batchCount = 50 + Math.floor(Math.random() * 30);
      const batch = [];
      for (let i = 0; i < batchCount; i++) {
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

      batchDuration.add(batchRes.timings.duration);
      check(batchRes, { 'batch ok': (r) => r.status >= 200 && r.status < 300 }) || cpuErrors.add(1);

    } else if (scenario < 0.75) {
      const query = `
        query($minAmount: Float, $status: String, $page: Int!, $size: Int!) {
          searchPayments(minAmount: $minAmount, status: $status, page: $page, size: $size) {
            id
            amount
            status
            createdAt
          }
        }
      `;
      const res = executeGraphQL(query, {
        minAmount: Math.random() * 100,
        status: 'SUCCESS',
        page: 0,
        size: 20,
      });
      searchDuration.add(res.timings.duration);
      check(res, { 'search ok': (r) => r.status === 200 }) || cpuErrors.add(1);

    } else if (scenario < 0.90) {
      const reportRes = http.get(
        `${BASE_URL}/v1/payments/reports/summary?startDate=${encodeURIComponent('2020-01-01T00:00:00Z')}&endDate=${encodeURIComponent('2030-12-31T23:59:59Z')}`
      );
      check(reportRes, { 'report ok': (r) => r.status >= 200 && r.status < 300 }) || cpuErrors.add(1);

    } else {
      const mutation = `
        mutation($input: ProcessPaymentInput!) {
          processPayment(input: $input) {
            id
            status
            createdAt
          }
        }
      `;
      const res = executeGraphQL(mutation, {
        input: {
          userId: userId,
          merchantId: merchantId,
          amount: Math.round(Math.random() * 200 * 100) / 100 + 0.01,
          type: 'DEBIT',
        },
      });
      check(res, { 'graphql ok': (r) => r.status === 200 }) || cpuErrors.add(1);
    }
  } catch (e) {
    cpuErrors.add(1);
    console.error(`CPU stress error: ${e.message}`);
  }

  activeVUs.add(-1);
  sleep(0.05);
}

export function teardown(data) {
  teardownTestEntities(BASE_URL, data);
}

export function handleSummary(data) {
  printScalingBox('CPU STRESS TEST', {
    'Target VUs': TARGET_VUS,
    'Total Requests': data.metrics.http_reqs.values.count,
    'P95 Duration': `${data.metrics.http_req_duration.values['p(95)']?.toFixed(2) || 'N/A'}ms`,
    'Error Rate': `${(data.metrics.cpu_stress_errors?.values.rate * 100 || 0).toFixed(2)}%`,
    'Avg Transfer': `${data.metrics.cpu_stress_wallet_transfer_duration?.values.avg?.toFixed(2) || 'N/A'}ms`,
    'Avg Batch': `${data.metrics.cpu_stress_batch_duration?.values.avg?.toFixed(2) || 'N/A'}ms`,
  });

  return {
    'results/cpu-stress-summary.json': JSON.stringify(data, null, 2),
  };
}
