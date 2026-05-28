import { check, sleep } from 'k6';
import http from 'k6/http';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const GRAPHQL_URL = `${BASE_URL}/graphql`;

export function generateUUID() {
  return uuidv4();
}

export function setupTestEntities(baseUrl) {
  const prefix = generateUUID().substring(0, 8);

  const userRes = http.post(`${baseUrl}/v1/users`, JSON.stringify({
    email: `loadtest-${prefix}@example.com`,
    fullName: `Load Test User ${prefix}`,
    status: 'ACTIVE',
  }), { headers: { 'Content-Type': 'application/json' } });

  const user = JSON.parse(userRes.body);
  const userId = user.id;

  const merchantRes = http.post(`${baseUrl}/v1/merchants`, JSON.stringify({
    name: `Merchant-${prefix}`,
    apiKey: `api-key-${prefix}`,
  }), { headers: { 'Content-Type': 'application/json' } });

  const merchant = JSON.parse(merchantRes.body);
  const merchantId = merchant.id;

  const walletRes = http.post(`${baseUrl}/v1/wallets`, JSON.stringify({
    userId: userId,
    balance: 999999.99,
    currency: 'USD',
  }), { headers: { 'Content-Type': 'application/json' } });

  const wallet = JSON.parse(walletRes.body);
  const walletId = wallet.id;

  return { userId, merchantId, walletId, prefix };
}

export function teardownTestEntities(baseUrl, data) {
  if (data) {
    http.del(`${baseUrl}/v1/users/${data.userId}`);
    http.del(`${baseUrl}/v1/merchants/${data.merchantId}`);
  }
}

export function buildRampStages(targetVUs, scaleSteps, stepDuration, cooldownDuration) {
  const stages = [];
  const stepSize = Math.ceil(targetVUs / scaleSteps);
  for (let i = 1; i <= scaleSteps; i++) {
    stages.push({ target: stepSize * i, duration: `${stepDuration}s` });
  }
  if (cooldownDuration > 0) {
    stages.push({ target: 0, duration: `${cooldownDuration}s` });
  }
  return stages;
}

export function printScalingBox(title, metrics) {
  console.log('');
  console.log('='.repeat(60));
  console.log(`  ${title}`);
  console.log('='.repeat(60));
  for (const [key, value] of Object.entries(metrics)) {
    console.log(`  ${key.padEnd(25)} ${value}`);
  }
  console.log('='.repeat(60));
}

export function refuelWalletIfNeeded(baseUrl, walletId, minBalance) {
  const walletRes = http.get(`${baseUrl}/v1/wallets/${walletId}`);
  const wallet = JSON.parse(walletRes.body);
  if (wallet.balance < minBalance) {
    http.post(`${baseUrl}/v1/payments/wallets/${walletId}/topup`, JSON.stringify({
      amount: 999999.99,
    }), { headers: { 'Content-Type': 'application/json' } });
  }
}

export function executeGraphQL(query, variables) {
  return http.post(GRAPHQL_URL, JSON.stringify({
    query: query,
    variables: variables || {},
  }), { headers: { 'Content-Type': 'application/json' } });
}
