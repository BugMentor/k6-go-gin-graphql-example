import { check, sleep } from 'k6';
import { Trend, Rate, Counter, Gauge } from 'k6/metrics';
import {
  setupTestEntities,
  teardownTestEntities,
  buildRampStages,
  printScalingBox,
  refuelWalletIfNeeded,
  executeGraphQL,
  generateUUID,
} from './_shared.js';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const TARGET_VUS = parseInt(__ENV.TARGET_VUS || '50');
const TEST_DURATION = __ENV.TEST_DURATION || '5m';

const errors = new Rate('errors');
const walletTransferLatency = new Trend('wallet_transfer_latency');
const paymentCreateLatency = new Trend('payment_create_latency');
const graphqlLatency = new Trend('graphql_latency');
const topUpLatency = new Trend('top_up_latency');
const searchLatency = new Trend('search_latency');
const concurrentVUs = new Gauge('concurrent_vus');

export const options = {
  stages: buildRampStages(TARGET_VUS, TARGET_VUS > 50 ? 10 : 5, TARGET_VUS > 50 ? 30 : 15, 30),
  thresholds: {
    http_req_duration: ['p(95)<2000', 'p(99)<5000'],
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

  const scenario = Math.random();
  const walletId = data.walletId;
  const merchantId = data.merchantId;
  const userId = data.userId;

  try {
    if (scenario < 0.35) {
      const transferRes = http.post(`${BASE_URL}/v1/payments/wallet-transfer`, JSON.stringify({
        walletId: walletId,
        merchantId: merchantId,
        amount: Math.round(Math.random() * 100 * 100) / 100 + 0.01,
      }), { headers: { 'Content-Type': 'application/json' } });

      walletTransferLatency.add(transferRes.timings.duration);
      check(transferRes, {
        'transfer 2xx': (r) => r.status >= 200 && r.status < 300,
      }) || errors.add(1);

    } else if (scenario < 0.55) {
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

    } else if (scenario < 0.70) {
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

    } else if (scenario < 0.80) {
      const topUpRes = http.post(`${BASE_URL}/v1/payments/wallets/${walletId}/topup`, JSON.stringify({
        amount: Math.round(Math.random() * 1000 * 100) / 100 + 1,
      }), { headers: { 'Content-Type': 'application/json' } });

      topUpLatency.add(topUpRes.timings.duration);
      check(topUpRes, {
        'topup 2xx': (r) => r.status >= 200 && r.status < 300,
      }) || errors.add(1);

    } else if (scenario < 0.90) {
      const searchRes = http.get(`${BASE_URL}/v1/payments/search?status=SUCCESS&page=0&size=10`);

      searchLatency.add(searchRes.timings.duration);
      check(searchRes, {
        'search 2xx': (r) => r.status >= 200 && r.status < 300,
      }) || errors.add(1);

    } else {
      const getRes = http.get(`${BASE_URL}/v1/payments/user/${userId}`);
      check(getRes, {
        'get 2xx': (r) => r.status >= 200 && r.status < 300,
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
  printScalingBox('BASELINE LOAD TEST', {
    'Total Requests': data.metrics.http_reqs.values.count,
    'Request Rate': `${data.metrics.http_reqs.values.rate.toFixed(2)}/s`,
    'P95 Duration': `${data.metrics.http_req_duration.values['p(95)']?.toFixed(2) || 'N/A'}ms`,
    'P99 Duration': `${data.metrics.http_req_duration.values['p(99)']?.toFixed(2) || 'N/A'}ms`,
    'Error Rate': `${(data.metrics.errors.values.rate * 100).toFixed(2)}%`,
    'Avg Transfer': `${data.metrics.wallet_transfer_latency?.values.avg?.toFixed(2) || 'N/A'}ms`,
    'Avg GraphQL': `${data.metrics.graphql_latency?.values.avg?.toFixed(2) || 'N/A'}ms`,
  });

  return {
    'results/load-summary.json': JSON.stringify(data, null, 2),
  };
}
