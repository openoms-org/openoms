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
  sort_by?: string;
  sort_order?: "asc" | "desc";
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
  tags: string[];
  ordered_at?: string;
  shipped_at?: string;
  delivered_at?: string;
  delivery_method?: string;
  pickup_point_id?: string;
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
  tags?: string[];
  ordered_at?: string;
  payment_status?: string;
  payment_method?: string;
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
  tags?: string[];
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
  tag?: string;
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
  service_type: string;
  parcel_size?: string;
  target_point?: string;
  label_format: string;
  weight_kg?: number;
  width_cm?: number;
  height_cm?: number;
  depth_cm?: number;
  cod_amount?: number;
  insured_value?: number;
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
  tags: string[];
  category?: string;
  description_short: string;
  description_long: string;
  weight?: number;
  width?: number;
  height?: number;
  depth?: number;
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
  tags?: string[];
  category?: string;
  description_short?: string;
  description_long?: string;
  weight?: number;
  width?: number;
  height?: number;
  depth?: number;
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
  tags?: string[];
  category?: string;
  description_short?: string;
  description_long?: string;
  weight?: number;
  width?: number;
  height?: number;
  depth?: number;
  image_url?: string;
  images?: ProductImage[];
}

export interface ProductListParams extends PaginationParams {
  name?: string;
  sku?: string;
  tag?: string;
  category?: string;
}

// === Returns/RMA ===
export interface Return {
  id: string;
  tenant_id: string;
  order_id: string;
  status: string;
  reason: string;
  items: Record<string, unknown>[];
  refund_amount: number;
  notes?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateReturnRequest {
  order_id: string;
  reason: string;
  items?: Record<string, unknown>[];
  refund_amount: number;
  notes?: string;
}

export interface UpdateReturnRequest {
  reason?: string;
  refund_amount?: number;
  notes?: string;
}

export interface ReturnStatusRequest {
  status: string;
}

export interface ReturnListParams extends PaginationParams {
  status?: string;
  order_id?: string;
}

// === Integrations (NOT paginated, admin only) ===
export interface Integration {
  id: string;
  tenant_id: string;
  provider: string;
  label?: string;
  status: "active" | "inactive" | "error";
  has_credentials: boolean;
  settings?: Record<string, unknown>;
  sync_cursor?: string;
  error_message?: string;
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
  label?: string;
  status?: "active" | "inactive" | "error";
  credentials?: Record<string, unknown>;
  settings?: Record<string, unknown>;
  sync_cursor?: string;
  error_message?: string;
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

// === Outgoing Webhooks ===
export interface WebhookEndpoint {
  id: string;
  name: string;
  url: string;
  secret: string;
  events: string[];
  active: boolean;
}

export interface WebhookConfig {
  endpoints: WebhookEndpoint[];
}

export interface WebhookDelivery {
  id: string;
  url: string;
  event_type: string;
  payload: Record<string, unknown>;
  status: string;
  response_code?: number;
  error?: string;
  created_at: string;
}

export interface WebhookDeliveryParams extends PaginationParams {
  event_type?: string;
  status?: string;
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
  entity_type: string;
  entity_id: string;
  changes: Record<string, string>;
  ip_address?: string;
  created_at: string;
}

export interface AuditListParams extends PaginationParams {
  entity_type?: string;
  action?: string;
  user_id?: string;
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

// === Order Status Config ===
export interface StatusDef {
  key: string;
  label: string;
  color: string; // preset name: "blue", "green", "red", etc.
  position: number;
}

export interface OrderStatusConfig {
  statuses: StatusDef[];
  transitions: Record<string, string[]>;
}

// === Custom Fields Config ===
export interface CustomFieldDef {
  key: string;
  label: string;
  type: "text" | "number" | "select" | "date" | "checkbox";
  required: boolean;
  position: number;
  options?: string[];
}

export interface CustomFieldsConfig {
  fields: CustomFieldDef[];
}

// === Product Categories Config ===
export interface CategoryDef {
  key: string;
  label: string;
  color: string;
  position: number;
}

export interface ProductCategoriesConfig {
  categories: CategoryDef[];
}

// === InPost Points ===
export interface InPostPointAddress {
  line1: string;
  line2: string;
}

export interface InPostPointAddressDetails {
  city: string;
  province: string;
  post_code: string;
  street: string;
  building_number: string;
}

export interface InPostPoint {
  name: string;
  type: string[];
  address: InPostPointAddress;
  address_details?: InPostPointAddressDetails;
  location_description: string;
  opening_hours: string;
  status: string;
}

export interface InPostPointSearchResponse {
  items: InPostPoint[];
  count: number;
  page: number;
  per_page: number;
  total_pages: number;
}

// === Product Listings ===
export interface ProductListing {
  id: string;
  tenant_id: string;
  product_id: string;
  integration_id: string;
  external_id?: string;
  status: string;
  url?: string;
  price_override?: number;
  stock_override?: number;
  sync_status: string;
  last_synced_at?: string;
  error_message?: string;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

// === Sync Jobs ===
export interface SyncJob {
  id: string;
  tenant_id: string;
  integration_id: string;
  job_type: string;
  status: string;
  started_at?: string;
  finished_at?: string;
  items_processed: number;
  items_failed: number;
  error_message?: string;
  metadata: Record<string, unknown>;
  created_at: string;
}

// === Suppliers ===
export interface Supplier {
  id: string;
  tenant_id: string;
  name: string;
  code?: string;
  feed_url?: string;
  feed_format: string;
  status: string;
  settings: Record<string, unknown>;
  last_sync_at?: string;
  error_message?: string;
  created_at: string;
  updated_at: string;
}

export interface SupplierProduct {
  id: string;
  tenant_id: string;
  supplier_id: string;
  product_id?: string;
  external_id: string;
  name: string;
  ean?: string;
  sku?: string;
  price?: number;
  stock_quantity: number;
  metadata: Record<string, unknown>;
  last_synced_at?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateSupplierRequest {
  name: string;
  code?: string;
  feed_url?: string;
  feed_format?: string;
  settings?: Record<string, unknown>;
}

export interface UpdateSupplierRequest {
  name?: string;
  code?: string;
  feed_url?: string;
  feed_format?: string;
  status?: string;
  settings?: Record<string, unknown>;
}

export interface SupplierListParams extends PaginationParams {
  status?: string;
}

export interface SupplierProductListParams extends PaginationParams {
  ean?: string;
  linked?: boolean;
}

// === Invoices ===
export interface Invoice {
  id: string;
  tenant_id: string;
  order_id: string;
  provider: string;
  external_id?: string;
  external_number?: string;
  status: string;
  invoice_type: string;
  total_net?: number;
  total_gross?: number;
  currency: string;
  issue_date?: string;
  due_date?: string;
  pdf_url?: string;
  metadata: Record<string, unknown>;
  error_message?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateInvoiceRequest {
  order_id: string;
  provider?: string;
  invoice_type?: string;
  customer_name?: string;
  customer_email?: string;
  nip?: string;
  payment_method?: string;
  notes?: string;
}

export interface InvoiceListParams extends PaginationParams {
  status?: string;
  provider?: string;
  order_id?: string;
}

// === Invoicing Settings ===
export interface InvoicingSettings {
  provider: string;
  auto_create_on_status: string[];
  default_tax_rate: number;
  payment_days: number;
  credentials: Record<string, string>;
}

// === SMS Settings ===
export interface SMSSettings {
  enabled: boolean;
  api_token: string;
  from: string;
  notify_on: string[];
  templates: Record<string, string>;
}

export interface SendTestSMSRequest {
  phone: string;
}

// === Automation Rules ===
export interface AutomationCondition {
  field: string;
  operator: string;
  value: unknown;
}

export interface AutomationAction {
  type: string;
  config: Record<string, unknown>;
}

export interface AutomationRule {
  id: string;
  tenant_id: string;
  name: string;
  description?: string;
  enabled: boolean;
  priority: number;
  trigger_event: string;
  conditions: AutomationCondition[];
  actions: AutomationAction[];
  last_fired_at?: string;
  fire_count: number;
  created_at: string;
  updated_at: string;
}

export interface CreateAutomationRuleRequest {
  name: string;
  description?: string;
  enabled?: boolean;
  priority?: number;
  trigger_event: string;
  conditions: AutomationCondition[];
  actions: AutomationAction[];
}

export interface UpdateAutomationRuleRequest {
  name?: string;
  description?: string;
  enabled?: boolean;
  priority?: number;
  trigger_event?: string;
  conditions?: AutomationCondition[];
  actions?: AutomationAction[];
}

export interface AutomationRuleLog {
  id: string;
  tenant_id: string;
  rule_id: string;
  trigger_event: string;
  entity_type: string;
  entity_id: string;
  conditions_met: boolean;
  actions_executed: Record<string, unknown>[];
  error_message?: string;
  executed_at: string;
}

export interface AutomationRuleListParams extends PaginationParams {
  trigger_event?: string;
  enabled?: boolean;
}

export interface TestAutomationRuleRequest {
  data: Record<string, unknown>;
}

export interface TestAutomationRuleResponse {
  condition_results: ConditionResult[];
  all_conditions_met: boolean;
  actions_to_execute: AutomationAction[];
}

export interface ConditionResult {
  condition: AutomationCondition;
  met: boolean;
}

// === Import ===
export interface ImportColumnMapping {
  csv_column: string;
  order_field: string;
}

export interface ImportPreviewRow {
  row: number;
  data: Record<string, unknown>;
  errors?: string[];
}

export interface ImportPreviewResponse {
  headers: string[];
  total_rows: number;
  sample_rows: ImportPreviewRow[];
  mappings?: ImportColumnMapping[];
}

export interface ImportResult {
  total_rows: number;
  imported: number;
  skipped: number;
  errors: ImportError[];
}

export interface ImportError {
  row: number;
  field?: string;
  message: string;
}

// === Product Variants ===
export interface ProductVariant {
  id: string;
  tenant_id: string;
  product_id: string;
  sku?: string;
  ean?: string;
  name: string;
  attributes: Record<string, string>;
  price_override?: number;
  stock_quantity: number;
  weight?: number;
  image_url?: string;
  position: number;
  active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateVariantRequest {
  sku?: string;
  ean?: string;
  name: string;
  attributes?: Record<string, string>;
  price_override?: number;
  stock_quantity: number;
  weight?: number;
  image_url?: string;
  position?: number;
  active?: boolean;
}

export interface UpdateVariantRequest {
  sku?: string;
  ean?: string;
  name?: string;
  attributes?: Record<string, string>;
  price_override?: number;
  stock_quantity?: number;
  weight?: number;
  image_url?: string;
  position?: number;
  active?: boolean;
}

export interface VariantListParams extends PaginationParams {
  active?: boolean;
}
