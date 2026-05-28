import { check, sleep } from 'k6';
import http from 'k6/http';
import { uuidv4 } from 'https://jslib.io/k6-utils/1.4.0/index.js';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const GRAPHQL_URL = `${BASE_URL}/graphql`;

export function generateUUID() {
  return uuidv4();
}

export function executeGraphQL(query, variables) {
  return http.post(GRAPHQL_URL, JSON.stringify({
    query: query,
    variables: variables || {},
  }), { headers: { 'Content-Type': 'application/json' } });
}

export function setupTestEntities(baseUrl) {
  const prefix = generateUUID().substring(0, 8);

  const createUserQuery = `
    mutation($email: String!, $fullName: String!, $status: String) {
      createUser(email: $email, fullName: $fullName, status: $status) {
        id
      }
    }
  `;
  const userRes = executeGraphQL(createUserQuery, {
    email: `loadtest-${prefix}@example.com`,
    fullName: `Load Test User ${prefix}`,
    status: 'ACTIVE',
  });
  check(userRes, { 'user created via graphql': r => r.status === 200 }) ||
    (() => { throw new Error(`Setup failed: create user (${userRes.status})`); })();
  const userId = userRes.json().data.createUser.id;

  const createMerchantQuery = `
    mutation($name: String!, $apiKey: String!) {
      createMerchant(name: $name, apiKey: $apiKey) {
        id
      }
    }
  `;
  const merchantRes = executeGraphQL(createMerchantQuery, {
    name: `Merchant-${prefix}`,
    apiKey: `api-key-${prefix}`,
  });
  check(merchantRes, { 'merchant created via graphql': r => r.status === 200 }) ||
    (() => { throw new Error(`Setup failed: create merchant (${merchantRes.status})`); })();
  const merchantId = merchantRes.json().data.createMerchant.id;

  const createWalletQuery = `
    mutation($userId: String!, $balance: Float, $currency: String) {
      createWallet(userId: $userId, balance: $balance, currency: $currency) {
        id
      }
    }
  `;
  const walletRes = executeGraphQL(createWalletQuery, {
    userId: userId,
    balance: 999999.99,
    currency: 'USD',
  });
  check(walletRes, { 'wallet created via graphql': r => r.status === 200 }) ||
    (() => { throw new Error(`Setup failed: create wallet (${walletRes.status})`); })();
  const walletId = walletRes.json().data.createWallet.id;

  const topUpQuery = `
    mutation($walletId: String!, $amount: Float!) {
      topUpWallet(walletId: $walletId, amount: $amount) {
        id
        balance
      }
    }
  `;
  const topUpRes = executeGraphQL(topUpQuery, {
    walletId: walletId,
    amount: 9999999.99,
  });
  check(topUpRes, { 'wallet funded via graphql': r => r.status === 200 });

  console.log(`SETUP: user=${userId} wallet=${walletId} merchant=${merchantId}`);
  return { userId, merchantId, walletId, prefix };
}

export function teardownTestEntities(baseUrl, data) {
  if (!data) return;

  const deleteUserQuery = `
    mutation($id: String!) {
      deleteUser(id: $id)
    }
  `;
  const deleteMerchantQuery = `
    mutation($id: String!) {
      deleteMerchant(id: $id)
    }
  `;

  if (data.userId) {
    executeGraphQL(deleteUserQuery, { id: data.userId });
  }
  if (data.merchantId) {
    executeGraphQL(deleteMerchantQuery, { id: data.merchantId });
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
  const getWalletQuery = `
    query($id: String!) {
      wallet(id: $id) {
        balance
      }
    }
  `;
  const walletRes = executeGraphQL(getWalletQuery, { id: walletId });
  if (walletRes.status === 200) {
    const balance = walletRes.json().data.wallet.balance;
    if (balance < minBalance) {
      const topUpQuery = `
        mutation($walletId: String!, $amount: Float!) {
          topUpWallet(walletId: $walletId, amount: $amount) {
            id
            balance
          }
        }
      `;
      executeGraphQL(topUpQuery, {
        walletId: walletId,
        amount: 999999.99,
      });
    }
  }
}
