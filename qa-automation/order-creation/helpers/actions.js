const { expect } = require('@playwright/test');

const ADD_ITEM = /^(Add item|Dodaj pozycję|Dodaj pozycje)$/i;
const CREATE_ORDER = /^(Create order|Utwórz zamówienie)$/i;

async function gotoWithAuth(page, url) {
  await page.goto(url, { waitUntil: 'domcontentloaded' });

  if (page.url().endsWith('/login')) {
    const { TEST_CREDENTIALS } =
      require('../../../apps/dashboard/e2e/fixtures/test-data');

    await page.getByRole('textbox', { name: 'Organization' })
      .fill(TEST_CREDENTIALS.tenant_slug);

    await page.getByRole('textbox', { name: 'Email' })
      .fill(TEST_CREDENTIALS.email);

    await page.getByRole('textbox', { name: 'Password' })
      .fill(TEST_CREDENTIALS.password);

    await page.getByRole('button', { name: 'Log in' }).click();

    await expect(page).not.toHaveURL(/\/login$/, {
      timeout: 15000,
    });

    await page.goto(url, { waitUntil: 'domcontentloaded' });
  }

  await expect(page).not.toHaveURL(/\/login$/, {
    timeout: 15000,
  });
}

async function waitForToast(page, text) {
  const toast = page.locator('[data-sonner-toast]').filter({ hasText: text });
  await expect(toast).toBeVisible({ timeout: 15000 });
}

/*
 * React-safe fill.
 * The form can re-render and replace the input DOM node.
 * Therefore we resolve the locator fresh on every attempt.
 */
async function stableFill(page, selector, value) {
  const text = String(value);
  const locatorFactory = () =>
    typeof selector === 'string'
      ? page.locator(selector)
      : selector;

  for (let attempt = 1; attempt <= 5; attempt++) {
    const locator = locatorFactory();

    try {
      await locator.waitFor({
        state: 'visible',
        timeout: 5000,
      });

      await locator.fill(text, {
        timeout: 5000,
      });

      await expect(locator).toHaveValue(text, {
        timeout: 2000,
      });

      return;
    } catch (error) {
      if (attempt === 5) {
        throw error;
      }

      await page.waitForTimeout(300);
    }
  }
}

async function setInput(locator, value) {
  await locator.fill(String(value), { timeout: 15000 });
}

function addItemButton(page) {
  return page.getByRole('button', { name: ADD_ITEM }).last();
}

function createOrderButton(page) {
  return page.getByRole('button', { name: CREATE_ORDER });
}

async function fillCustomer(page, data) {
  await stableFill(
    page,
    '#customer_name',
    data.customerName
  );

  await expect(addItemButton(page)).toBeVisible({
    timeout: 15000,
  });
}

async function clickAddItem(page) {
  const button = addItemButton(page);

  await expect(button).toBeVisible({
    timeout: 15000,
  });

  await expect(button).toBeEnabled({
    timeout: 15000,
  });

  await button.click({
    timeout: 15000,
  });
}

async function clickCreateOrder(page) {
  const button = createOrderButton(page);

  await expect(button).toBeVisible({
    timeout: 15000,
  });

  await expect(button).toBeEnabled({
    timeout: 15000,
  });

  await button.click();
}

async function addOrderItem(page, data) {
  await clickAddItem(page);

  const product = page
    .getByPlaceholder(/Nazwa produktu|Product name/i)
    .last();

  await expect(product).toBeVisible({
    timeout: 15000,
  });

  await stableFill(
    page,
    product,
    data.itemName
  );

  if (data.itemSku) {
    await stableFill(
      page,
      page.getByPlaceholder('SKU').last(),
      data.itemSku
    );
  }

  await stableFill(
    page,
    'input[type="number"][min="1"][step="1"]',
    data.itemQuantity
  );

  await stableFill(
    page,
    'input[type="number"][step="0.01"]:not(#total_amount)',
    data.itemPrice
  );
}

async function fillAndSubmitOrderForm(page, data) {
  await fillCustomer(page, data);
  await addOrderItem(page, data);
  await clickCreateOrder(page);
}

module.exports = {
  gotoWithAuth,
  waitForToast,
  stableFill,
  setInput,
  fillCustomer,
  addOrderItem,
  fillAndSubmitOrderForm,
  addItemButton,
  createOrderButton,
};

