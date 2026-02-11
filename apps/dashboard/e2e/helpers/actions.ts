import { type Page, expect } from '@playwright/test';

export async function waitForTableLoaded(page: Page) {
  // Wait for skeleton/loading to disappear and rows to appear
  await expect(page.locator('table tbody tr').first()).toBeVisible({ timeout: 15000 });
}

export async function waitForToast(page: Page, text: string) {
  await expect(page.locator('[data-sonner-toast]').filter({ hasText: text })).toBeVisible({ timeout: 5000 });
}

export async function waitForPageReady(page: Page) {
  // Wait for Next.js hydration to complete
  await page.waitForLoadState('domcontentloaded');
}

/**
 * Navigate to a URL and wait for the AuthProvider hydrate to complete.
 * AuthProvider calls POST /v1/auth/refresh on mount — if we don't wait
 * for it, a failed/pending refresh clears the has_session cookie and
 * subsequent client-side navigations redirect to /login.
 */
export async function gotoWithAuth(page: Page, url: string) {
  const refreshDone = page.waitForResponse(
    (resp) => resp.url().includes('/v1/auth/refresh'),
    { timeout: 15000 },
  );
  await page.goto(url);
  try {
    await refreshDone;
  } catch {
    // The refresh response may have arrived before the listener was set up
    // (race condition observed in Firefox). If so, the page has already loaded
    // with auth intact, so we can safely continue.
  }
}
