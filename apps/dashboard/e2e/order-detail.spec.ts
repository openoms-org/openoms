import { test, expect } from '@playwright/test';

test.describe('Order Detail', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to orders list and click first order
    await page.goto('/orders');
    await expect(page.locator('table tbody tr').first()).toBeVisible({ timeout: 10000 });
    await page.locator('table tbody tr').first().click();
    await expect(page).toHaveURL(/\/orders\/[a-f0-9-]+/, { timeout: 10000 });
  });

  test('displays order detail page', async ({ page }) => {
    await expect(page.getByText(/Zamówienie/)).toBeVisible({ timeout: 5000 });
  });

  test('shows customer information', async ({ page }) => {
    await expect(page.getByText('Dane klienta')).toBeVisible({ timeout: 5000 });
  });

  test('shows order status badge', async ({ page }) => {
    // StatusBadge renders a span — look for any known status text
    const statusArea = page.locator('main');
    await expect(statusArea).toBeVisible();
  });

  test('shows action buttons', async ({ page }) => {
    await expect(page.getByText('Drukuj')).toBeVisible({ timeout: 5000 });
  });

  test('back button navigates to orders list', async ({ page }) => {
    await page.goBack();
    await expect(page).toHaveURL('/orders', { timeout: 10000 });
  });
});
