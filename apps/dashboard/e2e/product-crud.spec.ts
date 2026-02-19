import { test, expect } from '@playwright/test';
import {
  gotoWithAuth,
  waitForToast,
  waitForTableLoaded,
  fillAndSubmitProductForm,
  confirmDeleteDialog,
} from './helpers/actions';
import { NEW_PRODUCT, SEED } from './fixtures/test-data';

test.describe.serial('Product CRUD', () => {
  let productUrl: string;

  test('create product with required fields', async ({ page }) => {
    await gotoWithAuth(page, '/products/new');
    await expect(
      page.getByRole('heading', { name: /Nowy produkt/ }),
    ).toBeVisible({ timeout: 10000 });

    await fillAndSubmitProductForm(page, NEW_PRODUCT);
    await waitForToast(page, 'Produkt został utworzony');

    // Should redirect to product detail page
    await expect(page).toHaveURL(/\/products\/[a-f0-9-]+/, { timeout: 10000 });
    productUrl = page.url();
  });

  test('verify created product detail', async ({ page }) => {
    await gotoWithAuth(page, productUrl);
    await expect(page.getByText(NEW_PRODUCT.name)).toBeVisible({
      timeout: 10000,
    });
    await expect(page.getByText(NEW_PRODUCT.sku!)).toBeVisible();
    // Polish locale price: 149,99
    await expect(page.getByText('149,99')).toBeVisible();
  });

  test('product appears in list', async ({ page }) => {
    await gotoWithAuth(page, '/products');
    await waitForTableLoaded(page);
    await expect(page.getByText(NEW_PRODUCT.name)).toBeVisible();
  });

  test('search product by name', async ({ page }) => {
    await gotoWithAuth(page, '/products');
    await waitForTableLoaded(page);

    const search = page.getByPlaceholder(/Szukaj/);
    await search.fill(NEW_PRODUCT.name);
    await page.waitForTimeout(500); // debounce
    await expect(page.getByText(NEW_PRODUCT.name)).toBeVisible();
  });

  test('edit product price', async ({ page }) => {
    await gotoWithAuth(page, productUrl);
    await expect(page.getByText(NEW_PRODUCT.name)).toBeVisible({
      timeout: 10000,
    });

    // Enter edit mode
    await page.getByRole('button', { name: 'Edytuj' }).click();
    await expect(
      page.getByRole('heading', { name: 'Edycja produktu' }),
    ).toBeVisible({ timeout: 5000 });

    // Update price
    await page.locator('#price').fill('199.99');
    await page.getByRole('button', { name: 'Zapisz zmiany' }).click();
    await waitForToast(page, /zaktualizowany|zapisany/i);

    // Verify updated price in detail view
    await expect(page.getByText('199,99')).toBeVisible({ timeout: 5000 });
  });

  test('delete product', async ({ page }) => {
    await gotoWithAuth(page, productUrl);
    await expect(page.getByText(NEW_PRODUCT.name)).toBeVisible({
      timeout: 10000,
    });

    await page.getByRole('button', { name: 'Usuń' }).click();
    await confirmDeleteDialog(page, 'Usuń');
    await waitForToast(page, /usunięt/i);

    // Should redirect to products list
    await expect(page).toHaveURL('/products', { timeout: 10000 });
  });

  test('search seed product by SKU', async ({ page }) => {
    await gotoWithAuth(page, '/products');
    await waitForTableLoaded(page);

    const search = page.getByPlaceholder(/Szukaj/);
    await search.fill(SEED.PRODUCT_SKU);
    await page.waitForTimeout(500);
    await expect(page.getByText(SEED.PRODUCT_NAME)).toBeVisible();
  });

  test('validation errors on empty product form', async ({ page }) => {
    await gotoWithAuth(page, '/products/new');
    await expect(
      page.getByRole('heading', { name: /Nowy produkt/ }),
    ).toBeVisible({ timeout: 10000 });

    // Clear default values and submit
    await page.locator('#name').fill('');
    await page.getByRole('button', { name: 'Utwórz produkt' }).click();

    await expect(
      page.getByText('Nazwa produktu jest wymagana'),
    ).toBeVisible();
  });
});
