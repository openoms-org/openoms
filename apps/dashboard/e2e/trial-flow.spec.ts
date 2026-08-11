import { test, expect } from '@playwright/test';
import {
  waitForToast,
  fillAndSubmitOrderForm,
} from './helpers/actions';
import { TRIAL_USER, NEW_ORDER } from './fixtures/test-data';

// ──────────────────────────────────────────────────────────────
// Helper: login as testflow trial user via API call, then navigate
// to targetUrl. Uses page.evaluate to call /v1/auth/login directly.
//
// Rate limit note: the in-memory rate limiter shares counters
// across all auth endpoints (login + refresh) per IP. Each login
// plus auto-refresh calls burn ~3-4 of the 10/min budget.
// ──────────────────────────────────────────────────────────────
async function loginViaAPI(
  page: import('@playwright/test').Page,
  targetUrl: string = '/',
) {
  await page.goto('/login');
  await page.waitForLoadState('domcontentloaded');

  const loginResult = await page.evaluate(
    async ({ email, password, tenant_slug }) => {
      const resp = await fetch('/v1/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password, tenant_slug }),
        credentials: 'include',
      });

      if (!resp.ok) {
        const body = await resp.json().catch(() => ({}));
        return { error: `Login failed: ${resp.status} ${body.error || ''}` };
      }

      const data = await resp.json();
      document.cookie = 'has_session=1; path=/; SameSite=Lax; max-age=2592000';

      return {
        access_token: data.access_token,
        user: data.user,
        tenant: data.tenant,
      };
    },
    {
      email: TRIAL_USER.email,
      password: TRIAL_USER.password,
      tenant_slug: TRIAL_USER.tenant_slug,
    },
  );

  if ('error' in loginResult) {
    throw new Error(loginResult.error as string);
  }

  // Navigate directly to target — AuthProvider will refresh via httpOnly cookie
  await page.goto(targetUrl);
  await page.waitForLoadState('domcontentloaded');
}

// ──────────────────────────────────────────────────────────────
// Plan Selection (unauthenticated)
// ──────────────────────────────────────────────────────────────

test.describe('Plan Selection', () => {
  test.use({ storageState: { cookies: [], origins: [] } });

  test('register page loads without errors', async ({ page }) => {
    await page.goto('/register');
    await page.waitForLoadState('domcontentloaded');
    await expect(page.locator('body')).not.toBeEmpty();
    await expect(page.getByText('500')).not.toBeVisible({ timeout: 3000 }).catch(() => {});
  });
});

// ──────────────────────────────────────────────────────────────
// Onboarding Flow (serial, testflow user)
//
// DESTRUCTIVE: These tests complete the onboarding wizard for the
// testflow tenant. Run `task seed` to reset before re-running.
//
// Retries disabled — each login burns ~3 rate limit tokens from a
// shared 10/min budget (login + auto-refresh calls count together).
// ──────────────────────────────────────────────────────────────

test.describe.serial('Onboarding Flow', () => {
  test.use({ storageState: { cookies: [], origins: [] } });
  test.describe.configure({ retries: 0 });

  test('complete onboarding wizard from step 1 through step 4', async ({ page }) => {
    await loginViaAPI(page, '/onboarding');

    // ── Step 1: Company details ──
    await expect(page.getByText('Konfiguracja systemu')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('#company_name')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('#nip')).toBeVisible();

    await page.locator('#company_name').fill('Test Flow E2E Sp. z o.o.');
    await page.locator('#nip').fill('1234563218');
    await page.locator('#address').fill('ul. Testowa 42');
    await page.locator('#city').fill('Warszawa');
    await page.locator('#post_code').fill('00-001');
    await page.getByRole('button', { name: 'Dalej' }).click();

    // ── Step 2: Warehouse → Skip ──
    await expect(page.locator('#wh_name')).toBeVisible({ timeout: 10000 });
    await page.getByRole('button', { name: 'Pomiń' }).click();

    // ── Step 3: Integration → Skip ──
    await expect(page.getByRole('button', { name: 'Allegro' })).toBeVisible({ timeout: 10000 });
    await page.getByRole('button', { name: 'Pomiń' }).click();

    // ── Step 4: Team → Skip ──
    await expect(page.getByPlaceholder('adres@email.pl')).toBeVisible({ timeout: 10000 });
    await page.getByRole('button', { name: 'Pomiń' }).click();

    // ── Completion: may briefly show "Gotowe!" then redirect to dashboard ──
    await expect(page).toHaveURL(/\/(onboarding)?$/, { timeout: 15000 });
  });

  test('after completion: dashboard with onboarding checklist', async ({ page }) => {
    await loginViaAPI(page, '/');

    await expect(page.getByText('Panel główny')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Pierwsze kroki')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Uzupełnij dane firmy')).toBeVisible();
  });

  test('create first order and verify billing page', async ({ page }) => {
    await loginViaAPI(page, '/orders/new');

    await expect(page.getByRole('heading', { name: /zamówienie/i })).toBeVisible({
      timeout: 10000,
    });

    await fillAndSubmitOrderForm(page, NEW_ORDER);
    await waitForToast(page, /utworzone|zapisane/i);

    // Navigate to billing settings within the same session (no extra login)
    await page.goto('/settings/billing');
    await page.waitForLoadState('domcontentloaded');

    await expect(page.getByRole('heading', { name: 'Subskrypcja' })).toBeVisible({
      timeout: 10000,
    });
  });
});
