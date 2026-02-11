import { test, expect } from '@playwright/test';

test.use({ storageState: { cookies: [], origins: [] } });

test.describe('Authentication', () => {
  test('shows login form with all fields', async ({ page }) => {
    await page.goto('/login');
    await expect(page.getByText('Logowanie')).toBeVisible();
    await expect(page.getByLabel('Organizacja')).toBeVisible();
    await expect(page.getByLabel('Email')).toBeVisible();
    await expect(page.locator('#password')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Zaloguj się' })).toBeVisible();
  });

  test('shows validation errors for empty form submission', async ({ page }) => {
    await page.goto('/login');
    await page.getByRole('button', { name: 'Zaloguj się' }).click();
    await expect(page.getByText('Slug organizacji jest wymagany')).toBeVisible();
    await expect(page.getByText('Hasło jest wymagane')).toBeVisible();
  });

  test('shows error for invalid credentials', async ({ page }) => {
    await page.goto('/login');
    await page.getByLabel('Organizacja').fill('mercpart');
    await page.getByLabel('Email').fill('wrong@example.com');
    await page.locator('#password').fill('wrongpassword');
    await page.getByRole('button', { name: 'Zaloguj się' }).click();
    // Wait for error toast
    await expect(page.locator('[data-sonner-toast]')).toBeVisible({ timeout: 5000 });
  });

  test('successfully logs in with valid credentials', async ({ page }) => {
    await page.goto('/login');
    await page.getByLabel('Organizacja').fill('mercpart');
    await page.getByLabel('Email').fill('rafal@mercpart.pl');
    await page.locator('#password').fill('password123');
    await page.getByRole('button', { name: 'Zaloguj się' }).click();
    await expect(page).toHaveURL('/', { timeout: 15000 });
    await expect(page.getByText('Panel główny')).toBeVisible({ timeout: 10000 });
  });

  test('can toggle password visibility', async ({ page }) => {
    await page.goto('/login');
    const passwordInput = page.locator('#password');
    await expect(passwordInput).toHaveAttribute('type', 'password');
    await page.getByRole('button', { name: 'Pokaż hasło' }).click();
    await expect(passwordInput).toHaveAttribute('type', 'text');
    await page.getByRole('button', { name: 'Ukryj hasło' }).click();
    await expect(passwordInput).toHaveAttribute('type', 'password');
  });

  test('register link navigates to registration page', async ({ page }) => {
    await page.goto('/login');
    await page.getByRole('link', { name: 'Zarejestruj się' }).click();
    await expect(page).toHaveURL('/register', { timeout: 10000 });
  });
});
