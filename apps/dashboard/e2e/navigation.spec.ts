import { test, expect } from '@playwright/test';
import { gotoWithAuth } from './helpers/actions';

/**
 * Expand a sidebar group by clicking its toggle button.
 * Groups are rendered as uppercase-labeled buttons in the sidebar.
 */
async function expandSidebarGroup(page: import('@playwright/test').Page, groupName: string) {
  const groupButton = page.locator('nav button').filter({ hasText: groupName });
  // Only click if the group is collapsed (next sibling with links not visible)
  const isExpanded = await groupButton.evaluate((btn) => {
    const chevron = btn.querySelector('svg');
    return chevron?.classList.contains('rotate-90') ?? false;
  });
  if (!isExpanded) {
    await groupButton.click();
  }
}

test.describe('Navigation', () => {
  test('sidebar renders main navigation items', async ({ page }) => {
    await gotoWithAuth(page, '/orders');
    // "Pulpit" is ungrouped — always visible
    await expect(page.getByRole('link', { name: 'Pulpit' })).toBeVisible();

    // "Zamówienia" and "Zwroty" are in "Sprzedaż" (defaultExpanded OR auto-expanded since we are on /orders)
    await expect(page.getByRole('link', { name: 'Zamówienia' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Zwroty' })).toBeVisible();

    // "Produkty" is in "Katalog" group — expand it first
    await expandSidebarGroup(page, 'Katalog');
    await expect(page.getByRole('link', { name: 'Produkty' })).toBeVisible();

    // "Przesyłki" is in "Logistyka" group — expand it first
    await expandSidebarGroup(page, 'Logistyka');
    await expect(page.getByRole('link', { name: 'Przesyłki' })).toBeVisible();
  });

  test('clicking sidebar links navigates correctly', async ({ page }) => {
    await gotoWithAuth(page, '/orders');
    // "Zamówienia" should already be visible (Sprzedaż auto-expanded)
    await page.getByRole('link', { name: 'Zamówienia' }).click();
    await expect(page).toHaveURL('/orders', { timeout: 10000 });
    await expect(page.getByRole('heading', { name: 'Zamówienia' })).toBeVisible();

    // Expand "Katalog" group to access "Produkty"
    await expandSidebarGroup(page, 'Katalog');
    await page.getByRole('link', { name: 'Produkty' }).click();
    await expect(page).toHaveURL('/products', { timeout: 10000 });
    await expect(page.getByRole('heading', { name: 'Produkty' })).toBeVisible();
  });

  test('admin sections are visible for owner role', async ({ page }) => {
    await gotoWithAuth(page, '/orders');

    // "Magazyny" is in "Logistyka"
    await expandSidebarGroup(page, 'Logistyka');
    await expect(page.getByRole('link', { name: 'Magazyny' })).toBeVisible();

    // "Użytkownicy" and "Webhooki" are in "Ustawienia"
    await expandSidebarGroup(page, 'Ustawienia');
    await expect(page.getByRole('link', { name: 'Użytkownicy' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Webhooki' })).toBeVisible();

    // "Automatyzacja" is in "Narzędzia"
    await expandSidebarGroup(page, 'Narzędzia');
    await expect(page.getByRole('link', { name: 'Automatyzacja' })).toBeVisible();
  });

  test('navigates to settings pages', async ({ page }) => {
    await gotoWithAuth(page, '/orders');

    // "Firma" and "Statusy zamówień" are in "Ustawienia" group
    await expandSidebarGroup(page, 'Ustawienia');

    await page.getByRole('link', { name: 'Firma' }).click();
    await expect(page).toHaveURL('/settings/company', { timeout: 10000 });

    // After navigation, the "Ustawienia" group should auto-expand since we're on a settings page
    await page.getByRole('link', { name: 'Statusy zamówień' }).click();
    await expect(page).toHaveURL('/settings/order-statuses', { timeout: 10000 });
  });
});
