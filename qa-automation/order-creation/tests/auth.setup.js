const { test: setup, expect } = require('@playwright/test');
const fs = require('fs');
const path = require('path');
const { TEST_CREDENTIALS } = require('../../../apps/dashboard/e2e/fixtures/test-data');

const authDir = path.resolve(__dirname, '../.auth');
const authFile = path.join(authDir, 'user.json');

fs.mkdirSync(authDir, { recursive: true });

setup('authenticate', async ({ page }) => {
  await page.goto('/login', { waitUntil: 'domcontentloaded' });

  await page.getByRole('textbox', { name: 'Organization' })
    .fill(TEST_CREDENTIALS.tenant_slug);

  await page.getByRole('textbox', { name: 'Email' })
    .fill(TEST_CREDENTIALS.email);

  await page.getByRole('textbox', { name: 'Password' })
    .fill(TEST_CREDENTIALS.password);

  await page.getByRole('button', { name: 'Log in' }).click();

  await expect(page).not.toHaveURL(/\/login$/, {
    timeout: 15000,
  });

  await page.waitForLoadState('domcontentloaded');

  await expect(page).not.toHaveURL(/\/login$/, {
    timeout: 15000,
  });

  await page.context().storageState({
    path: authFile,
  });

  console.log(`AUTH STATE SAVED: ${authFile}`);
});
