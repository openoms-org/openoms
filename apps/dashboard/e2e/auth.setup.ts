import { test as setup, expect } from '@playwright/test';
import { TEST_CREDENTIALS } from './fixtures/test-data';

const authFile = 'e2e/.auth/user.json';

setup('authenticate', async ({ page }) => {
  await page.goto('/login');
  await page.waitForLoadState('domcontentloaded');

  // Use API-based login to avoid react-hook-form reliability issues
  const loginResult = await page.evaluate(
    async ({ email, password, tenant_slug }) => {
      const apiUrl = 'http://localhost:8080';
      const resp = await fetch(`${apiUrl}/v1/auth/login`, {
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
      return { ok: true };
    },
    TEST_CREDENTIALS,
  );

  if ('error' in loginResult) {
    throw new Error(loginResult.error as string);
  }

  await page.goto('/');
  await page.waitForLoadState('networkidle');
  // Debug: log page content if dashboard text not found
  const dashboardVisible = await page.getByText('Panel główny').isVisible().catch(() => false);
  if (!dashboardVisible) {
    const html = await page.content();
    const title = await page.title();
    console.log(`[auth.setup] Page title: ${title}`);
    console.log(`[auth.setup] Page URL: ${page.url()}`);
    console.log(`[auth.setup] Page HTML (first 2000 chars): ${html.substring(0, 2000)}`);
  }
  await expect(page.getByText('Panel główny')).toBeVisible({ timeout: 15000 });
  await page.context().storageState({ path: authFile });
});
