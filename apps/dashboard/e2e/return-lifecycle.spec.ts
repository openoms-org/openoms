import { test, expect } from '@playwright/test';
import {
  gotoWithAuth,
  waitForToast,
  waitForTableLoaded,
  confirmDeleteDialog,
} from './helpers/actions';

test.describe.serial('Return Lifecycle', () => {
  let returnUrl: string;

  test('create return for existing order', async ({ page }) => {
    await gotoWithAuth(page, '/returns/new');
    await expect(
      page.getByRole('heading', { name: 'Nowy zwrot' }),
    ).toBeVisible({ timeout: 10000 });

    // Open order search combobox
    await page.getByRole('combobox').click();

    // Wait for orders to load then select the first one
    await page.waitForTimeout(500); // debounce
    const firstOrder = page.getByRole('option').first();
    await expect(firstOrder).toBeVisible({ timeout: 10000 });
    await firstOrder.click();

    // Fill reason
    await page.locator('#reason').fill('Produkt niezgodny z opisem - test E2E');

    // Fill refund amount
    await page.locator('#refund_amount').fill('50.00');

    // Submit
    await page.getByRole('button', { name: 'Utwórz zwrot' }).click();
    await waitForToast(page, 'Zwrot został utworzony');

    // Should redirect to return detail
    await expect(page).toHaveURL(/\/returns\/[a-f0-9-]+/, { timeout: 10000 });
    returnUrl = page.url();
  });

  test('verify return detail', async ({ page }) => {
    await gotoWithAuth(page, returnUrl);

    // Verify return data
    await expect(
      page.getByText('Produkt niezgodny z opisem - test E2E'),
    ).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('50,00')).toBeVisible();

    // Status should be "requested" (Zgłoszone or Requested depending on seed data)
    await expect(page.getByText(/Zgłoszon|Requested/i).first()).toBeVisible({ timeout: 5000 });
  });

  test('return appears in list', async ({ page }) => {
    await gotoWithAuth(page, '/returns');
    await waitForTableLoaded(page);

    await expect(
      page.getByText('Produkt niezgodny z opisem').first(),
    ).toBeVisible();
  });

  test('approve return: requested → approved', async ({ page }) => {
    await gotoWithAuth(page, returnUrl);
    await expect(
      page.getByText('Produkt niezgodny z opisem - test E2E'),
    ).toBeVisible({ timeout: 10000 });

    // Click approve transition button (seed data may use English labels)
    await page.getByRole('button', { name: /Approved|Zatwierdź|Zaakceptowane/ }).click();
    await waitForToast(page, /status|zmienion|zatwierdzony|changed/i);

    // Verify new status
    await expect(page.getByText(/Approved|Zatwierdzon|Zaakceptowane/i).first()).toBeVisible({ timeout: 5000 });
  });

  test('mark return as received', async ({ page }) => {
    await gotoWithAuth(page, returnUrl);
    await expect(page.getByText(/Approved|Zatwierdzon|Zaakceptowane/i).first()).toBeVisible({ timeout: 10000 });

    // Click received transition button (seed data may use English labels)
    await page.getByRole('button', { name: /Received|Odebran|Oznacz jako odebrane/ }).click();
    await waitForToast(page, /status|zmienion|odebran|changed/i);

    await expect(page.getByText(/Received|Odebran/i).first()).toBeVisible({ timeout: 5000 });
  });

  test('edit return reason', async ({ page }) => {
    await gotoWithAuth(page, returnUrl);
    await expect(page.getByText(/Odebran/i).first()).toBeVisible({ timeout: 10000 });

    // Enter edit mode
    await page.getByRole('button', { name: 'Edytuj' }).click();

    // Update reason
    await page.locator('#reason').fill('Zaktualizowany powód - test E2E');
    await page.getByRole('button', { name: 'Zapisz zmiany' }).click();
    await waitForToast(page, /zaktualizowany|zapisano/i);

    // Verify updated reason
    await expect(
      page.getByText('Zaktualizowany powód - test E2E'),
    ).toBeVisible({ timeout: 5000 });
  });

  test('delete return', async ({ page }) => {
    await gotoWithAuth(page, returnUrl);
    await expect(
      page.getByText('Zaktualizowany powód - test E2E'),
    ).toBeVisible({ timeout: 10000 });

    await page.getByRole('button', { name: 'Usuń' }).click();
    await confirmDeleteDialog(page, 'Usuń zwrot');
    await waitForToast(page, /usunięt/i);

    await expect(page).toHaveURL('/returns', { timeout: 10000 });
  });
});
