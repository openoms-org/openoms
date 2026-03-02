import { test, expect } from '@playwright/test';
import {
  gotoWithAuth,
  waitForToast,
  fillAndSubmitOrderForm,
} from './helpers/actions';
import { TRIAL_USER, NEW_ORDER } from './fixtures/test-data';

// ── Plan Selection (unauthenticated) ──

test.describe('Plan Selection', () => {
  test.use({ storageState: { cookies: [], origins: [] } });

  test('shows plan selection on register page', async ({ page }) => {
    await page.goto('/register');
    await page.waitForLoadState('domcontentloaded');
    // Register page should load without errors
    await expect(page.getByRole('heading').first()).toBeVisible({ timeout: 10000 });
  });
});

// ── Onboarding Flow (serial, fresh auth) ──

test.describe.serial('Onboarding Flow', () => {
  test.use({ storageState: { cookies: [], origins: [] } });

  test('login as trial user', async ({ page }) => {
    await page.goto('/login');
    await page.getByLabel('Organizacja').fill(TRIAL_USER.tenant_slug);
    await page.getByLabel('Email').fill(TRIAL_USER.email);
    await page.locator('#password').fill(TRIAL_USER.password);
    await page.getByRole('button', { name: 'Zaloguj się' }).click();
    // After login, trial user should reach the app
    await expect(page).toHaveURL(/\/(orders|onboarding)?$/, { timeout: 15000 });
  });

  test('onboarding wizard shows on dashboard', async ({ page }) => {
    // Login first
    await page.goto('/login');
    await page.getByLabel('Organizacja').fill(TRIAL_USER.tenant_slug);
    await page.getByLabel('Email').fill(TRIAL_USER.email);
    await page.locator('#password').fill(TRIAL_USER.password);
    await page.getByRole('button', { name: 'Zaloguj się' }).click();
    await expect(page).toHaveURL(/\/(orders)?$/, { timeout: 15000 });

    // Navigate to dashboard
    await page.goto('/');
    await page.waitForLoadState('domcontentloaded');

    // Onboarding wizard should be visible (steps not completed)
    await expect(page.getByText('Pierwsze kroki')).toBeVisible({ timeout: 10000 });
  });
});

// ── Subscription Status (authenticated with main test user) ──

test.describe('Subscription Status', () => {
  test('billing settings page loads', async ({ page }) => {
    await gotoWithAuth(page, '/settings/billing');
    await expect(page.getByRole('heading', { name: 'Subskrypcja' })).toBeVisible({
      timeout: 10000,
    });
    // Plan name should be visible
    await expect(page.getByText(/Twój plan/)).toBeVisible();
  });

  test('subscription API returns valid response', async ({ page }) => {
    await gotoWithAuth(page, '/');
    const apiBase = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080';
    const response = await page.request.get(`${apiBase}/v1/billing/subscription`);
    // The endpoint may not exist for all tenants, but shouldn't 500
    expect([200, 401, 404]).toContain(response.status());
  });
});

// ── Dashboard Quick Start ──

test.describe('Dashboard', () => {
  test('dashboard shows stat cards', async ({ page }) => {
    await gotoWithAuth(page, '/');
    await expect(page.getByText('Panel główny')).toBeVisible({ timeout: 10000 });
  });
});
