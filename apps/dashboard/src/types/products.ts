import type { PaginationParams } from "./common";

// === Products ===
interface ProductImage {
  url: string;
  alt?: string;
  position: number;
}

/** Normalize images array — backend may return strings or objects */
export function normalizeProductImages(
  images: (ProductImage | string)[] | null | undefined
): ProductImage[] {
  if (!images || !Array.isArray(images)) return [];
  return images.map((img, i) => {
    if (typeof img === "string") {
      return { url: img, position: i };
    }
    return img;
  });
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
  category_id?: string;
  description_short: string;
  description_long: string;
  weight?: number;
  width?: number;
  height?: number;
  depth?: number;
  image_url?: string;
  images: (ProductImage | string)[];
  has_variants: boolean;
  is_bundle: boolean;
  is_dropship: boolean;
  dropship_supplier_id?: string;
  supplier_name?: string;
  marketplace_providers: string[];
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
  category_id?: string;
  description_short?: string;
  description_long?: string;
  weight?: number;
  width?: number;
  height?: number;
  depth?: number;
  image_url?: string;
  images?: ProductImage[];
  is_dropship?: boolean;
  dropship_supplier_id?: string;
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
  category_id?: string;
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
  category_id?: string;
  supplier_id?: string;
  source?: string;
  search?: string;
  marketplace?: string;
}

// === Product Categories Config (legacy flat) ===
interface CategoryDef {
  key: string;
  label: string;
  color: string;
  position: number;
}

export interface ProductCategoriesConfig {
  categories: CategoryDef[];
}

// === Product Categories (hierarchical) ===
export interface ProductCategory {
  id: string;
  tenant_id: string;
  parent_id?: string;
  name: string;
  slug: string;
  color: string;
  icon?: string;
  position: number;
  depth: number;
  children?: ProductCategory[];
  created_at: string;
  updated_at: string;
}

export interface CreateCategoryRequest {
  name: string;
  parent_id?: string;
  color?: string;
  icon?: string;
}

export interface UpdateCategoryRequest {
  name?: string;
  parent_id?: string;
  color?: string;
  icon?: string;
  position?: number;
}

export interface SupplierCategoryMapping {
  id: string;
  tenant_id: string;
  supplier_id: string;
  source_category: string;
  category_id?: string;
  auto_matched: boolean;
  confirmed: boolean;
  created_at: string;
}

export interface UpsertCategoryMappingRequest {
  source_category: string;
  category_id?: string;
  confirmed: boolean;
}

export interface MarketplaceCategoryMapping {
  id: string;
  tenant_id: string;
  integration_id: string;
  external_category_id: string;
  external_category_name: string;
  category_id?: string;
  auto_created: boolean;
  confirmed: boolean;
  created_at: string;
  updated_at: string;
}

export interface UpsertMarketplaceCategoryMappingRequest {
  external_category_id: string;
  external_category_name?: string;
  category_id?: string;
  confirmed: boolean;
}

export interface AllegroParameterMapping {
  id: string;
  supplier_id: string;
  allegro_category_id: string;
  allegro_param_id: string;
  allegro_param_name: string;
  source_type: "attribute" | "field" | "static";
  source_key: string;
  value_mapping: Record<string, string>;
  created_at: string;
}

export interface BulkUpsertAllegroMappingsRequest {
  allegro_category_id: string;
  mappings: {
    allegro_param_id: string;
    allegro_param_name: string;
    source_type: "attribute" | "field" | "static";
    source_key: string;
    value_mapping?: Record<string, string>;
  }[];
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
  stock_sync_mode: 'auto' | 'manual';
  description_html?: string;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
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

// === Product Feed Settings ===
export interface ProductFeedConfig {
  ceneo_enabled: boolean;
  ceneo_feed_token: string;
  google_enabled: boolean;
  google_feed_token: string;
  default_currency: string;
  default_shipping_cost: string;
  excluded_categories: string[];
  exclude_out_of_stock: boolean;
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

// === Demand Forecast ===
export interface ProductForecast {
  product_id: string;
  product_name: string;
  sku: string;
  current_stock: number;
  days_ahead: number;
  forecasted_units: number;
  confidence_low: number;
  confidence_high: number;
  avg_daily_sales: number;
  trend_direction: "up" | "down" | "stable";
  trend_pct: number;
  days_of_stock: number;
  low_data_warning: boolean;
}

export interface ReorderRecommendation {
  product_id: string;
  product_name: string;
  sku: string;
  current_stock: number;
  forecasted_demand: number;
  safety_stock: number;
  reorder_point: number;
  recommended_qty: number;
  urgency: "critical" | "soon" | "planned";
  days_until_stockout: number;
  supplier_id?: string;
  supplier_name?: string;
  estimated_cost: number;
}

export interface SeasonalityData {
  product_id: string;
  product_name: string;
  by_day_of_week: DayOfWeekSales[];
  by_month: MonthSales[];
}

export interface DayOfWeekSales {
  day: string;
  avg_sales: number;
  index: number;
}

export interface MonthSales {
  month: string;
  avg_sales: number;
  index: number;
}

export interface ProductVelocity {
  product_id: string;
  product_name: string;
  sku: string;
  total_revenue: number;
  total_units: number;
  abc_class: "A" | "B" | "C";
  days_of_stock: number;
  current_stock: number;
}

export interface ForecastConfig {
  default_lead_time_days: number;
  safety_stock_days: number;
  forecast_days_ahead: number;
}

// === Repricing Engine ===
export interface RepricingRule {
  id: string;
  tenant_id: string;
  name: string;
  status: "active" | "paused" | "archived";
  strategy: "margin" | "time_based" | "stock_based";
  priority: number;
  scope_type: "all" | "category" | "tag" | "product_ids";
  scope_value?: unknown;
  params: Record<string, unknown>;
  min_price?: number;
  max_price?: number;
  channels: string[];
  last_applied_at?: string;
  products_affected: number;
  created_by?: string;
  created_at: string;
  updated_at: string;
}

export interface RepricingLog {
  id: string;
  tenant_id: string;
  rule_id: string;
  product_id: string;
  old_price: number;
  new_price: number;
  reason?: string;
  channel: string;
  applied_at: string;
}

export interface CreateRepricingRuleRequest {
  name: string;
  strategy: string;
  priority: number;
  scope_type: string;
  scope_value?: unknown;
  params: Record<string, unknown>;
  min_price?: number;
  max_price?: number;
  channels?: string[];
}

export interface UpdateRepricingRuleRequest {
  name?: string;
  status?: string;
  strategy?: string;
  priority?: number;
  scope_type?: string;
  scope_value?: unknown;
  params?: Record<string, unknown>;
  min_price?: number;
  max_price?: number;
  channels?: string[];
}

export interface RepricingSummary {
  active_rules: number;
  paused_rules: number;
  changes_today: number;
  changes_week: number;
  avg_change_pct: number;
  total_affected: number;
}

export interface SimulationResult {
  product_id: string;
  product_name: string;
  sku?: string;
  old_price: number;
  new_price: number;
  change_pct: number;
  reason: string;
}

export interface ApplyResult {
  rules_evaluated: number;
  products_affected: number;
  price_changes: number;
}

export interface RepricingRuleListParams extends PaginationParams {
  status?: string;
  strategy?: string;
}

// === Listing Sync ===
export interface ListingSyncConfig {
  id: string;
  tenant_id: string;
  integration_id: string;
  sync_direction: "push" | "pull" | "bidirectional";
  auto_sync: boolean;
  sync_interval_minutes: number;
  field_mapping: Record<string, unknown>;
  price_rule: "same" | "markup_pct" | "markup_fixed" | "custom";
  price_modifier: number;
  stock_buffer: number;
  status: "active" | "paused" | "error";
  last_sync_at?: string;
  last_error?: string;
  created_at: string;
  updated_at: string;
  integration_provider?: string;
  integration_label?: string;
}

export interface ListingSyncLog {
  id: string;
  tenant_id: string;
  config_id: string;
  direction: "push" | "pull";
  entity_type: "product" | "price" | "stock" | "offer";
  product_id?: string;
  external_id?: string;
  status: "success" | "failed" | "skipped";
  changes?: Record<string, unknown>;
  error_message?: string;
  created_at: string;
}

export interface CreateListingSyncConfigRequest {
  integration_id: string;
  sync_direction: "push" | "pull" | "bidirectional";
  auto_sync?: boolean;
  sync_interval_minutes?: number;
  field_mapping?: Record<string, unknown>;
  price_rule?: "same" | "markup_pct" | "markup_fixed" | "custom";
  price_modifier?: number;
  stock_buffer?: number;
}

export interface UpdateListingSyncConfigRequest {
  sync_direction?: "push" | "pull" | "bidirectional";
  auto_sync?: boolean;
  sync_interval_minutes?: number;
  field_mapping?: Record<string, unknown>;
  price_rule?: "same" | "markup_pct" | "markup_fixed" | "custom";
  price_modifier?: number;
  stock_buffer?: number;
  status?: "active" | "paused" | "error";
}

export interface SyncResult {
  items_processed: number;
  items_failed: number;
  message: string;
}

export interface ListingSyncConfigListParams extends PaginationParams {
  integration_id?: string;
  status?: string;
}

export interface ListingSyncLogListParams extends PaginationParams {
  direction?: string;
  entity_type?: string;
  status?: string;
}

// === Stock Sync ===
export interface StockSyncChannel {
  id: string;
  tenant_id: string;
  integration_id?: string;
  channel_type: string;
  enabled: boolean;
  stock_buffer: number;
  sync_mode: "realtime" | "scheduled" | "manual";
  priority: number;
  last_sync_at?: string;
  last_error?: string;
  created_at: string;
  updated_at: string;
}

export interface StockSyncEvent {
  id: string;
  tenant_id: string;
  product_id?: string;
  sku?: string;
  trigger_type: string;
  old_quantity: number;
  new_quantity: number;
  available_quantity: number;
  channels_notified: number;
  channels_failed: number;
  details: Record<string, unknown>;
  created_at: string;
}

export interface StockAllocation {
  channel_id: string;
  channel_type: string;
  total_stock: number;
  reserved: number;
  buffer: number;
  available_quantity: number;
}

export interface ChannelSummary {
  id: string;
  channel_type: string;
  enabled: boolean;
  sync_mode: string;
  stock_buffer: number;
  last_sync_at?: string;
  last_error?: string;
  status: "ok" | "warning" | "error" | "disabled";
  error_count: number;
  items_synced: number;
}

export interface StockSyncDashboard {
  total_products: number;
  active_channels: number;
  recent_errors: number;
  last_sync_at?: string;
  channel_summaries: ChannelSummary[];
}

export interface CreateStockSyncChannelRequest {
  integration_id?: string;
  channel_type: string;
  enabled?: boolean;
  stock_buffer?: number;
  sync_mode?: string;
  priority?: number;
}

export interface UpdateStockSyncChannelRequest {
  enabled?: boolean;
  stock_buffer?: number;
  sync_mode?: string;
  priority?: number;
}

export interface StockSyncChannelListParams extends PaginationParams {
  enabled?: boolean;
  channel_type?: string;
}

export interface StockSyncEventListParams extends PaginationParams {
  product_id?: string;
  trigger_type?: string;
}
