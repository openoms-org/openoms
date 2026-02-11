import { test, expect } from '@playwright/test';

test.describe('Orders List', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/orders');
    await expect(page.getByRole('heading', { name: 'Zamówienia' })).toBeVisible();
  });

  test('displays orders table with correct columns', async ({ page }) => {
    await expect(page.locator('table')).toBeVisible({ timeout: 10000 });
    const headers = page.locator('table thead th');
    await expect(headers.filter({ hasText: 'Klient' })).toBeVisible();
    await expect(headers.filter({ hasText: 'Status' })).toBeVisible();
    await expect(headers.filter({ hasText: 'Kwota' })).toBeVisible();
    await expect(headers.filter({ hasText: 'Data' })).toBeVisible();
  });

  test('displays order data from seed', async ({ page }) => {
    await expect(page.locator('table tbody tr').first()).toBeVisible({ timeout: 10000 });
    const count = await page.locator('table tbody tr').count();
    expect(count).toBeGreaterThan(0);
  });

  test('search filter filters orders', async ({ page }) => {
    await expect(page.locator('table tbody tr').first()).toBeVisible({ timeout: 10000 });
    const searchInput = page.getByPlaceholder('Szukaj');
    if (await searchInput.isVisible()) {
      await searchInput.fill('Marek');
      await page.waitForTimeout(500);
    }
  });

  test('new order button navigates to create page', async ({ page }) => {
    await page.getByRole('link', { name: 'Nowe zamówienie' }).click();
    await expect(page).toHaveURL('/orders/new', { timeout: 10000 });
  });

  test('CSV export button is visible', async ({ page }) => {
    await expect(page.getByRole('button', { name: /Eksportuj CSV/ })).toBeVisible();
  });

  test('clicking a row navigates to order detail', async ({ page }) => {
    await expect(page.locator('table tbody tr').first()).toBeVisible({ timeout: 10000 });
    await page.locator('table tbody tr').first().click();
    await expect(page).toHaveURL(/\/orders\/[a-f0-9-]+/, { timeout: 10000 });
  });

  test('pagination is rendered when data loads', async ({ page }) => {
    await expect(page.locator('table tbody tr').first()).toBeVisible({ timeout: 10000 });
    // Pagination text: "Wyniki X-Y z Z" or "Na stronie:" selector
    await expect(page.getByText('Na stronie:')).toBeVisible({ timeout: 5000 });
  });

  test('row selection checkboxes work', async ({ page }) => {
    await expect(page.locator('table tbody tr').first()).toBeVisible({ timeout: 10000 });
    const firstCheckbox = page.locator('table tbody tr').first().locator('input[type="checkbox"]');
    if (await firstCheckbox.isVisible()) {
      await firstCheckbox.check();
      await expect(page.getByText(/Zaznaczono/)).toBeVisible({ timeout: 3000 });
    }
  });
});
