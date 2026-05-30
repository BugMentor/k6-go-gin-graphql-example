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
const TARGET_VUS = parseInt(__ENV.TARGET_VUS || '300');

const errors = new Rate('errors');
const walletTransferLatency = new Trend('cpu_stress_wallet_transfer_duration', true);
const batchLatency = new Trend('cpu_stress_batch_duration', true);
const searchLatency = new Trend('cpu_stress_search_duration', true);
const summaryLatency = new Trend('cpu_stress_summary_duration', true);
const paymentCreateLatency = new Trend('cpu_stress_payment_create_duration', true);
const concurrentVUs = new Gauge('concurrent_vus');

export const options = {
  stages: [
    { duration: '1m', target: Math.floor(TARGET_VUS * 0.2) },
    { duration: '2m', target: Math.floor(TARGET_VUS * 0.5) },
    { duration: '3m', target: TARGET_VUS },
    { duration: '5m', target: TARGET_VUS },
    { duration: '1m', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<8000', 'p(99)<20000'],
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
    refuelWalletIfNeeded(BASE_URL, walletId, 50000);

    const scenario = Math.random();

    if (scenario < 0.30) {
      const mutation = `
        mutation($walletId: String!, $merchantId: String!, $amount: Float!) {
          walletTransfer(walletId: $walletId, merchantId: $merchantId, amount: $amount) {
            id
            status
          }
        }
      `;
      const start = Date.now();
      const responses = [];
      for (let i = 0; i < 3; i++) {
        const res = executeGraphQL(mutation, {
          walletId: walletId,
          merchantId: merchantId,
          amount: Math.round(Math.random() * 100 * 100) / 100 + 0.01,
        });
        responses.push(res);
      }
      walletTransferLatency.add((Date.now() - start) / responses.length);
      responses.forEach((r) => {
        check(r, { 'wallet transfer ok': (res) => res.status === 200 }) || errors.add(1);
      });

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
      const start = Date.now();
      const res = executeGraphQL(mutation, { payments });
      batchLatency.add(Date.now() - start);
      check(res, { 'batch ok': (r) => r.status === 200 }) || errors.add(1);

    } else if (scenario < 0.75) {
      const query = `
        query($minAmount: Float, $maxAmount: Float, $status: String, $page: Int, $size: Int) {
          searchPayments(minAmount: $minAmount, maxAmount: $maxAmount, status: $status, page: $page, size: $size) {
            id
            amount
            status
            createdAt
          }
        }
      `;
      const res = executeGraphQL(query, {
        minAmount: 1,
        maxAmount: 500,
        status: 'SUCCESS',
        page: 0,
        size: 10,
      });
      searchLatency.add(res.timings.duration);
      check(res, { 'search ok': (r) => r.status === 200 }) || errors.add(1);

    } else if (scenario < 0.90) {
      const query = `
        query($startDate: String!, $endDate: String!) {
          paymentSummary(startDate: $startDate, endDate: $endDate) {
            totalsByStatus {
              status
              total
            }
          }
        }
      `;
      const res = executeGraphQL(query, {
        startDate: '2020-01-01T00:00:00Z',
        endDate: '2030-12-31T23:59:59Z',
      });
      summaryLatency.add(res.timings.duration);
      check(res, { 'summary ok': (r) => r.status === 200 }) || errors.add(1);

    } else {
      const mutation = `
        mutation($input: ProcessPaymentInput!) {
          processPayment(input: $input) {
            id
            status
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
      paymentCreateLatency.add(res.timings.duration);
      check(res, { 'payment create ok': (r) => r.status === 200 }) || errors.add(1);
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
  printScalingBox('CPU STRESS TEST - GO', {
    'Target VUs': TARGET_VUS,
    'Total Requests': data.metrics.http_reqs.values.count,
    'Error Rate': `${(data.metrics.errors.values.rate * 100).toFixed(2)}%`,
    'P95 Duration': `${data.metrics.http_req_duration.values['p(95)']?.toFixed(2) || 'N/A'}ms`,
  });

  return {
    'results/go-cpu-stress-summary.json': JSON.stringify(data, null, 2),
  };
}
