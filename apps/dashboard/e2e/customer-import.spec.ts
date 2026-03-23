import { test, expect } from '@playwright/test';
import { gotoWithAuth } from './helpers/actions';

test.describe('Customer Import', () => {
  test('customer import page loads', async ({ page }) => {
    await gotoWithAuth(page, '/customers/import');
    await expect(
      page.getByText(/Import klientów/),
    ).toBeVisible({ timeout: 15000 });
    await expect(page.getByText('Przeciągnij plik CSV tutaj')).toBeVisible();
    await expect(page.getByText('lub kliknij, aby wybrać plik')).toBeVisible();
  });

  test('customers page has import CSV link', async ({ page }) => {
    await gotoWithAuth(page, '/customers');
    await expect(
      page.getByRole('heading', { name: 'Klienci' }),
    ).toBeVisible({ timeout: 20000 });
    const importLink = page.getByRole('link', { name: /Importuj CSV/ });
    await expect(importLink).toBeVisible();
    await importLink.click();
    await expect(page).toHaveURL('/customers/import', { timeout: 10000 });
  });

  test('upload CSV shows preview', async ({ page }) => {
    await gotoWithAuth(page, '/customers/import');
    await expect(
      page.getByText(/Import klientów/),
    ).toBeVisible({ timeout: 15000 });

    // Create a test CSV file in-memory
    const csvContent = [
      'name,email,phone',
      'Test Import E2E,test-import-e2e@example.com,+48 111 222 333',
    ].join('\n');
    const csvBuffer = Buffer.from(csvContent, 'utf-8');

    // Upload via hidden file input
    await page.locator('input[type="file"]').setInputFiles({
      name: 'test-customers.csv',
      mimeType: 'text/csv',
      buffer: csvBuffer,
    });

    // Wait for the preview to appear (requires API response)
    await expect(
      page.getByText(/Podgląd importu/),
    ).toBeVisible({ timeout: 15000 });

    // Verify preview badges are visible
    await expect(page.getByText(/Łącznie:/)).toBeVisible();
    await expect(page.getByText(/Nowi klienci:/)).toBeVisible();
    await expect(page.getByText(/Aktualizacje:/)).toBeVisible();

    // Verify the import button is visible
    await expect(
      page.getByRole('button', { name: /Importuj \d+ klientów/ }),
    ).toBeVisible();
  });
});
