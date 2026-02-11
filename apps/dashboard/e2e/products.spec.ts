import { test, expect } from '@playwright/test';
import { gotoWithAuth } from './helpers/actions';

test.describe('Products', () => {
  test.beforeEach(async ({ page }) => {
    await gotoWithAuth(page, '/products');
    await expect(page.getByRole('heading', { name: 'Produkty' })).toBeVisible();
  });

  test('displays products table with data', async ({ page }) => {
    await expect(page.locator('table tbody tr').first()).toBeVisible({ timeout: 10000 });
    const rows = page.locator('table tbody tr');
    const count = await rows.count();
    expect(count).toBeGreaterThan(0);
  });

  test('search input works', async ({ page }) => {
    const searchInput = page.getByPlaceholder('Szukaj po nazwie...');
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
    const firstRow = page.locator('table tbody tr').first();
    await expect(firstRow).toBeVisible({ timeout: 10000 });
    await firstRow.getByRole('link').first().click();
    await expect(page).toHaveURL(/\/products\/[a-f0-9-]+/, { timeout: 10000 });
  });

  test('pagination is rendered when data loads', async ({ page }) => {
    await expect(page.locator('table tbody tr').first()).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Na stronie:')).toBeVisible({ timeout: 5000 });
  });
});
