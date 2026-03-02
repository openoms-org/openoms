export const TEST_CREDENTIALS = {
  tenant_slug: 'mercpart',
  email: 'rafal@mercpart.pl',
  password: 'password123',
};

export const LABELS = {
  dashboard: 'Panel główny',
  orders: 'Zamówienia',
  products: 'Produkty',
  shipments: 'Przesyłki',
  returns: 'Zwroty',
  invoices: 'Faktury',
  customers: 'Klienci',
  warehouses: 'Magazyny',
  users: 'Użytkownicy',
  login: 'Logowanie',
  loginButton: 'Zaloguj się',
  logout: 'Wyloguj',
  newOrder: 'Nowe zamówienie',
  newProduct: 'Nowy produkt',
  exportCSV: 'Eksportuj CSV',
};

// Unique suffix per test run to avoid duplicate key conflicts
const runId = Date.now().toString(36);

export const NEW_ORDER = {
  customerName: `Test E2E Klient ${runId}`,
  customerEmail: `e2e-order-${runId}@example.com`,
  customerPhone: '+48 500 100 200',
  itemName: 'Produkt testowy E2E',
  itemSku: `E2E-SKU-${runId}`,
  itemQuantity: '2',
  itemPrice: '99.99',
};

export const NEW_PRODUCT = {
  name: `Produkt E2E ${runId}`,
  sku: `E2E-PROD-${runId}`,
  price: '149.99',
  stock: '50',
};

export const NEW_CUSTOMER = {
  name: `Anna Testowa ${runId}`,
  email: `anna.testowa.${runId}@example.com`,
  phone: '+48 600 200 300',
  company: 'Firma Testowa E2E',
  nip: '1234567890',
};

export const TRIAL_USER = {
  tenant_slug: 'testflow',
  email: 'trial@testflow.pl',
  password: 'password123',
};

export const SEED = {
  ORDER_CUSTOMER: 'Marek Jabłoński',
  PRODUCT_NAME: 'Klocki hamulcowe przód Audi A4 B8',
  PRODUCT_SKU: 'KH-AUDI-A4-P',
};
