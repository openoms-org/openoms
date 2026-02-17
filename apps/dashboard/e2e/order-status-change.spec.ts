import { test, expect } from '@playwright/test';
import { gotoWithAuth, waitForTableLoaded } from './helpers/actions';

test.describe('Order Status Change', () => {
  test('navigates to order detail and sees status badge', async ({ page }) => {
    await gotoWithAuth(page, '/orders');
    await waitForTableLoaded(page);

    // Click first order row
    await page.locator('table tbody tr').first().click();
    await expect(page).toHaveURL(/\/orders\/[a-f0-9-]+/, { timeout: 10000 });

    // Status badge should be visible
    await expect(page.locator('[data-testid="status-badge"], .inline-flex').first()).toBeVisible({ timeout: 5000 });
  });

  test('order detail page shows customer info', async ({ page }) => {
    await gotoWithAuth(page, '/orders');
    await waitForTableLoaded(page);
    await page.locator('table tbody tr').first().click();
    await expect(page).toHaveURL(/\/orders\/[a-f0-9-]+/, { timeout: 10000 });

    // Should show some customer-related content
    await expect(page.getByText(/Klient|Dane klienta/)).toBeVisible({ timeout: 5000 });
  });

  test('order detail page shows audit timeline', async ({ page }) => {
    await gotoWithAuth(page, '/orders');
    await waitForTableLoaded(page);
    await page.locator('table tbody tr').first().click();
    await expect(page).toHaveURL(/\/orders\/[a-f0-9-]+/, { timeout: 10000 });

    // Audit/timeline section
    await expect(page.getByText(/Historia|Timeline|Audit/)).toBeVisible({ timeout: 5000 });
  });
});
