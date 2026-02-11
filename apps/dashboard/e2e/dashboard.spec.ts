import { test, expect } from '@playwright/test';

test.describe('Dashboard', () => {
  test('displays dashboard heading and greeting', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByRole('heading', { name: 'Panel główny' })).toBeVisible();
    await expect(page.getByText('Witaj, Rafał Strzelczyk!')).toBeVisible();
  });

  test('displays stat cards', async ({ page }) => {
    await page.goto('/');
    // StatCards should show at least the "Zamówienia" card heading or similar stats
    // Wait for data to load
    await page.waitForLoadState('networkidle');
    // Check for stat card containers (grid of cards)
    const cards = page.locator('[class*="grid"] > div').filter({ has: page.locator('p') });
    await expect(cards.first()).toBeVisible({ timeout: 10000 });
  });

  test('displays charts section', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    // The recharts SVG elements should be rendered
    // Check for chart containers with card titles
    await expect(page.getByText('Przychody')).toBeVisible({ timeout: 10000 });
  });

  test('displays recent orders table', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    // RecentOrdersTable has a "Ostatnie zamówienia" heading
    await expect(page.getByText('Ostatnie zamówienia')).toBeVisible({ timeout: 10000 });
  });
});
