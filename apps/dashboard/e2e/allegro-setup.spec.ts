import { test, expect } from '@playwright/test';
import { gotoWithAuth } from './helpers/actions';

test.describe('Allegro Integration', () => {
  test('integrations page loads', async ({ page }) => {
    await gotoWithAuth(page, '/integrations');
    await expect(page.getByRole('heading', { name: /Integracje|Kanały sprzedaży|Połączenia/ })).toBeVisible({ timeout: 10000 });
  });

  test('allegro integration page loads', async ({ page }) => {
    await gotoWithAuth(page, '/marketplaces/allegro');
    await expect(page.getByRole('heading', { name: /Allegro/ })).toBeVisible({ timeout: 15000 });
  });

  test('allegro offers subpage loads', async ({ page }) => {
    await gotoWithAuth(page, '/marketplaces/allegro/offers');
    await expect(page.getByRole('heading', { name: /Oferty/ })).toBeVisible({ timeout: 10000 });
  });

  test('allegro returns subpage loads', async ({ page }) => {
    await gotoWithAuth(page, '/marketplaces/allegro/returns');
    await expect(page.getByRole('heading', { name: /Zwroty/ })).toBeVisible({ timeout: 10000 });
  });

  test('allegro messages subpage loads', async ({ page }) => {
    await gotoWithAuth(page, '/marketplaces/allegro/messages');
    await expect(page.getByRole('heading', { name: /Wiadomości/ })).toBeVisible({ timeout: 10000 });
  });
});
