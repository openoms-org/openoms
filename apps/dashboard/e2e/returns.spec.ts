import { test, expect } from '@playwright/test';
import { gotoWithAuth } from './helpers/actions';

test.describe('Returns', () => {
  test.beforeEach(async ({ page }) => {
    await gotoWithAuth(page, '/returns');
    await expect(page.getByRole('heading', { name: 'Zwroty' })).toBeVisible({ timeout: 10000 });
  });

  test('displays returns page', async ({ page }) => {
    // Either shows table or empty state
    const tableOrEmpty = page.locator('table, [class*="empty"]');
    await expect(tableOrEmpty.first()).toBeVisible({ timeout: 10000 });
  });

  test('new return link navigates correctly', async ({ page }) => {
    await page.getByRole('link', { name: /Nowy zwrot/ }).click();
    await expect(page).toHaveURL('/returns/new', { timeout: 10000 });
  });

  test('back navigation works', async ({ page }) => {
    await page.getByRole('link', { name: /Nowy zwrot/ }).click();
    await expect(page).toHaveURL('/returns/new', { timeout: 10000 });
    await page.goBack();
    await expect(page).toHaveURL('/returns', { timeout: 10000 });
  });
});
