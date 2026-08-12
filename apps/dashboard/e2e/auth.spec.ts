import { test, expect } from '@playwright/test';
import { LOGIN_COPY } from './helpers/messages';

/**
 * The login screen only renders the "register" link when the deployment
 * actually offers self-registration (see (auth)/login/page.tsx). The dev stack
 * runs in "invite" mode without a license, so the link is legitimately absent —
 * asserting it unconditionally tested the fixture, not the app.
 */
async function registrationLinkOffered(request: import('@playwright/test').APIRequestContext) {
  const resp = await request.get('/v1/config/public');
  expect(resp.ok()).toBeTruthy();
  const config = (await resp.json()) as {
    registration_mode: string;
    license_enabled: boolean;
  };
  return (
    config.registration_mode === 'open' ||
    (config.registration_mode === 'invite' && config.license_enabled)
  );
}

test.use({ storageState: { cookies: [], origins: [] } });

test.describe('Authentication', () => {
  test('shows login form with all fields', async ({ page }) => {
    await page.goto('/login');
    await expect(page.getByText(LOGIN_COPY.title)).toBeVisible({ timeout: 15000 });
    await expect(page.getByLabel(LOGIN_COPY.organization)).toBeVisible();
    await expect(page.getByLabel(LOGIN_COPY.email)).toBeVisible();
    await expect(page.locator('#password')).toBeVisible();
    await expect(page.getByRole('button', { name: LOGIN_COPY.submit })).toBeVisible();
  });

  test('shows validation errors for empty form submission', async ({ page }) => {
    await page.goto('/login');
    await expect(page.getByText(LOGIN_COPY.title)).toBeVisible({ timeout: 15000 });
    await page.getByRole('button', { name: LOGIN_COPY.submit }).click();
    await expect(page.getByText(LOGIN_COPY.orgRequired)).toBeVisible();
    await expect(page.getByText(LOGIN_COPY.passwordRequired)).toBeVisible();
  });

  test('shows error for invalid credentials', async ({ page }) => {
    await page.goto('/login');
    await expect(page.getByText(LOGIN_COPY.title)).toBeVisible({ timeout: 15000 });
    await page.getByLabel(LOGIN_COPY.organization).fill('dev');
    await page.getByLabel(LOGIN_COPY.email).fill('wrong@example.com');
    await page.locator('#password').fill('wrongpassword');
    await page.getByRole('button', { name: LOGIN_COPY.submit }).click();
    await expect(page.locator('[data-sonner-toast]')).toBeVisible({ timeout: 5000 });
  });

  test('successfully logs in with valid credentials', async ({ page }) => {
    await page.goto('/login');
    await expect(page.getByText(LOGIN_COPY.title)).toBeVisible({ timeout: 15000 });
    await page.getByLabel(LOGIN_COPY.organization).fill('dev');
    await page.getByLabel(LOGIN_COPY.email).fill('admin@dev.local');
    await page.locator('#password').fill('password123');

    const [response] = await Promise.all([
      page.waitForResponse((resp) => resp.url().includes('/v1/auth/login'), {
        timeout: 10000,
      }),
      page.getByRole('button', { name: LOGIN_COPY.submit }).click(),
    ]);

    expect(response.status()).toBe(200);
    await expect(page).toHaveURL(/\/(orders)?$/, { timeout: 15000 });
  });

  test('can toggle password visibility', async ({ page }) => {
    await page.goto('/login');
    await expect(page.getByText(LOGIN_COPY.title)).toBeVisible({ timeout: 15000 });
    const passwordInput = page.locator('#password');
    await expect(passwordInput).toHaveAttribute('type', 'password');
    await page.getByRole('button', { name: LOGIN_COPY.showPassword }).click();
    await expect(passwordInput).toHaveAttribute('type', 'text');
    await page.getByRole('button', { name: LOGIN_COPY.hidePassword }).click();
    await expect(passwordInput).toHaveAttribute('type', 'password');
  });

  test('register link matches the deployment registration mode', async ({ page, request }) => {
    const offered = await registrationLinkOffered(request);

    await page.goto('/login');
    await expect(page.getByText(LOGIN_COPY.title)).toBeVisible({ timeout: 15000 });

    const registerLink = page.getByRole('link', { name: LOGIN_COPY.register });

    if (!offered) {
      await expect(registerLink).toHaveCount(0);
      return;
    }

    await registerLink.click();
    await expect(page).toHaveURL(/\/register/, { timeout: 10000 });
  });
});
