import { test, expect } from '@playwright/test';
import { gotoWithAuth, waitForTableLoaded } from './helpers/actions';

/** Navigate to the first order in the list via link or row click. */
async function clickFirstOrderRow(page: import('@playwright/test').Page) {
  await waitForTableLoaded(page);
  const firstRow = page.locator('table tbody tr').first();
  await expect(firstRow).toBeVisible({ timeout: 10000 });
  // Prefer clicking a link inside the row (more reliable than cell click)
  const link = firstRow.getByRole('link').first();
  if (await link.isVisible().catch(() => false)) {
    await link.click();
  } else {
    // Fallback: click a non-checkbox cell
    await firstRow.locator('td').nth(1).click({ timeout: 10000 });
  }
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
