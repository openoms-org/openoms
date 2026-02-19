import { test, expect } from '@playwright/test';
import { gotoWithAuth, openUserMenu } from './helpers/actions';

test.describe('Auth Deep Flows', () => {
  test('session persists after page reload', async ({ page }) => {
    await gotoWithAuth(page, '/');
    await expect(page.getByText('Panel główny')).toBeVisible({ timeout: 10000 });

    await page.reload();
    await page.waitForLoadState('domcontentloaded');
    await expect(page.getByText('Panel główny')).toBeVisible({ timeout: 15000 });
  });

  test('unauthenticated user is redirected to login', async ({ browser }) => {
    // Create a fresh context without any saved auth state
    const context = await browser.newContext();
    const page = await context.newPage();

    await page.goto('/orders');
    await expect(page).toHaveURL(/\/login/, { timeout: 15000 });

    await context.close();
  });

  test('logout redirects to login page', async ({ page }) => {
    await gotoWithAuth(page, '/');
    await expect(page.getByText('Panel główny')).toBeVisible({ timeout: 10000 });

    // Open user menu and click logout
    await openUserMenu(page);
    await page.getByRole('menuitem', { name: 'Wyloguj się' }).click();

    await expect(page).toHaveURL(/\/login/, { timeout: 10000 });
    await expect(page.getByText('Logowanie')).toBeVisible();
  });

  test('after logout, dashboard is inaccessible', async ({ page }) => {
    await gotoWithAuth(page, '/');
    await expect(page.getByText('Panel główny')).toBeVisible({ timeout: 10000 });

    // Logout
    await openUserMenu(page);
    await page.getByRole('menuitem', { name: 'Wyloguj się' }).click();
    await expect(page).toHaveURL(/\/login/, { timeout: 10000 });

    // Try to access protected page — should redirect back to login
    await page.goto('/orders');
    await expect(page).toHaveURL(/\/login/, { timeout: 15000 });
  });

  test('register page is accessible from login', async ({ browser }) => {
    const context = await browser.newContext();
    const page = await context.newPage();

    await page.goto('/login');
    await expect(page.getByText('Logowanie')).toBeVisible({ timeout: 10000 });

    await page.getByRole('link', { name: 'Zarejestruj się' }).click();
    await expect(page).toHaveURL('/register', { timeout: 10000 });

    await context.close();
  });
});
