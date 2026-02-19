import { test, expect } from '@playwright/test';
import {
  gotoWithAuth,
  waitForToast,
  waitForTableLoaded,
  fillAndSubmitCustomerForm,
  confirmDeleteDialog,
} from './helpers/actions';
import { NEW_CUSTOMER } from './fixtures/test-data';

test.describe.serial('Customer CRUD', () => {
  let customerUrl: string;

  test('create customer', async ({ page }) => {
    await gotoWithAuth(page, '/customers/new');
    await expect(
      page.getByRole('heading', { name: /Nowy klient/ }),
    ).toBeVisible({ timeout: 10000 });

    await fillAndSubmitCustomerForm(page, NEW_CUSTOMER);
    await waitForToast(page, 'Klient został utworzony');

    // Should redirect to customers list
    await expect(page).toHaveURL('/customers', { timeout: 10000 });
  });

  test('customer appears in list', async ({ page }) => {
    await gotoWithAuth(page, '/customers');
    await waitForTableLoaded(page);
    await expect(page.getByText(NEW_CUSTOMER.name).first()).toBeVisible();
  });

  test('navigate to customer detail', async ({ page }) => {
    await gotoWithAuth(page, '/customers');
    await waitForTableLoaded(page);

    // Click the first matching customer row (duplicates may exist from prior runs)
    await page.locator('table tbody tr').filter({ hasText: NEW_CUSTOMER.name }).first().click();
    await expect(page).toHaveURL(/\/customers\/[a-f0-9-]+/, { timeout: 10000 });

    // Verify customer data
    await expect(page.getByText(NEW_CUSTOMER.name).first()).toBeVisible({ timeout: 10000 });
    await expect(page.getByText(NEW_CUSTOMER.email!).first()).toBeVisible();

    customerUrl = page.url();
  });

  test('edit customer', async ({ page }) => {
    await gotoWithAuth(page, customerUrl);
    await expect(page.getByText(NEW_CUSTOMER.name).first()).toBeVisible({ timeout: 10000 });

    // Click edit button
    await page.getByRole('button', { name: 'Edytuj' }).click();

    // Update phone number (edit form uses #edit-phone, not #phone)
    await page.locator('#edit-phone').fill('+48 700 800 900');
    await page.getByRole('button', { name: 'Zapisz' }).click();
    await waitForToast(page, /zaktualizowan|zapisan/i);

    // Verify updated phone
    await expect(page.getByText('+48 700 800 900')).toBeVisible({ timeout: 5000 });
  });

  test('search customer by name', async ({ page }) => {
    await gotoWithAuth(page, '/customers');
    await waitForTableLoaded(page);

    const search = page.getByPlaceholder(/Szukaj/);
    await search.fill(NEW_CUSTOMER.name);
    // Customer search uses form submit, not auto-debounce
    await page.getByRole('button', { name: /Szukaj/i }).click();
    await expect(page.getByText(NEW_CUSTOMER.name).first()).toBeVisible({ timeout: 10000 });
  });

  test('delete customer', async ({ page }) => {
    await gotoWithAuth(page, customerUrl);
    await expect(page.getByText(NEW_CUSTOMER.name).first()).toBeVisible({ timeout: 10000 });

    await page.getByRole('button', { name: 'Usuń' }).click();
    await confirmDeleteDialog(page, 'Usuń klienta');
    await waitForToast(page, /usunięt/i);

    // Should redirect to customers list
    await expect(page).toHaveURL('/customers', { timeout: 10000 });
  });
});
