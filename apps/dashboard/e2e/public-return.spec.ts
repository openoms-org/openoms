import { test, expect } from '@playwright/test';

test.use({ storageState: { cookies: [], origins: [] } });

test.describe('Public Return Request', () => {
  test('renders public return form', async ({ page }) => {
    await page.goto('/return-request');
    await expect(page.getByText('Formularz zwrotu towaru')).toBeVisible({ timeout: 10000 });
    // "Zglos zwrot" appears as both CardTitle and Button — check the button specifically
    await expect(page.getByRole('button', { name: /Zgłoś zwrot/ })).toBeVisible();
  });

  test('shows required form fields', async ({ page }) => {
    await page.goto('/return-request');
    await expect(page.getByText('Formularz zwrotu towaru')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('#order-id')).toBeVisible();
    await expect(page.locator('#email')).toBeVisible();
    await expect(page.locator('#reason')).toBeVisible();
  });

  test('submit button is disabled when fields are empty', async ({ page }) => {
    await page.goto('/return-request');
    await expect(page.getByText('Formularz zwrotu towaru')).toBeVisible({ timeout: 10000 });
    const submitButton = page.getByRole('button', { name: /Zgłoś zwrot/ });
    await expect(submitButton).toBeDisabled();
  });

  test('pre-fills order_id from URL parameter', async ({ page }) => {
    await page.goto('/return-request?order_id=test-order-123');
    await expect(page.getByText('Formularz zwrotu towaru')).toBeVisible({ timeout: 10000 });
    const orderIdInput = page.locator('#order-id');
    await expect(orderIdInput).toHaveValue('test-order-123', { timeout: 5000 });
  });
});
