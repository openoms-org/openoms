const { expect } = require('@playwright/test');

class OrderCreatePage {
  constructor(page) {
    this.page = page;
    this.customerName = page.locator('#customer_name');
    this.productNames = page.getByPlaceholder(/Nazwa produktu|Product name/i);
    this.skus = page.getByPlaceholder('SKU');
    this.addItemButton = page.getByRole('button', { name: /Dodaj produkt|Add item/i });
    this.createButton = page.getByRole('button', { name: /Utwórz zamówienie|Create order/i });
  }

  async goto() {
    await this.page.goto('/orders/new');
    await expect(this.customerName).toBeVisible({ timeout: 15000 });
  }

  async fillCustomer(name) {
    await this.customerName.fill(name);
  }

  async addItem() {
    await this.addItemButton.click();
  }

  async fillFirstItem({ name, sku, quantity, price }) {
    await this.productNames.first().fill(name);
    await this.skus.first().fill(sku);
    await this.page.locator('input[type="number"][min="1"][step="1"]').first().fill(String(quantity));
    await this.page.locator('input[type="number"][step="0.01"]:not(#total_amount)').first().fill(String(price));
  }

  async createOrder() {
    await this.createButton.click();
  }
}

module.exports = { OrderCreatePage };
