import { test, expect } from '@playwright/test';

test.describe('Settings', () => {
  test('warehouses page loads', async ({ page }) => {
    await page.goto('/settings/warehouses');
    await expect(page.getByRole('heading', { name: /Magazyny/ })).toBeVisible();
    await page.waitForLoadState('networkidle');
  });

  test('company settings page loads', async ({ page }) => {
    await page.goto('/settings/company');
    await expect(page.getByRole('heading', { name: /Firma|Dane firmy/ })).toBeVisible();
  });

  test('order statuses page loads', async ({ page }) => {
    await page.goto('/settings/order-statuses');
    await expect(page.getByRole('heading', { name: /Statusy/ })).toBeVisible();
  });

  test('webhooks page loads', async ({ page }) => {
    await page.goto('/settings/webhooks');
    await expect(page.getByRole('heading', { name: /Webhooki/ })).toBeVisible();
  });

  test('users page loads', async ({ page }) => {
    await page.goto('/settings/users');
    await expect(page.getByRole('heading', { name: /Użytkownicy/ })).toBeVisible();
  });

  test('roles page loads', async ({ page }) => {
    await page.goto('/settings/roles');
    await expect(page.getByRole('heading', { name: /Role/ })).toBeVisible();
  });

  test('automation page loads', async ({ page }) => {
    await page.goto('/settings/automation');
    await expect(page.getByRole('heading', { name: /Automatyzacja/ })).toBeVisible();
  });
});
