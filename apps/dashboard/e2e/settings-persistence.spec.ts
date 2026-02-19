import { test, expect } from '@playwright/test';
import { gotoWithAuth, waitForToast, waitForTableLoaded } from './helpers/actions';

test.describe('Settings Persistence', () => {
  test('company info saves and persists', async ({ page }) => {
    await gotoWithAuth(page, '/settings/company');
    await expect(
      page.getByRole('heading', { name: 'Dane firmy' }),
    ).toBeVisible({ timeout: 10000 });

    // Find the company name input (first input inside the "Nazwa firmy" label container)
    const nameInput = page
      .locator('label:has-text("Nazwa firmy")')
      .locator('..')
      .locator('input');
    await nameInput.fill('E2E Test Company Sp. z o.o.');

    // Save
    await page.getByRole('button', { name: /Zapisz dane firmy/ }).click();
    await waitForToast(page, /zapisane|zapisano/i);

    // Reload and verify persistence
    await page.reload();
    await expect(
      page.getByRole('heading', { name: 'Dane firmy' }),
    ).toBeVisible({ timeout: 10000 });
    await expect(
      page.locator('input[value="E2E Test Company Sp. z o.o."]'),
    ).toBeVisible({ timeout: 5000 });
  });

  test('order statuses page loads and displays config', async ({ page }) => {
    await gotoWithAuth(page, '/settings/order-statuses');
    await expect(
      page.getByRole('heading', { name: /Statusy zamówień|Status/ }),
    ).toBeVisible({ timeout: 10000 });

    // Verify default statuses are shown (key inputs)
    await expect(page.locator('input[value="new"]').first()).toBeVisible();
    await expect(page.locator('input[value="confirmed"]').first()).toBeVisible();
    await expect(page.locator('input[value="shipped"]').first()).toBeVisible();

    // "Dodaj status" button should be visible
    await expect(
      page.getByRole('button', { name: 'Dodaj status' }),
    ).toBeVisible();
  });

  test('add custom order status', async ({ page }) => {
    await gotoWithAuth(page, '/settings/order-statuses');
    await expect(
      page.getByRole('heading', { name: /Statusy zamówień|Status/ }),
    ).toBeVisible({ timeout: 10000 });

    // Wait for existing statuses to load from API before counting
    await expect(page.getByPlaceholder('Klucz (np. new)').first()).toBeVisible({ timeout: 10000 });

    // Remove any leftover e2e_test entries from prior runs to avoid duplicate key errors
    const keyInputs = page.getByPlaceholder('Klucz (np. new)');
    const count = await keyInputs.count();
    for (let i = count - 1; i >= 0; i--) {
      const val = await keyInputs.nth(i).inputValue();
      if (val === 'e2e_test') {
        // Click the trash button (last button in the row — first is the color Select)
        const row = keyInputs.nth(i).locator('..');
        await row.locator('button').last().click();
      }
    }

    const keysBefore = await keyInputs.count();

    // Add new status
    await page.getByRole('button', { name: 'Dodaj status' }).click();

    // Wait for new row to appear
    await expect(keyInputs).toHaveCount(keysBefore + 1, { timeout: 3000 });

    // Fill the last (newly added) key and label
    await page.getByPlaceholder('Klucz (np. new)').last().fill('e2e_test');
    await page.getByPlaceholder('Etykieta (np. Nowe)').last().fill('Test E2E');

    // Save
    await page.getByRole('button', { name: 'Zapisz zmiany' }).click();
    await waitForToast(page, /zapisane|zapisano/i);
  });

  test('custom fields page loads', async ({ page }) => {
    await gotoWithAuth(page, '/settings/custom-fields');
    await expect(
      page.getByRole('heading', { name: /Pola dodatkowe|Custom fields/ }),
    ).toBeVisible({ timeout: 10000 });

    // "Dodaj pole" button should be visible
    await expect(
      page.getByRole('button', { name: /Dodaj pole/ }),
    ).toBeVisible();
  });

  test('product categories page loads', async ({ page }) => {
    await gotoWithAuth(page, '/settings/product-categories');
    await expect(
      page.getByRole('heading', { name: /Kategorie produktów|Kategorie/ }),
    ).toBeVisible({ timeout: 10000 });

    // "Dodaj kategorię" button should be visible
    await expect(
      page.getByRole('button', { name: /Dodaj kategorię/ }),
    ).toBeVisible();
  });

  test('users page loads and shows user list', async ({ page }) => {
    await gotoWithAuth(page, '/settings/users');
    await expect(
      page.getByRole('heading', { name: /Użytkownicy/ }),
    ).toBeVisible({ timeout: 10000 });

    await waitForTableLoaded(page);

    // Seed users should be visible
    await expect(page.getByText('rafal@mercpart.pl')).toBeVisible();
    await expect(page.getByText('adam@mercpart.pl')).toBeVisible();
  });

  test('warehouses page loads', async ({ page }) => {
    await gotoWithAuth(page, '/settings/warehouses');
    await expect(
      page.getByRole('heading', { name: /Magazyny/ }),
    ).toBeVisible({ timeout: 10000 });

    // "Dodaj magazyn" or "Nowy magazyn" button should be visible
    await expect(
      page.getByRole('button', { name: /Dodaj magazyn|Nowy magazyn/ }),
    ).toBeVisible();
  });
});
