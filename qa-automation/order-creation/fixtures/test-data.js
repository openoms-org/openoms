const runId = Date.now().toString(36);

const TEST_CREDENTIALS = {
  tenant_slug: 'dev',
  email: 'admin@dev.local',
  password: 'password123',
};

const NEW_ORDER = {
  customerName: `Test E2E Klient ${runId}`,
  customerEmail: `e2e-order-${runId}@example.com`,
  customerPhone: '+48 500 100 200',
  itemName: 'Produkt testowy E2E',
  itemSku: `E2E-SKU-${runId}`,
  itemQuantity: '2',
  itemPrice: '99.99',
};

module.exports = {
  TEST_CREDENTIALS,
  NEW_ORDER,
};
