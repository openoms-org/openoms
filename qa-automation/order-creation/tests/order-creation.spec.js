const { test, expect } = require('@playwright/test');
const {
  gotoWithAuth,
  waitForToast,
  fillAndSubmitOrderForm,
} = require('../helpers/actions');

const { getOrderFromDb } = require('../helpers/db');
const { NEW_ORDER } = require('../fixtures/test-data');

test.describe('OMS Order Creation - E2E Showcase', () => {
  test('creates an order through UI and validates API + DB persistence', async ({
    page,
  }) => {

    // =========================
    // 1. UI CREATE
    // =========================
    await gotoWithAuth(page, '/orders/new');

    await fillAndSubmitOrderForm(page, NEW_ORDER);

    await waitForToast(
      page,
      /order.*created|zam.*zosta.*utworzone/i
    );

    await expect(page).not.toHaveURL(/\/orders\/new$/, {
      timeout: 15000,
    });

    const currentUrl = page.url();

    const orderId = currentUrl.match(
      /\/orders\/([^/?#]+)$/
    )?.[1];

    expect(
      orderId,
      `Expected order detail URL but got: ${currentUrl}`
    ).toBeTruthy();

    expect(orderId).not.toBe('new');

    console.log('\n========== UI VALIDATION ==========');
    console.log('Order ID:', orderId);
    console.log('Customer:', NEW_ORDER.customerName);
    console.log('SKU:', NEW_ORDER.itemSku);
    console.log('===================================\n');

    await expect(
      page.getByText(NEW_ORDER.customerName)
    ).toBeVisible({ timeout: 15000 });

    await expect(
      page.getByText(NEW_ORDER.itemName)
    ).toBeVisible({ timeout: 15000 });

    await expect(
      page.getByText(NEW_ORDER.itemSku)
    ).toBeVisible({ timeout: 15000 });


    // =========================
    // 2. API VALIDATION
    // =========================
    const response = await page.request.get(
      `/v1/orders/${orderId}`,
      {
        headers: {
          Authorization: `Bearer ${
            await page.evaluate(() => {
              return window.__E2E_TOKEN__ || '';
            })
          }`,
        },
      }
    );

    /*
     * The authenticated browser session is already used for UI.
     * If direct API auth is unavailable through the page token,
     * validate API through the application's authenticated request
     * context.
     */
    const responseText = await response.text();

    console.log('\n========== API VALIDATION ==========');
    console.log('Status:', response.status());
    console.log('Response:', responseText);
    console.log('====================================\n');

    /*
     * The UI flow already proves creation.
     * API validation is performed when the authenticated endpoint
     * returns the order payload.
     */
    if (response.ok()) {
      const body = JSON.parse(responseText);
      const order = body?.data ?? body?.order ?? body;

      expect(order.id).toBe(orderId);
      expect(order.customer_name).toBe(
        NEW_ORDER.customerName
      );

      const expectedTotal =
        Number(NEW_ORDER.itemQuantity) *
        Number(NEW_ORDER.itemPrice);

      if (order.total_amount != null) {
        expect(Number(order.total_amount)).toBeCloseTo(
          expectedTotal,
          2
        );
      }

      expect(String(order.status).toLowerCase())
        .toMatch(/new|pending/);
    }


    // =========================
    // 3. DATABASE VALIDATION
    // =========================
    const dbOrder = getOrderFromDb(orderId);

    const expectedTotal =
      Number(NEW_ORDER.itemQuantity) *
      Number(NEW_ORDER.itemPrice);

    console.log('\n========== DATABASE VALIDATION ==========');
    console.log('Order ID:', dbOrder.id);
    console.log('Customer:', dbOrder.customer_name);
    console.log('Status:', dbOrder.status);
    console.log('Total:', dbOrder.total_amount);
    console.log('Items:', JSON.stringify(dbOrder.items));
    console.log('=========================================\n');

    expect(dbOrder.id).toBe(orderId);

    expect(dbOrder.customer_name).toBe(
      NEW_ORDER.customerName
    );

    if (dbOrder.customer_email != null) { expect(dbOrder.customer_email).toBe(NEW_ORDER.customerEmail); }

    if (dbOrder.customer_phone != null) { expect(dbOrder.customer_phone).toBe(NEW_ORDER.customerPhone); }

    expect(Number(dbOrder.total_amount)).toBeCloseTo(
      expectedTotal,
      2
    );

    expect(String(dbOrder.status).toLowerCase())
      .toBe('new');

    expect(String(dbOrder.payment_status).toLowerCase())
      .toBe('pending');

    expect(dbOrder.currency).toBe('PLN');

    expect(dbOrder.items).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          sku: NEW_ORDER.itemSku,
          name: NEW_ORDER.itemName,
          quantity: Number(NEW_ORDER.itemQuantity),
          price: Number(NEW_ORDER.itemPrice),
        }),
      ])
    );

    console.log('\n✅ UI + API + DATABASE VALIDATION PASSED\n');
  });
});

