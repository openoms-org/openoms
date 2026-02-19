import { test, expect } from '@playwright/test';
import { gotoWithAuth } from './helpers/actions';

test.describe('Dashboard', () => {
  test('root path redirects to orders page', async ({ page }) => {
    await gotoWithAuth(page, '/');
    await expect(page).toHaveURL(/\/orders/, { timeout: 10000 });
    await expect(page.getByRole('heading', { name: 'Zamówienia' })).toBeVisible({ timeout: 10000 });
  });

  test('orders page displays stat-like content after redirect', async ({ page }) => {
    await gotoWithAuth(page, '/');
    await expect(page).toHaveURL(/\/orders/, { timeout: 10000 });
    // Wait for data to load
    await page.waitForLoadState('networkidle');
    // Orders page should have a table or some visible content
    await expect(page.locator('table').first()).toBeVisible({ timeout: 10000 });
  });

  test('orders page has a heading after redirect', async ({ page }) => {
    await gotoWithAuth(page, '/');
    await expect(page).toHaveURL(/\/orders/, { timeout: 10000 });
    await page.waitForLoadState('networkidle');
    await expect(page.getByRole('heading', { name: 'Zamówienia' })).toBeVisible({ timeout: 10000 });
  });

  test('orders page has action buttons', async ({ page }) => {
    await gotoWithAuth(page, '/');
    await expect(page).toHaveURL(/\/orders/, { timeout: 10000 });
    await page.waitForLoadState('networkidle');
    // Should have a "Nowe zamówienie" link/button
    await expect(page.getByRole('link', { name: /Nowe zamówienie/ })).toBeVisible({ timeout: 10000 });
  });
});
