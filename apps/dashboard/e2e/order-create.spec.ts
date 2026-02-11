import { test, expect } from '@playwright/test';
import { gotoWithAuth } from './helpers/actions';

test.describe('Create Order', () => {
  test.beforeEach(async ({ page }) => {
    await gotoWithAuth(page, '/orders/new');
  });

  test('renders create order form', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /Nowe zamówienie/ })).toBeVisible();
  });

  test('shows form fields', async ({ page }) => {
    await expect(page.getByLabel('Nazwa klienta')).toBeVisible();
  });

  test('can fill basic order fields', async ({ page }) => {
    const nameInput = page.getByLabel('Nazwa klienta');
    await nameInput.fill('Test E2E Klient');
    await expect(nameInput).toHaveValue('Test E2E Klient');
  });

  test('navigating from orders list and back works', async ({ page }) => {
    // Build history: go to orders list first, then click new order
    await gotoWithAuth(page, '/orders');
    await expect(page.getByRole('heading', { name: 'Zamówienia' })).toBeVisible();
    await page.getByRole('link', { name: 'Nowe zamówienie' }).click();
    await expect(page).toHaveURL('/orders/new', { timeout: 10000 });
    // Now goBack has /orders in history
    await page.goBack();
    await expect(page).toHaveURL('/orders', { timeout: 10000 });
  });
});
