import { test, expect } from '@playwright/test';
import { gotoWithAuth } from './helpers/actions';

test.describe('Products', () => {
  test.beforeEach(async ({ page }) => {
    await gotoWithAuth(page, '/products');
    await expect(page.getByRole('heading', { name: 'Produkty' })).toBeVisible({ timeout: 10000 });
  });

  test('displays products table with data', async ({ page }) => {
    await expect(page.locator('table tbody tr').first()).toBeVisible({ timeout: 10000 });
    const rows = page.locator('table tbody tr');
    const count = await rows.count();
    expect(count).toBeGreaterThan(0);
  });

  test('search input works', async ({ page }) => {
    const searchInput = page.getByPlaceholder('Szukaj po nazwie, SKU lub EAN...');
    await expect(searchInput).toBeVisible();
    await searchInput.fill('Test');
    await page.waitForTimeout(500);
    await expect(searchInput).toHaveValue('Test');
  });

  test('new product button navigates correctly', async ({ page }) => {
    await page.getByRole('link', { name: 'Nowy produkt' }).click();
    await expect(page).toHaveURL('/products/new', { timeout: 10000 });
  });

  test('clicking a product link navigates to detail', async ({ page }) => {
    const firstLink = page.locator('table tbody tr').first().getByRole('link').first();
    await expect(firstLink).toBeVisible({ timeout: 10000 });
    await firstLink.click();
    await expect(page).toHaveURL(/\/products\/[a-f0-9-]+/, { timeout: 10000 });
  });

  test('pagination is rendered when data loads', async ({ page }) => {
    await expect(page.locator('table tbody tr').first()).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Na stronie:')).toBeVisible({ timeout: 5000 });
  });

  test('product form page loads with required fields', async ({ page }) => {
    await gotoWithAuth(page, '/products/new');
    await expect(page.getByRole('heading', { name: /Nowy produkt|Dodaj produkt/ })).toBeVisible({ timeout: 5000 });
    // Name and Price fields should be present
    await expect(page.getByLabel(/Nazwa/i)).toBeVisible();
    await expect(page.getByLabel(/Cena/i)).toBeVisible();
  });

  test('product detail page loads correctly', async ({ page }) => {
    await gotoWithAuth(page, '/products');
    await expect(page.locator('table tbody tr').first()).toBeVisible({ timeout: 10000 });
    const link = page.locator('table tbody tr').first().getByRole('link').first();
    await expect(link).toBeVisible({ timeout: 10000 });
    await link.click();
    await expect(page).toHaveURL(/\/products\/[a-f0-9-]+/, { timeout: 10000 });
    // Should show product details
    await expect(page.getByText(/Szczegóły|SKU|Cena/).first()).toBeVisible({ timeout: 5000 });
  });
});
