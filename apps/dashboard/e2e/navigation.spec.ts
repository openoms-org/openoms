import { test, expect } from '@playwright/test';
import { gotoWithAuth } from './helpers/actions';

test.describe('Navigation', () => {
  test('sidebar renders main navigation items', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByRole('link', { name: 'Pulpit' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Zamówienia' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Produkty' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Przesyłki' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Zwroty' })).toBeVisible();
  });

  test('clicking sidebar links navigates correctly', async ({ page }) => {
    await gotoWithAuth(page, '/');
    await page.getByRole('link', { name: 'Zamówienia' }).click();
    await expect(page).toHaveURL('/orders', { timeout: 10000 });
    await expect(page.getByRole('heading', { name: 'Zamówienia' })).toBeVisible();

    await page.getByRole('link', { name: 'Produkty' }).click();
    await expect(page).toHaveURL('/products', { timeout: 10000 });
    await expect(page.getByRole('heading', { name: 'Produkty' })).toBeVisible();
  });

  test('admin sections are visible for owner role', async ({ page }) => {
    await gotoWithAuth(page, '/');
    // Admin links from nav-items.ts — they may need scrolling into view
    await expect(page.getByRole('link', { name: 'Magazyny' })).toBeAttached();
    await expect(page.getByRole('link', { name: 'Użytkownicy' })).toBeAttached();
    await expect(page.getByRole('link', { name: 'Webhooki' })).toBeAttached();
    await expect(page.getByRole('link', { name: 'Automatyzacja' })).toBeAttached();
  });

  test('navigates to settings pages', async ({ page }) => {
    await gotoWithAuth(page, '/');
    await page.getByRole('link', { name: 'Firma' }).click();
    await expect(page).toHaveURL('/settings/company', { timeout: 10000 });

    await page.getByRole('link', { name: 'Statusy' }).click();
    await expect(page).toHaveURL('/settings/order-statuses', { timeout: 10000 });
  });
});
