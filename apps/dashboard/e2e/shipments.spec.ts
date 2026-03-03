import { test, expect } from '@playwright/test';
import { gotoWithAuth } from './helpers/actions';

test.describe('Shipments', () => {
  test.beforeEach(async ({ page }) => {
    await gotoWithAuth(page, '/shipments');
    await expect(page.getByRole('heading', { name: 'Przesyłki' })).toBeVisible({ timeout: 10000 });
  });

  test('displays shipments page with table or empty state', async ({ page }) => {
    await expect(
      page.locator('table').or(page.getByText('Brak przesyłek')),
    ).toBeVisible({ timeout: 10000 });
  });

  test('new shipment link navigates correctly', async ({ page }) => {
    await page.getByRole('link', { name: /Nowa przesyłka/ }).first().click();
    await expect(page).toHaveURL('/shipments/new', { timeout: 10000 });
  });

  test('displays correct table headers or empty state', async ({ page }) => {
    const table = page.locator('table');
    const emptyState = page.getByText('Brak przesyłek');
    await expect(table.or(emptyState)).toBeVisible({ timeout: 10000 });
    if (await table.isVisible()) {
      const headers = page.locator('table thead th');
      await expect(headers.filter({ hasText: 'Status' })).toBeVisible();
    }
  });

  test('new shipment form page loads', async ({ page }) => {
    await gotoWithAuth(page, '/shipments/new');
    await expect(page.getByRole('heading', { name: /Nowa przesyłka/ })).toBeVisible({ timeout: 5000 });
  });

  test('shipment table has provider column or empty state', async ({ page }) => {
    const table = page.locator('table');
    const emptyState = page.getByText('Brak przesyłek');
    await expect(table.or(emptyState)).toBeVisible({ timeout: 10000 });
    if (await table.isVisible()) {
      const headers = page.locator('table thead th');
      await expect(headers.filter({ hasText: /Kurier|Przewoźnik|Provider|Dostawca/ })).toBeVisible();
    }
  });
});
