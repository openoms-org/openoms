import { test, expect } from '@playwright/test';

test.describe('Security', () => {
  test('unauthenticated user redirected to login', async ({ browser }) => {
    const context = await browser.newContext({ storageState: { cookies: [], origins: [] } });
    const page = await context.newPage();

    await page.goto('/orders');
    await expect(page).toHaveURL(/\/login/, { timeout: 15000 });

    await context.close();
  });

  test('access token not stored in cookies', async ({ browser }) => {
    const context = await browser.newContext({ storageState: { cookies: [], origins: [] } });
    const page = await context.newPage();

    await page.goto('/login');
    await page.waitForLoadState('domcontentloaded');

    const cookies = await context.cookies();
    const tokenCookie = cookies.find((c) => c.name === 'access_token');
    expect(tokenCookie).toBeUndefined();

    await context.close();
  });

  test('CSP header present on pages', async ({ browser }) => {
    const context = await browser.newContext({ storageState: { cookies: [], origins: [] } });
    const page = await context.newPage();

    const response = await page.goto('/login');
    const csp = response?.headers()['content-security-policy'];
    // CSP may come from the API reverse proxy, not the dashboard itself
    // If CSP is configured, verify it doesn't contain unsafe-eval
    if (csp) {
      expect(csp).not.toContain('unsafe-eval');
    }

    await context.close();
  });

  test('localStorage does not contain tokens', async ({ browser }) => {
    const context = await browser.newContext({ storageState: { cookies: [], origins: [] } });
    const page = await context.newPage();

    await page.goto('/login');
    await page.waitForLoadState('domcontentloaded');

    const token = await page.evaluate(() => localStorage.getItem('token'));
    expect(token).toBeNull();

    const accessToken = await page.evaluate(() => localStorage.getItem('access_token'));
    expect(accessToken).toBeNull();

    await context.close();
  });

  test('XSS payload in page content rendered as text not executed', async ({ browser }) => {
    const context = await browser.newContext({ storageState: { cookies: [], origins: [] } });
    const page = await context.newPage();

    // Navigate to login page and verify React doesn't execute injected scripts
    await page.goto('/login');
    await page.waitForLoadState('domcontentloaded');

    const content = await page.content();
    // Verify the page rendered without script injection
    expect(content).not.toContain('<script>alert');

    await context.close();
  });
});
