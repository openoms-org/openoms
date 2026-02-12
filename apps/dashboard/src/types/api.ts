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

export interface LoginResponse {
  access_token?: string;
  expires_in?: number;
  user?: User;
  tenant?: Tenant;
  requires_2fa?: boolean;
  temp_token?: string;
}

export interface TwoFASetupResponse {
  secret: string;
  qr_url: string;
}

export interface TwoFAStatusResponse {
  enabled: boolean;
  verified_at?: string;
}

export interface AITextResult {
  description: string;
}

export interface AIDescribeRequest {
  product_id: string;
  style?: "professional" | "promotional" | "casual" | "seo";
  language?: "pl" | "en" | "de";
  length?: "short" | "medium" | "long";
  marketplace?: "allegro" | "amazon" | "ebay";
}

export interface CreateUserRequest {
  email: string;
  name: string;
  role: "owner" | "admin" | "member";
}

export interface UpdateUserRequest {
  name?: string;
  role?: "owner" | "admin" | "member";
  role_id?: string;
}

// === Core Models ===
export interface User {
  id: string;
  tenant_id: string;
  email: string;
  name: string;
  role: "owner" | "admin" | "member";
  role_id?: string;
  last_login_at?: string;
  last_logout_at?: string;
  created_at: string;
  updated_at: string;
}

export interface Tenant {
  id: string;
  name: string;
  slug: string;
  plan: string;
  settings?: Record<string, unknown>;
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

export interface ReturnItem {
  name: string;
  quantity: number;
}

export interface Address {
  name?: string;
  street?: string;
  city?: string;
  postal_code?: string;
  country?: string;
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
  shipping_address?: Address;
  billing_address?: Address;
  items?: OrderItem[];
  total_amount: number;
  currency: string;
  notes?: string;
  internal_notes?: string;
  priority?: "urgent" | "high" | "normal" | "low";
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
  customer_id?: string;
  merged_into?: string;
  split_from?: string;
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
  shipping_address?: Address;
  billing_address?: Address;
  items?: OrderItem[];
  total_amount: number;
  currency?: string;
  notes?: string;
  internal_notes?: string;
  priority?: "urgent" | "high" | "normal" | "low";
  metadata?: Record<string, unknown>;
  tags?: string[];
  delivery_method?: string;
  pickup_point_id?: string;
  ordered_at?: string;
  payment_status?: string;
  payment_method?: string;
  shipment_provider?: string;
  auto_create_shipment?: boolean;
}

export interface UpdateOrderRequest {
  external_id?: string;
  customer_name?: string;
  customer_email?: string;
  customer_phone?: string;
  shipping_address?: Address;
  billing_address?: Address;
  items?: OrderItem[];
  total_amount?: number;
  currency?: string;
  notes?: string;
  internal_notes?: string;
  priority?: "urgent" | "high" | "normal" | "low";
  metadata?: Record<string, unknown>;
  tags?: string[];
  delivery_method?: string;
  pickup_point_id?: string;
  payment_status?: string;
  payment_method?: string;
  paid_at?: string;
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
  priority?: string;
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
  warehouse_id?: string;
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
  warehouse_id?: string;
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

export interface TrackingEvent {
  status: string;
  location?: string;
  timestamp: string;
  details?: string;
}

export interface GenerateLabelRequest {
  service_type: string;
  parcel_size?: string;
  target_point?: string;
  sending_method?: string;
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
  has_variants: boolean;
  is_bundle: boolean;
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
  is_bundle?: boolean;
}

export interface ProductListParams extends PaginationParams {
  name?: string;
  sku?: string;
  tag?: string;
  category?: string;
  search?: string;
}

// === Returns/RMA ===
export interface Return {
  id: string;
  tenant_id: string;
  order_id: string;
  status: string;
  reason: string;
  items: ReturnItem[];
  refund_amount: number;
  notes?: string;
  return_token?: string;
  customer_email?: string;
  customer_notes?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateReturnRequest {
  order_id: string;
  reason: string;
  items?: ReturnItem[];
  refund_amount: number;
  notes?: string;
}

export interface UpdateReturnRequest {
  reason?: string;
  items?: ReturnItem[];
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
  label?: string;
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
  tenant_id: string;
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

// === Advanced Reports ===
export interface TopProduct {
  name: string;
  sku?: string;
  total_quantity: number;
  total_revenue: number;
}

export interface SourceRevenue {
  source: string;
  revenue: number;
  count: number;
}

export interface DailyOrderTrend {
  date: string;
  count: number;
  avg_value: number;
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

export interface SyncJobListParams extends PaginationParams {
  integration_id?: string;
  job_type?: string;
  status?: string;
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
  error_message?: string;
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
  ksef_number?: string;
  ksef_status: string;
  ksef_sent_at?: string;
  ksef_response?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface CreateInvoiceRequest {
  order_id: string;
  provider: string;
  invoice_type?: string;
  customer_name: string;
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

// === KSeF Settings ===
export interface KSeFSettings {
  enabled: boolean;
  environment: string;
  nip: string;
  token: string;
  company_name: string;
  company_street: string;
  company_city: string;
  company_postal: string;
  company_country: string;
}

export interface KSeFTestResult {
  success: boolean;
  message: string;
  timestamp?: string;
  challenge?: string;
}

export interface KSeFBulkSendResult {
  sent: number;
  errors: string[];
  total: number;
}

// === SMS Settings ===
export interface SMSSettings {
  enabled: boolean;
  api_token: string;
  from: string;
  notify_on: string[];
  templates: Record<string, string>;
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
  delay_seconds?: number;
}

export interface DelayedAction {
  id: string;
  tenant_id: string;
  rule_id: string;
  action_index: number;
  order_id?: string;
  execute_at: string;
  executed: boolean;
  executed_at?: string;
  error?: string;
  created_at: string;
  action_data: Record<string, unknown>;
  event_data: Record<string, unknown>;
}

export interface BatchLabelsRequest {
  shipment_ids: string[];
}

export interface CreateDispatchOrderRequest {
  shipment_ids: string[];
  street?: string;
  building_number?: string;
  city?: string;
  post_code?: string;
  name?: string;
  phone?: string;
  email?: string;
  comment?: string;
}

export interface DispatchOrderResponse {
  id: number;
  status: string;
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

// === Warehouses ===
export interface Warehouse {
  id: string;
  tenant_id: string;
  name: string;
  code?: string;
  address: Record<string, unknown>;
  is_default: boolean;
  active: boolean;
  created_at: string;
  updated_at: string;
}

export interface WarehouseStock {
  id: string;
  tenant_id: string;
  warehouse_id: string;
  product_id: string;
  variant_id?: string;
  quantity: number;
  reserved: number;
  min_stock: number;
  created_at: string;
  updated_at: string;
}

export interface CreateWarehouseRequest {
  name: string;
  code?: string;
  address?: Record<string, unknown>;
  is_default?: boolean;
  active?: boolean;
}

export interface UpdateWarehouseRequest {
  name?: string;
  code?: string;
  address?: Record<string, unknown>;
  is_default?: boolean;
  active?: boolean;
}

export interface UpsertWarehouseStockRequest {
  product_id: string;
  variant_id?: string;
  quantity: number;
  reserved: number;
  min_stock: number;
}

export interface WarehouseListParams extends PaginationParams {
  active?: boolean;
}

export interface WarehouseStockListParams extends PaginationParams {}

// === Inventory Settings ===
export interface InventorySettings {
  strict_mode: boolean;
}

// === Product Import ===
export interface ProductImportPreview {
  headers: string[];
  total_rows: number;
  sample_rows: ImportPreviewRow[];
  new_count: number;
  update_count: number;
}

export interface ProductImportResult {
  created: number;
  updated: number;
  errors: ImportError[];
}

// === Customers ===
export interface Customer {
  id: string;
  tenant_id: string;
  email?: string;
  phone?: string;
  name: string;
  company_name?: string;
  nip?: string;
  default_shipping_address?: Record<string, unknown>;
  default_billing_address?: Record<string, unknown>;
  tags: string[];
  notes?: string;
  total_orders: number;
  total_spent: number;
  price_list_id?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateCustomerRequest {
  name: string;
  email?: string;
  phone?: string;
  company_name?: string;
  nip?: string;
  default_shipping_address?: Record<string, unknown>;
  default_billing_address?: Record<string, unknown>;
  tags?: string[];
  notes?: string;
}

export interface UpdateCustomerRequest {
  name?: string;
  email?: string;
  phone?: string;
  company_name?: string;
  nip?: string;
  default_shipping_address?: Record<string, unknown>;
  default_billing_address?: Record<string, unknown>;
  tags?: string[];
  notes?: string;
  price_list_id?: string;
}

export interface CustomerListParams extends PaginationParams {
  search?: string;
  tags?: string;
}

// === Print Templates ===
export interface PrintTemplatesConfig {
  packing_slip_html: string;
  order_summary_html: string;
  return_slip_html: string;
}

// === Order Groups (Merge/Split) ===
export interface OrderGroup {
  id: string;
  tenant_id: string;
  group_type: "merged" | "split";
  source_order_ids: string[];
  target_order_ids: string[];
  notes?: string;
  created_by?: string;
  created_at: string;
}

export interface MergeOrdersRequest {
  order_ids: string[];
  notes?: string;
}

export interface SplitSpec {
  items: OrderItem[];
  customer_name?: string;
  shipping_address?: Address;
}

export interface SplitOrderRequest {
  splits: SplitSpec[];
  notes?: string;
}

// === Product Bundles ===
export interface BundleComponent {
  id: string;
  tenant_id: string;
  bundle_product_id: string;
  component_product_id: string;
  component_variant_id?: string;
  quantity: number;
  position: number;
  component_name?: string;
  component_sku?: string;
  component_stock: number;
  created_at: string;
  updated_at: string;
}

export interface CreateBundleComponentRequest {
  component_product_id: string;
  component_variant_id?: string;
  quantity: number;
  position?: number;
}

export interface UpdateBundleComponentRequest {
  quantity?: number;
  position?: number;
}

export interface BundleStockResponse {
  stock: number;
}

// === Barcode / Packing Station ===
export interface BarcodeLookupResponse {
  product?: Product;
  variants?: ProductVariant[];
}

export interface ScannedItem {
  sku: string;
  quantity: number;
}

export interface PackOrderRequest {
  scanned_items: ScannedItem[];
}

export interface PackOrderResponse {
  order_id: string;
  packed_at: string;
  packed_by: string;
  status: string;
}

// === Price Lists (B2B) ===
export interface PriceList {
  id: string;
  tenant_id: string;
  name: string;
  description?: string;
  currency: string;
  is_default: boolean;
  discount_type: "percentage" | "fixed" | "override";
  active: boolean;
  valid_from?: string;
  valid_to?: string;
  created_at: string;
  updated_at: string;
}

export interface PriceListItem {
  id: string;
  tenant_id: string;
  price_list_id: string;
  product_id: string;
  variant_id?: string;
  price?: number;
  discount?: number;
  min_quantity: number;
  created_at: string;
  updated_at: string;
}

export interface CreatePriceListRequest {
  name: string;
  description?: string;
  currency?: string;
  is_default?: boolean;
  discount_type?: "percentage" | "fixed" | "override";
  active?: boolean;
  valid_from?: string;
  valid_to?: string;
}

export interface UpdatePriceListRequest {
  name?: string;
  description?: string;
  currency?: string;
  is_default?: boolean;
  discount_type?: "percentage" | "fixed" | "override";
  active?: boolean;
  valid_from?: string;
  valid_to?: string;
}

export interface CreatePriceListItemRequest {
  product_id: string;
  variant_id?: string;
  price?: number;
  discount?: number;
  min_quantity?: number;
}

export interface PriceListListParams extends PaginationParams {
  active?: boolean;
}

// === Warehouse Documents (PZ/WZ/MM) ===
export interface WarehouseDocument {
  id: string;
  tenant_id: string;
  document_number: string;
  document_type: "PZ" | "WZ" | "MM";
  status: "draft" | "confirmed" | "cancelled";
  warehouse_id: string;
  target_warehouse_id?: string;
  supplier_id?: string;
  order_id?: string;
  notes?: string;
  confirmed_at?: string;
  confirmed_by?: string;
  created_by?: string;
  items?: WarehouseDocItem[];
  created_at: string;
  updated_at: string;
}

export interface WarehouseDocItem {
  id: string;
  tenant_id: string;
  document_id: string;
  product_id: string;
  variant_id?: string;
  quantity: number;
  unit_price?: number;
  notes?: string;
  created_at: string;
}

export interface CreateWarehouseDocumentRequest {
  document_type: "PZ" | "WZ" | "MM";
  warehouse_id: string;
  target_warehouse_id?: string;
  supplier_id?: string;
  order_id?: string;
  notes?: string;
  items: CreateWarehouseDocItemRequest[];
}

export interface CreateWarehouseDocItemRequest {
  product_id: string;
  variant_id?: string;
  quantity: number;
  unit_price?: number;
  notes?: string;
}

export interface UpdateWarehouseDocumentRequest {
  notes?: string;
}

export interface WarehouseDocumentListParams extends PaginationParams {
  document_type?: string;
  status?: string;
  warehouse_id?: string;
}

// === Exchange Rates (Multi-Currency) ===
export interface ExchangeRate {
  id: string;
  tenant_id: string;
  base_currency: string;
  target_currency: string;
  rate: number;
  source: string;
  fetched_at: string;
  created_at: string;
}

export interface CreateExchangeRateRequest {
  base_currency: string;
  target_currency: string;
  rate: number;
  source?: string;
}

export interface UpdateExchangeRateRequest {
  rate?: number;
  source?: string;
}

export interface ConvertAmountRequest {
  amount: number;
  from: string;
  to: string;
}

export interface ConvertAmountResponse {
  original_amount: number;
  converted_amount: number;
  from: string;
  to: string;
  rate: number;
}

export interface ExchangeRateListParams extends PaginationParams {
  base_currency?: string;
  target_currency?: string;
}

export interface FetchNBPResponse {
  fetched: number;
  source: string;
}

// === Public Return Self-Service ===
export interface PublicReturnRequest {
  order_id: string;
  email: string;
  items?: ReturnItem[];
  reason: string;
  notes?: string;
}

export interface PublicReturnResponse {
  id: string;
  status: string;
  return_token: string;
  created_at: string;
}

export interface PublicReturnStatus {
  id: string;
  status: string;
  reason: string;
  items: ReturnItem[];
  created_at: string;
  updated_at: string;
}

// === Roles (RBAC) ===
export interface Role {
  id: string;
  tenant_id: string;
  name: string;
  description?: string;
  is_system: boolean;
  permissions: string[];
  created_at: string;
  updated_at: string;
}

export interface CreateRoleRequest {
  name: string;
  description?: string;
  permissions: string[];
}

export interface UpdateRoleRequest {
  name?: string;
  description?: string;
  permissions?: string[];
}

export interface RoleListParams extends PaginationParams {}

export interface PermissionGroup {
  group: string;
  permissions: string[];
}

// === AI Auto-Categorization (Phase 33) ===
export interface AISuggestion {
  product_id: string;
  categories: string[];
  tags: string[];
  description?: string;
  short_description?: string;
  long_description?: string;
}

export interface AIBulkCategorizeResult {
  product_id: string;
  categories: string[];
  tags: string[];
  error?: string;
}

export interface AIBulkCategorizeResponse {
  results: AIBulkCategorizeResult[];
}

// === Marketing / Mailchimp (Phase 34) ===
export interface MailchimpSettings {
  api_key: string;
  list_id: string;
  enabled: boolean;
}

export interface MarketingSyncResponse {
  synced: number;
  failed: number;
}

export interface MarketingStatusResponse {
  enabled: boolean;
  configured: boolean;
}

export interface CreateCampaignRequest {
  name: string;
  subject: string;
  content: string;
}

export interface CreateCampaignResponse {
  campaign_id: string;
}

// === Helpdesk / Freshdesk (Phase 34) ===
export interface FreshdeskSettings {
  domain: string;
  api_key: string;
  enabled: boolean;
}

export interface FreshdeskTicket {
  id: number;
  subject: string;
  description?: string;
  status: number;
  priority: number;
  created_at: string;
  updated_at: string;
}

export interface CreateTicketRequest {
  subject: string;
  description: string;
  email: string;
}

export interface TicketListResponse {
  tickets: FreshdeskTicket[];
}

// === Stocktakes (Inventory Counting) ===
export interface Stocktake {
  id: string;
  tenant_id: string;
  warehouse_id: string;
  name: string;
  status: "draft" | "in_progress" | "completed" | "cancelled";
  started_at?: string;
  completed_at?: string;
  notes?: string;
  created_by?: string;
  created_at: string;
  updated_at: string;
  stats?: StocktakeStats;
  items?: StocktakeItem[];
}

export interface StocktakeItem {
  id: string;
  tenant_id: string;
  stocktake_id: string;
  product_id: string;
  expected_quantity: number;
  counted_quantity: number | null;
  difference: number;
  notes?: string;
  counted_at?: string;
  counted_by?: string;
  created_at: string;
  product_name?: string;
  product_sku?: string;
}

export interface StocktakeStats {
  total_items: number;
  counted_items: number;
  discrepancies: number;
  surplus_count: number;
  shortage_count: number;
}

export interface CreateStocktakeRequest {
  warehouse_id: string;
  name: string;
  notes?: string;
  product_ids?: string[];
}

export interface UpdateStocktakeItemRequest {
  counted_quantity: number;
  notes?: string;
}

export interface StocktakeListParams extends PaginationParams {
  warehouse_id?: string;
  status?: string;
}

export interface StocktakeItemListParams extends PaginationParams {
  filter?: "all" | "uncounted" | "discrepancies";
}

// === Shipping Rate Shopping ===
export interface ShippingRate {
  carrier_name: string;
  carrier_code: string;
  service_name: string;
  price: number;
  currency: string;
  estimated_days: number;
  pickup_point: boolean;
}

export interface GetRatesRequest {
  from_postal_code: string;
  from_country: string;
  to_postal_code: string;
  to_country: string;
  weight: number;
  width: number;
  height: number;
  length: number;
  cod: number;
}

export interface GetRatesResponse {
  rates: ShippingRate[];
}

// === WebSocket Events ===
export interface WSEvent {
  type: string;
  tenant_id: string;
  payload?: Record<string, unknown>;
}
