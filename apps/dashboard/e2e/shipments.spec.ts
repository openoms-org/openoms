import { test, expect } from '@playwright/test';

test.describe('Shipments', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/shipments');
    await expect(page.getByRole('heading', { name: 'Przesyłki' })).toBeVisible();
  });

  test('displays shipments page with table', async ({ page }) => {
    await expect(page.locator('table')).toBeVisible({ timeout: 10000 });
  });

  test('new shipment link navigates correctly', async ({ page }) => {
    await page.getByRole('link', { name: /Nowa przesyłka/ }).click();
    await expect(page).toHaveURL('/shipments/new', { timeout: 10000 });
  });

  test('displays correct table headers', async ({ page }) => {
    await expect(page.locator('table')).toBeVisible({ timeout: 10000 });
    const headers = page.locator('table thead th');
    await expect(headers.filter({ hasText: 'Status' })).toBeVisible();
  });
});
