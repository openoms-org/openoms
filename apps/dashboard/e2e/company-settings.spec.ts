import { test, expect } from '@playwright/test';
import { gotoWithAuth } from './helpers/actions';

test.describe('Company Settings', () => {
  test('loads company settings page with form', async ({ page }) => {
    await gotoWithAuth(page, '/settings/company');
    await expect(page.getByRole('heading', { name: /Firma|Dane firmy/ })).toBeVisible({ timeout: 10000 });

    // Form fields should be visible
    await expect(page.getByText('Nazwa firmy')).toBeVisible({ timeout: 5000 });
  });

  test('NIP field is present', async ({ page }) => {
    await gotoWithAuth(page, '/settings/company');
    await expect(page.getByText('NIP')).toBeVisible({ timeout: 10000 });
  });

  test('save button is present', async ({ page }) => {
    await gotoWithAuth(page, '/settings/company');
    await expect(page.getByRole('button', { name: /Zapisz|Aktualizuj/ })).toBeVisible({ timeout: 5000 });
  });
});
