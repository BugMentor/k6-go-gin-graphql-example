import { check, sleep } from 'k6';
import { Trend, Rate, Gauge } from 'k6/metrics';
import {
  setupTestEntities,
  teardownTestEntities,
  printScalingBox,
  refuelWalletIfNeeded,
  executeGraphQL,
} from './_shared.js';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const TARGET_VUS = parseInt(__ENV.TARGET_VUS || '500');

const errors = new Rate('errors');
const walletTransferLatency = new Trend('wallet_transfer_latency', true);
const batchLatency = new Trend('batch_latency', true);
const topUpLatency = new Trend('top_up_latency', true);
const paymentCreateLatency = new Trend('payment_create_latency', true);
const restGetLatency = new Trend('rest_get_latency', true);
const concurrentVUs = new Gauge('concurrent_vus');

export const options = {
  stages: [
    { duration: '30s', target: Math.floor(TARGET_VUS * 0.1) },
    { duration: '30s', target: Math.floor(TARGET_VUS * 0.3) },
    { duration: '1m', target: Math.floor(TARGET_VUS * 0.5) },
    { duration: '1m', target: Math.floor(TARGET_VUS * 0.7) },
    { duration: '2m', target: TARGET_VUS },
    { duration: '5m', target: TARGET_VUS },
    { duration: '1m', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<15000', 'p(99)<30000'],
    http_req_failed: ['rate<0.15'],
  },
};

export function setup() {
  return setupTestEntities(BASE_URL);
}

export default function (data) {
  concurrentVUs.add(1);

  const walletId = data.walletId;
  const merchantId = data.merchantId;
  const userId = data.userId;

  try {
    refuelWalletIfNeeded(BASE_URL, walletId, 50000);

    const scenario = Math.random();

    if (scenario < 0.35) {
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
        amount: Math.round(Math.random() * 100 * 100) / 100 + 0.01,
      });
      walletTransferLatency.add(res.timings.duration);
      check(res, { 'wallet transfer ok': (r) => r.status === 200 }) || errors.add(1);

    } else if (scenario < 0.55) {
      const mutation = `
        mutation($payments: [ProcessPaymentInput!]!) {
          processBatchPayments(payments: $payments)
        }
      `;
      const payments = [
        {
          userId: userId,
          merchantId: merchantId,
          amount: Math.round(Math.random() * 100 * 100) / 100 + 0.01,
          type: 'DEBIT',
        },
      ];
      const res = executeGraphQL(mutation, { payments });
      batchLatency.add(res.timings.duration);
      check(res, { 'batch ok': (r) => r.status === 200 }) || errors.add(1);

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
        amount: Math.round(Math.random() * 50 * 100) / 100 + 0.01,
      });
      walletTransferLatency.add(res.timings.duration);
      check(res, { 'wallet transfer 2 ok': (r) => r.status === 200 }) || errors.add(1);

    } else if (scenario < 0.85) {
      const mutation = `
        mutation($walletId: String!, $amount: Float!) {
          topUpWallet(walletId: $walletId, amount: $amount) {
            id
            balance
          }
        }
      `;
      const res = executeGraphQL(mutation, {
        walletId: walletId,
        amount: Math.round(Math.random() * 1000 * 100) / 100 + 1,
      });
      topUpLatency.add(res.timings.duration);
      check(res, { 'topup ok': (r) => r.status === 200 }) || errors.add(1);

    } else {
      const query = `
        query($userId: String!, $limit: Int) {
          payments(userId: $userId, limit: $limit) {
            id
            amount
            status
          }
        }
      `;
      const res = executeGraphQL(query, { userId: userId, limit: 5 });
      restGetLatency.add(res.timings.duration);
      check(res, { 'list payments ok': (r) => r.status === 200 }) || errors.add(1);
    }
  } catch (e) {
    errors.add(1);
    console.error(`Error: ${e.message}`);
  }

  concurrentVUs.add(-1);
  sleep(0.1);
}

export function teardown(data) {
  teardownTestEntities(BASE_URL, data);
}

export function handleSummary(data) {
  printScalingBox('MAX CAPACITY TEST - GO', {
    'Target VUs': TARGET_VUS,
    'Total Requests': data.metrics.http_reqs.values.count,
    'Error Rate': `${(data.metrics.errors.values.rate * 100).toFixed(2)}%`,
    'P95 Duration': `${data.metrics.http_req_duration.values['p(95)']?.toFixed(2) || 'N/A'}ms`,
  });

  return {
    'results/go-max-capacity-summary.json': JSON.stringify(data, null, 2),
  };
}
