// === Auth ===
export interface LoginRequest {
  email: string;
  password: string;
  tenant_slug: string;
}

export interface RegisterRequest {
  email: string;
  password: string;
  name: string;
  tenant_name: string;
  tenant_slug: string;
}

export interface TokenResponse {
  access_token: string;
  expires_in: number;
  user: User;
  tenant: Tenant;
}

export interface CreateUserRequest {
  email: string;
  name: string;
  role: "owner" | "admin" | "member";
}

export interface UpdateUserRequest {
  name?: string;
  role?: "owner" | "admin" | "member";
}

// === Core Models ===
export interface User {
  id: string;
  tenant_id: string;
  email: string;
  name: string;
  role: "owner" | "admin" | "member";
  last_login_at?: string;
  created_at: string;
  updated_at: string;
}

export interface Tenant {
  id: string;
  name: string;
  slug: string;
  plan: string;
  created_at: string;
  updated_at: string;
}

// === Pagination ===
export interface ListResponse<T> {
  items: T[];
  total: number;
  limit: number;
  offset: number;
}

export interface PaginationParams {
  limit?: number;
  offset?: number;
}

// === Orders ===
export interface OrderItem {
  name: string;
  sku?: string;
  quantity: number;
  price: number;
}

export interface Order {
  id: string;
  tenant_id: string;
  external_id?: string;
  source: string;
  integration_id?: string;
  status: string;
  customer_name: string;
  customer_email?: string;
  customer_phone?: string;
  shipping_address?: Record<string, unknown>;
  billing_address?: Record<string, unknown>;
  items?: OrderItem[];
  total_amount: number;
  currency: string;
  notes?: string;
  metadata?: Record<string, unknown>;
  ordered_at?: string;
  shipped_at?: string;
  delivered_at?: string;
  payment_status: string;
  payment_method?: string;
  paid_at?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateOrderRequest {
  external_id?: string;
  source: string;
  integration_id?: string;
  customer_name: string;
  customer_email?: string;
  customer_phone?: string;
  shipping_address?: Record<string, unknown>;
  billing_address?: Record<string, unknown>;
  items?: Record<string, unknown>;
  total_amount: number;
  currency?: string;
  notes?: string;
  metadata?: Record<string, unknown>;
  ordered_at?: string;
}

export interface UpdateOrderRequest {
  external_id?: string;
  customer_name?: string;
  customer_email?: string;
  customer_phone?: string;
  shipping_address?: Record<string, unknown>;
  billing_address?: Record<string, unknown>;
  items?: Record<string, unknown>;
  total_amount?: number;
  currency?: string;
  notes?: string;
  metadata?: Record<string, unknown>;
}

export interface StatusTransitionRequest {
  status: string;
  force?: boolean;
}

export interface OrderListParams extends PaginationParams {
  status?: string;
  source?: string;
  search?: string;
  payment_status?: string;
}

// === Shipments ===
export interface Shipment {
  id: string;
  tenant_id: string;
  order_id: string;
  provider: string;
  integration_id?: string;
  tracking_number?: string;
  status: string;
  label_url?: string;
  carrier_data?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface CreateShipmentRequest {
  order_id: string;
  provider: string;
  integration_id?: string;
  tracking_number?: string;
  label_url?: string;
  carrier_data?: Record<string, unknown>;
}

export interface UpdateShipmentRequest {
  tracking_number?: string;
  label_url?: string;
  carrier_data?: Record<string, unknown>;
}

export interface ShipmentListParams extends PaginationParams {
  status?: string;
  provider?: string;
  order_id?: string;
}

export interface GenerateLabelRequest {
  service_type: "inpost_locker_standard" | "inpost_courier_standard";
  parcel_size: "small" | "medium" | "large";
  target_point?: string;
  label_format: "pdf" | "zpl" | "epl";
}

// === Products ===
export interface ProductImage {
  url: string;
  alt?: string;
  position: number;
}

export interface Product {
  id: string;
  tenant_id: string;
  external_id?: string;
  source: string;
  name: string;
  sku?: string;
  ean?: string;
  price: number;
  stock_quantity: number;
  metadata?: Record<string, unknown>;
  image_url?: string;
  images: ProductImage[];
  created_at: string;
  updated_at: string;
}

export interface CreateProductRequest {
  external_id?: string;
  source?: string;
  name: string;
  sku?: string;
  ean?: string;
  price: number;
  stock_quantity: number;
  metadata?: Record<string, unknown>;
  image_url?: string;
  images?: ProductImage[];
}

export interface UpdateProductRequest {
  external_id?: string;
  source?: string;
  name?: string;
  sku?: string;
  ean?: string;
  price?: number;
  stock_quantity?: number;
  metadata?: Record<string, unknown>;
  image_url?: string;
  images?: ProductImage[];
}

export interface ProductListParams extends PaginationParams {
  name?: string;
  sku?: string;
}

// === Integrations (NOT paginated, admin only) ===
export interface Integration {
  id: string;
  tenant_id: string;
  provider: string;
  status: "active" | "inactive" | "error";
  has_credentials: boolean;
  settings?: Record<string, unknown>;
  last_sync_at?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateIntegrationRequest {
  provider: string;
  credentials: Record<string, unknown>;
  settings?: Record<string, unknown>;
}

export interface UpdateIntegrationRequest {
  status?: "active" | "inactive" | "error";
  credentials?: Record<string, unknown>;
  settings?: Record<string, unknown>;
}

// === Webhooks ===
export interface WebhookEvent {
  id: string;
  tenant_id: string;
  provider: string;
  event_type: string;
  payload: Record<string, unknown>;
  status: string;
  created_at: string;
}

// === API Error ===
export interface ApiError {
  error: string;
}

// === Dashboard Stats ===
export interface DashboardStats {
  order_counts: OrderCounts;
  revenue: Revenue;
  recent_orders: OrderSummary[];
}

export interface OrderCounts {
  total: number;
  by_status: Record<string, number>;
  by_source: Record<string, number>;
}

export interface Revenue {
  total: number;
  currency: string;
  daily: DailyRevenue[];
}

export interface DailyRevenue {
  date: string;
  amount: number;
  count: number;
}

export interface OrderSummary {
  id: string;
  customer_name: string;
  status: string;
  source: string;
  total_amount: number;
  currency: string;
  created_at: string;
}

// === Audit Log ===
export interface AuditLogEntry {
  id: number;
  user_name?: string;
  action: string;
  changes: Record<string, string>;
  ip_address?: string;
  created_at: string;
}

// === Bulk Status ===
export interface BulkStatusResult {
  order_id: string;
  success: boolean;
  error?: string;
}

export interface BulkStatusTransitionResponse {
  results: BulkStatusResult[];
  succeeded: number;
  failed: number;
}

// === Email Settings ===
export interface EmailSettings {
  enabled: boolean;
  smtp_host: string;
  smtp_port: number;
  smtp_user: string;
  smtp_pass: string;
  from_email: string;
  from_name: string;
  notify_on: string[];
}

// === Company Settings ===
export interface CompanySettings {
  company_name: string;
  logo_url: string;
  address: string;
  city: string;
  post_code: string;
  nip: string;
  phone: string;
  email: string;
  website: string;
}
