import { type Page, expect } from '@playwright/test';

export async function waitForTableLoaded(page: Page) {
  // Wait for skeleton/loading to disappear and rows to appear
  await expect(page.locator('table tbody tr').first()).toBeVisible({ timeout: 15000 });
}

export async function waitForToast(page: Page, text: string | RegExp) {
  const locator =
    typeof text === 'string'
      ? page.locator('[data-sonner-toast]').filter({ hasText: text })
      : page.locator('[data-sonner-toast]').filter({ hasText: text });
  await expect(locator).toBeVisible({ timeout: 5000 });
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

export async function fillAndSubmitOrderForm(
  page: Page,
  data: {
    customerName: string;
    customerEmail?: string;
    customerPhone?: string;
    itemName: string;
    itemSku?: string;
    itemQuantity: string;
    itemPrice: string;
  },
) {
  await page.locator('#customer_name').fill(data.customerName);
  if (data.customerEmail) {
    await page.locator('#customer_email').fill(data.customerEmail);
  }
  if (data.customerPhone) {
    await page.locator('#customer_phone').fill(data.customerPhone);
  }

  // Add an order item
  await page.getByRole('button', { name: 'Dodaj pozycję' }).click();
  await page.getByPlaceholder('Nazwa produktu').fill(data.itemName);
  if (data.itemSku) {
    await page.getByPlaceholder('SKU').first().fill(data.itemSku);
  }
  // Quantity — only input with min=1 step=1
  await page
    .locator('input[type="number"][min="1"][step="1"]')
    .first()
    .fill(data.itemQuantity);
  // Price — exclude #total_amount which is auto-calculated
  await page
    .locator('input[type="number"][step="0.01"]:not(#total_amount)')
    .first()
    .fill(data.itemPrice);

  await page.getByRole('button', { name: 'Utwórz zamówienie' }).click();
}

export async function fillAndSubmitProductForm(
  page: Page,
  data: { name: string; sku?: string; price: string; stock: string },
) {
  await page.locator('#name').fill(data.name);
  if (data.sku) {
    await page.locator('#sku').fill(data.sku);
  }
  await page.locator('#price').fill(data.price);
  await page.locator('#stock_quantity').fill(data.stock);
  await page.getByRole('button', { name: 'Utwórz produkt' }).click();
}

export async function fillAndSubmitCustomerForm(
  page: Page,
  data: {
    name: string;
    email?: string;
    phone?: string;
    company?: string;
    nip?: string;
  },
) {
  await page.locator('#name').fill(data.name);
  if (data.email) {
    await page.locator('#email').fill(data.email);
  }
  if (data.phone) {
    await page.locator('#phone').fill(data.phone);
  }
  if (data.company) {
    await page.locator('#company_name').fill(data.company);
  }
  if (data.nip) {
    await page.locator('#nip').fill(data.nip);
  }
  await page.getByRole('button', { name: 'Utwórz klienta' }).click();
}

export async function confirmDeleteDialog(page: Page, confirmLabel: string) {
  const dialog = page.getByRole('alertdialog');
  await expect(dialog).toBeVisible({ timeout: 5000 });
  await dialog.getByRole('button', { name: confirmLabel }).click();
}

export async function openUserMenu(page: Page) {
  await page.locator('button.rounded-full').first().click();
}
