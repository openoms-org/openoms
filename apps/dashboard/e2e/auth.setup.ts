import { test as setup, expect } from '@playwright/test';
import { TEST_CREDENTIALS } from './fixtures/test-data';

const authFile = 'e2e/.auth/user.json';

setup('authenticate', async ({ page }) => {
  await page.goto('/login');
  await page.getByLabel('Organizacja').fill(TEST_CREDENTIALS.tenant_slug);
  await page.getByLabel('Email').fill(TEST_CREDENTIALS.email);
  await page.locator('#password').fill(TEST_CREDENTIALS.password);
  await page.getByRole('button', { name: 'Zaloguj się' }).click();
  await expect(page).toHaveURL('/', { timeout: 15000 });
  await expect(page.getByText('Panel główny')).toBeVisible({ timeout: 10000 });
  await page.context().storageState({ path: authFile });
});
