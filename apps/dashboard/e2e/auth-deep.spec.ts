import { test, expect } from '@playwright/test';
import { gotoWithAuth } from './helpers/actions';

// Use a DIFFERENT user for logout tests to avoid invalidating the shared auth state.
// The main auth setup uses admin@dev.local — logout updates last_logout_at which
// invalidates all tokens issued before that time.
const LOGOUT_USER = {
  tenant_slug: 'dev',
  email: 'e2e-logout@dev.local',
  password: 'password123',
};

test.describe('Auth Deep Flows', () => {
  test('session persists after page reload', async ({ page }) => {
    await gotoWithAuth(page, '/orders');
    await expect(page.getByRole('heading', { name: 'Zamówienia' })).toBeVisible({ timeout: 10000 });

    await page.reload();
    await page.waitForLoadState('domcontentloaded');
    await expect(page.getByRole('heading', { name: 'Zamówienia' })).toBeVisible({ timeout: 15000 });
  });

  test('unauthenticated user is redirected to login', async ({ browser }) => {
    const context = await browser.newContext({ storageState: { cookies: [], origins: [] } });
    const page = await context.newPage();

    await page.goto('/orders');
    await expect(page).toHaveURL(/\/login/, { timeout: 15000 });

    await context.close();
  });

  // Use a fresh login session for logout tests to avoid invalidating the shared auth state
  test('logout redirects to login page', async ({ browser }) => {
    const context = await browser.newContext({ storageState: { cookies: [], origins: [] } });
    const page = await context.newPage();

    // Login fresh
    await page.goto('/login');
    await page.getByLabel('Organizacja').fill(LOGOUT_USER.tenant_slug);
    await page.getByLabel('Email').fill(LOGOUT_USER.email);
    await page.locator('#password').fill(LOGOUT_USER.password);
    await page.getByRole('button', { name: 'Zaloguj się' }).click();
    await expect(page).toHaveURL(/\/(orders)?$/, { timeout: 15000 });

    // Open user menu and click logout
    await page.locator('button.rounded-full').first().click();
    await page.getByRole('menuitem', { name: 'Wyloguj się' }).click();

    await expect(page).toHaveURL(/\/login/, { timeout: 10000 });
    await expect(page.getByText('Logowanie')).toBeVisible();

    await context.close();
  });

  test('after logout, dashboard is inaccessible', async ({ browser }) => {
    const context = await browser.newContext({ storageState: { cookies: [], origins: [] } });
    const page = await context.newPage();

    // Login fresh
    await page.goto('/login');
    await page.getByLabel('Organizacja').fill(LOGOUT_USER.tenant_slug);
    await page.getByLabel('Email').fill(LOGOUT_USER.email);
    await page.locator('#password').fill(LOGOUT_USER.password);
    await page.getByRole('button', { name: 'Zaloguj się' }).click();
    await expect(page).toHaveURL(/\/(orders)?$/, { timeout: 15000 });

    // Logout
    await page.locator('button.rounded-full').first().click();
    await page.getByRole('menuitem', { name: 'Wyloguj się' }).click();
    await expect(page).toHaveURL(/\/login/, { timeout: 10000 });

    // Try to access protected page — should redirect back to login
    await page.goto('/orders');
    await expect(page).toHaveURL(/\/login/, { timeout: 15000 });

    await context.close();
  });

  test('register page is accessible from login', async ({ browser }) => {
    const context = await browser.newContext({ storageState: { cookies: [], origins: [] } });
    const page = await context.newPage();

    await page.goto('/login');
    await expect(page.getByText('Logowanie')).toBeVisible({ timeout: 10000 });

    await page.getByRole('link', { name: 'Zarejestruj się' }).click();
    await expect(page).toHaveURL(/\/register/, { timeout: 10000 });

    await context.close();
  });
});
