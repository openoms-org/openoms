import { test as setup, expect } from '@playwright/test';
import { TEST_CREDENTIALS } from './fixtures/test-data';

const authFile = 'e2e/.auth/user.json';

setup('authenticate', async ({ page }) => {
  await page.goto('/login');

  const response = await page.request.post('/v1/auth/login', {
    data: TEST_CREDENTIALS,
  });

  expect(response.ok()).toBeTruthy();

  // Navigate using the authenticated browser context.
  await page.goto('/');
  await page.waitForLoadState('domcontentloaded');

  // Make sure authentication actually survived.
  await expect(page).not.toHaveURL(/\/login/, {
    timeout: 15000,
  });

  await page.context().storageState({
    path: authFile,
  });
});