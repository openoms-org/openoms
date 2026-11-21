const { test, expect } = require('@playwright/test');
const { TEST_CREDENTIALS } = require('../../../apps/dashboard/e2e/fixtures/test-data');

async function directLogin(page) {
  await page.goto('/login');
  await page.getByRole('textbox', { name: 'Organization' }).fill(TEST_CREDENTIALS.tenant_slug);
  await page.getByRole('textbox', { name: 'Email' }).fill(TEST_CREDENTIALS.email);
  await page.getByRole('textbox', { name: 'Password' }).fill(TEST_CREDENTIALS.password);
  await page.getByRole('button', { name: 'Log in' }).click();
  await expect(page).not.toHaveURL(/\/login$/, { timeout: 15000 });
  await page.goto('/orders/new', { waitUntil: 'domcontentloaded' });
  await expect(page.locator('#customer_name')).toBeVisible({ timeout: 15000 });
}


const {
  gotoWithAuth,
  setInput,
  fillCustomer,
  addOrderItem,
  addItemButton,
  createOrderButton,
} = require('../helpers/actions');

test.describe('OMS Order Creation - Negative & Edge Cases', () => {

  test('rejects order creation when customer name is missing @smoke @regression', async ({ page }) => {
    await gotoWithAuth(page, '/orders/new');

    const customerName = page.locator('#customer_name');

    await expect(customerName).toBeVisible();
    await expect(customerName).toHaveValue('');

    await createOrderButton(page).click();

    await expect(page).toHaveURL(/\/orders\/new$/);
    await expect(customerName).toHaveValue('');
    await expect(customerName).toHaveAttribute('aria-invalid', 'true');
  });


  test('prevents submission when order item details are missing @regression', async ({ page }) => {
    await gotoWithAuth(page, '/orders/new');
const runId = Date.now();

    await fillCustomer(page, {
      customerName: `Missing Item E2E ${runId}`,
      customerEmail: `missing-item-${runId}@example.com`,
      customerPhone: '+48 500 111 444',
    });

    // Add an empty item row.
    await addItemButton(page).click();

    const products = page.getByPlaceholder(
      /Nazwa produktu|Product name/i
    );

    await expect(products).toHaveCount(1, {
      timeout: 15000,
    });

    // Submit without filling item details.
    await createOrderButton(page).click();

    // Browser/application validation must keep us on the form.
    await expect(page).toHaveURL(/\/orders\/new$/);
  });


  test('validates minimum quantity on order item @regression', async ({ page }) => {
    await gotoWithAuth(page, '/orders/new');
const runId = Date.now();

    await fillCustomer(page, {
      customerName: `Invalid Quantity E2E ${runId}`,
      customerEmail: `invalid-qty-${runId}@example.com`,
      customerPhone: '+48 500 111 555',
    });

    await addOrderItem(page, {
      itemName: 'Produkt testowy E2E',
      itemSku: `INVALID-QTY-${runId}`,
      itemQuantity: '1',
      itemPrice: '50.00',
    });

    const quantity = page.locator(
      'input[type="number"][min="1"][step="1"]'
    ).last();

    await expect(quantity).toHaveAttribute('min', '1');

    const minValue = await quantity.getAttribute('min');

    expect(Number(minValue)).toBe(1);

    // Attempt an invalid quantity.
    await quantity.fill('0');

    // The application/browser may normalize the controlled input
    // back to its minimum value. In either case, the field must
    // never accept a value below the declared minimum.
    const actualValue = Number(await quantity.inputValue());

    expect(actualValue).toBeGreaterThanOrEqual(1);
  });


  test('creates order with multiple valid items @smoke @regression', async ({ page }) => {
    await gotoWithAuth(page, '/orders/new');
const runId = Date.now();

    await fillCustomer(page, {
      customerName: `Multi Item E2E ${runId}`,
      customerEmail: `multi-${runId}@example.com`,
      customerPhone: '+48 500 111 666',
    });

    // =========================
    // FIRST ITEM
    // =========================
    await addOrderItem(page, {
      itemName: 'Produkt testowy E2E',
      itemSku: `MULTI-1-${runId}`,
      itemQuantity: '2',
      itemPrice: '50.00',
    });

    const products = page.getByPlaceholder(
      /Nazwa produktu|Product name/i
    );

    await expect(products).toHaveCount(1, {
      timeout: 15000,
    });

    // =========================
    // SECOND ITEM
    // =========================
    await addItemButton(page).click();

    await expect(products).toHaveCount(2, {
      timeout: 15000,
    });

    await setInput(
      products.nth(1),
      'Produkt testowy E2E 2'
    );

    await setInput(
      page.getByPlaceholder('SKU').nth(1),
      `MULTI-2-${runId}`
    );

    await setInput(
      page.locator(
        'input[type="number"][min="1"][step="1"]'
      ).nth(1),
      '1'
    );

    await setInput(
      page.locator(
        'input[type="number"][step="0.01"]:not(#total_amount)'
      ).nth(1),
      '25.00'
    );

    // =========================
    // SUBMIT
    // =========================
    await createOrderButton(page).click();

    await expect(page).not.toHaveURL(
      /\/orders\/new$/,
      { timeout: 15000 }
    );

    await expect(
      page.getByText(`Multi Item E2E ${runId}`)
    ).toBeVisible({ timeout: 15000 });

    await expect(
      page.getByText('Produkt testowy E2E', { exact: true })
    ).toBeVisible({ timeout: 15000 });

    await expect(
      page.getByText('Produkt testowy E2E 2')
    ).toBeVisible({ timeout: 15000 });
  });

});















