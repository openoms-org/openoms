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
];
