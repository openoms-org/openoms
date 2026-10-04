const { test, expect } = require('@playwright/test');
const { TEST_CREDENTIALS } = require('../../../apps/dashboard/e2e/fixtures/test-data');

test.describe('Independent API Tests', () => {
  test('authenticates and reads orders through API without UI', async ({ request }) => {
    const loginResponse = await request.post('/v1/auth/login', {
      data: TEST_CREDENTIALS,
    });

    expect(loginResponse.ok()).toBeTruthy();

    const loginBody = await loginResponse.json();
    const accessToken = loginBody?.access_token;

    expect(accessToken).toBeTruthy();

    const ordersResponse = await request.get('/v1/orders', {
      headers: {
        Authorization: `Bearer ${accessToken}`,
      },
    });

    expect(ordersResponse.ok()).toBeTruthy();

    const body = await ordersResponse.json();

    expect(body).toBeTruthy();
    console.log('API ORDER READ PASSED');
  });

  test('rejects protected orders endpoint without authentication', async ({ request }) => {
    const response = await request.get('/v1/orders');

    expect(response.status()).toBe(401);
  });
});
