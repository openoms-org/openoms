import type { Address, PaginationParams } from "./common";

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
  sync_interval_minutes: number;
  last_sync_at?: string;
  error_message?: string;
  portal_enabled: boolean;
  integration_id?: string;
  default_category_id?: string;
  created_at: string;
  updated_at: string;
}

// === Supplier Portal ===
export interface SupplierPortalPO {
  id: string;
  po_number: string;
  status: string;
  notes?: string;
  expected_delivery_date?: string;
  total_amount: number;
  currency: string;
  items?: PurchaseOrderItem[];
  created_at: string;
  updated_at: string;
}

interface SupplierPortalOrdersResponse {
  supplier: { id: string; name: string };
  orders: SupplierPortalPO[];
}

export interface SupplierMessage {
  id: string;
  tenant_id: string;
  purchase_order_id: string;
  supplier_id?: string;
  user_id?: string;
  message: string;
  is_from_supplier: boolean;
  created_at: string;
}

export interface SupplierPortalLinkResponse {
  url: string;
  expires_at: string;
}

export interface SupplierPortalStatus {
  portal_enabled: boolean;
  last_used_at?: string;
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
  source_category?: string;
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
  sync_interval_minutes?: number;
  settings?: Record<string, unknown>;
  integration_id?: string;
}

export interface UpdateSupplierRequest {
  name?: string;
  code?: string;
  feed_url?: string;
  feed_format?: string;
  sync_interval_minutes?: number;
  status?: string;
  settings?: Record<string, unknown>;
  error_message?: string;
  default_category_id?: string;
}

export interface SupplierListParams extends PaginationParams {
  status?: string;
}

export interface SupplierProductListParams extends PaginationParams {
  ean?: string;
  linked?: boolean;
  search?: string;
  category?: string;
}

export interface ImportSupplierProductsRequest {
  supplier_product_ids: string[];
}

export interface ImportSupplierProductsResponse {
  imported: number;
  skipped: number;
  errors?: { supplier_product_id: string; reason: string }[];
}

export interface BulkDeleteSupplierProductsRequest {
  supplier_product_ids: string[];
}

export interface SupplierProductWithSupplier extends SupplierProduct {
  supplier_name: string;
}

export interface SupplierProductListAllParams extends PaginationParams {
  search?: string;
  supplier_id?: string;
  category?: string;
  linked?: string;
}

// === BTP Wizard ===
export interface BTPImportProgressResponse {
  status: "pending" | "running" | "completed" | "failed";
  total: number;
  processed: number;
  error?: string;
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

export type WarehouseStockListParams = PaginationParams;

// === Inventory Settings ===
export interface InventorySettings {
  strict_mode: boolean;
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

interface WarehouseDocItem {
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

// === Purchase Orders ===
export interface PurchaseOrder {
  id: string;
  tenant_id: string;
  po_number: string;
  supplier_id?: string;
  supplier_name: string;
  warehouse_id?: string;
  status: string;
  notes?: string;
  expected_delivery_date?: string;
  total_amount: number;
  currency: string;
  created_by?: string;
  items?: PurchaseOrderItem[];
  created_at: string;
  updated_at: string;
}

export interface PurchaseOrderItem {
  id: string;
  tenant_id: string;
  purchase_order_id: string;
  product_id?: string;
  sku: string;
  product_name: string;
  quantity_ordered: number;
  quantity_received: number;
  unit_cost: number;
  total_cost: number;
  notes?: string;
  created_at: string;
}

export interface CreatePurchaseOrderItemReq {
  product_id?: string;
  sku: string;
  product_name: string;
  quantity: number;
  unit_cost: number;
  notes?: string;
}

export interface CreatePurchaseOrderRequest {
  supplier_id?: string;
  supplier_name: string;
  warehouse_id?: string;
  notes?: string;
  expected_delivery_date?: string;
  currency?: string;
  items: CreatePurchaseOrderItemReq[];
}

export interface UpdatePurchaseOrderRequest {
  supplier_id?: string;
  supplier_name?: string;
  warehouse_id?: string;
  notes?: string;
  expected_delivery_date?: string;
  currency?: string;
  items?: CreatePurchaseOrderItemReq[];
}

export interface ReceiveItemsRequest {
  items: ReceiveItemEntry[];
}

export interface ReceiveItemEntry {
  item_id: string;
  quantity_received: number;
}

export interface PurchaseOrderListParams extends PaginationParams {
  status?: string;
  supplier_id?: string;
}

// === Pick & Pack ===
export interface PickPackSession {
  id: string;
  tenant_id: string;
  session_type: "single" | "batch";
  status: "picking" | "packing" | "completed" | "cancelled";
  assigned_to?: string;
  started_at: string;
  completed_at?: string;
  notes?: string;
  created_at: string;
  updated_at: string;
  items?: PickPackItem[];
  stats?: PickPackStats;
}

export interface PickPackItem {
  id: string;
  tenant_id: string;
  session_id: string;
  order_id: string;
  product_id?: string;
  sku: string;
  product_name: string;
  quantity_required: number;
  quantity_picked: number;
  quantity_packed: number;
  pick_location?: string;
  picked_at?: string;
  packed_at?: string;
  created_at: string;
}

export interface PickPackStats {
  total_items: number;
  total_picked: number;
  total_packed: number;
  total_required: number;
  order_count: number;
  all_picked: boolean;
  all_packed: boolean;
}

export interface CreatePickPackSessionRequest {
  order_ids: string[];
  notes?: string;
}

export interface ScanItemRequest {
  barcode: string;
}

export interface ScanItemResponse {
  item: PickPackItem;
  remaining: number;
  message: string;
}

export interface MarkPackedRequest {
  quantity: number;
}

export interface PickPackSessionListParams extends PaginationParams {
  status?: string;
  assigned_to?: string;
}

// === Dropship Orders ===
export interface DropshipOrder {
  id: string;
  tenant_id: string;
  order_id: string;
  supplier_id: string;
  supplier_name: string;
  status: string;
  supplier_reference?: string;
  tracking_number?: string;
  carrier?: string;
  notes?: string;
  total_cost: number;
  currency: string;
  sent_at?: string;
  confirmed_at?: string;
  shipped_at?: string;
  delivered_at?: string;
  items?: DropshipOrderItem[];
  created_at: string;
  updated_at: string;
}

export interface DropshipOrderItem {
  id: string;
  tenant_id: string;
  dropship_order_id: string;
  product_id?: string;
  sku: string;
  product_name: string;
  quantity: number;
  unit_cost: number;
  created_at: string;
}

export interface CreateDropshipOrderRequest {
  order_id: string;
  supplier_id: string;
  notes?: string;
  currency?: string;
  items: CreateDropshipOrderItemReq[];
}

export interface CreateDropshipOrderItemReq {
  product_id?: string;
  sku: string;
  product_name: string;
  quantity: number;
  unit_cost: number;
}

export interface UpdateDropshipStatusRequest {
  status: string;
  tracking_number?: string;
  carrier?: string;
  supplier_reference?: string;
  notes?: string;
}

export interface DropshipOrderListParams extends PaginationParams {
  status?: string;
  supplier_id?: string;
  order_id?: string;
}

// === Recurring Orders (Subscriptions) ===
export interface RecurringOrder {
  id: string;
  tenant_id: string;
  customer_id?: string;
  customer_name: string;
  customer_email?: string;
  status: string;
  frequency: string;
  interval_days: number;
  next_order_date: string;
  last_order_date?: string;
  end_date?: string;
  total_orders_created: number;
  max_orders?: number;
  shipping_address?: Address;
  notes?: string;
  created_by?: string;
  created_at: string;
  updated_at: string;
  items?: RecurringOrderItem[];
}

export interface RecurringOrderItem {
  id: string;
  tenant_id: string;
  recurring_order_id: string;
  product_id?: string;
  sku: string;
  product_name: string;
  quantity: number;
  unit_price: number;
  created_at: string;
}

export interface CreateRecurringOrderItemRequest {
  product_id?: string;
  sku: string;
  product_name: string;
  quantity: number;
  unit_price: number;
}

export interface CreateRecurringOrderRequest {
  customer_id?: string;
  customer_name: string;
  customer_email?: string;
  frequency: string;
  next_order_date: string;
  end_date?: string;
  max_orders?: number;
  shipping_address?: Address;
  notes?: string;
  items: CreateRecurringOrderItemRequest[];
}

export interface UpdateRecurringOrderRequest {
  customer_id?: string;
  customer_name?: string;
  customer_email?: string;
  frequency?: string;
  next_order_date?: string;
  end_date?: string;
  max_orders?: number;
  shipping_address?: Address;
  notes?: string;
  items?: CreateRecurringOrderItemRequest[];
}

export interface RecurringOrderListParams extends PaginationParams {
  status?: string;
  customer_id?: string;
  next_date_before?: string;
}
