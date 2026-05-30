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
const TARGET_VUS = parseInt(__ENV.TARGET_VUS || '100');

const errors = new Rate('errors');
const walletTransferLatency = new Trend('wallet_transfer_latency');
const batchLatency = new Trend('batch_latency');
const topUpLatency = new Trend('top_up_latency');
const searchLatency = new Trend('search_latency');
const paymentCreateLatency = new Trend('payment_create_latency');
const concurrentVUs = new Gauge('concurrent_vus');

export const options = {
  stages: [
    { duration: '30s', target: Math.floor(TARGET_VUS * 0.2) },
    { duration: '1m', target: Math.floor(TARGET_VUS * 0.5) },
    { duration: '2m', target: TARGET_VUS },
    { duration: '5m', target: TARGET_VUS },
    { duration: '1m', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<10000', 'p(99)<30000'],
    http_req_failed: ['rate<0.10'],
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
    refuelWalletIfNeeded(BASE_URL, walletId, 100000);

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

      for (let i = 0; i < 8; i++) {
        const res = executeGraphQL(mutation, {
          walletId: walletId,
          merchantId: merchantId,
          amount: Math.round(Math.random() * 100 * 100) / 100 + 0.01,
        });
        walletTransferLatency.add(res.timings.duration);
        check(res, { 'wallet transfer ok': (r) => r.status === 200 }) || errors.add(1);
      }

    } else if (scenario < 0.60) {
      const mutation = `
        mutation($payments: [ProcessPaymentInput!]!) {
          processBatchPayments(payments: $payments)
        }
      `;

      const payments = [];
      for (let i = 0; i < 20; i++) {
        payments.push({
          userId: userId,
          merchantId: merchantId,
          amount: Math.round(Math.random() * 100 * 100) / 100 + 0.01,
          type: 'DEBIT',
        });
      }
      const res = executeGraphQL(mutation, { payments });
      batchLatency.add(res.timings.duration);
      check(res, { 'batch ok': (r) => r.status === 200 }) || errors.add(1);

    } else if (scenario < 0.80) {
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
        amount: Math.round(Math.random() * 5000 * 100) / 100 + 1,
      });
      topUpLatency.add(res.timings.duration);
      check(res, { 'topup ok': (r) => r.status === 200 }) || errors.add(1);

    } else if (scenario < 0.90) {
      const query = `
        query($userId: String!, $status: String, $limit: Int) {
          payments(userId: $userId, status: $status, limit: $limit) {
            id
            status
            amount
          }
        }
      `;
      const res = executeGraphQL(query, { userId: userId, status: 'SUCCESS', limit: 100 });
      searchLatency.add(res.timings.duration);
      check(res, { 'user payments ok': (r) => r.status === 200 }) || errors.add(1);

    } else {
      const query = `
        query($minAmount: Float, $maxAmount: Float, $status: String, $page: Int, $size: Int) {
          searchPayments(minAmount: $minAmount, maxAmount: $maxAmount, status: $status, page: $page, size: $size) {
            id
            amount
            status
          }
        }
      `;
      const res = executeGraphQL(query, {
        minAmount: 10,
        maxAmount: 1000,
        status: 'SUCCESS',
        page: 0,
        size: 50,
      });
      searchLatency.add(res.timings.duration);
      check(res, { 'search ok': (r) => r.status === 200 }) || errors.add(1);
    }
  } catch (e) {
    errors.add(1);
    console.error(`Error: ${e.message}`);
  }

  concurrentVUs.add(-1);
  sleep(0.05);
}

export function teardown(data) {
  teardownTestEntities(BASE_URL, data);
}

export function handleSummary(data) {
  printScalingBox('MEMORY STRESS TEST - GO', {
    'Target VUs': TARGET_VUS,
    'Total Requests': data.metrics.http_reqs.values.count,
    'Error Rate': `${(data.metrics.errors.values.rate * 100).toFixed(2)}%`,
    'P95 Duration': `${data.metrics.http_req_duration.values['p(95)']?.toFixed(2) || 'N/A'}ms`,
    'P99 Duration': `${data.metrics.http_req_duration.values['p(99)']?.toFixed(2) || 'N/A'}ms`,
  });

  return {
    'results/go-memory-stress-summary.json': JSON.stringify(data, null, 2),
  };
}
