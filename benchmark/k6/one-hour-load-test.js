import { check, sleep } from 'k6';
import http from 'k6/http';
import { Trend, Rate, Gauge } from 'k6/metrics';
import {
  setupTestEntities,
  teardownTestEntities,
  printScalingBox,
  refuelWalletIfNeeded,
  executeGraphQL,
} from './_shared.js';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const TARGET_VUS = parseInt(__ENV.TARGET_VUS || '50');

const errors = new Rate('errors');
const walletTransferLatency = new Trend('wallet_transfer_latency');
const paymentCreateLatency = new Trend('payment_create_latency');
const graphqlLatency = new Trend('graphql_latency');
const topUpLatency = new Trend('top_up_latency');
const searchLatency = new Trend('search_latency');
const restGetLatency = new Trend('rest_get_latency');
const batchLatency = new Trend('batch_latency');
const concurrentVUs = new Gauge('concurrent_vus');

export const options = {
  stages: [
    { target: Math.floor(TARGET_VUS * 0.3), duration: '2m' },
    { target: Math.floor(TARGET_VUS * 0.6), duration: '3m' },
    { target: TARGET_VUS, duration: '5m' },
    { target: TARGET_VUS, duration: '60m' },
    { target: 0, duration: '2m' },
  ],
  thresholds: {
    http_req_duration: ['p(95)<3000', 'p(99)<5000'],
    http_req_failed: ['rate<0.05'],
    wallet_transfer_latency: ['p(95)<3000'],
    payment_create_latency: ['p(95)<2000'],
    graphql_latency: ['p(95)<4000'],
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
    refuelWalletIfNeeded(BASE_URL, walletId, 1000);

    const scenario = Math.random();

    if (scenario < 0.25) {
      const transferRes = http.post(`${BASE_URL}/v1/payments/wallet-transfer`, JSON.stringify({
        walletId: walletId,
        merchantId: merchantId,
        amount: Math.round(Math.random() * 100 * 100) / 100 + 0.01,
      }), { headers: { 'Content-Type': 'application/json' } });

      walletTransferLatency.add(transferRes.timings.duration);
      check(transferRes, {
        'transfer 2xx': (r) => r.status >= 200 && r.status < 300,
      }) || errors.add(1);

    } else if (scenario < 0.40) {
      const paymentRes = http.post(`${BASE_URL}/v1/payments`, JSON.stringify({
        userId: userId,
        merchantId: merchantId,
        amount: Math.round(Math.random() * 500 * 100) / 100 + 0.01,
        type: 'DEBIT',
      }), { headers: { 'Content-Type': 'application/json' } });

      paymentCreateLatency.add(paymentRes.timings.duration);
      check(paymentRes, {
        'payment 2xx': (r) => r.status >= 200 && r.status < 300,
      }) || errors.add(1);

    } else if (scenario < 0.55) {
      const query = `
        query($userId: String!, $limit: Int) {
          payments(userId: $userId, limit: $limit) {
            id
            amount
            status
            createdAt
          }
        }
      `;
      const res = executeGraphQL(query, { userId: data.userId, limit: 5 });
      graphqlLatency.add(res.timings.duration);
      check(res, { 'graphql 2xx': (r) => r.status === 200 }) || errors.add(1);

    } else if (scenario < 0.65) {
      const topUpRes = http.post(`${BASE_URL}/v1/payments/wallets/${walletId}/topup`, JSON.stringify({
        amount: Math.round(Math.random() * 1000 * 100) / 100 + 1,
      }), { headers: { 'Content-Type': 'application/json' } });

      topUpLatency.add(topUpRes.timings.duration);
      check(topUpRes, {
        'topup 2xx': (r) => r.status >= 200 && r.status < 300,
      }) || errors.add(1);

    } else if (scenario < 0.75) {
      const searchRes = http.get(`${BASE_URL}/v1/payments/search?status=SUCCESS&page=0&size=10`);

      searchLatency.add(searchRes.timings.duration);
      check(searchRes, {
        'search 2xx': (r) => r.status >= 200 && r.status < 300,
      }) || errors.add(1);

    } else if (scenario < 0.85) {
      const getRes = http.get(`${BASE_URL}/v1/payments/user/${userId}`);
      restGetLatency.add(getRes.timings.duration);
      check(getRes, {
        'get 2xx': (r) => r.status >= 200 && r.status < 300,
      }) || errors.add(1);

    } else if (scenario < 0.95) {
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
      graphqlLatency.add(res.timings.duration);
      check(res, { 'graphql ok': (r) => r.status === 200 }) || errors.add(1);

    } else {
      const summaryRes = http.get(
        `${BASE_URL}/v1/payments/reports/summary?startDate=2020-01-01T00:00:00Z&endDate=2030-12-31T23:59:59Z`
      );
      check(summaryRes, {
        'summary 2xx': (r) => r.status >= 200 && r.status < 300,
      }) || errors.add(1);
    }
  } catch (e) {
    errors.add(1);
    console.error(`Error in VU iteration: ${e.message}`);
  }

  concurrentVUs.add(-1);
  sleep(0.1);
}

export function teardown(data) {
  teardownTestEntities(BASE_URL, data);
}

export function handleSummary(data) {
  printScalingBox('ONE HOUR LOAD TEST - GO', {
    'Target VUs': TARGET_VUS,
    'Total Requests': data.metrics.http_reqs.values.count,
    'Request Rate': `${data.metrics.http_reqs.values.rate.toFixed(2)}/s`,
    'P95 Duration': `${data.metrics.http_req_duration.values['p(95)']?.toFixed(2) || 'N/A'}ms`,
    'P99 Duration': `${data.metrics.http_req_duration.values['p(99)']?.toFixed(2) || 'N/A'}ms`,
    'Error Rate': `${(data.metrics.errors.values.rate * 100).toFixed(2)}%`,
    'Avg Transfer': `${data.metrics.wallet_transfer_latency?.values.avg?.toFixed(2) || 'N/A'}ms`,
    'Avg Payment': `${data.metrics.payment_create_latency?.values.avg?.toFixed(2) || 'N/A'}ms`,
    'Avg GraphQL': `${data.metrics.graphql_latency?.values.avg?.toFixed(2) || 'N/A'}ms`,
    'Avg TopUp': `${data.metrics.top_up_latency?.values.avg?.toFixed(2) || 'N/A'}ms`,
    'Avg Search': `${data.metrics.search_latency?.values.avg?.toFixed(2) || 'N/A'}ms`,
    'Avg Batch': `${data.metrics.batch_latency?.values.avg?.toFixed(2) || 'N/A'}ms`,
  });

  return {
    'results/go-1h-summary.json': JSON.stringify(data, null, 2),
  };
}
