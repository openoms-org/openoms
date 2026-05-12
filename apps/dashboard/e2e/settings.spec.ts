import { test, expect } from '@playwright/test';
import { gotoWithAuth } from './helpers/actions';

test.describe('Settings', () => {
  test('warehouses page loads', async ({ page }) => {
    await gotoWithAuth(page, '/settings/warehouses');
    await expect(page.getByRole('heading', { name: /Magazyny/ })).toBeVisible({ timeout: 10000 });
  });

  test('company settings page loads', async ({ page }) => {
    await gotoWithAuth(page, '/settings/company');
    await expect(page.getByRole('heading', { name: /Firma|Dane firmy/ })).toBeVisible({ timeout: 10000 });
  });

  test('order statuses page loads', async ({ page }) => {
    await gotoWithAuth(page, '/settings/order-statuses');
    await expect(page.getByRole('heading', { name: /Statusy/ })).toBeVisible({ timeout: 10000 });
  });

  test('webhooks page loads', async ({ page }) => {
    await gotoWithAuth(page, '/settings/webhooks');
    await expect(page.getByRole('heading', { name: /Webhooki/ })).toBeVisible({ timeout: 10000 });
  });

  test('users page loads', async ({ page }) => {
    await gotoWithAuth(page, '/settings/users');
    await expect(page.getByRole('heading', { name: /Użytkownicy/ })).toBeVisible({ timeout: 10000 });
  });

  test('roles page loads', async ({ page }) => {
    await gotoWithAuth(page, '/settings/roles');
    await expect(page.getByRole('heading', { name: /Role/ })).toBeVisible({ timeout: 10000 });
  });

  test('automation page loads', async ({ page }) => {
    await gotoWithAuth(page, '/settings/automation');
    await expect(page.getByRole('heading', { name: /Automatyzacja/ })).toBeVisible({ timeout: 10000 });
  });

  test('custom fields page loads', async ({ page }) => {
    await gotoWithAuth(page, '/settings/custom-fields');
    await expect(page.getByRole('heading', { name: /Pola|Niestandardowe/ })).toBeVisible({ timeout: 10000 });
  });

  test('inventory settings page loads', async ({ page }) => {
    await gotoWithAuth(page, '/settings/inventory');
    await expect(page.getByRole('heading', { name: /Inwentaryzacja|Magazyn|Ustawienia magazynowe/ })).toBeVisible({ timeout: 10000 });
  });
});
