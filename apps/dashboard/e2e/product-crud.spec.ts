import { test, expect } from '@playwright/test';
import {
  gotoWithAuth,
  waitForToast,
  waitForTableLoaded,
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

    // Fill form fields
    await page.locator('#name').fill(NEW_PRODUCT.name);
    if (NEW_PRODUCT.sku) {
      await page.locator('#sku').fill(NEW_PRODUCT.sku);
    }
    await page.locator('#price').fill(NEW_PRODUCT.price);
    await page.locator('#stock_quantity').fill(NEW_PRODUCT.stock);

    // Listen for the API call before clicking
    const apiResponsePromise = page.waitForResponse(
      (resp) => resp.url().includes('/v1/products') && resp.request().method() === 'POST',
      { timeout: 15000 },
    );

    // Scroll to and click the submit button
    const submitBtn = page.getByRole('button', { name: 'Utwórz produkt' });
    await submitBtn.scrollIntoViewIfNeeded();
    await submitBtn.click();

    // Wait for the API response
    const resp = await apiResponsePromise;
    expect(resp.status()).toBe(201);

    // Should redirect to product detail page
    await expect(page).toHaveURL(/\/products\/[a-f0-9-]+/, { timeout: 10000 });
    productUrl = page.url();
  });

  test('verify created product detail', async ({ page }) => {
    await gotoWithAuth(page, productUrl);
    await expect(
      page.getByText(NEW_PRODUCT.name),
    ).toBeVisible({ timeout: 15000 });
    await expect(page.getByText(NEW_PRODUCT.sku!).first()).toBeVisible();
    // Price format depends on locale
    await expect(page.getByText(/149[.,]99/).first()).toBeVisible();
  });

  test('product appears in list', async ({ page }) => {
    await gotoWithAuth(page, '/products');
    await waitForTableLoaded(page);
    await expect(page.getByText(NEW_PRODUCT.name).first()).toBeVisible();
  });

  test('search product by name', async ({ page }) => {
    await gotoWithAuth(page, '/products');
    await waitForTableLoaded(page);

    const search = page.getByPlaceholder(/Szukaj/);
    await search.fill(NEW_PRODUCT.name);
    await page.waitForTimeout(500); // debounce
    await expect(page.getByText(NEW_PRODUCT.name).first()).toBeVisible();
  });

  test('edit product price', async ({ page }) => {
    test.skip(!productUrl, 'productUrl not set — prior test did not run');
    await gotoWithAuth(page, productUrl);
    await expect(
      page.getByRole('heading', { name: NEW_PRODUCT.name }),
    ).toBeVisible({ timeout: 10000 });

    // Enter edit mode — click the edit button in the main content area
    await page.locator('main').getByRole('button', { name: /Edytuj|Edit/ }).click();
    await expect(page.locator('#price')).toBeVisible({ timeout: 10000 });

    // Update price
    await page.locator('#price').fill('199.99');

    // Listen for API response before clicking save
    const saveResponse = page.waitForResponse(
      (resp) => resp.url().includes('/v1/products/') && resp.request().method() === 'PUT',
      { timeout: 10000 },
    );
    await page.getByRole('button', { name: /Zapisz zmiany|Save/ }).click();
    const resp = await saveResponse;
    expect(resp.status()).toBeLessThan(400);

    // Verify updated price in detail view
    await expect(page.getByText('199,99')).toBeVisible({ timeout: 5000 });
  });

  test('delete product', async ({ page }) => {
    await gotoWithAuth(page, productUrl);
    await expect(
      page.getByRole('heading', { name: NEW_PRODUCT.name }),
    ).toBeVisible({ timeout: 10000 });

    await page.getByRole('button', { name: 'Usuń' }).click();
    await confirmDeleteDialog(page, 'Usuń');
    await waitForToast(page, /usunięt/i);

    // Should redirect to products list
    await expect(page).toHaveURL('/products', { timeout: 10000 });
  });

  test('search seed product by name', async ({ page }) => {
    await gotoWithAuth(page, '/products');
    await waitForTableLoaded(page);

    const search = page.getByPlaceholder(/Szukaj/);
    await search.fill('Klocki hamulcowe');
    await page.waitForTimeout(500);
    await expect(page.getByText(SEED.PRODUCT_NAME).first()).toBeVisible();
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
