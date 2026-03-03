import { test, expect } from '@playwright/test';
import { gotoWithAuth, waitForTableLoaded } from './helpers/actions';

/** Click a non-checkbox cell in the first data row to trigger onRowClick navigation. */
async function clickFirstOrderRow(page: import('@playwright/test').Page) {
  await waitForTableLoaded(page);
  // Wait for actual data — seed orders have customer names
  const firstRow = page.locator('table tbody tr').first();
  await expect(firstRow.locator('td').nth(1)).toBeVisible({ timeout: 10000 });
  // Click the second cell (skip checkbox column) to trigger row onClick
  await firstRow.locator('td').nth(1).click();
}

test.describe('Order Status Change', () => {
  test('navigates to order detail and sees status badge', async ({ page }) => {
    await gotoWithAuth(page, '/orders');
    await clickFirstOrderRow(page);
    await expect(page).toHaveURL(/\/orders\/[a-f0-9-]+/, { timeout: 10000 });

    // Status badge should be visible — locate the status label then find the badge span next to it
    const statusSection = page.locator('p:text("Status")').first().locator('..');
    await expect(statusSection.locator('span.inline-flex').first()).toBeVisible({ timeout: 5000 });
  });

  test('order detail page shows customer info', async ({ page }) => {
    await gotoWithAuth(page, '/orders');
    await clickFirstOrderRow(page);
    await expect(page).toHaveURL(/\/orders\/[a-f0-9-]+/, { timeout: 10000 });

    // Should show customer data card
    await expect(page.getByText('Dane klienta')).toBeVisible({ timeout: 5000 });
  });

  test('order detail page shows audit timeline', async ({ page }) => {
    await gotoWithAuth(page, '/orders');
    await clickFirstOrderRow(page);
    await expect(page).toHaveURL(/\/orders\/[a-f0-9-]+/, { timeout: 10000 });

    // Audit/timeline section
    await expect(page.getByText(/Historia|Timeline|Audit/)).toBeVisible({ timeout: 5000 });
  });
});
