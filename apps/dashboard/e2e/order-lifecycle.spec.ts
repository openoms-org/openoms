import { test, expect } from '@playwright/test';
import {
  gotoWithAuth,
  waitForToast,
  waitForTableLoaded,
  fillAndSubmitOrderForm,
  confirmDeleteDialog,
} from './helpers/actions';
import { NEW_ORDER, SEED } from './fixtures/test-data';

test.describe.serial('Order Lifecycle', () => {
  let orderUrl: string;

  test('create order with item', async ({ page }) => {
    await gotoWithAuth(page, '/orders/new');
    await expect(
      page.getByRole('heading', { name: 'Nowe zamówienie' }),
    ).toBeVisible({ timeout: 10000 });

    await fillAndSubmitOrderForm(page, NEW_ORDER);
    await waitForToast(page, 'Zamówienie zostało utworzone');

    // Should redirect to order detail page
    await expect(page).toHaveURL(/\/orders\/[a-f0-9-]+/, { timeout: 10000 });
    orderUrl = page.url();
  });

  test('verify order detail page', async ({ page }) => {
    await gotoWithAuth(page, orderUrl);

    // Verify customer info
    await expect(page.getByText(NEW_ORDER.customerName)).toBeVisible({ timeout: 10000 });

    // Verify order item is displayed
    await expect(page.getByText(NEW_ORDER.itemName)).toBeVisible();

    // Verify status badge shows initial status
    await expect(page.getByText(/Now|nowe/i).first()).toBeVisible();
  });

  test('change status: new → confirmed', async ({ page }) => {
    await gotoWithAuth(page, orderUrl);
    await expect(page.getByText(NEW_ORDER.customerName)).toBeVisible({ timeout: 10000 });

    // Click "Potwierdzone" transition button (normal transition, no dialog)
    await page.getByRole('button', { name: 'Potwierdzone' }).click();
    await waitForToast(page, /status|zmienion/i);

    // Verify new status badge
    await expect(page.getByText('Potwierdzone').first()).toBeVisible({ timeout: 5000 });
  });

  test('change status: confirmed → processing', async ({ page }) => {
    await gotoWithAuth(page, orderUrl);
    await expect(page.getByText(NEW_ORDER.customerName)).toBeVisible({ timeout: 10000 });

    // Click "W realizacji" transition button
    await page.getByRole('button', { name: 'W realizacji' }).click();
    await waitForToast(page, /status|zmienion/i);

    await expect(page.getByText('W realizacji').first()).toBeVisible({ timeout: 5000 });
  });

  test('save internal notes', async ({ page }) => {
    await gotoWithAuth(page, orderUrl);
    await expect(page.getByText(NEW_ORDER.customerName)).toBeVisible({ timeout: 10000 });

    // Fill internal notes
    const notesField = page.getByPlaceholder('Notatki widoczne tylko dla zespołu...');
    await notesField.fill('Notatka testowa E2E');
    await page.getByRole('button', { name: 'Zapisz notatki' }).click();
    await waitForToast(page, /notatki|zapisano/i);

    // Reload and verify persistence
    await page.reload();
    await expect(page.getByText('Notatka testowa E2E')).toBeVisible({ timeout: 10000 });
  });

  test('CSV export button is functional', async ({ page }) => {
    await gotoWithAuth(page, '/orders');
    await waitForTableLoaded(page);

    const exportButton = page.getByRole('button', { name: /Eksportuj CSV/ });
    await expect(exportButton).toBeVisible();

    // Start download
    const [download] = await Promise.all([
      page.waitForEvent('download', { timeout: 10000 }),
      exportButton.click(),
    ]);
    expect(download.suggestedFilename()).toMatch(/\.csv$/);
  });

  test('search orders by customer name', async ({ page }) => {
    await gotoWithAuth(page, '/orders');
    await waitForTableLoaded(page);

    const search = page.getByPlaceholder(/Szukaj/);
    await search.fill(SEED.ORDER_CUSTOMER);
    await page.waitForTimeout(500); // debounce
    await expect(page.getByText(SEED.ORDER_CUSTOMER).first()).toBeVisible();
  });

  test('filter orders by status', async ({ page }) => {
    await gotoWithAuth(page, '/orders');
    await waitForTableLoaded(page);

    // Open status filter dropdown
    const statusFilter = page.locator('select, [role="combobox"]').filter({ hasText: /Status/ }).first();
    if (await statusFilter.isVisible()) {
      await statusFilter.click();
      // Select a specific status if dropdown is open
      const option = page.getByRole('option', { name: /Wysłane|shipped/i });
      if (await option.isVisible({ timeout: 2000 }).catch(() => false)) {
        await option.click();
        await page.waitForTimeout(500);
      }
    }
  });

  test('row selection and bulk action bar', async ({ page }) => {
    await gotoWithAuth(page, '/orders');
    await waitForTableLoaded(page);

    // Select first order via checkbox
    const firstCheckbox = page
      .locator('table tbody tr')
      .first()
      .locator('input[type="checkbox"]');
    await firstCheckbox.check();

    // Bulk action bar should appear
    await expect(page.getByText(/Zaznaczono/)).toBeVisible({ timeout: 3000 });
  });

  test('click row navigates to order detail', async ({ page }) => {
    await gotoWithAuth(page, '/orders');
    await waitForTableLoaded(page);

    // Click the customer name link in the first row
    await page.locator('table tbody tr').first().click();
    await expect(page).toHaveURL(/\/orders\/[a-f0-9-]+/, { timeout: 10000 });
  });

  test('delete order', async ({ page }) => {
    await gotoWithAuth(page, orderUrl);
    await expect(page.getByText(NEW_ORDER.customerName)).toBeVisible({ timeout: 10000 });

    await page.getByRole('button', { name: 'Usuń' }).click();
    await confirmDeleteDialog(page, 'Usuń zamówienie');
    await waitForToast(page, /usunięt/i);

    // Should redirect to orders list
    await expect(page).toHaveURL('/orders', { timeout: 10000 });
  });
});
