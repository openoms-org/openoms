const { test, expect } = require('@playwright/test');
const { gotoWithAuth } = require('../helpers/actions');
const { OrderCreatePage } = require('../pages/OrderCreatePage');

test.describe('POM Smoke', () => {
  test('creates a basic order using Page Object Model', async ({ page }) => {
    const orderPage = new OrderCreatePage(page);

    await gotoWithAuth(page, '/orders/new');

    await expect(orderPage.customerName).toBeVisible();

    await orderPage.fillCustomer(`POM Smoke ${Date.now()}`);
    await orderPage.addItem();
    await expect(orderPage.productNames.first()).toBeVisible({ timeout: 15000 });

    await orderPage.fillFirstItem({
      name: 'POM Smoke Product',
      sku: `POM-${Date.now()}`,
      quantity: 1,
      price: 10.00,
    });

    await orderPage.createOrder();

    await expect(page).not.toHaveURL(/\/orders\/new$/,{ timeout: 15000 });
  });
});
