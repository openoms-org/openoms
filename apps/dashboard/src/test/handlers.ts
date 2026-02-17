import { http, HttpResponse } from "msw";

const API_URL = "http://localhost:8080";

export const mockOrders = [
  {
    id: "ord-001",
    tenant_id: "t-1",
    source: "manual",
    status: "new",
    customer_name: "Jan Kowalski",
    customer_email: "jan@example.com",
    total_amount: 199.99,
    currency: "PLN",
    tags: ["vip"],
    payment_status: "pending",
    created_at: "2025-01-15T10:00:00Z",
    updated_at: "2025-01-15T10:00:00Z",
  },
  {
    id: "ord-002",
    tenant_id: "t-1",
    source: "allegro",
    status: "shipped",
    customer_name: "Anna Nowak",
    customer_email: "anna@example.com",
    total_amount: 349.50,
    currency: "PLN",
    tags: [],
    payment_status: "paid",
    created_at: "2025-01-14T08:00:00Z",
    updated_at: "2025-01-15T12:00:00Z",
  },
];

export const mockProducts = [
  {
    id: "prod-001",
    tenant_id: "t-1",
    source: "manual",
    name: "Widget A",
    sku: "WA-001",
    ean: "5901234123457",
    price: 49.99,
    stock_quantity: 100,
    tags: ["electronics"],
    category: "gadgets",
    description_short: "A small widget",
    description_long: "A detailed description of Widget A",
    images: [],
    created_at: "2025-01-10T10:00:00Z",
    updated_at: "2025-01-10T10:00:00Z",
  },
  {
    id: "prod-002",
    tenant_id: "t-1",
    source: "allegro",
    name: "Widget B",
    sku: "WB-002",
    price: 89.99,
    stock_quantity: 50,
    tags: [],
    description_short: "Another widget",
    description_long: "A detailed description of Widget B",
    images: [],
    created_at: "2025-01-11T10:00:00Z",
    updated_at: "2025-01-11T10:00:00Z",
  },
];

export const mockDashboardStats = {
  order_counts: {
    total: 150,
    by_status: { new: 20, processing: 30, shipped: 50, delivered: 40, cancelled: 10 },
    by_source: { manual: 40, allegro: 80, amazon: 30 },
  },
  revenue: {
    total: 45000,
    currency: "PLN",
    daily: [
      { date: "2025-01-14", amount: 1500, count: 5 },
      { date: "2025-01-15", amount: 2000, count: 8 },
    ],
  },
  recent_orders: [
    {
      id: "ord-001",
      customer_name: "Jan Kowalski",
      status: "new",
      source: "manual",
      total_amount: 199.99,
      currency: "PLN",
      created_at: "2025-01-15T10:00:00Z",
    },
  ],
};

export const mockUser = {
  id: "usr-001",
  tenant_id: "t-1",
  email: "admin@example.com",
  name: "Admin User",
  role: "owner" as const,
  created_at: "2025-01-01T00:00:00Z",
  updated_at: "2025-01-01T00:00:00Z",
};

export const mockTenant = {
  id: "t-1",
  name: "Test Company",
  slug: "test-company",
  plan: "pro",
  created_at: "2025-01-01T00:00:00Z",
  updated_at: "2025-01-01T00:00:00Z",
};

export const mockShipments = [
  {
    id: "shp-001",
    tenant_id: "t-1",
    order_id: "ord-001",
    provider: "inpost",
    tracking_number: "INP123456789",
    status: "created",
    package_number: 1,
    weight: 1.5,
    created_at: "2025-01-15T11:00:00Z",
    updated_at: "2025-01-15T11:00:00Z",
  },
  {
    id: "shp-002",
    tenant_id: "t-1",
    order_id: "ord-002",
    provider: "dhl",
    tracking_number: "DHL987654321",
    status: "shipped",
    package_number: 1,
    weight: 2.3,
    created_at: "2025-01-14T09:00:00Z",
    updated_at: "2025-01-15T13:00:00Z",
  },
];

export const mockEmailSettings = {
  enabled: true,
  smtp_host: "smtp.example.com",
  smtp_port: 587,
  smtp_user: "user@example.com",
  smtp_pass: "",
  from_email: "noreply@example.com",
  from_name: "Test Company",
  notify_on: ["order_created", "order_shipped"],
};

export const mockCompanySettings = {
  name: "Test Company",
  company_name: "Test Company",
  logo_url: "",
  address: "ul. Testowa 1",
  city: "Warszawa",
  post_code: "00-001",
  nip: "1234567890",
  phone: "+48123456789",
  email: "contact@example.com",
  website: "https://example.com",
};

export const mockInventorySettings = {
  strict_mode: false,
};

export const handlers = [
  // Orders
  http.get(`${API_URL}/v1/orders`, () => {
    return HttpResponse.json({
      items: mockOrders,
      total: mockOrders.length,
      limit: 20,
      offset: 0,
    });
  }),

  // Products
  http.get(`${API_URL}/v1/products`, () => {
    return HttpResponse.json({
      items: mockProducts,
      total: mockProducts.length,
      limit: 20,
      offset: 0,
    });
  }),

  // Shipments
  http.get(`${API_URL}/v1/shipments`, () => {
    return HttpResponse.json({
      items: mockShipments,
      total: mockShipments.length,
      limit: 20,
      offset: 0,
    });
  }),

  // Dashboard stats
  http.get(`${API_URL}/v1/stats/dashboard`, () => {
    return HttpResponse.json(mockDashboardStats);
  }),

  // Auth login
  http.post(`${API_URL}/v1/auth/login`, () => {
    return HttpResponse.json({
      access_token: "mock-access-token",
      expires_in: 3600,
      user: mockUser,
      tenant: mockTenant,
    });
  }),

  // Auth refresh
  http.post(`${API_URL}/v1/auth/refresh`, () => {
    return HttpResponse.json({
      access_token: "mock-refreshed-token",
      expires_in: 3600,
      user: mockUser,
      tenant: mockTenant,
    });
  }),

  // Settings: onboarding
  http.get(`${API_URL}/v1/settings/onboarding`, () => {
    return HttpResponse.json({ dismissed: false, completed_at: "" });
  }),

  // Settings: company (used by onboarding hook)
  http.get(`${API_URL}/v1/settings/company`, () => {
    return HttpResponse.json(mockCompanySettings);
  }),

  // Settings: email
  http.get(`${API_URL}/v1/settings/email`, () => {
    return HttpResponse.json(mockEmailSettings);
  }),

  // Settings: inventory
  http.get(`${API_URL}/v1/settings/inventory`, () => {
    return HttpResponse.json(mockInventorySettings);
  }),

  // Settings: onboarding dismiss
  http.put(`${API_URL}/v1/settings/onboarding`, () => {
    return HttpResponse.json({ dismissed: true });
  }),

  // Integrations
  http.get(`${API_URL}/v1/integrations`, () => {
    return HttpResponse.json([]);
  }),
];
