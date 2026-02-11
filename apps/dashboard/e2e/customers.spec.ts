import { test, expect } from '@playwright/test';

test.describe('Customers', () => {
  test('displays customers page', async ({ page }) => {
    await page.goto('/customers');
    // The page shows LoadingSkeleton first, then PageHeader after API responds
    await expect(page.getByRole('heading', { name: 'Klienci' })).toBeVisible({ timeout: 20000 });
  });

  test('new customer link works', async ({ page }) => {
    await page.goto('/customers');
    await expect(page.getByRole('heading', { name: 'Klienci' })).toBeVisible({ timeout: 20000 });
    await page.getByRole('link', { name: /Nowy klient/ }).first().click();
    await expect(page).toHaveURL('/customers/new', { timeout: 10000 });
  });

  test('new customer form renders correctly', async ({ page }) => {
    await page.goto('/customers/new');
    await expect(page.getByRole('heading', { name: 'Nowy klient' })).toBeVisible();
    await expect(page.getByLabel('Imię i nazwisko')).toBeVisible();
    await expect(page.getByLabel('E-mail')).toBeVisible();
    await expect(page.getByLabel('Telefon')).toBeVisible();
  });

  test('customer form validates required fields', async ({ page }) => {
    await page.goto('/customers/new');
    await page.getByRole('button', { name: /Utwórz klienta/ }).click();
    await expect(page.getByText(/wymagane/i)).toBeVisible({ timeout: 3000 });
  });
});
